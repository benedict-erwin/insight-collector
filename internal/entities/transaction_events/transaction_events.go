package transactionevents

import (
	"encoding/json"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// FINANCIAL & BUSINESS FOCUSED
type (
	TransactionEvents struct {
		// === IDENTITY GROUP ===
		UserID    string `json:"user_id"`    // Primary user identifier
		SessionID string `json:"session_id"` // Session correlation key

		// === BUSINESS TRANSACTION GROUP ===
		TransactionType   string `json:"transaction_type"`   // transfer/payment/topup/withdraw/refund
		Currency          string `json:"currency"`           // IDR/USD/SGD/MYR/THB/PHP
		PaymentMethod     string `json:"payment_method"`     // bank_transfer/ewallet/virtual_account/qris/credit_card
		Status            string `json:"status"`             // initiated/validated/processing/completed/failed/cancelled/expired
		TransactionNature string `json:"transaction_nature"` // normal/reversal/chargeback/refund

		// === MERCHANT & CATEGORY GROUP ===
		MerchantCategory string `json:"merchant_category"` // retail/food/transport/utilities/healthcare/other

		// === TECHNICAL CONTEXT GROUP ===
		DeviceType string `json:"device_type"` // desktop/mobile/tablet
		OS         string `json:"os"`          // windows/linux/ios
		Channel    string `json:"channel"`     // web/mobile_app/api
		Browser    string `json:"browser"`     // chrome/firefox/safari

		// === GEOGRAPHIC & ASSESSMENT GROUP ===
		GeoCountry string `json:"geo_country"` // ID/SG/MY/TH/US/PH
		RiskLevel  string `json:"risk_level"`  // low/medium/high/critical

		// === CORRELATION GROUP ===
		RequestID           string `json:"request_id"`            // Request correlation identifier
		TraceID             string `json:"trace_id"`              // Distributed tracing identifier
		TransactionID       string `json:"transaction_id"`        // Business transaction unique identifier
		ExternalReferenceID string `json:"external_reference_id"` // Bank/payment gateway reference ID

		// === FINANCIAL DATA GROUP ===
		Amount       float64 `json:"amount"`        // Primary transaction amount
		FeeAmount    float64 `json:"fee_amount"`    // Fee charged to user
		NetAmount    float64 `json:"net_amount"`    // Final amount after fee deduction
		ExchangeRate float64 `json:"exchange_rate"` // Currency conversion rate (if applicable)

		// === PERFORMANCE METRICS GROUP ===
		ProcessingTimeMs int `json:"processing_time_ms"` // Business logic processing duration
		DurationMs       int `json:"duration_ms"`        // Total request processing duration
		RetryCount       int `json:"retry_count"`        // Number of retry attempts
		ResponseCode     int `json:"response_code"`      // HTTP response status code

		// === BUSINESS CONTROL GROUP ===
		ApprovalRequired bool    `json:"approval_required"` // Manual approval requirement flag
		ComplianceScore  float64 `json:"compliance_score"`  // AML/compliance risk assessment score

		// === DETECTION & SECURITY GROUP ===
		IsBot bool `json:"is_bot"` // Automated transaction detection

		// === MERCHANT & DESTINATION GROUP ===
		MerchantID         string `json:"merchant_id"`         // Merchant identifier (for payment transactions)
		DestinationAccount string `json:"destination_account"` // Target account identifier (for transfers)

		// === NETWORK & CLIENT CONTEXT GROUP ===
		IPAddress  string `json:"ip_address"`  // Source IP address
		UserAgent  string `json:"user_agent"`  // Client user agent
		AppVersion string `json:"app_version"` // Application version
		Endpoint   string `json:"endpoint"`    // API endpoint (debugging purpose)
		Method     string `json:"method"`      // HTTP method (debugging purpose)

		// === GEOGRAPHIC DETAILS GROUP ===
		GeoCity        string `json:"geo_city"`        // Transaction origination city
		GeoCoordinates string `json:"geo_coordinates"` // Precise geographic coordinates
		GeoTimezone    string `json:"geo_timezone"`    // Local timezone information
		GeoPostal      string `json:"geo_postal"`      // Postal code from geolocation
		GeoISP         string `json:"geo_isp"`         // Internet service provider details

		// === METADATA GROUP ===
		OSVersion      string                 `json:"os_version"`      // OS version
		BrowserVersion string                 `json:"browser_version"` // Browser version
		Details        map[string]interface{} `json:"details"`

		// Timestamp
		Timestamp time.Time
	}

	TransactionEventsRequest struct {
		UserID              string                 `json:"user_id" validate:"required"`
		SessionID           string                 `json:"session_id" validate:"required"`
		TransactionType     string                 `json:"transaction_type"`
		Currency            string                 `json:"currency" validate:"required"`
		PaymentMethod       string                 `json:"payment_method"`
		Status              string                 `json:"status"`
		TransactionNature   string                 `json:"transaction_nature"`
		MerchantCategory    string                 `json:"merchant_category"`
		Channel             string                 `json:"channel"`
		RiskLevel           string                 `json:"risk_level"`
		RequestID           string                 `json:"request_id"`
		TraceID             string                 `json:"trace_id"`
		TransactionID       string                 `json:"transaction_id"`
		ExternalReferenceID string                 `json:"external_reference_id"`
		Amount              float64                `json:"amount"`
		FeeAmount           float64                `json:"fee_amount"`
		NetAmount           float64                `json:"net_amount"`
		ExchangeRate        float64                `json:"exchange_rate"`
		ProcessingTimeMs    int                    `json:"processing_time_ms"`
		DurationMs          int                    `json:"duration_ms"`
		RetryCount          int                    `json:"retry_count"`
		ResponseCode        int                    `json:"response_code"`
		ApprovalRequired    bool                   `json:"approval_required"`
		ComplianceScore     float64                `json:"compliance_score"`
		MerchantID          string                 `json:"merchant_id"`
		DestinationAccount  string                 `json:"destination_account"`
		IPAddress           string                 `json:"ip_address" validate:"required"`
		UserAgent           string                 `json:"user_agent" validate:"required"`
		AppVersion          string                 `json:"app_version"`
		Endpoint            string                 `json:"endpoint" validate:"required"`
		Method              string                 `json:"method" validate:"required"`
		Details             map[string]interface{} `json:"details"`
		Timestamp           time.Time              `json:"time" validate:"required"`
	}
	TransactionEventsResponse struct {
		ID                  string                 `json:"id"`
		Time                string                 `json:"time"`
		UserID              string                 `json:"user_id"`
		SessionID           string                 `json:"session_id"`
		TransactionType     string                 `json:"transaction_type"`
		Currency            string                 `json:"currency"`
		PaymentMethod       string                 `json:"payment_method"`
		Status              string                 `json:"status"`
		TransactionNature   string                 `json:"transaction_nature"`
		MerchantCategory    string                 `json:"merchant_category"`
		DeviceType          string                 `json:"device_type"`
		OS                  string                 `json:"os"`
		Channel             string                 `json:"channel"`
		Browser             string                 `json:"browser"`
		GeoCountry          string                 `json:"geo_country"`
		RiskLevel           string                 `json:"risk_level"`
		RequestID           string                 `json:"request_id"`
		TraceID             string                 `json:"trace_id"`
		TransactionID       string                 `json:"transaction_id"`
		ExternalReferenceID string                 `json:"external_reference_id"`
		Amount              float64                `json:"amount"`
		FeeAmount           float64                `json:"fee_amount"`
		NetAmount           float64                `json:"net_amount"`
		ExchangeRate        float64                `json:"exchange_rate"`
		ProcessingTimeMs    int                    `json:"processing_time_ms"`
		DurationMs          int                    `json:"duration_ms"`
		RetryCount          int                    `json:"retry_count"`
		ResponseCode        int                    `json:"response_code"`
		ApprovalRequired    bool                   `json:"approval_required"`
		ComplianceScore     float64                `json:"compliance_score"`
		IsBot               bool                   `json:"is_bot"`
		MerchantID          string                 `json:"merchant_id"`
		DestinationAccount  string                 `json:"destination_account"`
		IPAddress           string                 `json:"ip_address"`
		UserAgent           string                 `json:"user_agent"`
		AppVersion          string                 `json:"app_version"`
		Endpoint            string                 `json:"endpoint"`
		Method              string                 `json:"method"`
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

func (te *TransactionEvents) ToPoint() interface{} {
	// Serialize details to JSON string for InfluxDB storage
	var detailsJSON string
	if len(te.Details) > 0 {
		if jsonBytes, err := json.Marshal(te.Details); err == nil {
			detailsJSON = string(jsonBytes)
		}
	}

	return influxdb.NewPoint(
		"transaction_events",
		map[string]string{
			// OPTIMIZED: 5 carefully selected tags for transaction analytics
			"transaction_type": safeString(te.TransactionType), // Core business logic
			"status":           safeString(te.Status),          // Operational status
			"currency":         safeString(te.Currency),        // Financial analysis
			"channel":          safeString(te.Channel),         // User journey tracking
			"risk_level":       safeString(te.RiskLevel),       // Security monitoring
		},
		map[string]interface{}{
			"user_id":               safeString(te.UserID),
			"session_id":            safeString(te.SessionID),
			"request_id":            safeString(te.RequestID),
			"trace_id":              safeString(te.TraceID),
			"transaction_id":        safeString(te.TransactionID),
			"external_reference_id": safeString(te.ExternalReferenceID),
			"amount":                float64(te.Amount),
			"fee_amount":            float64(te.FeeAmount),
			"net_amount":            float64(te.NetAmount),
			"exchange_rate":         float64(te.ExchangeRate),
			"processing_time_ms":    int(te.ProcessingTimeMs),
			"duration_ms":           int(te.DurationMs),
			"retry_count":           int(te.RetryCount),
			"response_code":         int(te.ResponseCode),
			"approval_required":     bool(te.ApprovalRequired),
			"compliance_score":      float64(te.ComplianceScore),
			"is_bot":                bool(te.IsBot),
			"merchant_id":           safeString(te.MerchantID),
			"destination_account":   safeString(te.DestinationAccount),
			"ip_address":            safeString(te.IPAddress),
			"user_agent":            safeString(te.UserAgent),
			"browser":               safeString(te.Browser),
			"os":                    safeString(te.OS),
			"app_version":           safeString(te.AppVersion),
			"endpoint":              safeString(te.Endpoint),
			"method":                safeString(te.Method),
			"geo_city":              safeString(te.GeoCity),
			"geo_coordinates":       safeString(te.GeoCoordinates),
			"geo_timezone":          safeString(te.GeoTimezone),
			"geo_postal":            safeString(te.GeoPostal),
			"geo_isp":               safeString(te.GeoISP),
			"os_version":            safeString(te.OSVersion),
			"browser_version":       safeString(te.BrowserVersion),
			"details":               detailsJSON,

			// Moved from tags to fields (high cardinality)
			"payment_method":     safeString(te.PaymentMethod),
			"transaction_nature": safeString(te.TransactionNature),
			"merchant_category":  safeString(te.MerchantCategory),
			"device_type":        safeString(te.DeviceType),
			"geo_country":        safeString(te.GeoCountry),
		},
		te.Timestamp,
	)
}

// GetName returns the measurement name for this entity
func (te *TransactionEvents) GetName() string {
	return "transaction_events"
}

// safeString ensures tag values are never empty (InfluxDB requirement)
func safeString(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// MapToTransactionEventsResponse converts raw InfluxDB record to TransactionEventsResponse struct
func MapToTransactionEventsResponse(record map[string]interface{}) TransactionEventsResponse {
	response := TransactionEventsResponse{}

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
	if v, ok := record["user_id"].(string); ok && v != "" && v != "-" {
		response.UserID = v
	}
	if v, ok := record["session_id"].(string); ok && v != "" && v != "-" {
		response.SessionID = v
	}

	// === BUSINESS TRANSACTION GROUP ===
	if v, ok := record["transaction_type"].(string); ok && v != "" && v != "-" {
		response.TransactionType = v
	}
	if v, ok := record["currency"].(string); ok && v != "" && v != "-" {
		response.Currency = v
	}
	if v, ok := record["payment_method"].(string); ok && v != "" && v != "-" {
		response.PaymentMethod = v
	}
	if v, ok := record["status"].(string); ok && v != "" && v != "-" {
		response.Status = v
	}
	if v, ok := record["transaction_nature"].(string); ok && v != "" && v != "-" {
		response.TransactionNature = v
	}

	// === MERCHANT & CATEGORY GROUP ===
	if v, ok := record["merchant_category"].(string); ok && v != "" && v != "-" {
		response.MerchantCategory = v
	}

	// === TECHNICAL CONTEXT GROUP ===
	if v, ok := record["device_type"].(string); ok && v != "" && v != "-" {
		response.DeviceType = v
	}
	if v, ok := record["os"].(string); ok && v != "" && v != "-" {
		response.OS = v
	}
	if v, ok := record["channel"].(string); ok && v != "" && v != "-" {
		response.Channel = v
	}
	if v, ok := record["browser"].(string); ok && v != "" && v != "-" {
		response.Browser = v
	}

	// === GEOGRAPHIC & ASSESSMENT GROUP ===
	if v, ok := record["geo_country"].(string); ok && v != "" && v != "-" {
		response.GeoCountry = v
	}
	if v, ok := record["risk_level"].(string); ok && v != "" && v != "-" {
		response.RiskLevel = v
	}

	// === CORRELATION GROUP ===
	if v, ok := record["request_id"].(string); ok && v != "" && v != "-" {
		response.RequestID = v
	}
	if v, ok := record["trace_id"].(string); ok && v != "" && v != "-" {
		response.TraceID = v
	}
	if v, ok := record["transaction_id"].(string); ok && v != "" && v != "-" {
		response.TransactionID = v
	}
	if v, ok := record["external_reference_id"].(string); ok && v != "" && v != "-" {
		response.ExternalReferenceID = v
	}

	// === MERCHANT & DESTINATION GROUP ===
	if v, ok := record["merchant_id"].(string); ok && v != "" && v != "-" {
		response.MerchantID = v
	}
	if v, ok := record["destination_account"].(string); ok && v != "" && v != "-" {
		response.DestinationAccount = v
	}

	// === NETWORK & CLIENT CONTEXT GROUP ===
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
	if v, ok := record["method"].(string); ok && v != "" && v != "-" {
		response.Method = v
	}

	// === GEOGRAPHIC DETAILS GROUP ===
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

	// === VERSION FIELDS ===
	if v, ok := record["os_version"].(string); ok && v != "" && v != "-" {
		response.OSVersion = v
	}
	if v, ok := record["browser_version"].(string); ok && v != "" && v != "-" {
		response.BrowserVersion = v
	}

	// === FINANCIAL DATA GROUP - Float64 fields ===
	if v, ok := record["amount"]; ok {
		switch amount := v.(type) {
		case float64:
			response.Amount = amount
		case float32:
			response.Amount = float64(amount)
		case int64:
			response.Amount = float64(amount)
		case int:
			response.Amount = float64(amount)
		}
	}

	if v, ok := record["fee_amount"]; ok {
		switch fee := v.(type) {
		case float64:
			response.FeeAmount = fee
		case float32:
			response.FeeAmount = float64(fee)
		case int64:
			response.FeeAmount = float64(fee)
		case int:
			response.FeeAmount = float64(fee)
		}
	}

	if v, ok := record["net_amount"]; ok {
		switch net := v.(type) {
		case float64:
			response.NetAmount = net
		case float32:
			response.NetAmount = float64(net)
		case int64:
			response.NetAmount = float64(net)
		case int:
			response.NetAmount = float64(net)
		}
	}

	if v, ok := record["exchange_rate"]; ok {
		switch rate := v.(type) {
		case float64:
			response.ExchangeRate = rate
		case float32:
			response.ExchangeRate = float64(rate)
		case int64:
			response.ExchangeRate = float64(rate)
		case int:
			response.ExchangeRate = float64(rate)
		}
	}

	if v, ok := record["compliance_score"]; ok {
		switch score := v.(type) {
		case float64:
			response.ComplianceScore = score
		case float32:
			response.ComplianceScore = float64(score)
		case int64:
			response.ComplianceScore = float64(score)
		case int:
			response.ComplianceScore = float64(score)
		}
	}

	// === PERFORMANCE METRICS GROUP - Integer fields ===
	if v, ok := record["processing_time_ms"]; ok {
		switch duration := v.(type) {
		case int64:
			response.ProcessingTimeMs = int(duration)
		case float64:
			response.ProcessingTimeMs = int(duration)
		case int:
			response.ProcessingTimeMs = duration
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

	// === BOOLEAN FIELDS ===
	if v, ok := record["approval_required"]; ok {
		switch approval := v.(type) {
		case bool:
			response.ApprovalRequired = approval
		case string:
			response.ApprovalRequired = approval == "true" || approval == "1"
		}
	}

	if v, ok := record["is_bot"]; ok {
		switch bot := v.(type) {
		case bool:
			response.IsBot = bot
		case string:
			response.IsBot = bot == "true" || bot == "1"
		}
	}

	// === MAP/OBJECT FIELDS - deserialize JSON string back to map ===
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
