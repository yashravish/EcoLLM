package auth

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// newHandlerApp wires up a Fiber app with a real Service backed by a fakeRepo
// and an in-process miniredis instance. auditRepo is intentionally nil — the
// handler guards all audit calls with `if h.auditRepo != nil`.
func newHandlerApp(t *testing.T, repo RepositoryI) (*fiber.App, *Handler) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	svc := NewService(repo, rc, "handler-test-secret")
	h := NewHandler(svc)

	app := fiber.New(fiber.Config{
		// Suppress fiber's default HTML error pages; we want JSON bodies.
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		},
	})
	app.Post("/auth/login", h.Login)
	// /me requires user_id in locals — inject it via a thin test middleware.
	app.Get("/me", func(c *fiber.Ctx) error {
		// Tests set "X-Test-UserID" / "X-Test-OrgID" headers to simulate JWT middleware.
		if uid := c.Get("X-Test-UserID"); uid != "" {
			c.Locals("user_id", uid)
		}
		if oid := c.Get("X-Test-OrgID"); oid != "" {
			c.Locals("org_id", oid)
		}
		return h.Me(c)
	})
	return app, h
}

// ── POST /auth/login ──────────────────────────────────────────────────────────

func TestHandler_Login(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	goodUser := &User{
		ID:           userID,
		OrgID:        orgID,
		Email:        "alice@example.com",
		PasswordHash: mustHashPassword(t, "secret123"),
		Role:         "admin",
		Name:         "Alice",
		AuthMethod:   "password",
	}
	goodOrg := &Organization{ID: orgID, Name: "Acme", Slug: "acme", Plan: "free"}

	tests := []struct {
		name       string
		body       string
		setup      func(*fakeRepo)
		wantStatus int
		checkBody  func(*testing.T, []byte)
	}{
		{
			name:       "missing / malformed JSON body returns 400",
			body:       "not-json",
			setup:      func(*fakeRepo) {},
			wantStatus: 400,
		},
		{
			name:       "empty email returns 400",
			body:       `{"email":"","password":"secret123"}`,
			setup:      func(*fakeRepo) {},
			wantStatus: 400,
		},
		{
			name:       "empty password returns 400",
			body:       `{"email":"alice@example.com","password":""}`,
			setup:      func(*fakeRepo) {},
			wantStatus: 400,
		},
		{
			name:       "nonexistent user returns 401",
			body:       `{"email":"ghost@example.com","password":"secret123"}`,
			setup:      func(*fakeRepo) {},
			wantStatus: 401,
		},
		{
			name: "wrong password returns 401",
			body: `{"email":"alice@example.com","password":"wrongpw"}`,
			setup: func(r *fakeRepo) {
				r.userByEmail[goodUser.Email] = goodUser
				r.orgs[orgID] = goodOrg
			},
			wantStatus: 401,
		},
		// ── edge-case / security cases ──────────────────────────────────────
		{
			name:       "empty JSON object {} returns 400",
			body:       `{}`,
			setup:      func(*fakeRepo) {},
			wantStatus: 400,
		},
		{
			name:       "invalid email format (no @) returns 400",
			body:       `{"email":"notanemail","password":"pw123"}`,
			setup:      func(*fakeRepo) {},
			wantStatus: 400,
		},
		{
			name:       "SQL injection in email field returns 400 (format check)",
			body:       `{"email":"'; DROP TABLE users; --","password":"pw123"}`,
			setup:      func(*fakeRepo) {},
			wantStatus: 400,
		},
		{
			name:       "unicode email with @ (valid format, user not found) returns 401",
			body:       `{"email":"тест@example.com","password":"pw123"}`,
			setup:      func(*fakeRepo) {},
			wantStatus: 401,
		},
		{
			name: "SQL injection in password field returns 401 (service rejects)",
			body: `{"email":"alice@example.com","password":"'; DROP TABLE users; --"}`,
			setup: func(r *fakeRepo) {
				r.userByEmail[goodUser.Email] = goodUser
				r.orgs[orgID] = goodOrg
			},
			wantStatus: 401,
		},
		{
			name: "success returns 200 with token user and org",
			body: `{"email":"alice@example.com","password":"secret123"}`,
			setup: func(r *fakeRepo) {
				r.userByEmail[goodUser.Email] = goodUser
				r.orgs[orgID] = goodOrg
			},
			wantStatus: 200,
			checkBody: func(t *testing.T, body []byte) {
				t.Helper()
				var resp loginResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("unmarshal loginResponse: %v", err)
				}
				if resp.Token == "" {
					t.Error("response.token is empty")
				}
				if resp.User == nil {
					t.Error("response.user is nil")
				} else {
					if resp.User.Email != goodUser.Email {
						t.Errorf("user.email = %q, want %q", resp.User.Email, goodUser.Email)
					}
					if resp.User.Role != goodUser.Role {
						t.Errorf("user.role = %q, want %q", resp.User.Role, goodUser.Role)
					}
				}
				if resp.Org == nil {
					t.Error("response.org is nil")
				} else if resp.Org.ID != goodOrg.ID.String() {
					t.Errorf("org.id = %q, want %q", resp.Org.ID, goodOrg.ID.String())
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			tc.setup(repo)
			app, _ := newHandlerApp(t, repo)

			req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, 5000)
			if err != nil {
				t.Fatalf("app.Test(): %v", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", resp.StatusCode, tc.wantStatus, body)
			}
			if tc.checkBody != nil {
				tc.checkBody(t, body)
			}
		})
	}

	// ── long-input edge cases (computed bodies) ───────────────────────────────

	t.Run("extremely long email (5000 chars) returns 400", func(t *testing.T) {
		longEmail := strings.Repeat("a", 5000) + "@example.com"
		body := `{"email":"` + longEmail + `","password":"pw123"}`
		repo := newFakeRepo()
		app, _ := newHandlerApp(t, repo)

		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("app.Test(): %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 400 {
			b, _ := io.ReadAll(resp.Body)
			t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, b)
		}
	})

	t.Run("extremely long password (5000 chars) returns 400", func(t *testing.T) {
		body := `{"email":"alice@example.com","password":"` + strings.Repeat("p", 5000) + `"}`
		repo := newFakeRepo()
		app, _ := newHandlerApp(t, repo)

		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("app.Test(): %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 400 {
			b, _ := io.ReadAll(resp.Body)
			t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, b)
		}
	})
}

