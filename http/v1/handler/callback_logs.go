package handler

import (
	"crypto/md5"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/benedict-erwin/insight-collector/internal/constants"
	clEntities "github.com/benedict-erwin/insight-collector/internal/entities/callback_logs"
	clJobs "github.com/benedict-erwin/insight-collector/internal/jobs/callback_logs"
	"github.com/benedict-erwin/insight-collector/pkg/asynq"
	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	v2oss "github.com/benedict-erwin/insight-collector/pkg/influxdb/v2-oss"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/response"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
	"github.com/labstack/echo/v4"
)

// SaveCallbackLogs handles saving data for security events
func SaveCallbackLogs(c echo.Context) error {
	var req clEntities.CallbackLogsRequest

	// set logger scope
	log := logger.WithScope("SaveCallbackLogs")

	// Bind JSON into struct
	if err := c.Bind(&req); err != nil {
		return response.FailWithCodeAndMessage(c, constants.CodeInvalidJSON, err.Error())
	}

	// Validate using echo.Validator (with struct tags)
	if err := c.Validate(&req); err != nil {
		return response.FailWithCodeAndMessage(c, constants.CodeValidationFailed, err.Error())
	}

	// Auto-generate CallbackID from request_id
	req.CallbackID = constants.GetRequestID(c)

	// Fallback if GetRequestID returns empty
	if req.CallbackID == "" {
		req.CallbackID = fmt.Sprintf("req-%d-%08x", time.Now().Unix(), rand.Uint32())
	}

	// Generate JobId
	jobID := generateCallbackLogsJobId(&req)

	// Job Payload
	payload := asynq.Payload{
		TaskId:   jobID,
		TaskType: clJobs.TypeCallbackLogsLogging,
		Data: clEntities.CallbackLogsRequest{
			TransactionID:  req.TransactionID,
			CallbackType:   req.CallbackType,
			Status:         req.Status,
			ErrorCategory:  req.ErrorCategory,
			CallbackID:     req.CallbackID,
			HTTPStatusCode: req.HTTPStatusCode,
			ErrorMessage:   req.ErrorMessage,
			ClientResponse: req.ClientResponse,
			DurationMs:     req.DurationMs,
			RetryCount:     req.RetryCount,
			DestinationURL: req.DestinationURL,
			Payloads:       req.Payloads,
			Timestamp:      req.Timestamp,
		},
	}

	// Dispatch the job
	err := asynq.DispatchJob(&payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("callback_id", req.CallbackID).
			Str("job_id", jobID).
			Msg("Failed to enqueue job")

		return response.FailWithCodeAndMessage(c, constants.CodeInternalError, "Failed to dispatch job")
	}

	// Return immediate response
	data := map[string]interface{}{
		"message":   "Job dispatched!",
		"job_id":    jobID,
		"timestamp": utils.NowFormatted(),
	}

	return response.Success(c, data)
}

// generateCallbackLogsJobId for unique jobid
func generateCallbackLogsJobId(payload *clEntities.CallbackLogsRequest) string {
	// Clean endpoint
	safeEndpoint := strings.ReplaceAll(payload.DestinationURL, "/", "_")
	safeEndpoint = strings.ReplaceAll(safeEndpoint, ":", "_")

	// Concat to make some unique activity
	uniqueId := fmt.Sprintf("%s-%s-%s-%s-%d",
		payload.TransactionID,
		payload.CallbackID,
		payload.CallbackType,
		safeEndpoint,
		payload.Timestamp.Unix(),
	)

	hash := md5.Sum([]byte(uniqueId))
	return fmt.Sprintf("cl_%x", hash[:8])
}

