package maxmind

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// DatabaseDownloader handles MaxMind database downloads
type DatabaseDownloader struct {
	client      *MaxMindClient
	config      *Config
	storagePath string
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewDatabaseDownloader creates a new database downloader
func NewDatabaseDownloader(config *Config) (*DatabaseDownloader, error) {
	if !config.Downloader.Enabled {
		return &DatabaseDownloader{
			config: config,
			stopCh: make(chan struct{}),
		}, nil
	}

	client, err := NewMaxMindClient(&config.Downloader)
	if err != nil {
		return nil, fmt.Errorf("failed to create MaxMind client: %w", err)
	}

	return &DatabaseDownloader{
		client:      client,
		config:      config,
		storagePath: config.StoragePath,
		stopCh:      make(chan struct{}),
	}, nil
}

// NewDatabaseDownloaderForCLI creates a downloader for CLI operations (no periodic checker)
func NewDatabaseDownloaderForCLI(config *Config) (*DatabaseDownloader, error) {
	return NewDatabaseDownloader(config)
}

// StartPeriodicCheck starts periodic database update checking
func (d *DatabaseDownloader) StartPeriodicCheck() error {
	if !d.config.Downloader.Enabled {
		logger.Info().Msg("MaxMind downloader is disabled, skipping periodic checks")
		return nil
	}

	checkInterval, err := time.ParseDuration(d.config.CheckInterval)
	if err != nil {
		return fmt.Errorf("invalid check interval: %w", err)
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		log := logger.WithScope("maxmind-scheduler")
		log.Info().
			Dur("interval", checkInterval).
			Bool("enabled", d.config.Downloader.Enabled).
			Msg("Started MaxMind database periodic checker")

		// Skip initial check to avoid auto-download on CLI commands
		// Initial download should be done manually via CLI or triggered by periodic timer

		for {
			select {
			case <-ticker.C:
				if err := d.CheckForUpdates(); err != nil {
					log.Error().Err(err).Msg("Periodic database check failed")
				}
			case <-d.stopCh:
				log.Debug().Msg("Stopped MaxMind database periodic checker")
				return
			}
		}
	}()

	return nil
}

// CheckForUpdates checks all configured databases for updates
func (d *DatabaseDownloader) CheckForUpdates() error {
	if !d.config.Downloader.Enabled {
		return nil
	}

	log := logger.WithScope("maxmind-checker")
	log.Debug().Msg("Checking for database updates")

	// Check City database
	if d.config.Databases.City != "" {
		if err := d.DownloadIfNeeded("city", d.config.Databases.City); err != nil {
			log.Error().
				Err(err).
				Str("database", "city").
				Str("name", d.config.Databases.City).
				Msg("Failed to update city database")
		}
	}

	// Check ASN database
	if d.config.Databases.ASN != "" {
		if err := d.DownloadIfNeeded("asn", d.config.Databases.ASN); err != nil {
			log.Error().
				Err(err).
				Str("database", "asn").
				Str("name", d.config.Databases.ASN).
				Msg("Failed to update ASN database")
		}
	}

	return nil
}

// DownloadIfNeeded checks and downloads database if needed
func (d *DatabaseDownloader) DownloadIfNeeded(dbType, dbName string) error {
	log := logger.WithScope("maxmind-downloader")

	// 1. HEAD request to check Last-Modified
	updateInfo, err := d.client.CheckLastModified(dbName)
	if err != nil {
		return fmt.Errorf("failed to check updates for %s: %w", dbName, err)
	}

	// 2. Get local file modification time
	localFilePath := filepath.Join(d.storagePath, dbName+".mmdb")
	localModTime, err := GetFileModTime(localFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to get local file info: %w", err)
	}

	updateInfo.LocalModTime = localModTime
	updateInfo.NeedsUpdate = updateInfo.RemoteModTime.After(localModTime)

	if !updateInfo.NeedsUpdate {
		log.Debug().
			Str("database", dbName).
			Time("remote", updateInfo.RemoteModTime).
			Time("local", localModTime).
			Msg("Database is up to date, skipping download")
		return nil
	}

	log.Info().
		Str("database", dbName).
		Time("remote", updateInfo.RemoteModTime).
		Time("local", localModTime).
		Int64("size", updateInfo.ContentLength).
		Msg("Database update available, starting download")

	// 3. Download SHA256 checksum
	expectedHash, err := d.client.DownloadChecksum(dbName)
	if err != nil {
		return fmt.Errorf("failed to download checksum for %s: %w", dbName, err)
	}

	// 4. Download database tar.gz to temporary file
	tempTarPath := filepath.Join(d.storagePath, fmt.Sprintf("%s_temp_%d.tar.gz", dbName, utils.Now().Unix()))
	tempFile, err := os.Create(tempTarPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	downloadErr := d.client.DownloadDatabase(dbName, tempFile)
	tempFile.Close()

	if downloadErr != nil {
		CleanupTempFiles(tempTarPath)
		return fmt.Errorf("failed to download database %s: %w", dbName, downloadErr)
	}

	// 5. Verify SHA256 checksum
	if err := VerifyChecksum(tempTarPath, expectedHash); err != nil {
		CleanupTempFiles(tempTarPath)
		return fmt.Errorf("checksum verification failed for %s: %w", dbName, err)
	}

	// 6. Extract .mmdb file
	extractedPath, err := ExtractTarGz(tempTarPath, d.storagePath)
	if err != nil {
		CleanupTempFiles(tempTarPath)
		return fmt.Errorf("extraction failed for %s: %w", dbName, err)
	}

	// 7. Rename extracted file to expected name (if different)
	expectedPath := filepath.Join(d.storagePath, dbName+".mmdb")
	if extractedPath != expectedPath {
		if err := os.Rename(extractedPath, expectedPath); err != nil {
			CleanupTempFiles(tempTarPath, extractedPath)
			return fmt.Errorf("failed to rename extracted file: %w", err)
		}
	}

	// 8. Cleanup temporary files
	CleanupTempFiles(tempTarPath)

	// 9. Update metadata
	updateMetadata(d.storagePath,
		filepath.Join(d.storagePath, d.config.Databases.City+".mmdb"),
		filepath.Join(d.storagePath, d.config.Databases.ASN+".mmdb"))

	log.Info().
		Str("database", dbName).
		Str("type", dbType).
		Str("hash", expectedHash[:16]+"...").
		Time("remote_mod_time", updateInfo.RemoteModTime).
		Msg("Database downloaded and verified successfully")

	return nil
}

// ForceDownload forces download of specific database
func (d *DatabaseDownloader) ForceDownload(dbType string) error {
	if !d.config.Downloader.Enabled {
		return fmt.Errorf("MaxMind downloader is disabled")
	}

	var dbName string
	switch dbType {
	case "city":
		dbName = d.config.Databases.City
	case "asn":
		dbName = d.config.Databases.ASN
	default:
		return fmt.Errorf("invalid database type: %s", dbType)
	}

	if dbName == "" {
		return fmt.Errorf("database %s is not configured", dbType)
	}

	log := logger.WithScope("maxmind-force-download")
	log.Info().
		Str("database", dbName).
		Str("type", dbType).
		Msg("Starting forced database download")

	// Remove local file to force download
	localFilePath := filepath.Join(d.storagePath, dbName+".mmdb")
	if err := os.Remove(localFilePath); err != nil && !os.IsNotExist(err) {
		log.Warn().
			Err(err).
			Str("file", localFilePath).
			Msg("Failed to remove existing database file")
	}

	return d.DownloadIfNeeded(dbType, dbName)
}

// GetDownloadStatus returns current download status
func (d *DatabaseDownloader) GetDownloadStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled":      d.config.Downloader.Enabled,
		"storage_path": d.storagePath,
	}

	if !d.config.Downloader.Enabled {
		status["message"] = "Downloader is disabled"
		return status
	}

	if d.client != nil {
		status["client"] = d.client.String()
	}

	databases := make(map[string]interface{})

	// City database status
	if d.config.Databases.City != "" {
		cityPath := filepath.Join(d.storagePath, d.config.Databases.City+".mmdb")
		cityStatus := d.getDatabaseFileStatus(cityPath)
		databases["city"] = map[string]interface{}{
			"name":   d.config.Databases.City,
			"status": cityStatus,
		}
	}

	// ASN database status
	if d.config.Databases.ASN != "" {
		asnPath := filepath.Join(d.storagePath, d.config.Databases.ASN+".mmdb")
		asnStatus := d.getDatabaseFileStatus(asnPath)
		databases["asn"] = map[string]interface{}{
			"name":   d.config.Databases.ASN,
			"status": asnStatus,
		}
	}

	status["databases"] = databases
	return status
}

// getDatabaseFileStatus gets status of a database file
func (d *DatabaseDownloader) getDatabaseFileStatus(filePath string) map[string]interface{} {
	info, err := os.Stat(filePath)
	if err != nil {
		return map[string]interface{}{
			"exists": false,
			"error":  err.Error(),
		}
	}

	return map[string]interface{}{
		"exists":   true,
		"size":     info.Size(),
		"mod_time": info.ModTime(),
		"readable": info.Mode().IsRegular(),
	}
}

// Stop gracefully stops the downloader
func (d *DatabaseDownloader) Stop() error {
	close(d.stopCh)
	d.wg.Wait()

	logger.Debug().Msg("MaxMind database downloader stopped")
	return nil
}
