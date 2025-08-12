package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/internal/constants"
	exampleEntity "github.com/benedict-erwin/insight-collector/internal/entities/example"
	exampleJob "github.com/benedict-erwin/insight-collector/internal/jobs/example"
	"github.com/benedict-erwin/insight-collector/internal/services/example"
	"github.com/benedict-erwin/insight-collector/pkg/asynq"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/response"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// ExamplePost demonstrates full validation with struct tags and echo.Validator
func ExamplePost(c echo.Context) error {
	var req exampleEntity.ExampleRequest

	// Bind JSON into struct
	if err := c.Bind(&req); err != nil {
		return response.Fail(c, http.StatusBadRequest, 1, "Invalid JSON payload")
	}

	// Validate using echo.Validator (with struct tags)
	if err := c.Validate(&req); err != nil {
		return response.Fail(c, http.StatusBadRequest, 2, "Validation failed: "+err.Error())
	}

	// Process business logic
	result, err := example.ProcessExample(&req)
	if err != nil {
		return response.Fail(c, http.StatusInternalServerError, 3, err.Error())
	}

	return response.Success(c, result)
}

// ExampleGetId handles GET requests with ID parameter
func ExampleGetId(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return response.Fail(c, http.StatusBadRequest, 1, "id is required!")
	}

	data := map[string]interface{}{
		"id": id,
	}

	return response.Success(c, data)
}

// ExampleJob handles POST requests to enqueue background jobs
func ExampleJob(c echo.Context) error {
	rid := constants.GetRequestID(c)

	// Generate unique job ID
	jobID := fmt.Sprintf("job_%d_%s", time.Now().Unix(), rid[:8])

	// Create job payload
	payload := asynq.Payload{
		TaskId:   jobID,
		TaskType: exampleJob.TypeExampleProcessing,
		Data: exampleJob.ExampleProcessingPayload{
			ID:      jobID,
			Message: "Example background processing job",
			UserData: map[string]interface{}{
				"triggered_by": "http_request",
				"endpoint":     "/v1/example/job",
				"timestamp":    utils.NowFormatted(),
			},
			ProcessedAt: utils.Now(),
			RequestID:   rid,
		},
	}

	// Dispatch the job
	err := asynq.DispatchJob(&payload)
	if err != nil {
		logger.Error().
			Err(err).
			Str("job_id", jobID).
			Str("request_id", rid).
			Msg("Failed to enqueue example job")

		return response.Fail(c, http.StatusInternalServerError, 1, "Failed to dispatch job")
	}

	// Return immediate response
	data := map[string]interface{}{
		"message":   "Job dispatched!",
		"job_id":    jobID,
		"timestamp": utils.NowFormatted(),
	}

	return response.Success(c, data)
}
