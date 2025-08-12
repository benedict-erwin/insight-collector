package callbacklogs

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	callbacklogs "github.com/benedict-erwin/insight-collector/internal/entities/callback_logs"
	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// Job processor function
func HandleCallbackLogsLogging(ctx context.Context, t *asynq.Task) error {
	var cl callbacklogs.CallbackLogs
	var req callbacklogs.CallbackLogsRequest

	// Logger scope
	log := logger.WithScope(TypeCallbackLogsLogging)

	// Unmarshal request payload
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal payload")
		return err
	}

	// Convert CallbackLogsRequest to CallbackLogs
	cl.TransactionID = req.TransactionID
	cl.CallbackType = req.CallbackType
	cl.Status = req.Status
	cl.ErrorCategory = req.ErrorCategory
	cl.CallbackID = req.CallbackID
	cl.HTTPStatusCode = req.HTTPStatusCode
	cl.ErrorMessage = req.ErrorMessage
	cl.ClientResponse = req.ClientResponse
	cl.DurationMs = req.DurationMs
	cl.RetryCount = req.RetryCount
	cl.DestinationURL = req.DestinationURL
	cl.Payloads = req.Payloads
	cl.Timestamp = req.Timestamp

	// point
	point := cl.ToPoint()
	err := influxdb.WritePoint(point)
	if err != nil {
		return err
	}

	log.Info().
		Str("task_id", t.ResultWriter().TaskID()).
		Str("task_type", t.Type()).
		Str("measurements", cl.GetName()).
		Msg("Job completed successfully")

	return nil
}
