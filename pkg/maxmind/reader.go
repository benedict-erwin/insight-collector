package maxmind

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
	"github.com/oschwald/geoip2-golang/v2"
)

// SafeGeoIPReader provides thread-safe access to GeoIP databases
type SafeGeoIPReader struct {
	mu              sync.RWMutex
	cityReader      *geoip2.Reader
	asnReader       *geoip2.Reader
	config          *Config
	lastCityModTime time.Time
	lastASNModTime  time.Time
	dbInfo          *DatabaseInfo
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// NewSafeGeoIPReader creates a new thread-safe GeoIP reader
func NewSafeGeoIPReader(config *Config) (*SafeGeoIPReader, error) {
	reader := &SafeGeoIPReader{
		config: config,
		stopCh: make(chan struct{}),
		dbInfo: &DatabaseInfo{
			Enabled: config.Enabled,
		},
	}

	if !config.Enabled {
		logger.Info().Msg("MaxMind GeoIP service is disabled")
		return reader, nil
	}

	// Initial database load
	if err := reader.loadDatabases(); err != nil {
		logger.Warn().Err(err).Msg("Failed to load GeoIP databases on startup, will use fallback")
		// Don't return error - service should continue with fallback
	}

	// Start periodic checker
	reader.startPeriodicChecker()

	return reader, nil
}

// loadDatabases loads or reloads GeoIP databases
func (r *SafeGeoIPReader) loadDatabases() error {
	cityDBPath := filepath.Join(r.config.StoragePath, r.config.Databases.City+".mmdb")
	asnDBPath := filepath.Join(r.config.StoragePath, r.config.Databases.ASN+".mmdb")

	var newCityReader, newASNReader *geoip2.Reader
	var err error

	// Load City database
	if cityInfo, statErr := os.Stat(cityDBPath); statErr == nil {
		if newCityReader, err = geoip2.Open(cityDBPath); err != nil {
			logger.Error().Err(err).Str("path", cityDBPath).Msg("Failed to load City database")
		} else {
			logger.Debug().Str("path", cityDBPath).Msg("City database loaded successfully")
		}
		r.lastCityModTime = cityInfo.ModTime()
	} else {
		logger.Warn().Err(statErr).Str("path", cityDBPath).Msg("City database file not found")
	}

	// Load ASN database
	if asnInfo, statErr := os.Stat(asnDBPath); statErr == nil {
		if newASNReader, err = geoip2.Open(asnDBPath); err != nil {
			logger.Error().Err(err).Str("path", asnDBPath).Msg("Failed to load ASN database")
		} else {
			logger.Debug().Str("path", asnDBPath).Msg("ASN database loaded successfully")
		}
		r.lastASNModTime = asnInfo.ModTime()
	} else {
		logger.Warn().Err(statErr).Str("path", asnDBPath).Msg("ASN database file not found")
	}

	// Atomic swap with write lock
	r.mu.Lock()
	oldCityReader := r.cityReader
	oldASNReader := r.asnReader

	r.cityReader = newCityReader
	r.asnReader = newASNReader

	// Update database info
	r.updateDatabaseInfo(cityDBPath, asnDBPath)

	r.mu.Unlock()

	// Close old readers after grace period for in-flight requests
	if oldCityReader != nil {
		time.AfterFunc(5*time.Second, func() {
			oldCityReader.Close()
		})
	}
	if oldASNReader != nil {
		time.AfterFunc(5*time.Second, func() {
			oldASNReader.Close()
		})
	}

	// Update metadata
	updateMetadata(r.config.StoragePath, cityDBPath, asnDBPath)

	logger.Debug().
		Bool("city_loaded", newCityReader != nil).
		Bool("asn_loaded", newASNReader != nil).
		Msg("GeoIP databases loaded")

	return nil
}

// updateDatabaseInfo updates internal database information
func (r *SafeGeoIPReader) updateDatabaseInfo(cityDBPath, asnDBPath string) {
	r.dbInfo.CityDBPath = cityDBPath
	r.dbInfo.ASNDBPath = asnDBPath
	r.dbInfo.LoadedAt = utils.Now()
	r.dbInfo.ReloadCount++

	if cityInfo, err := os.Stat(cityDBPath); err == nil {
		r.dbInfo.CityDBSize = cityInfo.Size()
		r.dbInfo.CityDBModTime = cityInfo.ModTime()
	}

	if asnInfo, err := os.Stat(asnDBPath); err == nil {
		r.dbInfo.ASNDBSize = asnInfo.Size()
		r.dbInfo.ASNDBModTime = asnInfo.ModTime()
	}
}

// needsReload checks if databases need to be reloaded
func (r *SafeGeoIPReader) needsReload() bool {
	cityDBPath := filepath.Join(r.config.StoragePath, r.config.Databases.City+".mmdb")
	asnDBPath := filepath.Join(r.config.StoragePath, r.config.Databases.ASN+".mmdb")

	// Check City database
	if cityInfo, err := os.Stat(cityDBPath); err == nil {
		if cityInfo.ModTime().After(r.lastCityModTime) {
			return true
		}
	}

	// Check ASN database
	if asnInfo, err := os.Stat(asnDBPath); err == nil {
		if asnInfo.ModTime().After(r.lastASNModTime) {
			return true
		}
	}

	return false
}

// startPeriodicChecker starts the periodic database checker
func (r *SafeGeoIPReader) startPeriodicChecker() {
	checkInterval, err := time.ParseDuration(r.config.CheckInterval)
	if err != nil {
		checkInterval = 5 * time.Minute // Default fallback
		logger.Warn().Err(err).Dur("fallback", checkInterval).Msg("Invalid check interval, using fallback")
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		logger.Debug().Dur("interval", checkInterval).Msg("Started MaxMind database periodic checker")

		for {
			select {
			case <-ticker.C:
				if r.needsReload() {
					logger.Info().Msg("Database files changed, reloading...")
					if err := r.loadDatabases(); err != nil {
						logger.Error().Err(err).Msg("Failed to reload databases")
					}
				}
			case <-r.stopCh:
				logger.Debug().Msg("Stopped MaxMind database periodic checker")
				return
			}
		}
	}()
}

// LookupCity performs city lookup with safe fallback
func (r *SafeGeoIPReader) LookupCity(ip net.IP) *GeoLocation {
	result := DefaultGeoLocation(ip)

	if !r.config.Enabled {
		return result
	}

	r.mu.RLock()
	reader := r.cityReader
	r.mu.RUnlock()

	if reader == nil {
		logger.Debug().Str("ip", ip.String()).Msg("City database unavailable, using defaults")
		return result
	}

	// Convert net.IP to netip.Addr for v2 API
	addr, err := netip.ParseAddr(ip.String())
	if err != nil {
		logger.Debug().Err(err).Str("ip", ip.String()).Msg("Invalid IP address format")
		return result
	}

	record, err := reader.City(addr)
	if err != nil {
		logger.Debug().Err(err).Str("ip", ip.String()).Msg("City lookup failed, using defaults")
		return result
	}

	// Populate available fields with safe access to pointers
	if record.Country.Names.English != "" {
		result.Country = record.Country.Names.English
	}
	result.CountryCode = record.Country.ISOCode
	if record.City.Names.English != "" {
		result.City = record.City.Names.English
	}
	if len(record.Subdivisions) > 0 && record.Subdivisions[0].Names.English != "" {
		result.Region = record.Subdivisions[0].Names.English
	}
	result.PostalCode = record.Postal.Code

	// Handle pointer values safely
	if record.Location.Latitude != nil {
		result.Latitude = *record.Location.Latitude
	}
	if record.Location.Longitude != nil {
		result.Longitude = *record.Location.Longitude
	}
	result.Timezone = record.Location.TimeZone
	result.LookedUpAt = utils.Now()

	return result
}

// LookupASN performs ASN lookup with safe fallback
func (r *SafeGeoIPReader) LookupASN(ip net.IP) *ASNInfo {
	result := DefaultASNInfo(ip)

	if !r.config.Enabled {
		return result
	}

	r.mu.RLock()
	reader := r.asnReader
	r.mu.RUnlock()

	if reader == nil {
		logger.Debug().Str("ip", ip.String()).Msg("ASN database unavailable, using defaults")
		return result
	}

	// Convert net.IP to netip.Addr for v2 API
	addr, err := netip.ParseAddr(ip.String())
	if err != nil {
		logger.Debug().Err(err).Str("ip", ip.String()).Msg("Invalid IP address format")
		return result
	}

	record, err := reader.ASN(addr)
	if err != nil {
		logger.Debug().Err(err).Str("ip", ip.String()).Msg("ASN lookup failed, using defaults")
		return result
	}

	// Populate available fields
	result.ASN = record.AutonomousSystemNumber
	result.Organization = record.AutonomousSystemOrganization
	result.LookedUpAt = utils.Now()

	return result
}

// GetDatabaseInfo returns current database information
func (r *SafeGeoIPReader) GetDatabaseInfo() *DatabaseInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return copy to prevent external modification
	info := *r.dbInfo
	return &info
}

// ReloadDatabases manually triggers database reload
func (r *SafeGeoIPReader) ReloadDatabases() error {
	if !r.config.Enabled {
		return nil
	}

	logger.Info().Msg("Manual database reload triggered")
	return r.loadDatabases()
}

// Health checks if the service is healthy
func (r *SafeGeoIPReader) Health() error {
	if !r.config.Enabled {
		return nil // Service disabled is not an error
	}

	r.mu.RLock()
	cityReaderOK := r.cityReader != nil
	asnReaderOK := r.asnReader != nil
	r.mu.RUnlock()

	if !cityReaderOK && !asnReaderOK {
		return fmt.Errorf("no GeoIP databases loaded")
	}

	return nil
}

// Close gracefully shuts down the reader
func (r *SafeGeoIPReader) Close() error {
	close(r.stopCh)
	r.wg.Wait()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cityReader != nil {
		r.cityReader.Close()
		r.cityReader = nil
	}

	if r.asnReader != nil {
		r.asnReader.Close()
		r.asnReader = nil
	}

	logger.Debug().Msg("MaxMind GeoIP reader closed")
	return nil
}
