package redis

import (
	"context"
	"time"
)

// RedisMode defines the Redis deployment mode
type RedisMode string

const (
	ModeSingle   RedisMode = "single"   // Single-node Redis
	ModeCluster  RedisMode = "cluster"  // Redis Cluster
	ModeSentinel RedisMode = "sentinel" // Redis Sentinel
)

// Database constants for single-node Redis logical separation
const (
	DBMain       = 0 // Main application data (worker config, settings)
	DBAsynq      = 0 // Job queue (same as main for now, could be separate)
	DBSessions   = 1 // User sessions and authentication state
	DBCache      = 2 // Application cache and temporary computations
	DBTempData   = 3 // Temporary data with TTL
	DBNonceStore = 4 // Authentication nonce storage for replay protection
)

// Key prefixes for Redis Cluster logical separation (since DB selection not supported)
const (
	PrefixMain      = "main:"
	PrefixWorker    = "worker:"
	PrefixSessions  = "sessions:"
	PrefixCache     = "cache:"
	PrefixTempData  = "temp:"
	PrefixNonce     = "nonce:"
	PrefixAsynq     = "asynq:"
)

// Client defines the unified Redis client interface
type Client interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	GetJSON(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Health() error
	Close() error
}

// RedisConfig holds Redis configuration for different modes
type RedisConfig struct {
	Mode     string        `json:"mode"`     // single, cluster, sentinel
	Single   SingleConfig  `json:"single"`   // Single-node configuration
	Cluster  ClusterConfig `json:"cluster"`  // Cluster configuration
	Sentinel SentinelConfig `json:"sentinel"` // Sentinel configuration
	Pool     PoolConfig    `json:"pool"`     // Connection pool settings
}

// SingleConfig holds single-node Redis configuration
type SingleConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// ClusterConfig holds Redis Cluster configuration
type ClusterConfig struct {
	Nodes    []string `json:"nodes"`
	Password string   `json:"password"`
	ReadOnly bool     `json:"read_only"`
}

// SentinelConfig holds Redis Sentinel configuration
type SentinelConfig struct {
	MasterName string   `json:"master_name"`
	Nodes      []string `json:"nodes"`
	Password   string   `json:"password"`
}

// PoolConfig holds connection pool configuration
type PoolConfig struct {
	Size         int           `json:"size"`
	Timeout      time.Duration `json:"timeout"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

// DefaultRedisConfig returns default Redis configuration
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Mode: string(ModeSingle),
		Single: SingleConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       DBMain,
		},
		Pool: PoolConfig{
			Size:         10,
			Timeout:      30 * time.Second,
			DialTimeout:  10 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
	}
}