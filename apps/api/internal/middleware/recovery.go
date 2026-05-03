package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Recovery catches any panic that escapes a handler, logs the stack trace, and
// returns a 500 Internal Server Error. This prevents a single bad handler from
// crashing the entire server process.
func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) (retErr error) {
		defer func() {
			if r := recover(); r != nil {
				requestID, _ := c.Locals("request_id").(string)
				stack := debug.Stack()

				log.Error().
					Str("request_id", requestID).
					Str("panic", fmt.Sprintf("%v", r)).
					Bytes("stack", stack).
					Msg("panic recovered")

				retErr = c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
			}
		}()
		return c.Next()
	}
}
