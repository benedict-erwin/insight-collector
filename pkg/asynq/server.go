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

	// Create advanced Redis client for server (same optimization as client)
	serverRedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Asynq.DB, // Use Asynq-specific DB from config

		// POOL OPTIMIZATION
		PoolSize:        cfg.Asynq.PoolSize,     // Basic pool size (from config)
		ConnMaxIdleTime: 5 * time.Minute,        // Keep idle connections for 5 minutes (reduce PING frequency)
		ConnMaxLifetime: 30 * time.Minute,       // Refresh connections every 30 minutes (prevent stale connections)
		PoolTimeout:     10 * time.Second,       // Timeout when getting connection from pool (prevent blocking)
		MinIdleConns:    2,                      // Minimum idle connections to maintain
		MaxIdleConns:    cfg.Asynq.PoolSize / 2, // Maximum idle connections (50% of pool size)

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
		Int("pool_size", cfg.Asynq.PoolSize).
		Dur("conn_max_idle_time", 5*time.Minute).
		Dur("conn_max_lifetime", 30*time.Minute).
		Dur("pool_timeout", 10*time.Second).
		Int("min_idle_conns", 2).
		Int("max_idle_conns", cfg.Asynq.PoolSize/2).
		Msg("Asynq server initialized with advanced Redis pool optimization")
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
