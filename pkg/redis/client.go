package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient implements the Client interface for both single-node and cluster modes
type RedisClient struct {
	mode          RedisMode
	singleClient  *redis.Client        // For single-node Redis
	clusterClient *redis.ClusterClient // For Redis Cluster
	keyPrefix     string               // Key prefix for logical separation in cluster mode
	db            int                  // Database number for single-node mode
}

// NewRedisClient creates a new Redis client based on configuration
func NewRedisClient(cfg RedisConfig, keyPrefix string, db int) (*RedisClient, error) {
	client := &RedisClient{
		mode:      RedisMode(cfg.Mode),
		keyPrefix: keyPrefix,
		db:        db,
	}

	switch client.mode {
	case ModeSingle:
		client.singleClient = redis.NewClient(&redis.Options{
			Addr:         fmt.Sprintf("%s:%d", cfg.Single.Host, cfg.Single.Port),
			Password:     cfg.Single.Password,
			DB:           db,
			DialTimeout:  cfg.Pool.DialTimeout,
			ReadTimeout:  cfg.Pool.ReadTimeout,
			WriteTimeout: cfg.Pool.WriteTimeout,
			PoolSize:     cfg.Pool.Size,
			PoolTimeout:  cfg.Pool.Timeout,
		})

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.singleClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to single-node Redis: %w", err)
		}

	case ModeCluster:
		client.clusterClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.Cluster.Nodes,
			Password:     cfg.Cluster.Password,
			DialTimeout:  cfg.Pool.DialTimeout,
			ReadTimeout:  cfg.Pool.ReadTimeout,
			WriteTimeout: cfg.Pool.WriteTimeout,
			PoolSize:     cfg.Pool.Size,
			PoolTimeout:  cfg.Pool.Timeout,
		})

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.clusterClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis cluster: %w", err)
		}

	case ModeSentinel:
		// todo: Implement Sentinel support in future
		return nil, fmt.Errorf("redis Sentinel mode not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported Redis mode: %s", cfg.Mode)
	}

	return client, nil
}

// buildKey constructs the final key with prefix for cluster mode
func (r *RedisClient) buildKey(key string) string {
	if r.keyPrefix != "" {
		return r.keyPrefix + key
	}
	return key
}

// Set sets a key-value pair with expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	finalKey := r.buildKey(key)

	switch r.mode {
	case ModeSingle:
		return r.singleClient.Set(ctx, finalKey, value, expiration).Err()
	case ModeCluster:
		return r.clusterClient.Set(ctx, finalKey, value, expiration).Err()
	default:
		return fmt.Errorf("unsupported mode: %s", r.mode)
	}
}

// Get retrieves a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	finalKey := r.buildKey(key)

	switch r.mode {
	case ModeSingle:
		return r.singleClient.Get(ctx, finalKey).Result()
	case ModeCluster:
		return r.clusterClient.Get(ctx, finalKey).Result()
	default:
		return "", fmt.Errorf("unsupported mode: %s", r.mode)
	}
}

// SetJSON stores JSON-serialized data with expiration
func (r *RedisClient) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return r.Set(ctx, key, data, expiration)
}

// GetJSON retrieves and deserializes JSON data
func (r *RedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// Delete removes one or more keys
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	// Build final keys with prefix
	finalKeys := make([]string, len(keys))
	for i, key := range keys {
		finalKeys[i] = r.buildKey(key)
	}

	switch r.mode {
	case ModeSingle:
		return r.singleClient.Del(ctx, finalKeys...).Err()
	case ModeCluster:
		return r.clusterClient.Del(ctx, finalKeys...).Err()
	default:
		return fmt.Errorf("unsupported mode: %s", r.mode)
	}
}

// Exists checks if a key exists
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	finalKey := r.buildKey(key)

	var count int64
	var err error

	switch r.mode {
	case ModeSingle:
		count, err = r.singleClient.Exists(ctx, finalKey).Result()
	case ModeCluster:
		count, err = r.clusterClient.Exists(ctx, finalKey).Result()
	default:
		return false, fmt.Errorf("unsupported mode: %s", r.mode)
	}

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Health checks the Redis connection
func (r *RedisClient) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	switch r.mode {
	case ModeSingle:
		return r.singleClient.Ping(ctx).Err()
	case ModeCluster:
		return r.clusterClient.Ping(ctx).Err()
	default:
		return fmt.Errorf("unsupported mode: %s", r.mode)
	}
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	switch r.mode {
	case ModeSingle:
		if r.singleClient != nil {
			return r.singleClient.Close()
		}
	case ModeCluster:
		if r.clusterClient != nil {
			return r.clusterClient.Close()
		}
	}
	return nil
}