// ── GET /me ───────────────────────────────────────────────────────────────────

func TestHandler_Me(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	user := &User{ID: userID, OrgID: orgID, Email: "alice@example.com", Role: "admin", Name: "Alice", AuthMethod: "password"}
	org := &Organization{ID: orgID, Name: "Acme", Slug: "acme", Plan: "free"}

	t.Run("no user_id in context returns 401", func(t *testing.T) {
		repo := newFakeRepo()
		app, _ := newHandlerApp(t, repo)

		req := httptest.NewRequest("GET", "/me", nil)
		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("app.Test(): %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 401 {
			t.Errorf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("valid user and org returns 200", func(t *testing.T) {
		repo := newFakeRepo()
		repo.userByID[userID] = user
		repo.orgs[orgID] = org
		app, _ := newHandlerApp(t, repo)

		req := httptest.NewRequest("GET", "/me", nil)
		req.Header.Set("X-Test-UserID", userID.String())
		req.Header.Set("X-Test-OrgID", orgID.String())

		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("app.Test(): %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			t.Errorf("status = %d, want 200; body: %s", resp.StatusCode, body)
		}

		var result map[string]json.RawMessage
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("unmarshal /me response: %v", err)
		}
		if _, ok := result["user"]; !ok {
			t.Error("/me response missing 'user' key")
		}
		if _, ok := result["org"]; !ok {
			t.Error("/me response missing 'org' key")
		}
	})

	t.Run("oauth user without org returns 200 with no org key", func(t *testing.T) {
		repo := newFakeRepo()
		oauthUser := *user
		oauthUser.OrgID = uuid.Nil
		oauthUser.AuthMethod = "oauth"
		repo.userByID[userID] = &oauthUser
		app, _ := newHandlerApp(t, repo)

		req := httptest.NewRequest("GET", "/me", nil)
		req.Header.Set("X-Test-UserID", userID.String())
		// No X-Test-OrgID → simulates JWT with empty org_id

		resp, err := app.Test(req, 5000)
		if err != nil {
			t.Fatalf("app.Test(): %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			t.Errorf("status = %d, want 200; body: %s", resp.StatusCode, body)
		}

		var result map[string]json.RawMessage
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("unmarshal /me response: %v", err)
		}
		if _, ok := result["user"]; !ok {
			t.Error("/me response missing 'user' key")
		}
		if _, ok := result["org"]; ok {
			t.Error("/me response should not contain 'org' key for org-less user")
		}
	})
}
