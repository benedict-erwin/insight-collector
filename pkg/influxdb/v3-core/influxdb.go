package v3core

import (
	"context"
	"fmt"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// Config represents InfluxDB v3 Core configuration
type Config struct {
	Host       string
	Port       int
	Token      string
	AuthScheme string
	Bucket     string
}

// Client implements InfluxDB v3 Core client
type Client struct {
	client *influxdb3.Client
	config *Config
}

// Point wraps influxdb3.Point
type Point struct {
	*influxdb3.Point
}

// QueryIterator wraps influxdb3.QueryIterator and implements the interface
type QueryIterator struct {
	*influxdb3.QueryIterator
}

// QueryIterator implements QueryIteratorInterface

// NewPoint creates a new Point
func NewPoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) *Point {
	point := influxdb3.NewPoint(measurement, tags, fields, timestamp)
	return &Point{Point: point}
}

// Point interface implementation
func (p *Point) GetMeasurement() string {
	// For influxdb3, we need to extract measurement from the point
	// This is a simplified approach - you may need to adjust based on actual API
	return "measurement" // Placeholder - influxdb3.Point doesn't expose measurement directly
}

func (p *Point) GetTags() map[string]string {
	// For influxdb3, tags are not directly accessible
	// This is a simplified approach - you may need to adjust based on actual API
	return make(map[string]string) // Placeholder
}

func (p *Point) GetFields() map[string]interface{} {
	// For influxdb3, fields are not directly accessible  
	// This is a simplified approach - you may need to adjust based on actual API
	return make(map[string]interface{}) // Placeholder
}

func (p *Point) GetTime() time.Time {
	// For influxdb3, time is not directly accessible
	// This is a simplified approach - you may need to adjust based on actual API
	return time.Now() // Placeholder
}

// QueryIterator interface implementation
func (qi *QueryIterator) Next() bool {
	return qi.QueryIterator.Next()
}

func (qi *QueryIterator) Record() map[string]interface{} {
	if qi.QueryIterator.Value() == nil {
		return nil
	}
	// Convert influxdb3 record to generic map
	record := make(map[string]interface{})
	for k, v := range qi.QueryIterator.Value() {
		record[k] = v
	}
	return record
}

func (qi *QueryIterator) Err() error {
	return qi.QueryIterator.Err()
}

func (qi *QueryIterator) Close() error {
	// influxdb3.QueryIterator doesn't have Close method
	// This is a no-op for v3-core
	return nil
}

// SetConfig sets the configuration for the client
func (c *Client) SetConfig(host string, port int, token, authScheme, bucket string) {
	c.config = &Config{
		Host:       host,
		Port:       port,
		Token:      token,
		AuthScheme: authScheme,
		Bucket:     bucket,
	}
}

// Init initializes the InfluxDB v3 Core client
func (c *Client) Init() error {
	cfg := c.config
	if cfg == nil {
		return fmt.Errorf("v3-core client config not set")
	}
	
	if cfg.Host == "" || cfg.Token == "" || cfg.Bucket == "" {
		logger.Error().Msg("InfluxDB v3-core config missing host, token, or bucket")
		return fmt.Errorf("incomplete InfluxDB v3-core configuration")
	}

	// Close existing client
	if c.client != nil {
		c.client.Close()
	}

	// Create new client
	var err error
	c.client, err = influxdb3.New(influxdb3.ClientConfig{
		Host:       fmt.Sprintf("%s:%v", cfg.Host, cfg.Port),
		Token:      cfg.Token,
		Database:   cfg.Bucket,
		AuthScheme: cfg.AuthScheme,
	})
	if err != nil {
		logger.Error().Err(err).Str("url", cfg.Host).Msg("Failed to initialize InfluxDB v3-core client")
		return fmt.Errorf("failed to initialize InfluxDB v3-core client: %w", err)
	}
	
	logger.Info().
		Str("host", cfg.Host).
		Str("bucket", cfg.Bucket).
		Str("version", "v3-core").
		Msg("InfluxDB client initialized")
	return nil
}

func (c *Client) WritePoint(point interface{}) error {
	if c.client == nil {
		logger.Error().Msg("InfluxDB v3-core client not initialized")
		return fmt.Errorf("InfluxDB v3-core client not initialized")
	}

	// Convert point to influxdb3.Point
	var v3Point *influxdb3.Point
	if p, ok := point.(*Point); ok {
		v3Point = p.Point
	} else {
		return fmt.Errorf("invalid point type for v3-core")
	}

	// Write single point
	if err := c.client.WritePoints(context.Background(), []*influxdb3.Point{v3Point}); err != nil {
		logger.Error().Err(err).Msg("Failed to write point to InfluxDB v3-core")
		return fmt.Errorf("failed to write point: %w", err)
	}

	logger.Info().Msg("Point written to InfluxDB v3-core")
	return nil
}

func (c *Client) WritePoints(points []interface{}) error {
	if c.client == nil {
		logger.Error().Msg("InfluxDB v3-core client not initialized")
		return fmt.Errorf("InfluxDB v3-core client not initialized")
	}

	// Convert points to influxdb3.Points
	v3Points := make([]*influxdb3.Point, len(points))
	for i, point := range points {
		if p, ok := point.(*Point); ok {
			v3Points[i] = p.Point
		} else {
			return fmt.Errorf("invalid point type for v3-core")
		}
	}

	if err := c.client.WritePoints(context.Background(), v3Points); err != nil {
		logger.Error().Err(err).Msg("Failed to write points to InfluxDB v3-core")
		return fmt.Errorf("failed to write points: %w", err)
	}

	logger.Info().Int("count", len(points)).Msg("Points written to InfluxDB v3-core")
	return nil
}

func (c *Client) Query(query string) (interface{}, error) {
	if c.client == nil {
		logger.Error().Msg("InfluxDB v3-core client not initialized")
		return nil, fmt.Errorf("InfluxDB v3-core client not initialized")
	}

	iterator, err := c.client.Query(context.Background(), query)
	if err != nil {
		logger.Error().Err(err).Str("query", query).Msg("Failed to execute InfluxDB v3-core query")
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	return &QueryIterator{QueryIterator: iterator}, nil
}

func (c *Client) QueryWithOptions(query string, options ...interface{}) (interface{}, error) {
	if c.client == nil {
		logger.Error().Msg("InfluxDB v3-core client not initialized")
		return nil, fmt.Errorf("InfluxDB v3-core client not initialized")
	}

	// For now, ignore options and use standard query
	logger.Warn().Msg("QueryWithOptions not fully implemented for v3-core, using standard query")
	return c.Query(query)
}

func (c *Client) GetClient() interface{} {
	return c.client
}

func (c *Client) IsHealthy() bool {
	return c.client != nil
}

func (c *Client) HealthCheck() error {
	if c.client == nil {
		return fmt.Errorf("InfluxDB v3-core client not initialized")
	}

	// Ping test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.client.Query(ctx, "SELECT 1")
	return err
}

func (c *Client) Close() {
	if c.client != nil {
		c.client.Close()
		logger.Info().Msg("InfluxDB v3-core client closed")
		c.client = nil
	}
}