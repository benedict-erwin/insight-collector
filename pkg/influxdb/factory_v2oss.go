package influxdb

import (
	"time"
	
	v2oss "github.com/benedict-erwin/insight-collector/pkg/influxdb/v2-oss"
)

// createV2OSSClient creates a new v2-oss client instance
func createV2OSSClient() Client {
	client := &v2oss.Client{}
	cfg := GetConfig()
	client.SetConfig(cfg.URL, cfg.Token, cfg.Org, cfg.Bucket)
	return client
}

// createV2OSSPoint creates a new v2-oss Point
func createV2OSSPoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) interface{} {
	return v2oss.NewPoint(measurement, tags, fields, timestamp)
}