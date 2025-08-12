package influxdb

import (
	"time"
	
	v3core "github.com/benedict-erwin/insight-collector/pkg/influxdb/v3-core"
)

// createV3CoreClient creates a new v3-core client instance
func createV3CoreClient() Client {
	client := &v3core.Client{}
	cfg := GetConfig()
	client.SetConfig(cfg.Host, cfg.Port, cfg.Token, cfg.AuthScheme, cfg.Bucket)
	return client
}

// createV3CorePoint creates a new v3-core Point
func createV3CorePoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) interface{} {
	return v3core.NewPoint(measurement, tags, fields, timestamp)
}