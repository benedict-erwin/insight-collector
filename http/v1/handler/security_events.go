package handler

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/benedict-erwin/insight-collector/internal/constants"
	seEntities "github.com/benedict-erwin/insight-collector/internal/entities/security_events"
	seJobs "github.com/benedict-erwin/insight-collector/internal/jobs/security_events"
	"github.com/benedict-erwin/insight-collector/pkg/asynq"
	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	v2oss "github.com/benedict-erwin/insight-collector/pkg/influxdb/v2-oss"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/response"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// SaveSecurityEvents handles saving data for security events
func SaveSecurityEvents(c echo.Context) error {
	var req seEntities.SecurityEventsRequest

	// set logger scope
	log := logger.WithScope("SaveSecurityEvents")

	// Bind JSON into struct
	if err := c.Bind(&req); err != nil {
		return response.FailWithCodeAndMessage(c, constants.CodeInvalidJSON, err.Error())
	}

	// Validate using echo.Validator (with struct tags)
	if err := c.Validate(&req); err != nil {
		return response.FailWithCodeAndMessage(c, constants.CodeValidationFailed, err.Error())
	}

	// Generate JobId
	jobID := generateSecurityEventsJobId(&req)

	// Job Payload
	payload := asynq.Payload{
		TaskId:   jobID,
		TaskType: seJobs.TypeSecurityEventsLogging,
		Data: seEntities.SecurityEventsRequest{
			UserID:              req.UserID,
			SessionID:           req.SessionID,
			IdentifierType:      req.IdentifierType,
			EventType:           req.EventType,
			Severity:            req.Severity,
			AuthStage:           req.AuthStage,
			ActionTaken:         req.ActionTaken,
			DetectionMethod:     req.DetectionMethod,
			Channel:             req.Channel,
			EndpointGroup:       req.EndpointGroup,
			Method:              req.Method,
			RequestID:           req.RequestID,
			TraceID:             req.TraceID,
			IdentifierValue:     req.IdentifierType,
			AttemptCount:        req.AttemptCount,
			RiskScore:           req.RiskScore,
			ConfidenceScore:     req.ConfidenceScore,
			PreviousSuccessTime: req.PreviousSuccessTime,
			AffectedResource:    req.AffectedResource,
			DurationMs:          req.DurationMs,
			ResponseCode:        req.ResponseCode,
			IPAddress:           req.IPAddress,
			UserAgent:           req.UserAgent,
			AppVersion:          req.AppVersion,
			Endpoint:            req.Endpoint,
			Details:             req.Details,
			Timestamp:           req.Timestamp,
		},
	}

	// Dispatch the job
	err := asynq.DispatchJob(&payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("request_id", req.RequestID).
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

// generateSecurityEventsJobId for unique jobid
func generateSecurityEventsJobId(payload *seEntities.SecurityEventsRequest) string {
	// Clean endpoint
	safeEndpoint := strings.ReplaceAll(payload.Endpoint, "/", "_")
	safeEndpoint = strings.ReplaceAll(safeEndpoint, ":", "_")

	// Concat to make some unique activity
	uniqueId := fmt.Sprintf("%s-%s-%s-%s-%d",
		payload.UserID,
		payload.SessionID,
		payload.EventType,
		safeEndpoint,
		payload.Timestamp.Unix(),
	)

	hash := md5.Sum([]byte(uniqueId))
	return fmt.Sprintf("se_%x", hash[:8])
}

// ListSecurityEvents handles paginated listing of security events
func ListSecurityEvents(c echo.Context) error {
	var req v2oss.PaginationRequest

	// set logger scope
	log := logger.WithScope("ListSecurityEvents")

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
	queryConfig := seEntities.GetQueryConfig()

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
	var records []seEntities.SecurityEventsResponse
	for _, record := range results {
		records = append(records, seEntities.MapToSecurityEventsResponse(record))
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

func DetailSecurityEvents(c echo.Context) error {
	// Get encoded ID from path parameter
	encodedID := c.Param("id")

	// set logger scope
	log := logger.WithScope("DetailSecurityEvents")

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
	queryConfig := seEntities.GetQueryConfig()

	// Create query builder
	qb := v2oss.NewQueryBuilder(queryConfig)

	// Get record by timestamp & request_id
	record, err := qb.GetByTimestampAndUniqueID(timestamp, "request_id", requestID, v2ossClient)
	if err != nil {
		log.Error().Err(err).
			Str("timestamp", timestamp).
			Str("request_id", requestID).
			Msg("Failed to retrieve security events")
		return response.FailWithCodeAndMessage(c, constants.CodeNotFound, "Record not found")
	}

	// Convert raw record to structured response
	structuredResponse := seEntities.MapToSecurityEventsResponse(record)

	// Success
	return response.Success(c, structuredResponse)
}
