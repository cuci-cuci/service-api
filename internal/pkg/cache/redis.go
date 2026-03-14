package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a Redis client from a URL string.
// URL format: redis://<user>:<password>@<host>:<port>/<db>
func NewRedisClient(url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opts), nil
}

// IsAvailable pings the Redis server and returns true if it responds.
func IsAvailable(client *redis.Client) bool {
	if client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return client.Ping(ctx).Err() == nil
}
