package callbacklogs

import (
	"encoding/json"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// DELIVERY & INTEGRATION FOCUSED
type (
	CallbackLogs struct {
		// === CORE IDENTIFICATION ===
		TransactionID string `json:"transaction_id"` // Reference ke original transaction
		CallbackType  string `json:"callback_type"`  // transaction_success/transaction_failed/payment_confirmed
		Status        string `json:"status"`         // delivered/failed/timeout

		// === ERROR CATEGORIZATION ===
		ErrorCategory string `json:"error_category"` // http_error/timeout/network_error/client_error/success

		// === BASIC CORRELATION ===
		CallbackID string `json:"callback_id"` // Unique callback identifier

		// === ERROR DETAILS ===
		HTTPStatusCode int    `json:"http_status_code"` // 200/404/500/0 (0 for timeout)
		ErrorMessage   string `json:"error_message"`    // Error description untuk debugging
		ClientResponse string `json:"client_response"`  // Response body dari client (untuk debugging)

		// === MINIMAL PERFORMANCE ===
		DurationMs int `json:"duration_ms"` // Total callback request time
		RetryCount int `json:"retry_count"` // Total retry attempts made

		// === CALLBACK CONTEXT ===
		DestinationURL string                 `json:"destination_url"` // Client callback URL
		Payloads       map[string]interface{} `json:"payloads"`        // JSON untuk callback payload

		// Timestamp
		Timestamp time.Time
	}

	CallbackLogsRequest struct {
		TransactionID  string                 `json:"transaction_id"`
		CallbackType   string                 `json:"callback_type"`
		Status         string                 `json:"status"`
		ErrorCategory  string                 `json:"error_category"`
		CallbackID     string                 `json:"-"` // Excluded from JSON input, will be auto-generated from request_id
		HTTPStatusCode int                    `json:"http_status_code"`
		ErrorMessage   string                 `json:"error_message"`
		ClientResponse string                 `json:"client_response"`
		DurationMs     int                    `json:"duration_ms"`
		RetryCount     int                    `json:"retry_count"`
		DestinationURL string                 `json:"destination_url" validate:"required"`
		Payloads       map[string]interface{} `json:"payloads"`
		Timestamp      time.Time              `json:"time" validate:"required"`
	}
	CallbackLogsResponse struct {
		ID             string                 `json:"id"`
		Time           string                 `json:"time"`
		TransactionID  string                 `json:"transaction_id"`
		CallbackType   string                 `json:"callback_type"`
		Status         string                 `json:"status"`
		ErrorCategory  string                 `json:"error_category"`
		CallbackID     string                 `json:"callback_id"`
		HTTPStatusCode int                    `json:"http_status_code"`
		ErrorMessage   string                 `json:"error_message"`
		ClientResponse string                 `json:"client_response"`
		DurationMs     int                    `json:"duration_ms"`
		RetryCount     int                    `json:"retry_count"`
		DestinationURL string                 `json:"destination_url"`
		Payloads       map[string]interface{} `json:"payloads"`
	}
)

// ToPoint converts CallbackLogs to InfluxDB point with tags and fields
func (cl *CallbackLogs) ToPoint() interface{} {
	// Serialize payloads to JSON string for InfluxDB storage
	var payloadsJSON string
	if len(cl.Payloads) > 0 {
		if jsonBytes, err := json.Marshal(cl.Payloads); err == nil {
			payloadsJSON = string(jsonBytes)
		}
	}
	return influxdb.NewPoint(
		"callback_logs",
		map[string]string{
			"callback_type":  safeString(cl.CallbackType),
			"status":         safeString(cl.Status),
			"error_category": safeString(cl.ErrorCategory),
		},
		map[string]interface{}{
			"transaction_id":   safeString(cl.TransactionID),
			"callback_id":      string(cl.CallbackID),
			"http_status_code": int(cl.HTTPStatusCode),
			"error_message":    string(cl.ErrorMessage),
			"client_response":  string(cl.ClientResponse),
			"duration_ms":      int(cl.DurationMs),
			"retry_count":      int(cl.RetryCount),
			"destination_url":  string(cl.DestinationURL),
			"payloads":         payloadsJSON,
		},
		cl.Timestamp,
	)
}

// GetName returns the measurement name for this entity
func (se *CallbackLogs) GetName() string {
	return "callback_logs"
}

// safeString ensures tag values are never empty (InfluxDB requirement)
func safeString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// MapToCallbackLogsResponse converts raw InfluxDB record to CallbackLogsResponse struct
func MapToCallbackLogsResponse(record map[string]interface{}) CallbackLogsResponse {
	response := CallbackLogsResponse{}

	// Parse time field
	if v, ok := record["_time"]; ok {
		switch timeVal := v.(type) {
		case string:
			response.Time = timeVal
		case time.Time:
			response.Time = timeVal.Format(time.RFC3339)
		}
	}

	// === IDENTITY GROUP ===
	if v, ok := record["callback_type"].(string); ok && v != "" && v != "-" {
		response.CallbackType = v
	}
	if v, ok := record["callback_id"].(string); ok && v != "" && v != "-" {
		response.CallbackID = v
	}

	// === Client Response ===
	if v, ok := record["client_response"].(string); ok && v != "" && v != "-" {
		response.ClientResponse = v
	}
	if v, ok := record["status"].(string); ok && v != "" && v != "-" {
		response.Status = v
	}
	if v, ok := record["error_category"].(string); ok && v != "" && v != "-" {
		response.ErrorCategory = v
	}
	if v, ok := record["error_message"].(string); ok && v != "" && v != "-" {
		response.ErrorMessage = v
	}

	if v, ok := record["http_status_code"]; ok {
		switch code := v.(type) {
		case int64:
			response.HTTPStatusCode = int(code)
		case float64:
			response.HTTPStatusCode = int(code)
		case int:
			response.HTTPStatusCode = code
		}
	}

	if v, ok := record["duration_ms"]; ok {
		switch duration := v.(type) {
		case int64:
			response.DurationMs = int(duration)
		case float64:
			response.DurationMs = int(duration)
		case int:
			response.DurationMs = duration
		}
	}

	if v, ok := record["retry_count"]; ok {
		switch count := v.(type) {
		case int64:
			response.RetryCount = int(count)
		case float64:
			response.RetryCount = int(count)
		case int:
			response.RetryCount = count
		}
	}

	if v, ok := record["destination_url"].(string); ok && v != "" && v != "-" {
		response.DestinationURL = v
	}

	// === MAP/OBJECT FIELDS - deserialize JSON string back to map ===
	if v, ok := record["payloads"].(string); ok && v != "" {
		var payloads map[string]interface{}
		if err := json.Unmarshal([]byte(v), &payloads); err == nil {
			response.Payloads = payloads
		}
	}

	// Generate ID from timestamp and request_id
	if response.Time != "" && response.CallbackID != "" {
		response.ID = utils.CreateRecordID(response.Time, response.CallbackID)
	}

	return response
}
