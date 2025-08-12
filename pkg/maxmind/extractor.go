package maxmind

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// ExtractTarGz extracts .mmdb file from tar.gz archive
func ExtractTarGz(src, destDir string) (string, error) {
	log := logger.WithScope("maxmind-extractor")
	
	// Open tar.gz file
	file, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()
	
	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	
	// Create tar reader
	tarReader := tar.NewReader(gzipReader)
	
	var extractedFile string
	
	// Iterate through tar entries
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar entry: %w", err)
		}
		
		// Skip directories and non-mmdb files
		if header.Typeflag != tar.TypeReg {
			continue
		}
		
		if !strings.HasSuffix(header.Name, ".mmdb") {
			log.Debug().
				Str("file", header.Name).
				Msg("Skipping non-mmdb file")
			continue
		}
		
		// Extract file name from path
		fileName := filepath.Base(header.Name)
		destPath := filepath.Join(destDir, fileName)
		
		log.Info().
			Str("source", header.Name).
			Str("destination", destPath).
			Int64("size", header.Size).
			Msg("Extracting mmdb file")
		
		// Create destination file
		destFile, err := os.Create(destPath)
		if err != nil {
			return "", fmt.Errorf("failed to create destination file %s: %w", destPath, err)
		}
		
		// Copy file content
		bytesWritten, err := io.Copy(destFile, tarReader)
		destFile.Close()
		
		if err != nil {
			os.Remove(destPath) // Cleanup on error
			return "", fmt.Errorf("failed to extract file %s: %w", fileName, err)
		}
		
		if bytesWritten != header.Size {
			os.Remove(destPath) // Cleanup on error
			return "", fmt.Errorf("file size mismatch for %s: expected %d, got %d", 
				fileName, header.Size, bytesWritten)
		}
		
		// Set file permissions
		if err := os.Chmod(destPath, 0644); err != nil {
			log.Warn().Err(err).Str("file", destPath).Msg("Failed to set file permissions")
		}
		
		log.Info().
			Str("file", fileName).
			Int64("bytes", bytesWritten).
			Msg("Successfully extracted mmdb file")
		
		extractedFile = destPath
		break // Only extract first .mmdb file found
	}
	
	if extractedFile == "" {
		return "", fmt.Errorf("no .mmdb file found in archive")
	}
	
	return extractedFile, nil
}

// VerifyChecksum verifies file against SHA256 hash
func VerifyChecksum(filePath, expectedHash string) error {
	log := logger.WithScope("maxmind-verifier")
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum verification: %w", err)
	}
	defer file.Close()
	
	// Calculate SHA256 hash
	hasher := sha256.New()
	bytesRead, err := io.Copy(hasher, file)
	if err != nil {
		return fmt.Errorf("failed to read file for checksum calculation: %w", err)
	}
	
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	
	log.Debug().
		Str("file", filepath.Base(filePath)).
		Int64("bytes_read", bytesRead).
		Str("expected_hash", expectedHash[:16]+"...").
		Str("actual_hash", actualHash[:16]+"...").
		Msg("Checksum verification")
	
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", 
			filepath.Base(filePath), expectedHash, actualHash)
	}
	
	log.Info().
		Str("file", filepath.Base(filePath)).
		Str("hash", actualHash[:16]+"...").
		Msg("Checksum verification successful")
	
	return nil
}

// CleanupTempFiles removes temporary files
func CleanupTempFiles(files ...string) {
	log := logger.WithScope("maxmind-cleanup")
	
	for _, file := range files {
		if file == "" {
			continue
		}
		
		if err := os.Remove(file); err != nil {
			log.Warn().
				Err(err).
				Str("file", file).
				Msg("Failed to cleanup temporary file")
		} else {
			log.Debug().
				Str("file", file).
				Msg("Cleaned up temporary file")
		}
	}
}

// GetFileModTime safely gets file modification time
func GetFileModTime(filePath string) (time.Time, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// EnsureDir creates directory if it doesn't exist
func EnsureDir(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}
	return nil
}