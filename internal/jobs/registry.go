package jobs

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/benedict-erwin/insight-collector/internal/constants"
	cl "github.com/benedict-erwin/insight-collector/internal/jobs/callback_logs"
	"github.com/benedict-erwin/insight-collector/internal/jobs/example"
	se "github.com/benedict-erwin/insight-collector/internal/jobs/security_events"
	te "github.com/benedict-erwin/insight-collector/internal/jobs/transaction_events"
	ua "github.com/benedict-erwin/insight-collector/internal/jobs/user_activities"
)

// JobRegistration holds job metadata for registration and worker generation
type JobRegistration struct {
	TaskType string                                   `json:"task_type"`
	Handler  func(context.Context, *asynq.Task) error `json:"-"` // Not serialized
	Queue    string                                   `json:"queue"`
}

// RegisterHandlers registers all job handlers with the asynq server mux and returns job metadata
func RegisterHandlers(mux *asynq.ServeMux) ([]JobRegistration, error) {
	jobs := []JobRegistration{
		// Critical
		{
			TaskType: ua.TypeUserActivitiesLogging,
			Handler:  ua.HandleUserActivitiesLogging,
			Queue:    constants.QueueCritical,
		},
		{
			TaskType: se.TypeSecurityEventsLogging,
			Handler:  se.HandleSecurityEventsLogging,
			Queue:    constants.QueueCritical,
		},
		{
			TaskType: te.TypeTransactionEventsLogging,
			Handler:  te.HandleTransactionEventsLogging,
			Queue:    constants.QueueCritical,
		},
		{
			TaskType: cl.TypeCallbackLogsLogging,
			Handler:  cl.HandleCallbackLogsLogging,
			Queue:    constants.QueueCritical,
		},

		// Default

		// Low
		{
			TaskType: example.TypeExampleProcessing,
			Handler:  example.HandleExampleProcessing,
			Queue:    constants.QueueLow,
		},
	}

	// Validate queue names
	for _, job := range jobs {
		if !constants.IsValidQueue(job.Queue) {
			return nil, fmt.Errorf("invalid queue '%s' for job '%s'. Valid queues: %v",
				job.Queue, job.TaskType, constants.GetAllQueues())
		}
	}

	// Register handlers with mux (if provided)
	if mux != nil {
		for _, job := range jobs {
			mux.HandleFunc(job.TaskType, job.Handler)
		}
	}

	return jobs, nil
}

// GetRegisteredJobs returns job metadata without handlers (for worker generation)
func GetRegisteredJobs() ([]JobRegistration, error) {
	return RegisterHandlers(nil) // No mux = no registration, just return metadata
}
