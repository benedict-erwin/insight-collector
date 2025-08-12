package influxdb

import (
	"time"
)

// Interface compatibility will be checked at runtime

// Factory functions are implemented in separate files to avoid import cycles
// factory_v2oss.go and factory_v3core.go contain the actual implementations

// createNewPoint creates a new Point using the active client implementation
func createNewPoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) interface{} {
	cfg := GetConfig()
	
	switch cfg.Version {
	case VersionV2OSS:
		return createV2OSSPoint(measurement, tags, fields, timestamp)
	case VersionV3Core:
		return createV3CorePoint(measurement, tags, fields, timestamp)
	default:
		return createV2OSSPoint(measurement, tags, fields, timestamp)
	}
}
