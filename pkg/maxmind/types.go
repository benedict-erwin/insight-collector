package maxmind

import (
	"net"
	"time"
)

// GeoLocation holds geographical information for an IP address
type GeoLocation struct {
	IP          string    `json:"ip"`
	Country     string    `json:"country"`
	CountryCode string    `json:"country_code"`
	City        string    `json:"city"`
	Region      string    `json:"region"`
	PostalCode  string    `json:"postal_code,omitempty"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Timezone    string    `json:"timezone,omitempty"`
	LookedUpAt  time.Time `json:"looked_up_at"`
}

// ASNInfo holds Autonomous System Number information
type ASNInfo struct {
	IP           string    `json:"ip"`
	ASN          uint      `json:"asn"`
	Organization string    `json:"organization"`
	LookedUpAt   time.Time `json:"looked_up_at"`
}

// DatabaseInfo holds information about loaded databases
type DatabaseInfo struct {
	CityDBPath     string    `json:"city_db_path"`
	CityDBSize     int64     `json:"city_db_size"`
	CityDBModTime  time.Time `json:"city_db_modified"`
	ASNDBPath      string    `json:"asn_db_path"`
	ASNDBSize      int64     `json:"asn_db_size"`
	ASNDBModTime   time.Time `json:"asn_db_modified"`
	LoadedAt       time.Time `json:"loaded_at"`
	ReloadCount    int       `json:"reload_count"`
	Enabled        bool      `json:"enabled"`
}

// Config holds MaxMind service configuration
type Config struct {
	Enabled       bool   `json:"enabled"`
	StoragePath   string `json:"storage_path"`
	CheckInterval string `json:"check_interval"`
	Databases     struct {
		City string `json:"city"`
		ASN  string `json:"asn"`
	} `json:"databases"`
	Downloader DownloaderConfig `json:"downloader"`
}

// DownloaderConfig holds downloader-specific configuration
type DownloaderConfig struct {
	Enabled       bool   `json:"enabled"`
	AccountID     string `json:"account_id"`
	LicenseKey    string `json:"license_key"`
	BaseURL       string `json:"base_url"`
	Timeout       string `json:"timeout"`
	RetryAttempts int    `json:"retry_attempts"`
	RetryDelay    string `json:"retry_delay"`
}

// UpdateInfo holds information about database updates
type UpdateInfo struct {
	DatabaseName   string    `json:"database_name"`
	RemoteModTime  time.Time `json:"remote_mod_time"`
	LocalModTime   time.Time `json:"local_mod_time"`
	NeedsUpdate    bool      `json:"needs_update"`
	ContentLength  int64     `json:"content_length"`
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
}

// GeoIPService defines the interface for GeoIP operations
type GeoIPService interface {
	LookupCity(ip net.IP) *GeoLocation
	LookupASN(ip net.IP) *ASNInfo
	GetDatabaseInfo() *DatabaseInfo
	ReloadDatabases() error
	Health() error
	Close() error
}

// DefaultGeoLocation returns a GeoLocation with safe defaults
func DefaultGeoLocation(ip net.IP) *GeoLocation {
	return &GeoLocation{
		IP:         ip.String(),
		Country:    "",
		City:       "",
		Region:     "",
		PostalCode: "",
		Latitude:   0.0,
		Longitude:  0.0,
		Timezone:   "",
		LookedUpAt: time.Now(),
	}
}

// DefaultASNInfo returns an ASNInfo with safe defaults
func DefaultASNInfo(ip net.IP) *ASNInfo {
	return &ASNInfo{
		IP:           ip.String(),
		ASN:          0,
		Organization: "",
		LookedUpAt:   time.Now(),
	}
}