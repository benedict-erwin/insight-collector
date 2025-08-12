package maxmind

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// MaxMindClient handles HTTP requests to MaxMind API
type MaxMindClient struct {
	httpClient  *http.Client
	baseURL     string
	accountID   string
	licenseKey  string
	retryConfig RetryConfig
}

// NewMaxMindClient creates a new MaxMind HTTP client
func NewMaxMindClient(config *DownloaderConfig) (*MaxMindClient, error) {
	// Parse timeout
	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout duration: %w", err)
	}

	// Parse retry delay
	retryDelay, err := time.ParseDuration(config.RetryDelay)
	if err != nil {
		return nil, fmt.Errorf("invalid retry delay duration: %w", err)
	}

	// Support environment variables for credentials
	accountID := config.AccountID
	if envAccountID := os.Getenv("MAXMIND_ACCOUNT_ID"); envAccountID != "" {
		accountID = envAccountID
	}

	licenseKey := config.LicenseKey
	if envLicenseKey := os.Getenv("MAXMIND_LICENSE_KEY"); envLicenseKey != "" {
		licenseKey = envLicenseKey
	}

	// Validate credentials
	if accountID == "" {
		return nil, fmt.Errorf("MaxMind account ID is required")
	}
	if licenseKey == "" {
		return nil, fmt.Errorf("MaxMind license key is required")
	}

	return &MaxMindClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL:    config.BaseURL,
		accountID:  accountID,
		licenseKey: licenseKey,
		retryConfig: RetryConfig{
			MaxAttempts: config.RetryAttempts,
			Delay:       retryDelay,
		},
	}, nil
}

// setBasicAuth adds basic authentication to request
func (c *MaxMindClient) setBasicAuth(req *http.Request) {
	req.SetBasicAuth(c.accountID, c.licenseKey)
}

// CheckLastModified performs HEAD request to get Last-Modified header
func (c *MaxMindClient) CheckLastModified(dbName string) (*UpdateInfo, error) {
	url := fmt.Sprintf("%s/%s/download?suffix=tar.gz", c.baseURL, dbName)
	
	var lastErr error
	for attempt := 1; attempt <= c.retryConfig.MaxAttempts; attempt++ {
		req, err := http.NewRequest("HEAD", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create HEAD request: %w", err)
		}
		
		c.setBasicAuth(req)
		
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.retryConfig.MaxAttempts {
				logger.Warn().
					Err(err).
					Str("database", dbName).
					Int("attempt", attempt).
					Dur("retry_delay", c.retryConfig.Delay).
					Msg("HEAD request failed, retrying")
				time.Sleep(c.retryConfig.Delay)
				continue
			}
			return nil, fmt.Errorf("HEAD request failed after %d attempts: %w", c.retryConfig.MaxAttempts, err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HEAD request returned status %d", resp.StatusCode)
			if attempt < c.retryConfig.MaxAttempts {
				logger.Warn().
					Int("status_code", resp.StatusCode).
					Str("database", dbName).
					Int("attempt", attempt).
					Msg("HEAD request failed with non-200 status, retrying")
				time.Sleep(c.retryConfig.Delay)
				continue
			}
			return nil, lastErr
		}
		
		// Parse Last-Modified header
		lastModifiedStr := resp.Header.Get("Last-Modified")
		if lastModifiedStr == "" {
			return nil, fmt.Errorf("Last-Modified header not found in response")
		}
		
		remoteModTime, err := http.ParseTime(lastModifiedStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Last-Modified header '%s': %w", lastModifiedStr, err)
		}
		
		contentLength := resp.ContentLength
		
		logger.Debug().
			Str("database", dbName).
			Time("remote_mod_time", remoteModTime).
			Int64("content_length", contentLength).
			Msg("Successfully retrieved database metadata")
		
		return &UpdateInfo{
			DatabaseName:  dbName,
			RemoteModTime: remoteModTime,
			ContentLength: contentLength,
		}, nil
	}
	
	return nil, lastErr
}

