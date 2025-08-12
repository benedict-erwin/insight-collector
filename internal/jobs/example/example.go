package example

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/hibiken/asynq"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// Task payload
type ExampleProcessingPayload struct {
	ID          string                 `json:"id"`
	Message     string                 `json:"message"`
	UserData    map[string]interface{} `json:"user_data"`
	ProcessedAt time.Time              `json:"processed_at"`
	RequestID   string                 `json:"request_id"`
}

// Job processor function
func HandleExampleProcessing(ctx context.Context, t *asynq.Task) error {
	var payload ExampleProcessingPayload
	log := logger.WithScope(TypeExampleProcessing)
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal example processing payload")
		return err
	}

	log.Info().
		Str("id", payload.ID).
		Str("request_id", payload.RequestID).
		Msg("Starting example processing job")

	// Random delay 3-5 seconds
	delay := time.Duration(rand.Intn(3)+3) * time.Second
	log.Info().
		Str("id", payload.ID).
		Dur("delay", delay).
		Msg("Processing with random delay")

	// Simulate processing
	select {
	case <-time.After(delay):
		// Completed
	case <-ctx.Done():
		// Cancelled
		log.Warn().
			Str("id", payload.ID).
			Msg("Example processing job cancelled")
		return ctx.Err()
	}

	// Log completion
	log.Info().
		Str("id", payload.ID).
		Str("message", payload.Message).
		Str("request_id", payload.RequestID).
		Str("completed_at", utils.NowFormatted()).
		Interface("user_data", payload.UserData).
		Dur("processing_duration", delay).
		Msg("Example processing job completed successfully")

	return nil
}
