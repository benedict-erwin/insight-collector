package asynq

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

var client *asynq.Client

// InitClient initializes the Asynq Redis client
func InitClient() error {
	cfg := config.Get()
	redisOpt := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Asynq.DB, // Use Asynq-specific DB from config
	}

	client = asynq.NewClient(redisOpt)
	logger.Info().
		Str("host", cfg.Redis.Host).
		Int("port", cfg.Redis.Port).
		Int("db", cfg.Asynq.DB).
		Msg("Asynq client initialized")

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
	_, err = client.Enqueue(
		task,
		asynq.Queue(queue),
		asynq.Unique(5*time.Minute), // unique taskId for 5minutes
		asynq.TaskID(payload.TaskId),
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

// CloseClient closes the Asynq client connection
func CloseClient() {
	if client != nil {
		if err := client.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close Asynq client")
		} else {
			logger.Info().Msg("Asynq client closed")
		}
		client = nil
	}
}
