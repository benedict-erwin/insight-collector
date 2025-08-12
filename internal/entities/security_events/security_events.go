package securityevents

import (
	"encoding/json"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// SECURITY & RISK FOCUSED
type (
	SecurityEvents struct {
		// === IDENTITY GROUP ===
		UserID         string `json:"user_id"`         // User identifier (empty string if pre-auth)
		SessionID      string `json:"session_id"`      // Session correlation key
		IdentifierType string `json:"identifier_type"` // user_id/username/email/phone/device_id/anonymous

		// === SECURITY CONTEXT GROUP ===
		EventType       string `json:"event_type"`       // failed_login/suspicious_login/device_change/brute_force/account_lockout
		Severity        string `json:"severity"`         // info/warning/critical/alert
		AuthStage       string `json:"auth_stage"`       // pre_auth/post_auth/session_management
		ActionTaken     string `json:"action_taken"`     // none/alert_sent/account_locked/session_terminated/investigation_triggered
		DetectionMethod string `json:"detection_method"` // rule_based/ml_model/manual_review/threshold_based

		// === TECHNICAL CONTEXT GROUP ===
		DeviceType    string `json:"device_type"`    // desktop/mobile/tablet
		OS            string `json:"os"`             // windows/linux/ios
		Browser       string `json:"browser"`        // chrome/firefox/safari
		Channel       string `json:"channel"`        // web/mobile_app/api
		EndpointGroup string `json:"endpoint_group"` // auth_api/transfer_api/payment_api/account_api/other_api
		Method        string `json:"method"`         // GET/POST/PUT/DELETE

		// === GEOGRAPHIC GROUP ===
		GeoCountry string `json:"geo_country"` // ID/SG/MY/TH/US/PH

		// === CORRELATION GROUP ===
		RequestID       string `json:"request_id"`       // Request correlation ID
		TraceID         string `json:"trace_id"`         // Distributed tracing ID
		IdentifierValue string `json:"identifier_value"` // Value identifier

		// === SECURITY METRICS GROUP ===
		AttemptCount        int     `json:"attempt_count"`         // Number of attempts (e.g. brute force, failed login)
		RiskScore           float64 `json:"risk_score"`            // Risk assessment score (0.0 - 1.0)
		ConfidenceScore     float64 `json:"confidence_score"`      // ML model confidence score (0.0 - 1.0)
		PreviousSuccessTime int64   `json:"previous_success_time"` // Epoch timestamp of last successful authentication
		AffectedResource    string  `json:"affected_resource"`     // Resource affected (account_id/transaction_id/session_id)

		// === PERFORMANCE METRICS GROUP ===
		DurationMs   int `json:"duration_ms"`   // Security processing time in milliseconds
		ResponseCode int `json:"response_code"` // HTTP response code

		// === DETECTION & CONTEXT GROUP ===
		IsBot bool `json:"is_bot"` // Automated threat detection flag

		// === NETWORK & CLIENT CONTEXT GROUP ===
		IPAddress  string `json:"ip_address"`  // Source IP address
		UserAgent  string `json:"user_agent"`  // Client user agent string
		AppVersion string `json:"app_version"` // Application version
		Endpoint   string `json:"endpoint"`    // Full endpoint path for security context

		// === GEOGRAPHIC DETAILS GROUP ===
		GeoCity        string `json:"geo_city"`        // City from IP geolocation
		GeoCoordinates string `json:"geo_coordinates"` // Latitude,Longitude format
		GeoTimezone    string `json:"geo_timezone"`    // Timezone from geolocation
		GeoPostal      string `json:"geo_postal"`      // Postal code from geolocation
		GeoISP         string `json:"geo_isp"`         // Internet service provider information

		// === METADATA GROUP ===
		OSVersion      string                 `json:"os_version"`      // OS version
		BrowserVersion string                 `json:"browser_version"` // Browser version
		Details        map[string]interface{} `json:"details"`

		// Timestamp
		Timestamp time.Time
	}

	SecurityEventsRequest struct {
		UserID              string                 `json:"user_id"`
		SessionID           string                 `json:"session_id"`
		IdentifierType      string                 `json:"identifier_type"`
		EventType           string                 `json:"event_type" validate:"required"`
		Severity            string                 `json:"severity" validate:"required"`
		AuthStage           string                 `json:"auth_stage" validate:"required"`
		ActionTaken         string                 `json:"action_taken" validate:"required"`
		DetectionMethod     string                 `json:"detection_method"`
		Channel             string                 `json:"channel"`
		EndpointGroup       string                 `json:"endpoint_group"`
		Method              string                 `json:"method" validate:"required"`
		RequestID           string                 `json:"request_id"`
		TraceID             string                 `json:"trace_id"`
		IdentifierValue     string                 `json:"identifier_value"`
		AttemptCount        int                    `json:"attempt_count"`
		RiskScore           float64                `json:"risk_score"`
		ConfidenceScore     float64                `json:"confidence_score"`
		PreviousSuccessTime int64                  `json:"previous_success_time"`
		AffectedResource    string                 `json:"affected_resource"`
		DurationMs          int                    `json:"duration_ms"`
		ResponseCode        int                    `json:"response_code"`
		IPAddress           string                 `json:"ip_address" validate:"required"`
		UserAgent           string                 `json:"user_agent" validate:"required"`
		AppVersion          string                 `json:"app_version"`
		Endpoint            string                 `json:"endpoint" validate:"required"`
		Details             map[string]interface{} `json:"details"`
		Timestamp           time.Time              `json:"time" validate:"required"`
	}

	SecurityEventsResponse struct {
		ID                  string                 `json:"id"`
		Time                string                 `json:"time"`
		UserID              string                 `json:"user_id"`
		SessionID           string                 `json:"session_id"`
		IdentifierType      string                 `json:"identifier_type"`
		EventType           string                 `json:"event_type"`
		Severity            string                 `json:"severity"`
		AuthStage           string                 `json:"auth_stage"`
		ActionTaken         string                 `json:"action_taken"`
		DetectionMethod     string                 `json:"detection_method"`
		DeviceType          string                 `json:"device_type"`
		OS                  string                 `json:"os"`
		Browser             string                 `json:"browser"`
		Channel             string                 `json:"channel"`
		EndpointGroup       string                 `json:"endpoint_group"`
		Method              string                 `json:"method"`
		GeoCountry          string                 `json:"geo_country"`
		RequestID           string                 `json:"request_id"`
		TraceID             string                 `json:"trace_id"`
		IdentifierValue     string                 `json:"identifier_value"`
		AttemptCount        int                    `json:"attempt_count"`
		RiskScore           float64                `json:"risk_score"`
		ConfidenceScore     float64                `json:"confidence_score"`
		PreviousSuccessTime int64                  `json:"previous_success_time"`
		AffectedResource    string                 `json:"affected_resource"`
		DurationMs          int                    `json:"duration_ms"`
		ResponseCode        int                    `json:"response_code"`
		IsBot               bool                   `json:"is_bot"`
		IPAddress           string                 `json:"ip_address"`
		UserAgent           string                 `json:"user_agent"`
		AppVersion          string                 `json:"app_version"`
		Endpoint            string                 `json:"endpoint"`
		GeoCity             string                 `json:"geo_city"`
		GeoCoordinates      string                 `json:"geo_coordinates"`
		GeoTimezone         string                 `json:"geo_timezone"`
		GeoPostal           string                 `json:"geo_postal"`
		GeoISP              string                 `json:"geo_isp"`
		OSVersion           string                 `json:"os_version"`
		BrowserVersion      string                 `json:"browser_version"`
		Details             map[string]interface{} `json:"details"`
	}
)

// ToPoint converts SecurityEvents to InfluxDB point with tags and fields
func (se *SecurityEvents) ToPoint() interface{} {
	// Serialize details to JSON string for InfluxDB storage
	var detailsJSON string
	if len(se.Details) > 0 {
		if jsonBytes, err := json.Marshal(se.Details); err == nil {
			detailsJSON = string(jsonBytes)
		}
	}

	return influxdb.NewPoint(
		"security_events",
		map[string]string{
			"user_id":          safeString(se.UserID),
			"session_id":       safeString(se.SessionID),
			"identifier_type":  safeString(se.IdentifierType),
			"event_type":       safeString(se.EventType),
			"severity":         safeString(se.Severity),
			"auth_stage":       safeString(se.AuthStage),
			"action_taken":     safeString(se.ActionTaken),
			"detection_method": safeString(se.DetectionMethod),
			"device_type":      safeString(se.DeviceType),
			"os":               safeString(se.OS),
			"browser":          safeString(se.Browser),
			"channel":          safeString(se.Channel),
			"endpoint_group":   safeString(se.EndpointGroup),
			"method":           safeString(se.Method),
			"geo_country":      safeString(se.GeoCountry),
		},
		map[string]interface{}{
			"request_id":            safeString(se.RequestID),
			"trace_id":              safeString(se.TraceID),
			"identifier_value":      safeString(se.IdentifierValue),
			"attempt_count":         int(se.AttemptCount),
			"risk_score":            float64(se.RiskScore),
			"confidence_score":      float64(se.ConfidenceScore),
			"previous_success_time": int64(se.PreviousSuccessTime),
			"affected_resource":     safeString(se.AffectedResource),
			"duration_ms":           int(se.DurationMs),
			"response_code":         int(se.ResponseCode),
			"is_bot":                bool(se.IsBot),
			"ip_address":            safeString(se.IPAddress),
			"user_agent":            safeString(se.UserAgent),
			"app_version":           safeString(se.AppVersion),
			"endpoint":              safeString(se.Endpoint),
			"geo_city":              safeString(se.GeoCity),
			"geo_coordinates":       safeString(se.GeoCoordinates),
			"geo_timezone":          safeString(se.GeoTimezone),
			"geo_postal":            safeString(se.GeoPostal),
			"geo_isp":               safeString(se.GeoISP),
			"os_version":            safeString(se.OSVersion),
			"browser_version":       safeString(se.BrowserVersion),
			"details":               detailsJSON,
		},
		se.Timestamp,
	)
}

// GetName returns the measurement name for this entity
func (se *SecurityEvents) GetName() string {
	return "security_events"
}

// safeString ensures tag values are never empty (InfluxDB requirement)
func safeString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// MapToSecurityEventsResponse converts raw InfluxDB record to SecurityEventsResponse struct
func MapToSecurityEventsResponse(record map[string]interface{}) SecurityEventsResponse {
	response := SecurityEventsResponse{}

	// Parse time field
	if v, ok := record["_time"]; ok {
		switch timeVal := v.(type) {
		case string:
			response.Time = timeVal
		case time.Time:
			response.Time = timeVal.Format(time.RFC3339)
		}
	}

	// Core identity fields
	if v, ok := record["user_id"].(string); ok && v != "" && v != "-" {
		response.UserID = v
	}
	if v, ok := record["session_id"].(string); ok && v != "" && v != "-" {
		response.SessionID = v
	}
	if v, ok := record["identifier_type"].(string); ok && v != "" && v != "-" {
		response.IdentifierType = v
	}

	// Security context fields
	if v, ok := record["event_type"].(string); ok && v != "" && v != "-" {
		response.EventType = v
	}
	if v, ok := record["severity"].(string); ok && v != "" && v != "-" {
		response.Severity = v
	}
	if v, ok := record["auth_stage"].(string); ok && v != "" && v != "-" {
		response.AuthStage = v
	}
	if v, ok := record["action_taken"].(string); ok && v != "" && v != "-" {
		response.ActionTaken = v
	}
	if v, ok := record["detection_method"].(string); ok && v != "" && v != "-" {
		response.DetectionMethod = v
	}

	// Technical context fields
	if v, ok := record["device_type"].(string); ok && v != "" && v != "-" {
		response.DeviceType = v
	}
	if v, ok := record["os"].(string); ok && v != "" && v != "-" {
		response.OS = v
	}
	if v, ok := record["browser"].(string); ok && v != "" && v != "-" {
		response.Browser = v
	}
	if v, ok := record["channel"].(string); ok && v != "" && v != "-" {
		response.Channel = v
	}
	if v, ok := record["endpoint_group"].(string); ok && v != "" && v != "-" {
		response.EndpointGroup = v
	}
	if v, ok := record["method"].(string); ok && v != "" && v != "-" {
		response.Method = v
	}

	// Geographic fields
	if v, ok := record["geo_country"].(string); ok && v != "" && v != "-" {
		response.GeoCountry = v
	}
	if v, ok := record["geo_city"].(string); ok && v != "" && v != "-" {
		response.GeoCity = v
	}
	if v, ok := record["geo_coordinates"].(string); ok && v != "" && v != "-" {
		response.GeoCoordinates = v
	}
	if v, ok := record["geo_timezone"].(string); ok && v != "" && v != "-" {
		response.GeoTimezone = v
	}
	if v, ok := record["geo_postal"].(string); ok && v != "" && v != "-" {
		response.GeoPostal = v
	}
	if v, ok := record["geo_isp"].(string); ok && v != "" && v != "-" {
		response.GeoISP = v
	}

	// Correlation fields
	if v, ok := record["request_id"].(string); ok && v != "" && v != "-" {
		response.RequestID = v
	}
	if v, ok := record["trace_id"].(string); ok && v != "" && v != "-" {
		response.TraceID = v
	}
	if v, ok := record["identifier_value"].(string); ok && v != "" && v != "-" {
		response.IdentifierValue = v
	}
	if v, ok := record["affected_resource"].(string); ok && v != "" && v != "-" {
		response.AffectedResource = v
	}

	// Network and client context fields
	if v, ok := record["ip_address"].(string); ok && v != "" && v != "-" {
		response.IPAddress = v
	}
	if v, ok := record["user_agent"].(string); ok && v != "" && v != "-" {
		response.UserAgent = v
	}
	if v, ok := record["app_version"].(string); ok && v != "" && v != "-" {
		response.AppVersion = v
	}
	if v, ok := record["endpoint"].(string); ok && v != "" && v != "-" {
		response.Endpoint = v
	}

	// Version fields
	if v, ok := record["os_version"].(string); ok && v != "" && v != "-" {
		response.OSVersion = v
	}
	if v, ok := record["browser_version"].(string); ok && v != "" && v != "-" {
		response.BrowserVersion = v
	}

	// Security metrics - numeric fields with type conversion
	if v, ok := record["attempt_count"]; ok {
		switch count := v.(type) {
		case int64:
			response.AttemptCount = int(count)
		case float64:
			response.AttemptCount = int(count)
		case int:
			response.AttemptCount = count
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

	if v, ok := record["response_code"]; ok {
		switch code := v.(type) {
		case int64:
			response.ResponseCode = int(code)
		case float64:
			response.ResponseCode = int(code)
		case int:
			response.ResponseCode = code
		}
	}

	if v, ok := record["previous_success_time"]; ok {
		switch timestamp := v.(type) {
		case int64:
			response.PreviousSuccessTime = timestamp
		case float64:
			response.PreviousSuccessTime = int64(timestamp)
		case int:
			response.PreviousSuccessTime = int64(timestamp)
		}
	}

	// Float64 fields - security scores
	if v, ok := record["risk_score"]; ok {
		switch score := v.(type) {
		case float64:
			response.RiskScore = score
		case float32:
			response.RiskScore = float64(score)
		case int64:
			response.RiskScore = float64(score)
		case int:
			response.RiskScore = float64(score)
		}
	}

	if v, ok := record["confidence_score"]; ok {
		switch score := v.(type) {
		case float64:
			response.ConfidenceScore = score
		case float32:
			response.ConfidenceScore = float64(score)
		case int64:
			response.ConfidenceScore = float64(score)
		case int:
			response.ConfidenceScore = float64(score)
		}
	}

	// Boolean fields
	if v, ok := record["is_bot"]; ok {
		switch bot := v.(type) {
		case bool:
			response.IsBot = bot
		case string:
			response.IsBot = bot == "true" || bot == "1"
		}
	}

	// Map/Object fields - deserialize JSON string back to map
	if v, ok := record["details"].(string); ok && v != "" {
		var details map[string]interface{}
		if err := json.Unmarshal([]byte(v), &details); err == nil {
			response.Details = details
		}
	}

	// Generate ID from timestamp and request_id
	if response.Time != "" && response.RequestID != "" {
		response.ID = utils.CreateRecordID(response.Time, response.RequestID)
	}

	return response
}