// ListCallbackLogs handles paginated listing of security events
func ListCallbackLogs(c echo.Context) error {
	var req v2oss.PaginationRequest

	// set logger scope
	log := logger.WithScope("ListCallbackLogs")

	// Bind JSON into struct
	if err := c.Bind(&req); err != nil {
		return response.FailWithCodeAndMessage(c, constants.CodeInvalidJSON, err.Error())
	}

	// Validate using echo.Validator (with struct tags)
	if err := c.Validate(&req); err != nil {
		return response.FailWithCodeAndMessage(c, constants.CodeValidationFailed, err.Error())
	}

	// Get InfluxDB client and configuration
	client := influxdb.GetCurrentClient()
	if client == nil {
		log.Warn().Msg("InfluxDB client not initialized")
		return response.FailWithCode(c, constants.CodeInfluxDBError)
	}

	// Type assert to v2-oss client (assuming v2-oss is default)
	v2ossClient, ok := client.(*v2oss.Client)
	if !ok {
		log.Warn().Msg("Invalid InfluxDB client type")
		return response.FailWithCode(c, constants.CodeInfluxDBError)
	}

	// Get query config for security events
	queryConfig := clEntities.GetQueryConfig()

	// Create query builder
	qb := v2oss.NewQueryBuilder(queryConfig)

	// Get total count using client and bucket
	totalRecords := qb.GetTotalCount(&req, v2ossClient)

	// Execute data query and get results using client and bucket
	results, err := qb.ExecuteDataQuery(&req, v2ossClient)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute data query")
		return response.FailWithCode(c, constants.CodeInfluxDBError)
	}

	// Convert raw results to structured response
	var records []clEntities.CallbackLogsResponse
	for _, record := range results {
		records = append(records, clEntities.MapToCallbackLogsResponse(record))
	}

	// Get cursor-based pagination info
	paginationInfo := qb.GetPaginationInfo(&req, results, totalRecords)

	// Build response
	responseData := v2oss.PaginationResponse{
		Data:       records,
		Pagination: paginationInfo,
	}
	return response.Success(c, responseData)
}

func DetailCallbackLogs(c echo.Context) error {
	// Get encoded ID from path parameter
	encodedID := c.Param("id")

	// set logger scope
	log := logger.WithScope("DetailCallbackLogs")

	// Decode timestamp and request_id
	// time RFC3339 format: "2025-08-06T12:30:00Z"
	// request_id e.g., "req-1234567890-abcdef12"
	timestamp, requestID, err := utils.ParseRecordID(encodedID)
	if err != nil {
		log.Error().Err(err).Str("encoded_id", encodedID).Msg("Invalid record ID format")
		return response.FailWithCodeAndMessage(c, constants.CodeUnprocessable, "Invalid record ID format")
	}

	// Validate timestamp format
	if _, parseErr := time.Parse(time.RFC3339, timestamp); parseErr != nil {
		return response.FailWithCodeAndMessage(c, constants.CodeUnprocessable, "Invalid timestamp format")
	}

	// Get InfluxDB client and configuration
	client := influxdb.GetCurrentClient()
	if client == nil {
		log.Error().Str("request_id", requestID).Msg("InfluxDB client not initialized")
		return response.FailWithCode(c, constants.CodeInfluxDBError)
	}

	// Type assert to v2-oss client (assuming v2-oss is default)
	v2ossClient, ok := client.(*v2oss.Client)
	if !ok {
		log.Error().Str("request_id", requestID).Msg("Invalid InfluxDB client type")
		return response.FailWithCode(c, constants.CodeInfluxDBError)
	}

	// Get query config for security events
	queryConfig := clEntities.GetQueryConfig()

	// Create query builder
	qb := v2oss.NewQueryBuilder(queryConfig)

	// Get record by timestamp & request_id
	record, err := qb.GetByTimestampAndUniqueID(timestamp, "callback_id", requestID, v2ossClient)
	if err != nil {
		log.Error().Err(err).
			Str("timestamp", timestamp).
			Str("request_id", requestID).
			Msg("Failed to retrieve security events")
		return response.FailWithCodeAndMessage(c, constants.CodeNotFound, "Record not found")
	}

	// Convert raw record to structured response
	structuredResponse := clEntities.MapToCallbackLogsResponse(record)

	// Success
	return response.Success(c, structuredResponse)
}
