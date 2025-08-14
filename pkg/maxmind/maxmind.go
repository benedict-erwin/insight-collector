package maxmind

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

var (
	service    GeoIPService
	downloader *DatabaseDownloader
	mu         sync.RWMutex
	once       sync.Once
)

// Init initializes the MaxMind GeoIP service with full features
func Init() error {
	return initWithOptions(true)
}

// InitForCLI initializes MaxMind service for CLI commands (no periodic downloader)
func InitForCLI() error {
	return initWithOptions(false)
}

// initWithOptions initializes the MaxMind GeoIP service with configurable options
func initWithOptions(enablePeriodicDownloader bool) error {
	var initErr error
	
	once.Do(func() {
		cfg := config.Get()
		
		// Check if MaxMind configuration exists
		maxmindConfig := getMaxMindConfig(cfg)
		if maxmindConfig == nil {
			logger.Info().Msg("MaxMind configuration not found, service will be disabled")
			service = &DisabledService{}
			return
		}
		
		// Validate configuration
		if err := validateConfig(maxmindConfig); err != nil {
			initErr = fmt.Errorf("invalid MaxMind configuration: %w", err)
			return
		}
		
		// Create storage directory if it doesn't exist
		if maxmindConfig.Enabled {
			if err := os.MkdirAll(maxmindConfig.StoragePath, 0755); err != nil {
				initErr = fmt.Errorf("failed to create storage directory: %w", err)
				return
			}
		}
		
		// Initialize service
		reader, err := NewSafeGeoIPReader(maxmindConfig)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize MaxMind service: %w", err)
			return
		}
		
		// Store service reference
		mu.Lock()
		service = reader
		mu.Unlock()
		
		// Initialize downloader if enabled and not CLI mode
		if maxmindConfig.Downloader.Enabled && enablePeriodicDownloader {
			downloaderInstance, err := NewDatabaseDownloader(maxmindConfig)
			if err != nil {
				initErr = fmt.Errorf("failed to initialize downloader: %w", err)
				return
			}
			
			downloader = downloaderInstance
			
			// Start periodic checking
			if err := downloader.StartPeriodicCheck(); err != nil {
				initErr = fmt.Errorf("failed to start periodic downloader: %w", err)
				return
			}
			
			logger.Info().
				Bool("downloader_enabled", true).
				Str("account_id", maxmindConfig.Downloader.AccountID[:2]+"****").
				Msg("MaxMind downloader initialized")
		} else if maxmindConfig.Downloader.Enabled {
			// Initialize downloader without periodic checker (for CLI manual operations)
			downloaderInstance, err := NewDatabaseDownloader(maxmindConfig)
			if err != nil {
				logger.Debug().Err(err).Msg("Failed to initialize downloader for CLI mode")
			} else {
				downloader = downloaderInstance
				logger.Debug().Msg("MaxMind downloader initialized for CLI mode (no periodic checker)")
			}
		}
		
		if enablePeriodicDownloader {
			logger.Info().
				Bool("enabled", maxmindConfig.Enabled).
				Bool("downloader_enabled", maxmindConfig.Downloader.Enabled).
				Str("storage_path", maxmindConfig.StoragePath).
				Str("check_interval", maxmindConfig.CheckInterval).
				Msg("MaxMind GeoIP service initialized")
		} else {
			logger.Debug().
				Bool("enabled", maxmindConfig.Enabled).
				Str("storage_path", maxmindConfig.StoragePath).
				Msg("MaxMind GeoIP service initialized for CLI mode")
		}
	})
	
	return initErr
}

// getMaxMindConfig extracts MaxMind configuration from app config
func getMaxMindConfig(cfg *config.Config) *Config {
	maxmindCfg := &Config{
		Enabled:       cfg.MaxMind.Enabled,
		StoragePath:   cfg.MaxMind.StoragePath,
		CheckInterval: cfg.MaxMind.CheckInterval,
		Databases: struct {
			City string `json:"city"`
			ASN  string `json:"asn"`
		}{
			City: cfg.MaxMind.Databases.City,
			ASN:  cfg.MaxMind.Databases.ASN,
		},
		Downloader: DownloaderConfig{
			Enabled:       cfg.MaxMind.Downloader.Enabled,
			AccountID:     cfg.MaxMind.Downloader.AccountID,
			LicenseKey:    cfg.MaxMind.Downloader.LicenseKey,
			BaseURL:       cfg.MaxMind.Downloader.BaseURL,
			Timeout:       cfg.MaxMind.Downloader.Timeout,
			RetryAttempts: cfg.MaxMind.Downloader.RetryAttempts,
			RetryDelay:    cfg.MaxMind.Downloader.RetryDelay,
		},
		Cache: CacheConfig{
			Enabled:    cfg.MaxMind.Cache.Enabled,
			MaxEntries: cfg.MaxMind.Cache.MaxEntries,
			TTL:        cfg.MaxMind.Cache.TTL,
		},
	}
	
	// Apply defaults if configuration is missing
	if maxmindCfg.StoragePath == "" {
		maxmindCfg.StoragePath = "storage/maxmind"
	}
	if maxmindCfg.CheckInterval == "" {
		maxmindCfg.CheckInterval = "1h"
	}
	if maxmindCfg.Databases.City == "" {
		maxmindCfg.Databases.City = "GeoLite2-City"
	}
	if maxmindCfg.Databases.ASN == "" {
		maxmindCfg.Databases.ASN = "GeoLite2-ASN"
	}
	
	// Apply downloader defaults
	if maxmindCfg.Downloader.BaseURL == "" {
		maxmindCfg.Downloader.BaseURL = "https://download.maxmind.com/geoip/databases"
	}
	if maxmindCfg.Downloader.Timeout == "" {
		maxmindCfg.Downloader.Timeout = "30s"
	}
	if maxmindCfg.Downloader.RetryAttempts == 0 {
		maxmindCfg.Downloader.RetryAttempts = 3
	}
	if maxmindCfg.Downloader.RetryDelay == "" {
		maxmindCfg.Downloader.RetryDelay = "5s"
	}
	
	// Apply cache defaults
	if maxmindCfg.Cache.MaxEntries == 0 {
		maxmindCfg.Cache.MaxEntries = 10000 // Default to 10,000 entries
	}
	if maxmindCfg.Cache.TTL == "" {
		maxmindCfg.Cache.TTL = "1h" // Default 1 hour TTL
	}
	
	return maxmindCfg
}

