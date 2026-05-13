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

// Auth validates a Bearer token on /v1/* routes. It accepts either:
//   - An API key (for external developer access)
//   - A JWT token (for dashboard access)
//
// JWT is tried first; API key is the fallback.
func Auth(svc AuthService, jwtSecret string, redisClient *redis.Client) fiber.Handler {
	secret := []byte(jwtSecret)
	return func(c *fiber.Ctx) error {
		rawToken, err := extractBearer(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}

		// Try JWT first (dashboard users).
		if claims, err := parseJWT(rawToken, secret); err == nil {
			if !checkSession(c, redisClient, claims) {
				return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
			}
			setJWTLocals(c, claims)
			return c.Next()
		}

		// Fall back to API key (external developer access).
		orgID, scopes, err := svc.ValidateAPIKey(c.UserContext(), rawToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}
		c.Locals("org_id", orgID)
		c.Locals("api_key", rawToken)
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
		rawToken, err := extractBearer(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}
		claims, err := parseJWT(rawToken, secret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}
		if !checkSession(c, redisClient, claims) {
			return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
		}
		setJWTLocals(c, claims)
		return c.Next()
	}
}

func extractBearer(c *fiber.Ctx) (string, error) {
	const prefix = "Bearer "
	h := c.Get("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return "", fmt.Errorf("missing bearer token")
	}
	tok := strings.TrimPrefix(h, prefix)
	if tok == "" {
		return "", fmt.Errorf("empty token")
	}
	return tok, nil
}

func parseJWT(rawToken string, secret []byte) (jwt.MapClaims, error) {
	token, err := jwt.Parse(rawToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}

// checkSession verifies the JWT's jti is present in Redis (not revoked).
// Returns true if the session is valid (or has no jti to check).
func checkSession(c *fiber.Ctx, redisClient *redis.Client, claims jwt.MapClaims) bool {
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return true
	}
	return redisClient.Exists(c.UserContext(), "session:"+jti).Val() != 0
}

func setJWTLocals(c *fiber.Ctx, claims jwt.MapClaims) {
	c.Locals("user_id", claims["sub"])
	c.Locals("org_id", claims["org_id"])
	c.Locals("role", claims["role"])
	jti, _ := claims["jti"].(string)
	c.Locals("jti", jti)
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
