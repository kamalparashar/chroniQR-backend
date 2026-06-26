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
		log.Printf("[cache] Get: %q (Redis client disabled/uninitialized)", key)
		return "", false
	}
	log.Printf("[cache] Get: querying key %q from Redis", key)
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			log.Printf("[cache] Get: %q -> CACHE MISS (key does not exist)", key)
		} else {
			log.Printf("[cache] Get: %q -> ERROR querying Redis: %v", key, err)
		}
		return "", false
	}
	log.Printf("[cache] Get: %q -> CACHE HIT (%d bytes)", key, len(val))
	return val, true
}

// Set stores value under key with the given TTL.
func Set(ctx context.Context, key, value string, ttl time.Duration) {
	if client == nil {
		log.Printf("[cache] Set: %q (Redis client disabled/uninitialized)", key)
		return
	}
	log.Printf("[cache] Set: storing key %q in Redis with TTL %v (%d bytes)", key, ttl, len(value))
	if err := client.Set(ctx, key, value, ttl).Err(); err != nil {
		log.Printf("[cache] Set: ERROR storing key %q in Redis: %v", key, err)
	} else {
		log.Printf("[cache] Set: successfully stored key %q in Redis", key)
	}
}

var loadGroup flightGroup

// GetOrLoad returns the value for key, fetching it via loader on a cache miss and storing it.
func GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader func() (string, bool, error)) (string, bool, error) {
	log.Printf("[cache] GetOrLoad: request initiated for key %q", key)
	if v, ok := Get(ctx, key); ok {
		log.Printf("[cache] GetOrLoad: returned %q directly from cache", key)
		return v, true, nil
	}
	log.Printf("[cache] GetOrLoad: cache miss for key %q, passing to loader callback", key)

	return loadGroup.do(key, func() (string, bool, error) {
		log.Printf("[cache] GetOrLoad/singleflight: running loader closure for key %q", key)
		// Double check cache in case a parallel request loaded it already
		if v, ok := Get(ctx, key); ok {
			log.Printf("[cache] GetOrLoad/singleflight: duplicate load avoided, key %q resolved from cache", key)
			return v, true, nil
		}
		
		log.Printf("[cache] GetOrLoad/singleflight: calling loader function for key %q", key)
		val, found, err := loader()
		if err != nil {
			log.Printf("[cache] GetOrLoad/singleflight: loader returned error for key %q: %v", key, err)
			return val, found, err
		}
		if !found {
			log.Printf("[cache] GetOrLoad/singleflight: loader returned not_found for key %q", key)
			return val, found, nil
		}
		
		log.Printf("[cache] GetOrLoad/singleflight: loader successfully fetched %q (%d bytes)", key, len(val))
		Set(ctx, key, val, ttl)
		return val, true, nil
	})
}

// Delete removes key from the cache.
func Delete(ctx context.Context, key string) {
	if client == nil {
		log.Printf("[cache] Delete: %q (Redis client disabled/uninitialized)", key)
		return
	}
	log.Printf("[cache] Delete: deleting key %q from Redis", key)
	if err := client.Del(ctx, key).Err(); err != nil {
		log.Printf("[cache] Delete: ERROR deleting key %q from Redis: %v", key, err)
	} else {
		log.Printf("[cache] Delete: successfully deleted key %q from Redis", key)
	}
}

// QRKey builds the cache key for a QR code looked up by its short code.
func QRKey(shortCode string) string {
	return "qr:" + shortCode
}
