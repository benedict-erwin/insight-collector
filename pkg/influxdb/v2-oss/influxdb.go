package v2oss

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// Config represents InfluxDB v2 OSS configuration
type Config struct {
	URL    string
	Token  string
	Org    string
	Bucket string
}

// Client implements the InfluxDB v2 OSS client
type Client struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	queryAPI api.QueryAPI
	config   *Config
}

// Point wraps write.Point
type Point struct {
	*write.Point
}

// QueryIterator wraps api.QueryTableResult and implements the interface
type QueryIterator struct {
	result *api.QueryTableResult
	closed bool
}

// QueryIterator implements QueryIteratorInterface

// NewPoint creates a new Point
func NewPoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) *Point {
	point := write.NewPoint(measurement, tags, fields, timestamp)
	return &Point{Point: point}
}

// Point interface implementation
func (p *Point) GetMeasurement() string {
	return p.Point.Name()
}

func (p *Point) GetTags() map[string]string {
	tags := make(map[string]string)
	for _, tag := range p.Point.TagList() {
		tags[tag.Key] = tag.Value
	}
	return tags
}

func (p *Point) GetFields() map[string]interface{} {
	fields := make(map[string]interface{})
	for _, field := range p.Point.FieldList() {
		fields[field.Key] = field.Value
	}
	return fields
}

func (p *Point) GetTime() time.Time {
	return p.Point.Time()
}

// QueryIterator interface implementation
func (qi *QueryIterator) Next() bool {
	if qi.closed || qi.result == nil {
		return false
	}
	return qi.result.Next()
}

func (qi *QueryIterator) Record() map[string]interface{} {
	if qi.closed || qi.result == nil {
		return nil
	}

	record := qi.result.Record()
	if record == nil {
		return nil
	}

	// Convert InfluxDB v2 record to generic map
	result := make(map[string]interface{})
	result["_measurement"] = record.Measurement()
	result["_time"] = record.Time()
	result["_field"] = record.Field()
	result["_value"] = record.Value()

	// Add all other values
	for key, value := range record.Values() {
		result[key] = value
	}

	return result
}

func (qi *QueryIterator) Err() error {
	if qi.result == nil {
		return nil
	}
	return qi.result.Err()
}

func (qi *QueryIterator) Close() error {
	qi.closed = true
	if qi.result != nil {
		qi.result.Close()
	}
	return nil
}

// SetConfig sets the configuration for the client
func (c *Client) SetConfig(url, token, org, bucket string) {
	c.config = &Config{
		URL:    url,
		Token:  token,
		Org:    org,
		Bucket: bucket,
	}
}

// Init initializes the InfluxDB v2 OSS client
func (c *Client) Init() error {
	cfg := c.config
	if cfg == nil {
		return fmt.Errorf("v2-oss client config not set")
	}

	if cfg.URL == "" || cfg.Token == "" || cfg.Bucket == "" || cfg.Org == "" {
		logger.Error().Msg("InfluxDB v2-oss config missing url, token, bucket, or org")
		return fmt.Errorf("incomplete InfluxDB v2-oss configuration")
	}

	// Close existing client
	if c.client != nil {
		c.client.Close()
	}

	// Create new client
	c.client = influxdb2.NewClient(cfg.URL, cfg.Token)

	// Initialize write and query APIs
	c.writeAPI = c.client.WriteAPIBlocking(cfg.Org, cfg.Bucket)
	c.queryAPI = c.client.QueryAPI(cfg.Org)

	logger.Info().
		Str("url", cfg.URL).
		Str("org", cfg.Org).
		Str("bucket", cfg.Bucket).
		Str("version", "v2-oss").
		Msg("InfluxDB client initialized")
	return nil
}

func (c *Client) WritePoint(point interface{}) error {
	if c.client == nil || c.writeAPI == nil {
		logger.Error().Msg("InfluxDB v2-oss client not initialized")
		return fmt.Errorf("InfluxDB v2-oss client not initialized")
	}

	// Convert point to write.Point
	var v2Point *write.Point
	if p, ok := point.(*Point); ok {
		v2Point = p.Point
	} else {
		return fmt.Errorf("invalid point type for v2-oss")
	}

	// Write single point
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.writeAPI.WritePoint(ctx, v2Point); err != nil {
		logger.Error().Err(err).Str("measurement", v2Point.Name()).Msg("Failed to write point to InfluxDB v2-oss")
		return fmt.Errorf("failed to write point: %w", err)
	}

	logger.Info().Str("measurement", v2Point.Name()).Msg("Point written to InfluxDB v2-oss")
	return nil
}

func (c *Client) WritePoints(points []interface{}) error {
	if c.client == nil || c.writeAPI == nil {
		logger.Error().Msg("InfluxDB v2-oss client not initialized")
		return fmt.Errorf("InfluxDB v2-oss client not initialized")
	}

	// Convert points to write.Points
	v2Points := make([]*write.Point, len(points))
	for i, point := range points {
		if p, ok := point.(*Point); ok {
			v2Points[i] = p.Point
		} else {
			return fmt.Errorf("invalid point type for v2-oss")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.writeAPI.WritePoint(ctx, v2Points...); err != nil {
		logger.Error().Err(err).Msg("Failed to write points to InfluxDB v2-oss")
		return fmt.Errorf("failed to write points: %w", err)
	}

	logger.Info().Int("count", len(points)).Msg("Points written to InfluxDB v2-oss")
	return nil
}

func (c *Client) Query(query string) (interface{}, error) {
	if c.client == nil || c.queryAPI == nil {
		logger.Error().Msg("InfluxDB v2-oss client not initialized")
		return nil, fmt.Errorf("InfluxDB v2-oss client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := c.queryAPI.Query(ctx, query)
	if err != nil {
		logger.Error().Err(err).Str("query", query).Msg("Failed to execute InfluxDB v2-oss query")
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	return &QueryIterator{result: result}, nil
}

func (c *Client) QueryWithOptions(query string, options ...interface{}) (interface{}, error) {
	// For v2-oss, we'll ignore options for now and use standard query
	// This can be extended later if specific v2 query options are needed
	logger.Warn().Msg("QueryWithOptions not fully implemented for v2-oss, using standard query")
	return c.Query(query)
}

func (c *Client) GetClient() interface{} {
	return c.client
}

func (c *Client) IsHealthy() bool {
	return c.client != nil && c.writeAPI != nil && c.queryAPI != nil
}

func (c *Client) HealthCheck() error {
	if c.client == nil {
		return fmt.Errorf("InfluxDB v2-oss client not initialized")
	}

	// Ping test using health API
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := c.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("InfluxDB v2-oss health check failed: %w", err)
	}

	if health.Status != "pass" {
		return fmt.Errorf("InfluxDB v2-oss is not healthy: %s", health.Status)
	}

	return nil
}

func (c *Client) Close() {
	if c.client != nil {
		c.client.Close()
		logger.Info().Msg("InfluxDB v2-oss client closed")
		c.client = nil
		c.writeAPI = nil
		c.queryAPI = nil
	}
}
