package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

var (
	mainClient Client
	mu         sync.RWMutex
	once       sync.Once
)

// Init initializes the main Redis client from configuration
func Init() error {
	var initErr error

	once.Do(func() {
		cfg := config.Get().Redis

		// Validate Redis configuration
		if err := validateConfig(cfg); err != nil {
			initErr = fmt.Errorf("invalid Redis configuration: %w", err)
			return
		}

		// Create main Redis client
		client, err := NewClientForMain()
		if err != nil {
			initErr = fmt.Errorf("failed to initialize main Redis client: %w", err)
			return
		}

		// Test connection
		if err := client.Health(); err != nil {
			initErr = fmt.Errorf("redis health check failed: %w", err)
			return
		}

		// Store client reference
		mu.Lock()
		mainClient = client
		mu.Unlock()

		logger.Info().
			Str("mode", cfg.Mode).
			Str("host", cfg.Host).
			Int("port", cfg.Port).
			Msg("Redis client initialized successfully")
	})

	return initErr
}

// GetClient returns the main Redis client instance
func GetClient() Client {
	mu.RLock()
	defer mu.RUnlock()
	return mainClient
}

// Close closes all Redis connections
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if mainClient != nil {
		err := mainClient.Close()
		mainClient = nil
		return err
	}

	return nil
}

// Health checks the main Redis connection
func Health() error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	return client.Health()
}

// validateConfig validates the Redis configuration
func validateConfig(cfg config.RedisConfig) error {
	// Default to single mode if not specified
	if cfg.Mode == "" {
		cfg.Mode = string(ModeSingle)
	}

	switch RedisMode(cfg.Mode) {
	case ModeSingle:
		if cfg.Host == "" {
			return fmt.Errorf("redis host not specified for single-node mode")
		}
		if cfg.Port <= 0 || cfg.Port > 65535 {
			return fmt.Errorf("invalid Redis port: %d", cfg.Port)
		}

	case ModeCluster:
		if len(cfg.Cluster.Nodes) == 0 {
			return fmt.Errorf("redis cluster nodes not specified")
		}
		for _, node := range cfg.Cluster.Nodes {
			if node == "" {
				return fmt.Errorf("empty Redis cluster node")
			}
		}

	case ModeSentinel:
		return fmt.Errorf("redis Sentinel mode not yet implemented")

	default:
		return fmt.Errorf("unsupported Redis mode: %s", cfg.Mode)
	}

	return nil
}

// Helper functions for common operations using the main client

// SetJSON stores JSON data with the main client
func SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return client.SetJSON(ctx, key, value, expiration)
}

// GetJSON retrieves JSON data with the main client
func GetJSON(ctx context.Context, key string, dest interface{}) error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return client.GetJSON(ctx, key, dest)
}

// Set stores a key-value pair with the main client
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return client.Set(ctx, key, value, expiration)
}

// Get retrieves a value with the main client
func Get(ctx context.Context, key string) (string, error) {
	client := GetClient()
	if client == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	return client.Get(ctx, key)
}

// Delete removes keys with the main client
func Delete(ctx context.Context, keys ...string) error {
	client := GetClient()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return client.Delete(ctx, keys...)
}

// Exists checks if key exists with the main client
func Exists(ctx context.Context, key string) (bool, error) {
	client := GetClient()
	if client == nil {
		return false, fmt.Errorf("redis client not initialized")
	}
	return client.Exists(ctx, key)
}
