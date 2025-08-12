package maxmind

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

const metadataFileName = ".metadata.json"

// Metadata holds database version and tracking information
type Metadata struct {
	CityDBVersion string    `json:"city_db_version"`
	ASNDBVersion  string    `json:"asn_db_version"`
	LastUpdated   time.Time `json:"last_updated"`
	UpdateCount   int       `json:"update_count"`
}

// loadMetadata loads metadata from storage directory
func loadMetadata(storagePath string) *Metadata {
	metadataPath := filepath.Join(storagePath, metadataFileName)
	
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		logger.Debug().Err(err).Str("path", metadataPath).Msg("Metadata file not found, using defaults")
		return &Metadata{
			LastUpdated: time.Now(),
			UpdateCount: 0,
		}
	}
	
	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		logger.Warn().Err(err).Msg("Failed to parse metadata file, using defaults")
		return &Metadata{
			LastUpdated: time.Now(),
			UpdateCount: 0,
		}
	}
	
	return &metadata
}

// saveMetadata saves metadata to storage directory
func saveMetadata(storagePath string, metadata *Metadata) error {
	metadataPath := filepath.Join(storagePath, metadataFileName)
	
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(metadataPath, data, 0644)
}

// updateMetadata updates metadata with current database information
func updateMetadata(storagePath string, cityDBPath, asnDBPath string) {
	metadata := loadMetadata(storagePath)
	
	// Update versions based on file modification times
	if cityInfo, err := os.Stat(cityDBPath); err == nil {
		metadata.CityDBVersion = cityInfo.ModTime().Format("2006-01-02")
	}
	
	if asnInfo, err := os.Stat(asnDBPath); err == nil {
		metadata.ASNDBVersion = asnInfo.ModTime().Format("2006-01-02")
	}
	
	metadata.LastUpdated = time.Now()
	metadata.UpdateCount++
	
	if err := saveMetadata(storagePath, metadata); err != nil {
		logger.Error().Err(err).Msg("Failed to save metadata")
	} else {
		logger.Debug().
			Str("city_version", metadata.CityDBVersion).
			Str("asn_version", metadata.ASNDBVersion).
			Int("update_count", metadata.UpdateCount).
			Msg("Metadata updated")
	}
}