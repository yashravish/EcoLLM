package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// AuthService is the interface the API-key auth middleware depends on.
type AuthService interface {
	ValidateAPIKey(ctx context.Context, rawKey string) (orgID string, scopes []string, err error)
}

// Auth validates a Bearer API key and populates org_id/scopes locals.
// Used on /v1/* inference routes.
func Auth(svc AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}

		rawKey := strings.TrimPrefix(authHeader, prefix)
		if rawKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}

		orgID, scopes, err := svc.ValidateAPIKey(c.UserContext(), rawKey)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}

		c.Locals("org_id", orgID)
		c.Locals("api_key", rawKey)
		c.Locals("scopes", scopes)

		return c.Next()
	}
}

// JWTAuth validates a signed JWT in the Authorization header and populates
// user_id, org_id, role, and jti locals. Checks Redis for session revocation.
// Used on dashboard and admin routes.
func JWTAuth(jwtSecret string, redisClient *redis.Client) fiber.Handler {
	secret := []byte(jwtSecret)
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}

		rawToken := strings.TrimPrefix(authHeader, prefix)

		token, err := jwt.Parse(rawToken, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}

		jti, _ := claims["jti"].(string)
		if jti != "" {
			if redisClient.Exists(c.UserContext(), "session:"+jti).Val() == 0 {
				return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
			}
		}

		c.Locals("user_id", claims["sub"])
		c.Locals("org_id", claims["org_id"])
		c.Locals("role", claims["role"])
		c.Locals("jti", jti)

		return c.Next()
	}
}

// RequireRole returns a middleware that enforces that the authenticated user
// has one of the allowed roles stored in Fiber locals.
func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *fiber.Ctx) error {
		userRole, _ := c.Locals("role").(string)
		if !allowed[userRole] {
			return c.Status(fiber.StatusForbidden).JSON(apierror.ErrForbidden)
		}
		return c.Next()
	}
}
