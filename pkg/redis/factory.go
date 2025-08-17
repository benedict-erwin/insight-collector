package redis

import (
	"fmt"
	"time"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// getPoolConfigFromRedis gets pool configuration for specific client type from Redis pools config
func getPoolConfigFromRedis(clientType string) config.PoolConfig {
	cfg := config.Get()

	// Return specific pool config if exists
	if cfg.Redis.Pools != nil {
		if pool, exists := cfg.Redis.Pools[clientType]; exists {
			return pool
		}
		// Fallback to default pool if available
		if defaultPool, exists := cfg.Redis.Pools["default"]; exists {
			return defaultPool
		}
	}

	// Final fallback to reasonable defaults
	return config.PoolConfig{
		Size:         10,
		Timeout:      "30s",
		DialTimeout:  "10s",
		ReadTimeout:  "5s",
		WriteTimeout: "5s",
		MaxLifetime:  "30m",
		IdleTimeout:  "10m",
	}
}

// convertToRedisPoolConfig converts config.PoolConfig to types.PoolConfig with duration parsing
func convertToRedisPoolConfig(poolCfg config.PoolConfig) PoolConfig {
	timeout, _ := time.ParseDuration(poolCfg.Timeout)
	dialTimeout, _ := time.ParseDuration(poolCfg.DialTimeout)
	readTimeout, _ := time.ParseDuration(poolCfg.ReadTimeout)
	writeTimeout, _ := time.ParseDuration(poolCfg.WriteTimeout)
	maxLifetime, _ := time.ParseDuration(poolCfg.MaxLifetime)
	idleTimeout, _ := time.ParseDuration(poolCfg.IdleTimeout)

	return PoolConfig{
		Size:         poolCfg.Size,
		Timeout:      timeout,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		MaxLifetime:  maxLifetime,
		IdleTimeout:  idleTimeout,
	}
}

// buildRedisConfig creates RedisConfig from application config with specific client pool config
func buildRedisConfig(dbOverride *int, clientType string) RedisConfig {
	cfg := config.Get()

	// Default to single mode if not specified
	mode := cfg.Redis.Mode
	if mode == "" {
		mode = string(ModeSingle)
	}

	// Use override DB if provided, otherwise use config DB
	db := cfg.Redis.DB
	if dbOverride != nil {
		db = *dbOverride
	}

	// Get pool configuration for specific client type
	poolConfigFromRedis := getPoolConfigFromRedis(clientType)
	poolConfig := convertToRedisPoolConfig(poolConfigFromRedis)

	return RedisConfig{
		Mode:    mode,
		Single:  SingleConfig{Host: cfg.Redis.Host, Port: cfg.Redis.Port, Password: cfg.Redis.Password, DB: db},
		Cluster: ClusterConfig{Nodes: cfg.Redis.Cluster.Nodes, Password: cfg.Redis.Cluster.Password},
		Pool:    poolConfig,
	}
}

// NewClientForMain returns Redis client for main application data
func NewClientForMain() (Client, error) {
	dbMain := DBMain
	redisConfig := buildRedisConfig(&dbMain, "default")

	var keyPrefix string
	var db int

	switch RedisMode(redisConfig.Mode) {
	case ModeSingle:
		db = DBMain
		keyPrefix = ""
	case ModeCluster:
		db = 0 // Cluster doesn't use DB selection
		keyPrefix = PrefixMain
	default:
		return nil, fmt.Errorf("unsupported Redis mode: %s", redisConfig.Mode)
	}

	client, err := NewRedisClient(redisConfig, keyPrefix, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create main Redis client: %w", err)
	}

	logger.Info().
		Str("mode", redisConfig.Mode).
		Str("prefix", keyPrefix).
		Int("db", db).
		Msg("Main Redis client initialized")

	return client, nil
}

// NewClientForWorkerConfig returns Redis client for worker configuration
func NewClientForWorkerConfig() (Client, error) {
	dbMain := DBMain
	redisConfig := buildRedisConfig(&dbMain, "default")

	var keyPrefix string
	var db int

	switch RedisMode(redisConfig.Mode) {
	case ModeSingle:
		db = DBMain // Worker config in main DB for single-node
		keyPrefix = ""
	case ModeCluster:
		db = 0
		keyPrefix = PrefixWorker
	default:
		return nil, fmt.Errorf("unsupported Redis mode: %s", redisConfig.Mode)
	}

	client, err := NewRedisClient(redisConfig, keyPrefix, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker config Redis client: %w", err)
	}

	logger.Debug().
		Str("mode", redisConfig.Mode).
		Str("prefix", keyPrefix).
		Int("db", db).
		Msg("Worker config Redis client initialized")

	return client, nil
}

// NewClientForSessions returns Redis client for session storage
func NewClientForSessions() (Client, error) {
	dbSessions := DBSessions
	redisConfig := buildRedisConfig(&dbSessions, "sessions")

	var keyPrefix string
	var db int

	switch RedisMode(redisConfig.Mode) {
	case ModeSingle:
		db = DBSessions // Dedicated DB for sessions
		keyPrefix = ""
	case ModeCluster:
		db = 0
		keyPrefix = PrefixSessions
	default:
		return nil, fmt.Errorf("unsupported Redis mode: %s", redisConfig.Mode)
	}

	client, err := NewRedisClient(redisConfig, keyPrefix, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create sessions Redis client: %w", err)
	}

	logger.Debug().
		Str("mode", redisConfig.Mode).
		Str("prefix", keyPrefix).
		Int("db", db).
		Msg("Sessions Redis client initialized")

	return client, nil
}

// NewClientForCache returns Redis client for application cache
func NewClientForCache() (Client, error) {
	dbCache := DBCache
	redisConfig := buildRedisConfig(&dbCache, "cache")

	var keyPrefix string
	var db int

	switch RedisMode(redisConfig.Mode) {
	case ModeSingle:
		db = DBCache // Dedicated DB for cache
		keyPrefix = ""
	case ModeCluster:
		db = 0
		keyPrefix = PrefixCache
	default:
		return nil, fmt.Errorf("unsupported Redis mode: %s", redisConfig.Mode)
	}

	client, err := NewRedisClient(redisConfig, keyPrefix, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache Redis client: %w", err)
	}

	logger.Debug().
		Str("mode", redisConfig.Mode).
		Str("prefix", keyPrefix).
		Int("db", db).
		Msg("Cache Redis client initialized")

	return client, nil
}

// NewClientForNonceStore returns Redis client for nonce storage (replay protection)
func NewClientForNonceStore() (Client, error) {
	dbNonce := DBNonceStore
	redisConfig := buildRedisConfig(&dbNonce, "nonce")

	var keyPrefix string
	var db int

	switch RedisMode(redisConfig.Mode) {
	case ModeSingle:
		db = DBNonceStore // Dedicated DB for nonce storage
		keyPrefix = ""
	case ModeCluster:
		db = 0
		keyPrefix = PrefixNonce
	default:
		return nil, fmt.Errorf("unsupported Redis mode: %s", redisConfig.Mode)
	}

	client, err := NewRedisClient(redisConfig, keyPrefix, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create nonce store Redis client: %w", err)
	}

	logger.Debug().
		Str("mode", redisConfig.Mode).
		Str("prefix", keyPrefix).
		Int("db", db).
		Msg("Nonce store Redis client initialized")

	return client, nil
}

// NewClientForAsynq returns Redis client optimized for Asynq job queue
func NewClientForAsynq(heartbeat ...bool) (Client, error) {
	cfg := config.Get()
	asynqDB := cfg.Asynq.DB
	redisConfig := buildRedisConfig(&asynqDB, "asynq")

	// Default hertbeat check
	isHeartbeat := false
	if len(heartbeat) > 0 {
		isHeartbeat = heartbeat[0]
	}

	var keyPrefix string
	var db int

	switch RedisMode(redisConfig.Mode) {
	case ModeSingle:
		db = cfg.Asynq.DB // Use Asynq-specific DB from config
		keyPrefix = ""
	case ModeCluster:
		db = 0
		keyPrefix = PrefixAsynq
	default:
		return nil, fmt.Errorf("unsupported Redis mode: %s", redisConfig.Mode)
	}

	client, err := NewRedisClient(redisConfig, keyPrefix, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create Asynq Redis client: %w", err)
	}

	// Logger for heartbeat or init
	if isHeartbeat {
		logger.Info().
			Str("mode", redisConfig.Mode).
			Str("prefix", keyPrefix).
			Int("db", db).
			Int("concurrency", cfg.Asynq.Concurrency).
			Int("pool_size", cfg.Asynq.PoolSize).
			Msg("Asynq Redis client heartbeat check")
	} else {
		logger.Info().
			Str("mode", redisConfig.Mode).
			Str("prefix", keyPrefix).
			Int("db", db).
			Int("concurrency", cfg.Asynq.Concurrency).
			Int("pool_size", cfg.Asynq.PoolSize).
			Msg("Asynq Redis client initialized")
	}

	return client, nil
}
