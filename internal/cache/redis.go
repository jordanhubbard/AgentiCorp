package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements Cache using Redis as the backend
type RedisCache struct {
	client *redis.Client
	config *Config
	stats  *Stats
}

// NewRedisCache creates a new Redis-backed cache
func NewRedisCache(redisURL string, config *Config) (*RedisCache, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Parse Redis URL and create client
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		config: config,
		stats:  &Stats{},
	}, nil
}

// Get retrieves a cached response from Redis
func (rc *RedisCache) Get(ctx context.Context, key string) (*Entry, bool) {
	if !rc.config.Enabled {
		return nil, false
	}

	// Get from Redis
	val, err := rc.client.Get(ctx, "cache:"+key).Result()
	if err == redis.Nil {
		// Cache miss
		rc.stats.Misses++
		return nil, false
	}
	if err != nil {
		// Redis error - treat as cache miss
		rc.stats.Misses++
		return nil, false
	}

	// Deserialize entry
	var entry Entry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		// Corrupted entry - delete and treat as miss
		rc.client.Del(ctx, "cache:"+key)
		rc.stats.Misses++
		return nil, false
	}

	// Cache hit
	rc.stats.Hits++
	rc.stats.TokensSaved += entry.TokensSaved

	// Increment hit counter
	entry.Hits++

	// Update entry in Redis with new hit count
	if data, err := json.Marshal(entry); err == nil {
		ttl := time.Until(entry.ExpiresAt)
		if ttl > 0 {
			rc.client.Set(ctx, "cache:"+key, data, ttl)
		}
	}

	return &entry, true
}

// Set stores a response in Redis
func (rc *RedisCache) Set(ctx context.Context, key string, response interface{}, ttl time.Duration, metadata map[string]interface{}) error {
	if !rc.config.Enabled {
		return nil
	}

	if ttl == 0 {
		ttl = rc.config.DefaultTTL
	}

	entry := &Entry{
		Key:         key,
		Response:    response,
		Metadata:    metadata,
		CachedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(ttl),
		Hits:        0,
		ProviderID:  getStringFromMap(metadata, "provider_id"),
		ModelName:   getStringFromMap(metadata, "model_name"),
		TokensSaved: getInt64FromMap(metadata, "total_tokens"),
	}

	// Serialize entry
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Store in Redis with TTL
	return rc.client.Set(ctx, "cache:"+key, data, ttl).Err()
}

// Delete removes an entry from Redis
func (rc *RedisCache) Delete(ctx context.Context, key string) {
	if !rc.config.Enabled {
		return
	}

	rc.client.Del(ctx, "cache:"+key)
}

// Clear removes all cache entries from Redis
func (rc *RedisCache) Clear(ctx context.Context) {
	if !rc.config.Enabled {
		return
	}

	// Delete all keys matching cache:* pattern
	iter := rc.client.Scan(ctx, 0, "cache:*", 0).Iterator()
	for iter.Next(ctx) {
		rc.client.Del(ctx, iter.Val())
	}
}

// GetStats returns cache statistics
func (rc *RedisCache) GetStats(ctx context.Context) *Stats {
	stats := *rc.stats

	// Get count of cache entries from Redis
	count := int64(0)
	iter := rc.client.Scan(ctx, 0, "cache:*", 0).Iterator()
	for iter.Next(ctx) {
		count++
	}
	stats.TotalEntries = count

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	return &stats
}

// InvalidateByProvider removes all cache entries for a specific provider
func (rc *RedisCache) InvalidateByProvider(ctx context.Context, providerID string) int {
	if !rc.config.Enabled {
		return 0
	}

	return rc.invalidateByMetadata(ctx, "provider_id", providerID)
}

// InvalidateByModel removes all cache entries for a specific model
func (rc *RedisCache) InvalidateByModel(ctx context.Context, modelName string) int {
	if !rc.config.Enabled {
		return 0
	}

	return rc.invalidateByMetadata(ctx, "model_name", modelName)
}

// InvalidateByAge removes entries older than the specified duration
func (rc *RedisCache) InvalidateByAge(ctx context.Context, maxAge time.Duration) int {
	if !rc.config.Enabled {
		return 0
	}

	threshold := time.Now().Add(-maxAge)
	removed := 0

	iter := rc.client.Scan(ctx, 0, "cache:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		val, err := rc.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(val), &entry); err != nil {
			continue
		}

		if entry.CachedAt.Before(threshold) {
			rc.client.Del(ctx, key)
			removed++
		}
	}

	return removed
}

// InvalidateByPattern removes all entries matching a key pattern
func (rc *RedisCache) InvalidateByPattern(ctx context.Context, pattern string) int {
	if !rc.config.Enabled {
		return 0
	}

	removed := 0
	iter := rc.client.Scan(ctx, 0, "cache:"+pattern+"*", 0).Iterator()
	for iter.Next(ctx) {
		rc.client.Del(ctx, iter.Val())
		removed++
	}

	return removed
}

// invalidateByMetadata is a helper to invalidate by metadata field
func (rc *RedisCache) invalidateByMetadata(ctx context.Context, field, value string) int {
	removed := 0

	iter := rc.client.Scan(ctx, 0, "cache:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		val, err := rc.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(val), &entry); err != nil {
			continue
		}

		// Check metadata field
		shouldInvalidate := false
		switch field {
		case "provider_id":
			shouldInvalidate = entry.ProviderID == value
		case "model_name":
			shouldInvalidate = entry.ModelName == value
		}

		if shouldInvalidate {
			rc.client.Del(ctx, key)
			removed++
		}
	}

	return removed
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}
