package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logging returns a Fiber middleware that emits a structured zerolog entry for
// every request: method, path, status, latency, and request_id.
// org_id is included when available (set by auth middleware downstream).
// Requests below minLevel are not logged (e.g. "warn" suppresses 2xx/3xx lines).
func Logging(logLevel string) fiber.Handler {
	minLevel, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		minLevel = zerolog.InfoLevel
	}

	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		status := c.Response().StatusCode()

		var eventLevel zerolog.Level
		switch {
		case status >= 500:
			eventLevel = zerolog.ErrorLevel
		case status >= 400:
			eventLevel = zerolog.WarnLevel
		default:
			eventLevel = zerolog.InfoLevel
		}

		if eventLevel < minLevel {
			return err
		}

		requestID, _ := c.Locals("request_id").(string)
		orgID, _ := c.Locals("org_id").(string)

		log.WithLevel(eventLevel).
			Str("request_id", requestID).
			Str("org_id", orgID).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", status).
			Int64("latency_ms", time.Since(start).Milliseconds()).
			Str("ip", c.IP()).
			Msg("http request")

		return err
	}
}
