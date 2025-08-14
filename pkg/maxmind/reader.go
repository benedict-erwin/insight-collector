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
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/oschwald/geoip2-golang/v2"
)

// cacheEntry wraps cached data with expiration time
type cacheEntry[T any] struct {
	Data      T
	ExpiresAt time.Time
}

// isExpired checks if cache entry is expired
func (e *cacheEntry[T]) isExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// SafeGeoIPReader provides thread-safe access to GeoIP databases with LRU caching
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
	
	// LRU caches for performance optimization
	cityCache *lru.Cache[string, *cacheEntry[*GeoLocation]]
	asnCache  *lru.Cache[string, *cacheEntry[*ASNInfo]]
	cacheTTL  time.Duration
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
	
	// Initialize LRU cache if enabled
	if err := reader.initCaches(); err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize MaxMind caches, continuing without cache")
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

// initCaches initializes LRU caches if cache is enabled
func (r *SafeGeoIPReader) initCaches() error {
	if !r.config.Cache.Enabled {
		logger.Debug().Msg("MaxMind cache disabled")
		return nil
	}
	
	// Parse TTL
	ttl, err := time.ParseDuration(r.config.Cache.TTL)
	if err != nil {
		logger.Warn().Err(err).Str("ttl", r.config.Cache.TTL).Msg("Invalid cache TTL, disabling cache")
		return err
	}
	r.cacheTTL = ttl
	
	// Initialize City cache
	cityCache, err := lru.New[string, *cacheEntry[*GeoLocation]](r.config.Cache.MaxEntries)
	if err != nil {
		return fmt.Errorf("failed to create city cache: %w", err)
	}
	r.cityCache = cityCache
	
	// Initialize ASN cache
	asnCache, err := lru.New[string, *cacheEntry[*ASNInfo]](r.config.Cache.MaxEntries)
	if err != nil {
		return fmt.Errorf("failed to create ASN cache: %w", err)
	}
	r.asnCache = asnCache
	
	logger.Info().
		Bool("enabled", true).
		Int("max_entries", r.config.Cache.MaxEntries).
		Dur("ttl", ttl).
		Msg("MaxMind LRU cache initialized")
		
	return nil
}

// clearCaches clears all LRU caches (called during database reload)
func (r *SafeGeoIPReader) clearCaches() {
	if r.cityCache != nil {
		r.cityCache.Purge()
		logger.Debug().Msg("MaxMind city cache cleared")
	}
	if r.asnCache != nil {
		r.asnCache.Purge()
		logger.Debug().Msg("MaxMind ASN cache cleared")
	}
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

	// Clear caches when databases are reloaded
	r.clearCaches()

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

// LookupCity performs city lookup with LRU cache and safe fallback
func (r *SafeGeoIPReader) LookupCity(ip net.IP) *GeoLocation {
	if !r.config.Enabled {
		return DefaultGeoLocation(ip)
	}
	
	ipStr := ip.String()
	
	// Check cache first if enabled
	if r.cityCache != nil {
		if cached, found := r.cityCache.Get(ipStr); found && !cached.isExpired() {
			return cached.Data
		}
	}
	
	// Cache miss or expired - perform database lookup
	result := r.performCityDBLookup(ip)
	
	// Cache the result if cache is enabled
	if r.cityCache != nil && result != nil {
		entry := &cacheEntry[*GeoLocation]{
			Data:      result,
			ExpiresAt: time.Now().Add(r.cacheTTL),
		}
		r.cityCache.Add(ipStr, entry)
	}
	
	return result
}

// performCityDBLookup performs actual database lookup (extracted from original LookupCity)
func (r *SafeGeoIPReader) performCityDBLookup(ip net.IP) *GeoLocation {
	result := DefaultGeoLocation(ip)

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

// LookupASN performs ASN lookup with LRU cache and safe fallback
func (r *SafeGeoIPReader) LookupASN(ip net.IP) *ASNInfo {
	if !r.config.Enabled {
		return DefaultASNInfo(ip)
	}
	
	ipStr := ip.String()
	
	// Check cache first if enabled
	if r.asnCache != nil {
		if cached, found := r.asnCache.Get(ipStr); found && !cached.isExpired() {
			return cached.Data
		}
	}
	
	// Cache miss or expired - perform database lookup
	result := r.performASNDBLookup(ip)
	
	// Cache the result if cache is enabled
	if r.asnCache != nil && result != nil {
		entry := &cacheEntry[*ASNInfo]{
			Data:      result,
			ExpiresAt: time.Now().Add(r.cacheTTL),
		}
		r.asnCache.Add(ipStr, entry)
	}
	
	return result
}

// performASNDBLookup performs actual database lookup (extracted from original LookupASN)
func (r *SafeGeoIPReader) performASNDBLookup(ip net.IP) *ASNInfo {
	result := DefaultASNInfo(ip)

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
	
	// Clear caches on close
	if r.cityCache != nil {
		r.cityCache.Purge()
		r.cityCache = nil
	}
	if r.asnCache != nil {
		r.asnCache.Purge()
		r.asnCache = nil
	}

	logger.Debug().Msg("MaxMind GeoIP reader closed")
	return nil
}
