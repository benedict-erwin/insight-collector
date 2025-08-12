package useractivities

import (
	"encoding/json"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

type (

	// TECH & BEHAVIOR FOCUSED
	UserActivities struct {
		// TAGS
		// === IDENTITY GROUP ===
		UserID    string `json:"user_id"`    // Primary user identifier
		SessionID string `json:"session_id"` // Session correlation key

		// === BUSINESS CONTEXT GROUP ===
		ActivityType string `json:"activity_type"` // login/logout/page_view/form_submit/api_call/file_upload
		Category     string `json:"category"`      // auth/payment/account/security/admin/general
		Subcategory  string `json:"subcategory"`   // transfer/topup/withdraw/kyc/profile_update
		Status       string `json:"status"`        // success/failed/pending/timeout/cancelled

		// === TECHNICAL CONTEXT GROUP ===
		Browser       string `json:"browser"`        // e.g. chrome, firefox, safari
		DeviceType    string `json:"device_type"`    // e.g. desktop, mobile, tablet
		OS            string `json:"os"`             // e.g. windows, linux, ios
		Channel       string `json:"channel"`        // web/mobile_app/api
		EndpointGroup string `json:"endpoint_group"` // auth_api/transfer_api/payment_api/account_api/other_api
		Method        string `json:"method"`         // GET/POST/PUT/DELETE/VIEW

		// === GEOGRAPHIC & ASSESSMENT GROUP ===
		GeoCountry string `json:"geo_country"` // ID/SG/MY/TH/US/PH
		RiskLevel  string `json:"risk_level"`  // low/medium/high/critical

		// FIELDS
		// === CORRELATION GROUP ===
		RequestID string `json:"request_id"` // Unique request identifier (debugging)
		TraceID   string `json:"trace_id"`   // Distributed tracing ID

		// === PERFORMANCE METRICS GROUP ===
		DurationMs        int `json:"duration_ms"`         // Total request duration (in milliseconds)
		ResponseCode      int `json:"response_code"`       // HTTP status code (e.g. 200, 400, 500)
		RequestSizeBytes  int `json:"request_size_bytes"`  // Request payload size (in bytes)
		ResponseSizeBytes int `json:"response_size_bytes"` // Response payload size (in bytes)

		// === SECURITY & DETECTION GROUP ===
		IsBot bool `json:"is_bot"` // Bot detection flag

		// === NETWORK & CLIENT CONTEXT GROUP ===
		IPAddress   string `json:"ip_address"`   // Source IP address
		UserAgent   string `json:"user_agent"`   // Full user agent string
		AppVersion  string `json:"app_version"`  // Application version (e.g. v1.2.3)
		ReferrerURL string `json:"referrer_url"` // Previous page URL
		Endpoint    string `json:"endpoint"`     // Full endpoint path (e.g. /api/v1/transfer)

		// === GEOGRAPHIC DETAILS GROUP ===
		GeoCity        string `json:"geo_city"`        // jakarta/singapore/kuala_lumpur
		GeoCoordinates string `json:"geo_coordinates"` // Coordinates format: "lat,lng"
		GeoTimezone    string `json:"geo_timezone"`    // Asia/Jakarta/Asia/Singapore
		GeoPostal      string `json:"geo_postal"`      // Postal/ZIP code
		GeoISP         string `json:"geo_isp"`         // ISP name or ASN info

		// === METADATA GROUP ===
		OSVersion      string                 `json:"os_version"`      // OS version
		BrowserVersion string                 `json:"browser_version"` // Browser version
		Details        map[string]interface{} `json:"details"`

		// Timestamp
		Timestamp time.Time
	}

	UserActivitiesRequest struct {
		UserID            string                 `json:"user_id" validate:"required"`
		SessionID         string                 `json:"session_id" validate:"required"`
		ActivityType      string                 `json:"activity_type"`
		Category          string                 `json:"category"`
		Subcategory       string                 `json:"subcategory"`
		Status            string                 `json:"status"`
		Channel           string                 `json:"channel"`
		EndpointGroup     string                 `json:"endpoint_group"`
		Method            string                 `json:"method" validate:"required"`
		RiskLevel         string                 `json:"risk_level"`
		RequestID         string                 `json:"request_id,omitempty"`
		TraceID           string                 `json:"trace_id" validate:"required"`
		DurationMs        int                    `json:"duration_ms"`
		ResponseCode      int                    `json:"response_code"`
		RequestSizeBytes  int                    `json:"request_size_bytes"`
		ResponseSizeBytes int                    `json:"response_size_bytes"`
		IPAddress         string                 `json:"ip_address" validate:"required"`
		UserAgent         string                 `json:"user_agent" validate:"required"`
		AppVersion        string                 `json:"app_version"`
		ReferrerURL       string                 `json:"referrer_url"`
		Endpoint          string                 `json:"endpoint" validate:"required"`
		Details           map[string]interface{} `json:"details"`
		Timestamp         time.Time              `json:"time" validate:"required"`
	}

	// UserActivitiesResponse represents the response structure for user activities
	UserActivitiesResponse struct {
		ID             string                 `json:"id"`
		Time           string                 `json:"time"`
		UserID         string                 `json:"user_id"`
		SessionID      string                 `json:"session_id"`
		ActivityType   string                 `json:"activity_type"`
		Category       string                 `json:"category"`
		Subcategory    string                 `json:"subcategory"`
		Status         string                 `json:"status"`
		Browser        string                 `json:"browser"`
		DeviceType     string                 `json:"device_type"`
		OS             string                 `json:"os"`
		Channel        string                 `json:"channel"`
		EndpointGroup  string                 `json:"endpoint_group"`
		Method         string                 `json:"method"`
		GeoCountry     string                 `json:"geo_country"`
		RiskLevel      string                 `json:"risk_level"`
		RequestID      string                 `json:"request_id"`
		TraceID        string                 `json:"trace_id"`
		DurationMs     int                    `json:"duration_ms"`
		ResponseCode   int                    `json:"response_code"`
		IsBot          bool                   `json:"is_bot"`
		IPAddress      string                 `json:"ip_address"`
		UserAgent      string                 `json:"user_agent"`
		AppVersion     string                 `json:"app_version"`
		ReferrerURL    string                 `json:"referrer_url"`
		Endpoint       string                 `json:"endpoint"`
		GeoCity        string                 `json:"geo_city"`
		GeoCoordinates string                 `json:"geo_coordinates"`
		GeoTimezone    string                 `json:"geo_timezone"`
		GeoPostal      string                 `json:"geo_postal"`
		GeoISP         string                 `json:"geo_isp"`
		OSVersion      string                 `json:"os_version"`
		BrowserVersion string                 `json:"browser_version"`
		Details        map[string]interface{} `json:"details"`
	}
)

// ToPoint converts UserActivities to InfluxDB point with tags and fields
func (ua *UserActivities) ToPoint() interface{} {
	// Serialize details to JSON string for InfluxDB storage
	var detailsJSON string
	if len(ua.Details) > 0 {
		if jsonBytes, err := json.Marshal(ua.Details); err == nil {
			detailsJSON = string(jsonBytes)
		}
	}

	return influxdb.NewPoint(
		"user_activities",
		map[string]string{
			// Ensure all tags are non-empty strings
			"user_id":        safeString(ua.UserID),
			"activity_type":  safeString(ua.ActivityType),
			"category":       safeString(ua.Category),
			"subcategory":    safeString(ua.Subcategory),
			"status":         safeString(ua.Status),
			"browser":        safeString(ua.Browser),
			"device_type":    safeString(ua.DeviceType),
			"os":             safeString(ua.OS),
			"channel":        safeString(ua.Channel),
			"endpoint_group": safeString(ua.EndpointGroup),
			"method":         safeString(ua.Method),
			"geo_country":    safeString(ua.GeoCountry),
			"risk_level":     safeString(ua.RiskLevel),
		},
		map[string]interface{}{
			// String fields - consistent type
			"session_id":      safeString(ua.SessionID),
			"request_id":      safeString(ua.RequestID),
			"trace_id":        safeString(ua.TraceID),
			"ip_address":      safeString(ua.IPAddress),
			"user_agent":      safeString(ua.UserAgent),
			"app_version":     safeString(ua.AppVersion),
			"referrer_url":    safeString(ua.ReferrerURL),
			"endpoint":        safeString(ua.Endpoint),
			"geo_city":        safeString(ua.GeoCity),
			"geo_coordinates": safeString(ua.GeoCoordinates),
			"geo_timezone":    safeString(ua.GeoTimezone),
			"geo_postal":      safeString(ua.GeoPostal),
			"geo_isp":         safeString(ua.GeoISP),
			"os_version":      safeString(ua.OSVersion),
			"browser_version": safeString(ua.BrowserVersion),

			// Integer fields - consistent type
			"duration_ms":         int64(ua.DurationMs),
			"response_code":       int64(ua.ResponseCode),
			"request_size_bytes":  int64(ua.RequestSizeBytes),
			"response_size_bytes": int64(ua.ResponseSizeBytes),

			// Boolean fields - consistent type
			"is_bot": bool(ua.IsBot),

			// Map/Object fields - serialize to JSON string
			"details": detailsJSON,
		},
		ua.Timestamp,
	)
}

// GetName returns the measurement name for this entity
func (ua *UserActivities) GetName() string {
	return "user_activities"
}

// safeString ensures tag values are never empty (InfluxDB requirement)
func safeString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// MapToUserActivitiesResponse converts raw InfluxDB record to UserActivitiesResponse struct
func MapToUserActivitiesResponse(record map[string]interface{}) UserActivitiesResponse {
	response := UserActivitiesResponse{}

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

	// Business context fields
	if v, ok := record["activity_type"].(string); ok && v != "" && v != "-" {
		response.ActivityType = v
	}
	if v, ok := record["category"].(string); ok && v != "" && v != "-" {
		response.Category = v
	}
	if v, ok := record["subcategory"].(string); ok && v != "" && v != "-" {
		response.Subcategory = v
	}
	if v, ok := record["status"].(string); ok && v != "" && v != "-" {
		response.Status = v
	}

	// Technical context fields
	if v, ok := record["browser"].(string); ok && v != "" && v != "-" {
		response.Browser = v
	}
	if v, ok := record["device_type"].(string); ok && v != "" && v != "-" {
		response.DeviceType = v
	}
	if v, ok := record["os"].(string); ok && v != "" && v != "-" {
		response.OS = v
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

	// Security and assessment fields
	if v, ok := record["risk_level"].(string); ok && v != "" && v != "-" {
		response.RiskLevel = v
	}

	// Correlation fields
	if v, ok := record["request_id"].(string); ok && v != "" && v != "-" {
		response.RequestID = v
	}
	if v, ok := record["trace_id"].(string); ok && v != "" && v != "-" {
		response.TraceID = v
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
	if v, ok := record["referrer_url"].(string); ok && v != "" && v != "-" {
		response.ReferrerURL = v
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

	// Numeric fields with type conversion
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
