package middleware

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// newApp creates a minimal Fiber app wired with the given middleware and a
// trailing handler that always returns 200 OK.
func newApp(mw fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(mw)
	app.Use(func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	return app
}

func do(app *fiber.App, method, path string, headers map[string]string) (int, map[string]string) {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req, 3000)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	respHeaders := make(map[string]string)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}
	return resp.StatusCode, respHeaders
}

// ── MaxBody ───────────────────────────────────────────────────────────────────
//
// MaxBody checks c.Request().Header.ContentLength() which fasthttp populates
// from the wire Content-Length header. We must send an actual body so that
// Go's http.Request.Write() emits the correct Content-Length header; setting
// the header manually on a nil-body request causes fasthttp to stall waiting
// for body bytes that never arrive.

func TestMaxBody_NoBody_Passes(t *testing.T) {
	app := newApp(MaxBody())
	status, _ := do(app, "GET", "/", nil)
	if status != 200 {
		t.Errorf("status = %d, want 200 for request with no body", status)
	}
}

func TestMaxBody_SmallBody_Passes(t *testing.T) {
	app := newApp(MaxBody())
	req := httptest.NewRequest("POST", "/", strings.NewReader("small payload"))
	resp, err := app.Test(req, 3000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestMaxBody_OverLimit_Returns413(t *testing.T) {
	app := newApp(MaxBody())
	// Provide maxBodyBytes+1 bytes so Content-Length is set automatically.
	req := httptest.NewRequest("POST", "/", bytes.NewReader(make([]byte, maxBodyBytes+1)))
	resp, err := app.Test(req, 10000)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 413 {
		t.Errorf("status = %d, want 413", resp.StatusCode)
	}
}

// ── RequestID ─────────────────────────────────────────────────────────────────

func TestRequestID_GeneratesIDWhenMissing(t *testing.T) {
	app := newApp(RequestID())
	status, headers := do(app, "GET", "/", nil)
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
	if headers["X-Request-Id"] == "" {
		t.Error("X-Request-ID header should be set when missing from request")
	}
}

func TestRequestID_ReusesProvidedID(t *testing.T) {
	const myID = "my-custom-request-id"
	app := newApp(RequestID())
	status, headers := do(app, "GET", "/", map[string]string{"X-Request-Id": myID})
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
	if headers["X-Request-Id"] != myID {
		t.Errorf("X-Request-ID = %q, want %q", headers["X-Request-Id"], myID)
	}
}

func TestRequestID_GeneratedIDIsValidUUID(t *testing.T) {
	app := newApp(RequestID())
	_, headers := do(app, "GET", "/", nil)
	id := headers["X-Request-Id"]
	if _, err := uuid.Parse(id); err != nil {
		t.Errorf("generated X-Request-ID %q is not a valid UUID: %v", id, err)
	}
}

func TestRequestID_DifferentRequestsDifferentIDs(t *testing.T) {
	app := newApp(RequestID())
	_, h1 := do(app, "GET", "/", nil)
	_, h2 := do(app, "GET", "/", nil)
	if h1["X-Request-Id"] == h2["X-Request-Id"] {
		t.Error("successive requests should get different request IDs")
	}
}

// ── CORS ──────────────────────────────────────────────────────────────────────

func TestCORS_AllowedOrigin_SetsHeaders(t *testing.T) {
	const origin = "https://app.example.com"
	app := newApp(CORS([]string{origin}))
	status, headers := do(app, "GET", "/", map[string]string{"Origin": origin})
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
	if headers["Access-Control-Allow-Origin"] != origin {
		t.Errorf("ACAO = %q, want %q", headers["Access-Control-Allow-Origin"], origin)
	}
	if headers["Access-Control-Allow-Credentials"] != "true" {
		t.Error("ACAC should be true for allowed origin")
	}
}

func TestCORS_UnknownOrigin_NoHeaders(t *testing.T) {
	app := newApp(CORS([]string{"https://allowed.example.com"}))
	status, headers := do(app, "GET", "/", map[string]string{"Origin": "https://evil.example.com"})
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
	if headers["Access-Control-Allow-Origin"] != "" {
		t.Errorf("ACAO should be empty for unknown origin, got %q", headers["Access-Control-Allow-Origin"])
	}
}

func TestCORS_NoOriginHeader_Passes(t *testing.T) {
	app := newApp(CORS([]string{"https://app.example.com"}))
	status, _ := do(app, "GET", "/", nil)
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestCORS_OPTIONS_Returns204(t *testing.T) {
	const origin = "https://app.example.com"
	app := newApp(CORS([]string{origin}))
	status, _ := do(app, "OPTIONS", "/", map[string]string{"Origin": origin})
	if status != 204 {
		t.Errorf("status = %d, want 204 for OPTIONS preflight", status)
	}
}

func TestCORS_WhitespaceTrimmedOnInit(t *testing.T) {
	app := newApp(CORS([]string{"  https://app.example.com  ", " https://other.com "}))
	status, headers := do(app, "GET", "/", map[string]string{"Origin": "https://app.example.com"})
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
	if headers["Access-Control-Allow-Origin"] != "https://app.example.com" {
		t.Error("origin with surrounding whitespace in config should still match")
	}
}

func TestCORS_VaryHeaderSet(t *testing.T) {
	const origin = "https://app.example.com"
	app := newApp(CORS([]string{origin}))
	_, headers := do(app, "GET", "/", map[string]string{"Origin": origin})
	if headers["Vary"] != "Origin" {
		t.Errorf("Vary = %q, want Origin", headers["Vary"])
	}
}

// ── RequireRole ───────────────────────────────────────────────────────────────

func TestRequireRole_MatchingRole_Passes(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "admin")
		return c.Next()
	})
	app.Use(RequireRole("admin"))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	status, _ := do(app, "GET", "/", nil)
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestRequireRole_MultipleAllowedRoles(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "member")
		return c.Next()
	})
	app.Use(RequireRole("admin", "member"))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	status, _ := do(app, "GET", "/", nil)
	if status != 200 {
		t.Errorf("status = %d, want 200 for member role", status)
	}
}

