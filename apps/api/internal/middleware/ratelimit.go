package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RateLimit implements a Redis-backed sliding window rate limiter keyed on the
// API key. On Redis failure it fails open — requests are allowed through — so
// that a cache outage does not become a service outage.
//
// Redis key pattern: rl:{api_key}:{window_start_unix}  TTL: window
func RateLimit(redisClient *redis.Client, limit int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rawKey, _ := c.Locals("api_key").(string)
		if rawKey == "" {
			// No key yet (unauthenticated); let auth middleware reject it.
			return c.Next()
		}

		// Bucket by window start so old buckets expire naturally.
		windowStart := time.Now().Truncate(window).Unix()
		redisKey := fmt.Sprintf("rl:%s:%d", rawKey[:min(10, len(rawKey))], windowStart)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		count, err := redisClient.Incr(ctx, redisKey).Result()
		if err != nil {
			// Fail open: log but allow the request.
			log.Warn().Err(err).Str("key", redisKey).Msg("rate limit redis error, failing open")
			return c.Next()
		}

		if count == 1 {
			// First hit in this window — set expiry.
			redisClient.Expire(ctx, redisKey, window)
		}

		remaining := limit - int(count)
		if remaining < 0 {
			remaining = 0
		}

		c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(windowStart+int64(window.Seconds()), 10))

		if int(count) > limit {
			return c.Status(fiber.StatusTooManyRequests).JSON(apierror.ErrRateLimited)
		}

		return c.Next()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
