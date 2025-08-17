package asynq

import (
	"context"
	"fmt"
	"time"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

var (
	server            *asynq.Server
	serverRedisClient *redis.Client
)

// InitServer initializes and configures the Asynq server with advanced Redis pool optimization
func InitServer() *asynq.Server {
	cfg := config.Get()

	// Setup logger scope
	log := logger.WithScope("InitServer")

	// Initialize concurrency from config or runtime setting
	InitConcurrency()

	// Get pool size from Redis pools config with fallback to Asynq config for backward compatibility
	var poolSize int
	var maxLifetime, idleTimeout time.Duration = 30 * time.Minute, 5 * time.Minute

	if cfg.Redis.Pools != nil {
		if asynqPool, exists := cfg.Redis.Pools["asynq"]; exists {
			poolSize = asynqPool.Size
			if asynqPool.MaxLifetime != "" {
				if parsed, err := time.ParseDuration(asynqPool.MaxLifetime); err == nil {
					maxLifetime = parsed
				}
			}
			if asynqPool.IdleTimeout != "" {
				if parsed, err := time.ParseDuration(asynqPool.IdleTimeout); err == nil {
					idleTimeout = parsed
				}
			}
		} else if defaultPool, exists := cfg.Redis.Pools["default"]; exists {
			poolSize = defaultPool.Size
		} else {
			poolSize = cfg.Asynq.PoolSize // Fallback to old config
		}
	} else {
		poolSize = cfg.Asynq.PoolSize // Fallback to old config
	}

	// Create advanced Redis client for server (same optimization as client)
	serverRedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Asynq.DB, // Use Asynq-specific DB from config

		// POOL OPTIMIZATION
		PoolSize:        poolSize,         // Pool size from Redis pools config
		ConnMaxIdleTime: idleTimeout,      // Configurable idle timeout
		ConnMaxLifetime: maxLifetime,      // Configurable max lifetime
		PoolTimeout:     3 * time.Second,  // Reduced timeout to prevent blocking during high load
		MinIdleConns:    poolSize / 10,    // 10% minimum idle connections
		MaxIdleConns:    poolSize / 3,     // 33% maximum idle connections for better reuse

		// Aggressive connection timeouts for high-load performance
		DialTimeout:  2 * time.Second, // Faster connection establishment
		ReadTimeout:  1 * time.Second, // Faster read operations
		WriteTimeout: 1 * time.Second, // Faster write operations

		// Reduced retries for high-load performance
		MaxRetries:      1,                      // Single retry to prevent cascading delays
		MinRetryBackoff: 5 * time.Millisecond,   // Faster retry backoff
		MaxRetryBackoff: 100 * time.Millisecond, // Shorter maximum backoff
	})

	// Create Asynq server using the optimized Redis client
	server = asynq.NewServerFromRedisClient(
		serverRedisClient,
		asynq.Config{
			Concurrency:     GetConcurrency(),
			Queues:          GenerateQueues(),
			ShutdownTimeout: 30 * time.Second, // Wait 30s for running tasks
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Error().
					Err(err).
					Str("task_type", task.Type()).
					Bytes("payload", task.Payload()).
					Msg("Task processing failed")
			}),
		},
	)

	// Store server reference for graceful restart
	SetCurrentServer(server)

	log.Info().
		Int("concurrency", GetConcurrency()).
		Interface("queues", GenerateQueues()).
		Int("pool_size", poolSize).
		Dur("conn_max_idle_time", idleTimeout).
		Dur("conn_max_lifetime", maxLifetime).
		Dur("pool_timeout", 10*time.Second).
		Int("min_idle_conns", 2).
		Int("max_idle_conns", poolSize/2).
		Msg("Asynq server initialized with advanced Redis pool optimization")

	// Verify actual concurrency setting without non-serializable fields
	log.Warn().
		Int("actual_concurrency", GetConcurrency()).
		Int("config_concurrency", cfg.Asynq.Concurrency).
		Interface("queues", GenerateQueues()).
		Dur("shutdown_timeout", 30*time.Second).
		Bool("concurrency_match", GetConcurrency() == cfg.Asynq.Concurrency).
		Msg("Asynq server concurrency verification")

	// Additional debug: Verify server will process jobs with correct concurrency
	log.Info().
		Str("server_type", "NewServerFromRedisClient").
		Int("max_concurrent_jobs", GetConcurrency()).
		Int("queue_count", len(GenerateQueues())).
		Msg("Asynq server ready to process jobs with full concurrency")
	return server
}

// GetServer returns the current Asynq server instance
func GetServer() *asynq.Server {
	return server
}

// CloseServer closes the Asynq server and underlying Redis client connections
func CloseServer() {
	if server != nil {
		server.Shutdown()
		logger.Info().Msg("Asynq server shut down")
		server = nil
	}

	// Close the underlying Redis client (required for NewServerFromRedisClient)
	if serverRedisClient != nil {
		if err := serverRedisClient.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close server Redis client")
		} else {
			logger.Info().Msg("Server Redis client closed")
		}
		serverRedisClient = nil
	}
}
