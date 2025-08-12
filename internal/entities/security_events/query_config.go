package securityevents

import (
	v2oss "github.com/benedict-erwin/insight-collector/pkg/influxdb/v2-oss"
)

// GetQueryConfig returns query builder configuration for security events
func GetQueryConfig() v2oss.QueryBuilderConfig {
	return v2oss.QueryBuilderConfig{
		Measurement: "security_events",
		ValidTags: map[string]bool{
			// Identity Group - Tags from ToPoint() method
			"user_id":         true,
			"session_id":      true,
			"identifier_type": true,

			// Security Context Group
			"event_type":       true,
			"severity":         true,
			"auth_stage":       true,
			"action_taken":     true,
			"detection_method": true,

			// Technical Context Group
			"device_type":    true,
			"os":             true,
			"browser":        true,
			"channel":        true,
			"endpoint_group": true,
			"method":         true,

			// Geographic Group
			"geo_country": true,
		},
		ValidFields: map[string]bool{
			// Correlation Group - Fields from ToPoint() method
			"request_id":        true,
			"trace_id":          true,
			"identifier_value":  true,
			"affected_resource": true,

			// Security Metrics Group
			"attempt_count":         true,
			"risk_score":            true,
			"confidence_score":      true,
			"previous_success_time": true,

			// Performance Metrics Group
			"duration_ms":   true,
			"response_code": true,

			// Detection & Context Group
			"is_bot": true,

			// Network & Client Context Group
			"ip_address":  true,
			"user_agent":  true,
			"app_version": true,
			"endpoint":    true,

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
			// Essential columns for security events list view
			"_time",
			"user_id",
			"session_id",
			"identifier_type",
			"event_type",
			"severity",
			"auth_stage",
			"action_taken",
			"detection_method",
			"device_type",
			"os",
			"os_version",
			"browser",
			"browser_version",
			"channel",
			"endpoint_group",
			"method",
			"geo_country",
			"geo_city",
			"request_id",
			"trace_id",
			"identifier_value",
			"attempt_count",
			"risk_score",
			"confidence_score",
			"previous_success_time",
			"affected_resource",
			"duration_ms",
			"response_code",
			"is_bot",
			"ip_address",
			"user_agent",
			"app_version",
			"endpoint",
			"geo_coordinates",
			"geo_timezone",
			"geo_postal",
			"geo_isp",
			"details",
		},
		CountField: "request_id", // Use request_id for counting unique security events
	}
}