package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client

// DefaultTTL is the lifetime of a cached QR record.
const DefaultTTL = 60 * time.Second

// Init connects to Redis using a connection URL (e.g. "redis://localhost:6379/0").
func Init(url string) {
	if url == "" {
		log.Printf("[cache] REDIS_URL not set — QR caching disabled")
		return
	}

	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Printf("[cache] invalid REDIS_URL (%v) — QR caching disabled", err)
		return
	}

	c := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		log.Printf("[cache] Redis unreachable (%v) — QR caching disabled", err)
		return
	}

	client = c
	log.Printf("[cache] Connected to Redis — QR caching enabled (TTL %s)", DefaultTTL)
}

// Close releases the Redis connection.
func Close() {
	if client != nil {
		_ = client.Close()
	}
}

// Enabled reports whether caching is active.
func Enabled() bool { return client != nil }

// Get returns the cached value for key and whether it was found.
func Get(ctx context.Context, key string) (string, bool) {
	if client == nil {
		return "", false
	}
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

// Set stores value under key with the given TTL.
func Set(ctx context.Context, key, value string, ttl time.Duration) {
	if client == nil {
		return
	}
	if err := client.Set(ctx, key, value, ttl).Err(); err != nil {
		log.Printf("[cache] set %q failed: %v", key, err)
	}
}

var loadGroup flightGroup

// GetOrLoad returns the value for key, fetching it via loader on a cache miss and storing it.
func GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader func() (string, bool, error)) (string, bool, error) {
	if v, ok := Get(ctx, key); ok {
		return v, true, nil
	}

	return loadGroup.do(key, func() (string, bool, error) {
		if v, ok := Get(ctx, key); ok {
			return v, true, nil
		}
		val, found, err := loader()
		if err != nil || !found {
			return val, found, err
		}
		Set(ctx, key, val, ttl)
		return val, true, nil
	})
}

// Delete removes key from the cache.
func Delete(ctx context.Context, key string) {
	if client == nil {
		return
	}
	if err := client.Del(ctx, key).Err(); err != nil {
		log.Printf("[cache] delete %q failed: %v", key, err)
	}
}

// QRKey builds the cache key for a QR code looked up by its short code.
func QRKey(shortCode string) string {
	return "qr:" + shortCode
}
