package transactionevents

import (
	v2oss "github.com/benedict-erwin/insight-collector/pkg/influxdb/v2-oss"
)

// GetQueryConfig returns query builder configuration for transaction events
func GetQueryConfig() v2oss.QueryBuilderConfig {
	return v2oss.QueryBuilderConfig{
		Measurement: "transaction_events",
		ValidTags: map[string]bool{
			// Identity Group - Tags from ToPoint() method
			"user_id":    true,
			"session_id": true,

			// Business Transaction Group
			"transaction_type":   true,
			"currency":           true,
			"payment_method":     true,
			"status":             true,
			"transaction_nature": true,

			// Merchant & Category Group
			"merchant_category": true,

			// Technical Context Group
			"device_type": true,
			"os":          true,
			"channel":     true,
			"browser":     true,

			// Geographic & Assessment Group
			"geo_country": true,
			"risk_level":  true,
		},
		ValidFields: map[string]bool{
			// Correlation Group - Fields from ToPoint() method
			"request_id":            true,
			"trace_id":              true,
			"transaction_id":        true,
			"external_reference_id": true,

			// Financial Data Group
			"amount":         true,
			"fee_amount":     true,
			"net_amount":     true,
			"exchange_rate":  true,
			"compliance_score": true,

			// Performance Metrics Group
			"processing_time_ms": true,
			"duration_ms":        true,
			"retry_count":        true,
			"response_code":      true,

			// Business Control Group
			"approval_required": true,

			// Detection & Security Group
			"is_bot": true,

			// Merchant & Destination Group
			"merchant_id":         true,
			"destination_account": true,

			// Network & Client Context Group
			"ip_address":  true,
			"user_agent":  true,
			"app_version": true,
			"endpoint":    true,
			"method":      true,

			// Geographic Details Group
			"geo_city":        true,
			"geo_coordinates": true,
			"geo_timezone":    true,
			"geo_postal":      true,
			"geo_isp":         true,
			"os_version":      true,
			"browser_version": true,

			// Metadata Group
			"details": true,
		},
		Columns: []string{
			// Essential columns for transaction events list view
			"_time",
			"user_id",
			"session_id",
			"transaction_type",
			"currency",
			"payment_method",
			"status",
			"transaction_nature",
			"merchant_category",
			"device_type",
			"os",
			"os_version",
			"channel",
			"browser",
			"browser_version",
			"geo_country",
			"risk_level",
			"request_id",
			"trace_id",
			"transaction_id",
			"external_reference_id",
			"amount",
			"fee_amount",
			"net_amount",
			"exchange_rate",
			"processing_time_ms",
			"duration_ms",
			"retry_count",
			"response_code",
			"approval_required",
			"compliance_score",
			"is_bot",
			"merchant_id",
			"destination_account",
			"ip_address",
			"user_agent",
			"app_version",
			"endpoint",
			"method",
			"geo_city",
			"geo_coordinates",
			"geo_timezone",
			"geo_postal",
			"geo_isp",
			"details",
		},
		CountField: "request_id", // Use request_id for counting unique transaction events
	}
}