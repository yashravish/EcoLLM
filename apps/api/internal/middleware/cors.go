package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// CORS returns a middleware that sets Access-Control-Allow-* headers based on
// a whitelist of allowed origins. Wildcard "*" is never used in production.
func CORS(allowedOrigins []string) fiber.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = true
	}

	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		if origin != "" && allowed[origin] {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Credentials", "true")
			c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			c.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			c.Set("Vary", "Origin")
		}

		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}
