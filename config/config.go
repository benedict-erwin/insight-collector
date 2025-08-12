package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type (
	app struct {
		Name     string `json:"name" mapstructure:"name"`
		Env      string `json:"env" mapstructure:"env"`
		Port     int    `json:"port" mapstructure:"port"`
		Timezone string `json:"timezone" mapstructure:"timezone"`
		Version  string `json:"version" mapstructure:"version"`
	}

	influxDb struct {
		// Version selection - determines which InfluxDB implementation to use
		Version string `json:"version,omitempty" mapstructure:"version"` // "v2-oss" or "v3-core"

		// v2-oss fields (InfluxDB v2 OSS)
		URL string `json:"url,omitempty" mapstructure:"url"` // Complete URL like http://localhost:8086
		Org string `json:"org,omitempty" mapstructure:"org"` // Organization name

		// Common fields (used by both versions)
		Token  string `json:"token" mapstructure:"token"`
		Bucket string `json:"bucket" mapstructure:"bucket"`

		// v3-core fields (legacy InfluxDB v3 Core) - kept for backward compatibility
		Host       string `json:"host,omitempty" mapstructure:"host"`
		Port       int    `json:"port,omitempty" mapstructure:"port"`
		AuthScheme string `json:"auth_scheme,omitempty" mapstructure:"auth_scheme"`
		Node       string `json:"node,omitempty" mapstructure:"node"`
	}

	redis struct {
		Mode     string `json:"mode" mapstructure:"mode"` // "single", "cluster", "sentinel"
		Host     string `json:"host" mapstructure:"host"`
		Port     int    `json:"port" mapstructure:"port"`
		Password string `json:"password" mapstructure:"password"`
		DB       int    `json:"db" mapstructure:"db"`
		Cluster  struct {
			Nodes    []string `json:"nodes" mapstructure:"nodes"`
			Password string   `json:"password" mapstructure:"password"`
		} `json:"cluster" mapstructure:"cluster"`
	}

	asynq struct {
		Concurrency int `json:"concurrency" mapstructure:"concurrency"`
		DB          int `json:"db" mapstructure:"db"`
		PoolSize    int `json:"pool_size" mapstructure:"pool_size"`
	}

	auth struct {
		Enabled   bool           `json:"enabled" mapstructure:"enabled"`
		Algorithm string         `json:"algorithm" mapstructure:"algorithm"`
		Clients   []ClientConfig `json:"clients" mapstructure:"clients"`
	}

	maxmind struct {
		Enabled       bool   `json:"enabled" mapstructure:"enabled"`
		StoragePath   string `json:"storage_path" mapstructure:"storage_path"`
		CheckInterval string `json:"check_interval" mapstructure:"check_interval"`
		Databases     struct {
			City string `json:"city" mapstructure:"city"`
			ASN  string `json:"asn" mapstructure:"asn"`
		} `json:"databases" mapstructure:"databases"`
		Downloader struct {
			Enabled       bool   `json:"enabled" mapstructure:"enabled"`
			AccountID     string `json:"account_id" mapstructure:"account_id"`
			LicenseKey    string `json:"license_key" mapstructure:"license_key"`
			BaseURL       string `json:"base_url" mapstructure:"base_url"`
			Timeout       string `json:"timeout" mapstructure:"timeout"`
			RetryAttempts int    `json:"retry_attempts" mapstructure:"retry_attempts"`
			RetryDelay    string `json:"retry_delay" mapstructure:"retry_delay"`
		} `json:"downloader" mapstructure:"downloader"`
	}

	ClientConfig struct {
		ClientID    string   `json:"client_id" mapstructure:"client_id"`
		ClientName  string   `json:"client_name" mapstructure:"client_name"`
		AuthType    string   `json:"auth_type" mapstructure:"auth_type"`             // "rsa" or "hmac"
		KeyPath     string   `json:"key_path,omitempty" mapstructure:"key_path"`     // for RSA public key
		SecretKey   string   `json:"secret_key,omitempty" mapstructure:"secret_key"` // for HMAC
		Permissions []string `json:"permissions" mapstructure:"permissions"`
		Active      bool     `json:"active" mapstructure:"active"`
	}

	Config struct {
		App      app      `json:"app" mapstructure:"app"`
		InfluxDB influxDb `json:"influxdb" mapstructure:"influxdb"`
		Redis    redis    `json:"redis" mapstructure:"redis"`
		Asynq    asynq    `json:"asynq" mapstructure:"asynq"`
		Auth     auth     `json:"auth" mapstructure:"auth"`
		MaxMind  maxmind  `json:"maxmind" mapstructure:"maxmind"`
	}

	// RedisConfig is an alias for the internal redis struct for external access
	RedisConfig = redis
)

var cfg *Config

// Init loads configuration from .config file
func Init() error {
	viper.SetConfigName(".config")
	viper.SetConfigType("json")
	viper.AddConfigPath("./")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

// Get returns the current configuration instance
func Get() *Config {
	return cfg
}
