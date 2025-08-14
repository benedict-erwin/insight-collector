package asynq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

var (
	client      *asynq.Client
	redisClient *redis.Client
)

// InitClient initializes the Asynq Redis client with advanced pool optimization
func InitClient() error {
	cfg := config.Get()

	// Create advanced Redis client with optimization parameters
	redisClient = redis.NewClient(&redis.Options{
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

	// Create Asynq client using the optimized Redis client
	client = asynq.NewClientFromRedisClient(redisClient)

	logger.Info().
		Str("host", cfg.Redis.Host).
		Int("port", cfg.Redis.Port).
		Int("db", cfg.Asynq.DB).
		Int("pool_size", cfg.Asynq.PoolSize).
		Dur("conn_max_idle_time", 5*time.Minute).
		Dur("conn_max_lifetime", 30*time.Minute).
		Dur("pool_timeout", 10*time.Second).
		Int("min_idle_conns", 2).
		Int("max_idle_conns", cfg.Asynq.PoolSize/2).
		Msg("Asynq client initialized with advanced Redis pool optimization")

	return nil
}

// GetClient returns the current Asynq client instance
func GetClient() *asynq.Client {
	return client
}

// DispathJob enqueue helper function
func DispatchJob(payload *Payload) error {
	// Validate payload first
	if payload == nil {
		return fmt.Errorf("payload cannot be nil")
	}

	// Setup logger scope
	log := logger.WithScope("DispathJob")

	// Process payload
	data, err := json.Marshal(payload.Data)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal example processing payload")
		return err
	}

	// Create new task
	task := asynq.NewTask(payload.TaskType, data)
	client := GetClient()

	if client == nil {
		log.Error().Msg("Asynq client not initialized")
		return fmt.Errorf("queue client not available")
	}

	// Route to appropriate queue
	queue := GetQueueForTaskType(payload.TaskType)

	// Add timeout and reduced uniqueness check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5s timeout for enqueue
	defer cancel()

	_, err = client.EnqueueContext(
		ctx,
		task,
		asynq.Queue(queue),
		asynq.Unique(1*time.Minute), // Reduced uniqueness window from 5min to 1min
		asynq.TaskID(payload.TaskId),
		asynq.Retention(10*time.Minute), // Retain completed tasks for 10min only (reduce memory)
	)
	if err != nil {
		// Duplicate task
		if errors.Is(err, asynq.ErrDuplicateTask) {
			log.Warn().
				Str("taskId", payload.TaskId).
				Str("taskType", payload.TaskType).
				Msg("Duplicate task ignored - already in queue")
			return nil
		}

		// Conflict task
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			log.Warn().
				Str("taskId", payload.TaskId).
				Str("taskType", payload.TaskType).
				Msg("Task ID conflict - duplicate task")
			return nil
		}

		// Other errors
		log.Error().
			Err(err).
			Str("taskId", payload.TaskId).
			Str("taskType", payload.TaskType).
			Msg("Failed to enqueue task")
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	// Success
	log.Info().
		Str("taskId", payload.TaskId).
		Str("taskType", payload.TaskType).
		Str("queue", queue).
		Msg("Task enqueued successfully")

	return nil
}

// CloseClient closes the Asynq client and Redis client connections
func CloseClient() {
	if client != nil {
		if err := client.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close Asynq client")
		} else {
			logger.Info().Msg("Asynq client closed")
		}
		client = nil
	}

	// Close the underlying Redis client (required for NewClientFromRedisClient)
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close Redis client")
		} else {
			logger.Info().Msg("Redis client closed")
		}
		redisClient = nil
	}
}