func TestRequireRole_WrongRole_Returns403(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "viewer")
		return c.Next()
	})
	app.Use(RequireRole("admin"))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	status, _ := do(app, "GET", "/", nil)
	if status != 403 {
		t.Errorf("status = %d, want 403", status)
	}
}

func TestRequireRole_NoRole_Returns403(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(RequireRole("admin"))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	status, _ := do(app, "GET", "/", nil)
	if status != 403 {
		t.Errorf("status = %d, want 403 when no role set", status)
	}
}

// ── RateLimit ─────────────────────────────────────────────────────────────────

func newRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func appWithRateLimit(rc *redis.Client, limit int) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	// Inject a fake api_key local so RateLimit actually runs.
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("api_key", "eco-sk-testkey12345")
		return c.Next()
	})
	app.Use(RateLimit(rc, limit, time.Minute))
	app.Use(func(c *fiber.Ctx) error { return c.SendStatus(200) })
	return app
}

func TestRateLimit_UnderLimit_Passes(t *testing.T) {
	rc := newRedisClient(t)
	app := appWithRateLimit(rc, 5)
	status, _ := do(app, "GET", "/", nil)
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestRateLimit_SetsHeaders(t *testing.T) {
	rc := newRedisClient(t)
	app := appWithRateLimit(rc, 10)
	_, headers := do(app, "GET", "/", nil)
	if headers["X-Ratelimit-Limit"] == "" {
		t.Error("X-RateLimit-Limit header missing")
	}
	if headers["X-Ratelimit-Remaining"] == "" {
		t.Error("X-RateLimit-Remaining header missing")
	}
	if headers["X-Ratelimit-Reset"] == "" {
		t.Error("X-RateLimit-Reset header missing")
	}
}

func TestRateLimit_ExceedsLimit_Returns429(t *testing.T) {
	rc := newRedisClient(t)
	const limit = 3
	app := appWithRateLimit(rc, limit)
	for i := 0; i < limit; i++ {
		status, _ := do(app, "GET", "/", nil)
		if status != 200 {
			t.Fatalf("request %d: status = %d, want 200", i+1, status)
		}
	}
	// (limit+1)th request should be rejected.
	status, _ := do(app, "GET", "/", nil)
	if status != 429 {
		t.Errorf("status = %d, want 429 after exceeding limit", status)
	}
}

func TestRateLimit_NoAPIKey_PassesThrough(t *testing.T) {
	rc := newRedisClient(t)
	// Do NOT inject api_key local — middleware should skip rate limiting.
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(RateLimit(rc, 1, time.Minute))
	app.Use(func(c *fiber.Ctx) error { return c.SendStatus(200) })

	for i := 0; i < 5; i++ {
		status, _ := do(app, "GET", "/", nil)
		if status != 200 {
			t.Errorf("request %d without api_key: status = %d, want 200", i+1, status)
		}
	}
}

// ── JWTAuth ───────────────────────────────────────────────────────────────────

const testJWTSecret = "middleware-test-secret"

func makeJWT(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	return s
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub":    uuid.NewString(),
		"org_id": uuid.NewString(),
		"role":   "admin",
		"jti":    uuid.NewString(),
		"exp":    time.Now().Add(time.Hour).Unix(),
	}
}

func newJWTApp(mr *miniredis.Miniredis) *fiber.App {
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(JWTAuth(testJWTSecret, rc))
	app.Use(func(c *fiber.Ctx) error { return c.SendStatus(200) })
	return app
}

