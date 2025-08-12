package influxdb

import (
	"time"
)

// Point interface defines a generic point for time-series data
type Point interface {
	GetMeasurement() string
	GetTags() map[string]string
	GetFields() map[string]interface{}
	GetTime() time.Time
}

// QueryIterator interface defines a generic query result iterator
type QueryIterator interface {
	Next() bool
	Record() map[string]interface{}
	Err() error
	Close() error
}

// QueryOption defines query options interface
type QueryOption interface {
	Apply() interface{}
}

// Client interface defines the InfluxDB client operations
type Client interface {
	// Initialization and cleanup
	Init() error
	Close()
	IsHealthy() bool
	HealthCheck() error

	// Write operations
	WritePoint(point interface{}) error
	WritePoints(points []interface{}) error

	// Query operations
	Query(query string) (interface{}, error)
	QueryWithOptions(query string, options ...interface{}) (interface{}, error)

	// Client access
	GetClient() interface{}
}

// BaseQueryIterator provides a concrete implementation that other types can embed
type BaseQueryIterator struct{}

func (bqi *BaseQueryIterator) Next() bool                     { return false }
func (bqi *BaseQueryIterator) Record() map[string]interface{} { return nil }
func (bqi *BaseQueryIterator) Err() error                     { return nil }
func (bqi *BaseQueryIterator) Close() error                   { return nil }

// PointEntity interface for entities that can convert to Points
type PointEntity interface {
	ToPoint() interface{}
	GetName() string
}

// NewPointFunc defines function signature for creating new points
type NewPointFunc func(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) Point

// InfluxDBVersion represents supported InfluxDB versions
type InfluxDBVersion string

const (
	VersionV2OSS  InfluxDBVersion = "v2-oss"
	VersionV3Core InfluxDBVersion = "v3-core"
)

// Config represents InfluxDB configuration
type Config struct {
	Version InfluxDBVersion

	// v2-oss fields
	URL    string
	Token  string
	Org    string
	Bucket string

	// v3-core fields (legacy)
	Host       string
	Port       int
	AuthScheme string
}
