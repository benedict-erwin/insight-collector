package influxdb

import (
	"fmt"
	"time"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// Global client instance
var currentClient Client
var currentConfig *Config

// GetConfig returns the current InfluxDB configuration
func GetConfig() *Config {
	if currentConfig != nil {
		return currentConfig
	}

	cfg := config.Get().InfluxDB
	currentConfig = &Config{
		Token:  cfg.Token,
		Bucket: cfg.Bucket,
	}

	// Determine version from config
	if cfg.Version != "" {
		// Explicit version specified in config
		switch cfg.Version {
		case "v2-oss":
			currentConfig.Version = VersionV2OSS
		case "v3-core":
			currentConfig.Version = VersionV3Core
		default:
			logger.Warn().Str("version", cfg.Version).Msg("Unknown InfluxDB version, defaulting to v2-oss")
			currentConfig.Version = VersionV2OSS
		}
	} else {
		// Auto-detect version based on config fields
		if cfg.URL != "" || cfg.Org != "" {
			// v2-oss config detected
			currentConfig.Version = VersionV2OSS
		} else if cfg.Host != "" && cfg.AuthScheme != "" {
			// v3-core config detected
			currentConfig.Version = VersionV3Core
		} else {
			// Default to v2-oss
			currentConfig.Version = VersionV2OSS
		}
	}

	// Set version-specific fields
	switch currentConfig.Version {
	case VersionV2OSS:
		if cfg.URL != "" {
			currentConfig.URL = cfg.URL
		} else {
			// Fallback: construct URL from host:port
			currentConfig.URL = fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
		}
		if cfg.Org != "" {
			currentConfig.Org = cfg.Org
		} else {
			currentConfig.Org = "insight" // Default org
		}
	case VersionV3Core:
		currentConfig.Host = cfg.Host
		currentConfig.Port = cfg.Port
		currentConfig.AuthScheme = cfg.AuthScheme
	}

	return currentConfig
}

// NewPoint creates a new Point using the active client implementation
// Implementation is in factory.go to avoid import cycles
func NewPoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) interface{} {
	return createNewPoint(measurement, tags, fields, timestamp)
}

// Init initializes the InfluxDB client based on configuration
func Init() error {
	cfg := GetConfig()
	
	logger.Info().
		Str("version", string(cfg.Version)).
		Msg("Initializing InfluxDB client")
	
	switch cfg.Version {
	case VersionV2OSS:
		// Import v2-oss dynamically
		v2oss := getV2OSSClient()
		currentClient = v2oss
	case VersionV3Core:
		// Import v3-core dynamically  
		v3core := getV3CoreClient()
		currentClient = v3core
	default:
		// Default to v2-oss
		v2oss := getV2OSSClient()
		currentClient = v2oss
		logger.Warn().Msg("Unknown InfluxDB version, defaulting to v2-oss")
	}
	
	return currentClient.Init()
}

// WritePoint writes a single point to InfluxDB
func WritePoint(point interface{}) error {
	if currentClient == nil {
		logger.Error().Msg("InfluxDB client not initialized")
		return fmt.Errorf("InfluxDB client not initialized")
	}
	return currentClient.WritePoint(point)
}

// WritePoints writes multiple points to InfluxDB in batch
func WritePoints(points []interface{}) error {
	if currentClient == nil {
		logger.Error().Msg("InfluxDB client not initialized")
		return fmt.Errorf("InfluxDB client not initialized")
	}
	return currentClient.WritePoints(points)
}

// Query executes a query and returns results as an iterator
func Query(query string) (QueryIterator, error) {
	if currentClient == nil {
		logger.Error().Msg("InfluxDB client not initialized")
		return nil, fmt.Errorf("InfluxDB client not initialized")
	}
	result, err := currentClient.Query(query)
	if err != nil {
		return nil, err
	}
	// Type assert to QueryIterator interface
	if qi, ok := result.(QueryIterator); ok {
		return qi, nil
	}
	return nil, fmt.Errorf("invalid query iterator type")
}

// QueryWithOptions executes a query with additional options
func QueryWithOptions(query string, options ...interface{}) (QueryIterator, error) {
	if currentClient == nil {
		logger.Error().Msg("InfluxDB client not initialized")
		return nil, fmt.Errorf("InfluxDB client not initialized")
	}
	result, err := currentClient.QueryWithOptions(query, options...)
	if err != nil {
		return nil, err
	}
	// Type assert to QueryIterator interface
	if qi, ok := result.(QueryIterator); ok {
		return qi, nil
	}
	return nil, fmt.Errorf("invalid query iterator type")
}

// GetClient returns the underlying InfluxDB client
func GetClient() interface{} {
	if currentClient == nil {
		return nil
	}
	return currentClient.GetClient()
}

// GetCurrentClient returns the current InfluxDB client instance
func GetCurrentClient() Client {
	return currentClient
}

// GetV2OSSClient returns the current client as v2-oss client (type assertion)
func GetV2OSSClient() interface{} {
	return currentClient
}

// IsHealthy checks if client is initialized and ready
func IsHealthy() bool {
	if currentClient == nil {
		return false
	}
	return currentClient.IsHealthy()
}

// HealthCheck performs connectivity test
func HealthCheck() error {
	if currentClient == nil {
		return fmt.Errorf("InfluxDB client not initialized")
	}
	return currentClient.HealthCheck()
}

// Close shuts down the InfluxDB client
func Close() {
	if currentClient != nil {
		currentClient.Close()
		currentClient = nil
	}
	currentConfig = nil
}

// Dynamic client creation functions to avoid import cycles
func getV2OSSClient() Client {
	// This will be implemented in a separate file to avoid import cycles
	return createV2OSSClient()
}

func getV3CoreClient() Client {
	// This will be implemented in a separate file to avoid import cycles
	return createV3CoreClient()
}
