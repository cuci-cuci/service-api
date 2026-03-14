package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/bangun-ekosistem/service-api/internal/pkg/apperror"
	"github.com/bangun-ekosistem/service-api/internal/pkg/cache"
	"github.com/bangun-ekosistem/service-api/internal/pkg/response"
)

// RateLimiterBackend is the interface for rate limiter implementations.
type RateLimiterBackend interface {
	// Allow returns true if the request identified by key is within the rate limit.
	Allow(key string) bool
}

// --- In-memory backend ---

type memoryRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

func newMemoryRateLimiter(max int, window time.Duration) *memoryRateLimiter {
	rl := &memoryRateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
	// Cleanup stale entries every minute
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *memoryRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Filter to only recent attempts
	recent := make([]time.Time, 0)
	for _, t := range rl.attempts[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.max {
		rl.attempts[key] = recent
		return false
	}

	rl.attempts[key] = append(recent, now)
	return true
}

func (rl *memoryRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.window)
	for key, times := range rl.attempts {
		recent := make([]time.Time, 0)
		for _, t := range times {
			if t.After(cutoff) {
				recent = append(recent, t)
			}
		}
		if len(recent) == 0 {
			delete(rl.attempts, key)
		} else {
			rl.attempts[key] = recent
		}
	}
}

// --- Redis backend ---

type redisRateLimiter struct {
	client *redis.Client
	max    int
	window time.Duration
}

func newRedisRateLimiter(client *redis.Client, max int, window time.Duration) *redisRateLimiter {
	return &redisRateLimiter{
		client: client,
		max:    max,
		window: window,
	}
}

func (rl *redisRateLimiter) Allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	redisKey := fmt.Sprintf("rl:%s", key)

	// Use INCR + EXPIRE for a simple sliding-window-ish counter.
	count, err := rl.client.Incr(ctx, redisKey).Result()
	if err != nil {
		slog.Warn("redis rate limiter INCR failed, allowing request", "error", err)
		return true // fail open
	}

	// Set expiry only on the first request in the window (count == 1).
	if count == 1 {
		if err := rl.client.Expire(ctx, redisKey, rl.window).Err(); err != nil {
			slog.Warn("redis rate limiter EXPIRE failed", "error", err)
		}
	}

	return count <= int64(rl.max)
}

// --- Middleware constructors ---

// rateLimitMiddleware builds the Chi middleware from any RateLimiterBackend.
func rateLimitMiddleware(backend RateLimiterBackend) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if forwarded := r.Header.Get("X-Real-Ip"); forwarded != "" {
				ip = forwarded
			}

			if !backend.Allow(ip) {
				response.Error(w, apperror.NewAppError(429, "terlalu banyak permintaan, coba lagi nanti"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit applies per-IP rate limiting using an in-memory backend.
// max: maximum requests allowed in the window.
// This is kept for backwards compatibility.
func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimitMiddleware(newMemoryRateLimiter(max, window))
}

// RateLimitWithRedis applies per-IP rate limiting.
// If redisClient is non-nil and reachable, it uses the Redis backend;
// otherwise it falls back to in-memory.
func RateLimitWithRedis(redisClient *redis.Client, max int, window time.Duration) func(http.Handler) http.Handler {
	if redisClient != nil && cache.IsAvailable(redisClient) {
		slog.Info("rate limiter using redis backend")
		return rateLimitMiddleware(newRedisRateLimiter(redisClient, max, window))
	}
	slog.Info("rate limiter using in-memory backend")
	return rateLimitMiddleware(newMemoryRateLimiter(max, window))
}