// validateConfig validates MaxMind configuration
func validateConfig(cfg *Config) error {
	if cfg.StoragePath == "" {
		return fmt.Errorf("storage_path cannot be empty")
	}
	
	if cfg.CheckInterval == "" {
		cfg.CheckInterval = "5m" // Default fallback
	}
	
	if cfg.Databases.City == "" {
		return fmt.Errorf("city database filename cannot be empty")
	}
	
	if cfg.Databases.ASN == "" {
		return fmt.Errorf("asn database filename cannot be empty")
	}
	
	return nil
}

// GetService returns the MaxMind service instance
func GetService() GeoIPService {
	mu.RLock()
	defer mu.RUnlock()
	return service
}

// LookupCity performs city lookup using the service
func LookupCity(ip net.IP) *GeoLocation {
	service := GetService()
	if service == nil {
		return DefaultGeoLocation(ip)
	}
	return service.LookupCity(ip)
}

// LookupASN performs ASN lookup using the service
func LookupASN(ip net.IP) *ASNInfo {
	service := GetService()
	if service == nil {
		return DefaultASNInfo(ip)
	}
	return service.LookupASN(ip)
}

// GetDatabaseInfo returns database information
func GetDatabaseInfo() *DatabaseInfo {
	service := GetService()
	if service == nil {
		return &DatabaseInfo{Enabled: false}
	}
	return service.GetDatabaseInfo()
}

// ReloadDatabases manually triggers database reload
func ReloadDatabases() error {
	service := GetService()
	if service == nil {
		return fmt.Errorf("MaxMind service not initialized")
	}
	return service.ReloadDatabases()
}

// Health checks service health
func Health() error {
	service := GetService()
	if service == nil {
		return fmt.Errorf("MaxMind service not initialized")
	}
	return service.Health()
}

// Close gracefully shuts down the service
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	
	// Stop downloader first
	if downloader != nil {
		downloader.Stop()
		downloader = nil
	}
	
	// Close service
	if service != nil {
		err := service.Close()
		service = nil
		logger.Debug().Msg("MaxMind GeoIP service closed")
		return err
	}
	
	return nil
}

// DisabledService implements GeoIPService for when MaxMind is disabled
type DisabledService struct{}

func (d *DisabledService) LookupCity(ip net.IP) *GeoLocation {
	return DefaultGeoLocation(ip)
}

func (d *DisabledService) LookupASN(ip net.IP) *ASNInfo {
	return DefaultASNInfo(ip)
}

func (d *DisabledService) GetDatabaseInfo() *DatabaseInfo {
	return &DatabaseInfo{Enabled: false}
}

func (d *DisabledService) ReloadDatabases() error {
	return nil // No-op for disabled service
}

func (d *DisabledService) Health() error {
	return nil // Disabled service is always "healthy"
}

func (d *DisabledService) Close() error {
	return nil // No-op for disabled service
}

// Helper functions for common IP parsing

// LookupCityFromString parses IP string and performs city lookup
func LookupCityFromString(ipStr string) *GeoLocation {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		result := DefaultGeoLocation(nil)
		result.IP = ipStr
		return result
	}
	return LookupCity(ip)
}

// LookupASNFromString parses IP string and performs ASN lookup
func LookupASNFromString(ipStr string) *ASNInfo {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		result := DefaultASNInfo(nil)
		result.IP = ipStr
		return result
	}
	return LookupASN(ip)
}

// Downloader access functions

// CheckForUpdates manually triggers database update check
func CheckForUpdates() error {
	mu.RLock()
	d := downloader
	mu.RUnlock()
	
	if d == nil {
		return fmt.Errorf("MaxMind downloader not initialized")
	}
	return d.CheckForUpdates()
}

// ForceDownload forces download of specific database
func ForceDownload(dbType string) error {
	mu.RLock()
	d := downloader
	mu.RUnlock()
	
	if d == nil {
		return fmt.Errorf("MaxMind downloader not initialized")
	}
	return d.ForceDownload(dbType)
}

// GetDownloadStatus returns current download status
func GetDownloadStatus() map[string]interface{} {
	mu.RLock()
	d := downloader
	mu.RUnlock()
	
	if d == nil {
		return map[string]interface{}{
			"enabled": false,
			"message": "Downloader not initialized",
		}
	}
	return d.GetDownloadStatus()
}