// DownloadChecksum downloads SHA256 checksum file
func (c *MaxMindClient) DownloadChecksum(dbName string) (string, error) {
	url := fmt.Sprintf("%s/%s/download?suffix=tar.gz.sha256", c.baseURL, dbName)
	
	var lastErr error
	for attempt := 1; attempt <= c.retryConfig.MaxAttempts; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create checksum request: %w", err)
		}
		
		c.setBasicAuth(req)
		
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.retryConfig.MaxAttempts {
				logger.Warn().
					Err(err).
					Str("database", dbName).
					Int("attempt", attempt).
					Msg("Checksum download failed, retrying")
				time.Sleep(c.retryConfig.Delay)
				continue
			}
			return "", fmt.Errorf("checksum download failed after %d attempts: %w", c.retryConfig.MaxAttempts, err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("checksum download returned status %d", resp.StatusCode)
			if attempt < c.retryConfig.MaxAttempts {
				logger.Warn().
					Int("status_code", resp.StatusCode).
					Str("database", dbName).
					Int("attempt", attempt).
					Msg("Checksum download failed with non-200 status, retrying")
				time.Sleep(c.retryConfig.Delay)
				continue
			}
			return "", lastErr
		}
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read checksum response: %w", err)
		}
		
		// Extract hash from "hash filename" format
		bodyStr := strings.TrimSpace(string(body))
		parts := strings.Fields(bodyStr)
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid checksum format: %s", bodyStr)
		}
		
		hash := strings.TrimSpace(parts[0])
		if len(hash) != 64 { // SHA256 should be 64 characters
			return "", fmt.Errorf("invalid SHA256 hash length: expected 64, got %d", len(hash))
		}
		
		logger.Debug().
			Str("database", dbName).
			Str("hash", hash[:16]+"...").
			Msg("Successfully downloaded checksum")
		
		return hash, nil
	}
	
	return "", lastErr
}

// DownloadDatabase downloads tar.gz database file
func (c *MaxMindClient) DownloadDatabase(dbName string, writer io.Writer) error {
	url := fmt.Sprintf("%s/%s/download?suffix=tar.gz", c.baseURL, dbName)
	
	var lastErr error
	for attempt := 1; attempt <= c.retryConfig.MaxAttempts; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create database request: %w", err)
		}
		
		c.setBasicAuth(req)
		
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.retryConfig.MaxAttempts {
				logger.Warn().
					Err(err).
					Str("database", dbName).
					Int("attempt", attempt).
					Msg("Database download failed, retrying")
				time.Sleep(c.retryConfig.Delay)
				continue
			}
			return fmt.Errorf("database download failed after %d attempts: %w", c.retryConfig.MaxAttempts, err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("database download returned status %d", resp.StatusCode)
			if attempt < c.retryConfig.MaxAttempts {
				logger.Warn().
					Int("status_code", resp.StatusCode).
					Str("database", dbName).
					Int("attempt", attempt).
					Msg("Database download failed with non-200 status, retrying")
				time.Sleep(c.retryConfig.Delay)
				continue
			}
			return lastErr
		}
		
		// Copy response to writer with progress tracking
		bytesWritten, err := io.Copy(writer, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write database file: %w", err)
		}
		
		logger.Info().
			Str("database", dbName).
			Int64("bytes_written", bytesWritten).
			Msg("Successfully downloaded database")
		
		return nil
	}
	
	return lastErr
}

// String returns string representation with masked credentials
func (c *MaxMindClient) String() string {
	return fmt.Sprintf("MaxMindClient{AccountID: %s, LicenseKey: ****}", 
		c.maskAccountID())
}

// maskAccountID masks account ID for logging
func (c *MaxMindClient) maskAccountID() string {
	if len(c.accountID) <= 4 {
		return "****"
	}
	return c.accountID[:2] + "****" + c.accountID[len(c.accountID)-2:]
}