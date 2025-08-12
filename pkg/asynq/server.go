package asynq

import (
	"context"
	"fmt"
	"time"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/hibiken/asynq"
)

var server *asynq.Server

// InitServer initializes and configures the Asynq server
func InitServer() *asynq.Server {
	cfg := config.Get()

	// Setup logger scope
	log := logger.WithScope("InitServer")

	// Initialize concurrency from config or runtime setting
	InitConcurrency()

	redisOpt := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Asynq.DB,
		PoolSize: cfg.Asynq.PoolSize,
	}

	server = asynq.NewServer(
		redisOpt,
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
		Msg("Asynq server initialized with dynamic configuration")
	return server
}

// GetServer returns the current Asynq server instance
func GetServer() *asynq.Server {
	return server
}
