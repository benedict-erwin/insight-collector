package maxmind

import (
	"fmt"
	"os"

	"github.com/benedict-erwin/insight-collector/config"
)

// InitMinimalForCLI initializes only MaxMind service without other dependencies
func InitMinimalForCLI() error {
	// Initialize config silently if not already done
	if config.Get() == nil {
		// Set environment variable to suppress logs during init
		os.Setenv("CLI_QUIET_MODE", "true")
		defer os.Unsetenv("CLI_QUIET_MODE")
		
		if err := config.Init(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}
	}

	// Initialize MaxMind service without periodic downloader
	return InitForCLI()
}

// GetDatabaseInfoCLI gets database info without full service initialization
func GetDatabaseInfoCLI() (*DatabaseInfo, error) {
	// Check if service is already initialized
	service := GetService()
	if service != nil {
		return service.GetDatabaseInfo(), nil
	}

	// If not initialized, initialize minimally
	if err := InitMinimalForCLI(); err != nil {
		return nil, err
	}

	service = GetService()
	if service == nil {
		return &DatabaseInfo{Enabled: false}, nil
	}

	return service.GetDatabaseInfo(), nil
}

// GetDownloadStatusCLI gets download status without full service initialization
func GetDownloadStatusCLI() map[string]interface{} {
	// Check if service is already initialized
	status := GetDownloadStatus()
	if enabled, ok := status["enabled"].(bool); ok && enabled {
		return status
	}

	// If not initialized, initialize minimally
	if err := InitMinimalForCLI(); err != nil {
		return map[string]interface{}{
			"enabled": false,
			"error":   err.Error(),
		}
	}

	return GetDownloadStatus()
}

// CheckDatabaseFiles checks if database files exist without service initialization
func CheckDatabaseFiles() map[string]interface{} {
	cfg := config.Get()
	if cfg == nil {
		return map[string]interface{}{
			"error": "Configuration not loaded",
		}
	}

	maxmindConfig := getMaxMindConfig(cfg)

	result := map[string]interface{}{
		"storage_path": maxmindConfig.StoragePath,
		"databases":    make(map[string]interface{}),
	}

	databases := result["databases"].(map[string]interface{})

	// Check City database
	if maxmindConfig.Databases.City != "" {
		cityPath := maxmindConfig.StoragePath + "/" + maxmindConfig.Databases.City + ".mmdb"
		databases["city"] = checkSingleDatabaseFile(cityPath, maxmindConfig.Databases.City)
	}

	// Check ASN database
	if maxmindConfig.Databases.ASN != "" {
		asnPath := maxmindConfig.StoragePath + "/" + maxmindConfig.Databases.ASN + ".mmdb"
		databases["asn"] = checkSingleDatabaseFile(asnPath, maxmindConfig.Databases.ASN)
	}

	return result
}

// checkSingleDatabaseFile checks a single database file status
func checkSingleDatabaseFile(filePath, name string) map[string]interface{} {
	info, err := os.Stat(filePath)
	if err != nil {
		return map[string]interface{}{
			"name":   name,
			"exists": false,
			"error":  err.Error(),
		}
	}

	return map[string]interface{}{
		"name":     name,
		"exists":   true,
		"size":     info.Size(),
		"mod_time": info.ModTime(),
		"path":     filePath,
	}
}
