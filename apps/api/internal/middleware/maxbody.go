package middleware

import (
	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
)

const maxBodyBytes = 1 * 1024 * 1024 // 1 MB

// MaxBody rejects any request whose Content-Length exceeds 1 MB.
// Oversized requests are rejected before body parsing to prevent memory abuse.
func MaxBody() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Request().Header.ContentLength() > maxBodyBytes {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(apierror.ErrRequestTooLarge)
		}
		return c.Next()
	}
}
