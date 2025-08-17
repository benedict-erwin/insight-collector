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
		PoolTimeout:     10 * time.Second, // Timeout when getting connection from pool (prevent blocking)
		MinIdleConns:    2,                // Minimum idle connections to maintain
		MaxIdleConns:    poolSize / 2,     // Maximum idle connections (50% of pool size)

		// Connection timeouts optimization
		DialTimeout:  5 * time.Second, // Connection establishment timeout
		ReadTimeout:  3 * time.Second, // Read operation timeout
		WriteTimeout: 3 * time.Second, // Write operation timeout

		// Retry configuration for reliability
		MaxRetries:      2,                      // Retry failed commands up to 2 times
		MinRetryBackoff: 8 * time.Millisecond,   // Minimum backoff between retries
		MaxRetryBackoff: 512 * time.Millisecond, // Maximum backoff between retries
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
