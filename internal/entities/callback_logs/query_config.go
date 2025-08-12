package callbacklogs

import (
	v2oss "github.com/benedict-erwin/insight-collector/pkg/influxdb/v2-oss"
)

// GetQueryConfig returns query builder configuration for callback logs
func GetQueryConfig() v2oss.QueryBuilderConfig {
	return v2oss.QueryBuilderConfig{
		Measurement: "callback_logs",
		ValidTags: map[string]bool{
			// Core Identification - Tags from ToPoint() method
			"transaction_id": true,
			"callback_type":  true,
			"status":         true,

			// Error Categorization
			"error_category": true,
		},
		ValidFields: map[string]bool{
			// Basic Correlation - Fields from ToPoint() method
			"callback_id": true,

			// Error Details
			"http_status_code": true,
			"error_message":    true,
			"client_response":  true,

			// Performance Metrics
			"duration_ms": true,
			"retry_count": true,

			// Callback Context
			"destination_url": true,

			// Payload Data
			"payloads": true,
		},
		Columns: []string{
			// Essential columns for callback logs list view
			"_time",
			"transaction_id",
			"callback_type",
			"status",
			"error_category",
			"callback_id",
			"http_status_code",
			"error_message",
			"client_response",
			"duration_ms",
			"retry_count",
			"destination_url",
			"payloads",
		},
		CountField: "callback_id", // Use callback_id for counting unique callback logs
	}
}