func TestJWTAuth_ValidToken_Passes(t *testing.T) {
	mr := miniredis.RunT(t)
	claims := validClaims()
	token := makeJWT(t, testJWTSecret, claims)
	jti := claims["jti"].(string)
	mr.Set("session:"+jti, "1") //nolint:errcheck

	app := newJWTApp(mr)
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Bearer " + token})
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestJWTAuth_NoAuthHeader_Returns401(t *testing.T) {
	mr := miniredis.RunT(t)
	app := newJWTApp(mr)
	status, _ := do(app, "GET", "/", nil)
	if status != 401 {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestJWTAuth_WrongPrefix_Returns401(t *testing.T) {
	mr := miniredis.RunT(t)
	app := newJWTApp(mr)
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Token sometoken"})
	if status != 401 {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestJWTAuth_InvalidToken_Returns401(t *testing.T) {
	mr := miniredis.RunT(t)
	app := newJWTApp(mr)
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Bearer not.a.jwt"})
	if status != 401 {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestJWTAuth_WrongSecret_Returns401(t *testing.T) {
	mr := miniredis.RunT(t)
	claims := validClaims()
	token := makeJWT(t, "wrong-secret", claims)
	mr.Set("session:"+claims["jti"].(string), "1") //nolint:errcheck

	app := newJWTApp(mr)
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Bearer " + token})
	if status != 401 {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestJWTAuth_ExpiredToken_Returns401(t *testing.T) {
	mr := miniredis.RunT(t)
	claims := validClaims()
	claims["exp"] = time.Now().Add(-time.Hour).Unix() // already expired
	token := makeJWT(t, testJWTSecret, claims)
	mr.Set("session:"+claims["jti"].(string), "1") //nolint:errcheck

	app := newJWTApp(mr)
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Bearer " + token})
	if status != 401 {
		t.Errorf("status = %d, want 401 for expired token", status)
	}
}

func TestJWTAuth_RevokedSession_Returns401(t *testing.T) {
	mr := miniredis.RunT(t)
	claims := validClaims()
	token := makeJWT(t, testJWTSecret, claims)
	// Do NOT add session key → simulates revoked/logged-out session.

	app := newJWTApp(mr)
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Bearer " + token})
	if status != 401 {
		t.Errorf("status = %d, want 401 for revoked session", status)
	}
}

// ── Auth (JWT + API key fallback) ─────────────────────────────────────────────

type fakeAuthSvc struct {
	validKey string
	orgID    string
}

func (f *fakeAuthSvc) ValidateAPIKey(_ context.Context, rawKey string) (string, []string, error) {
	if rawKey == f.validKey {
		return f.orgID, []string{"inference"}, nil
	}
	return "", nil, fiber.ErrUnauthorized
}

func newAuthApp(mr *miniredis.Miniredis, svc AuthService) *fiber.App {
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(Auth(svc, testJWTSecret, rc))
	app.Use(func(c *fiber.Ctx) error { return c.SendStatus(200) })
	return app
}

func TestAuth_ValidJWT_Passes(t *testing.T) {
	mr := miniredis.RunT(t)
	claims := validClaims()
	token := makeJWT(t, testJWTSecret, claims)
	mr.Set("session:"+claims["jti"].(string), "1") //nolint:errcheck

	app := newAuthApp(mr, &fakeAuthSvc{})
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Bearer " + token})
	if status != 200 {
		t.Errorf("status = %d, want 200 for valid JWT", status)
	}
}

func TestAuth_ValidAPIKey_Passes(t *testing.T) {
	mr := miniredis.RunT(t)
	const apiKey = "eco-sk-validkey123"
	svc := &fakeAuthSvc{validKey: apiKey, orgID: uuid.NewString()}
	app := newAuthApp(mr, svc)
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Bearer " + apiKey})
	if status != 200 {
		t.Errorf("status = %d, want 200 for valid API key", status)
	}
}

func TestAuth_NoHeader_Returns401(t *testing.T) {
	mr := miniredis.RunT(t)
	app := newAuthApp(mr, &fakeAuthSvc{})
	status, _ := do(app, "GET", "/", nil)
	if status != 401 {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestAuth_InvalidBothJWTAndAPIKey_Returns401(t *testing.T) {
	mr := miniredis.RunT(t)
	app := newAuthApp(mr, &fakeAuthSvc{validKey: "correct-key"})
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Bearer totally-invalid-token"})
	if status != 401 {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestAuth_NonBearerScheme_Returns401(t *testing.T) {
	mr := miniredis.RunT(t)
	app := newAuthApp(mr, &fakeAuthSvc{})
	status, _ := do(app, "GET", "/", map[string]string{"Authorization": "Basic dXNlcjpwYXNz"})
	if status != 401 {
		t.Errorf("status = %d, want 401 for non-Bearer scheme", status)
	}
}
