package useractivities

import (
	v2oss "github.com/benedict-erwin/insight-collector/pkg/influxdb/v2-oss"
)

// GetQueryConfig returns query builder configuration for user activities
func GetQueryConfig() v2oss.QueryBuilderConfig {
	return v2oss.QueryBuilderConfig{
		Measurement: "user_activities",
		ValidTags: map[string]bool{
			// Identity Group - Tags from ToPoint() method
			"user_id": true,

			// Business Context Group
			"activity_type": true,
			"category":      true,
			"subcategory":   true,
			"status":        true,

			// Technical Context Group
			"browser":        true,
			"device_type":    true,
			"os":             true,
			"channel":        true,
			"endpoint_group": true,
			"method":         true,

			// Geographic & Assessment Group
			"geo_country": true,
			"risk_level":  true,
		},
		ValidFields: map[string]bool{
			// Correlation Group - Fields from ToPoint() method
			"request_id": true,
			"trace_id":   true,
			"session_id": true,

			// Performance Metrics Group
			"duration_ms":         true,
			"response_code":       true,
			"request_size_bytes":  true,
			"response_size_bytes": true,

			// Security & Detection Group
			"is_bot": true,

			// Network & Client Context Group
			"ip_address":   true,
			"user_agent":   true,
			"app_version":  true,
			"referrer_url": true,
			"endpoint":     true,

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
			// Essential columns for list view
			"_time",
			"user_id",
			"session_id",
			"activity_type",
			"category",
			"subcategory",
			"status",
			"browser",
			"browser_version",
			"device_type",
			"os",
			"os_version",
			"channel",
			"endpoint_group",
			"method",
			"geo_country",
			"geo_city",
			"risk_level",
			"request_id",
			"trace_id",
			"duration_ms",
			"response_code",
			"is_bot",
			"ip_address",
			"user_agent",
			"app_version",
			"referrer_url",
			"endpoint",
			"geo_coordinates",
			"geo_timezone",
			"geo_postal",
			"geo_isp",
			"request_size_bytes",
			"response_size_bytes",
			"details",
		},
		CountField: "request_id", // Use request_id for counting unique records
	}
}
