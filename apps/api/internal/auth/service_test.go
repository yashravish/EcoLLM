package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// ── fakeRepo ─────────────────────────────────────────────────────────────────

// fakeRepo is an in-memory RepositoryI used exclusively in unit tests.
type fakeRepo struct {
	userByEmail map[string]*User
	userByID    map[uuid.UUID]*User
	orgs        map[uuid.UUID]*Organization

	userByEmailErr error
	userByIDErr    error
	orgErr         error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		userByEmail: make(map[string]*User),
		userByID:    make(map[uuid.UUID]*User),
		orgs:        make(map[uuid.UUID]*Organization),
	}
}

func (f *fakeRepo) FindUserByEmail(_ context.Context, email string) (*User, error) {
	if f.userByEmailErr != nil {
		return nil, f.userByEmailErr
	}
	u, ok := f.userByEmail[email]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return u, nil
}

func (f *fakeRepo) FindUserByID(_ context.Context, id uuid.UUID) (*User, error) {
	if f.userByIDErr != nil {
		return nil, f.userByIDErr
	}
	u, ok := f.userByID[id]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return u, nil
}

func (f *fakeRepo) FindOrgByID(_ context.Context, id uuid.UUID) (*Organization, error) {
	if f.orgErr != nil {
		return nil, f.orgErr
	}
	o, ok := f.orgs[id]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return o, nil
}

// Remaining methods satisfy the interface but are not exercised by login tests.
func (f *fakeRepo) FindAPIKeyByPrefix(_ context.Context, _ string) (*APIKey, error) {
	return nil, pgx.ErrNoRows
}
func (f *fakeRepo) TouchAPIKeyLastUsed(_ context.Context, _ uuid.UUID) error { return nil }
func (f *fakeRepo) CreateOrgAndUser(_ context.Context, _ OrgInput, _ UserInput, _ *APIKey) (*Organization, *User, error) {
	return nil, nil, errors.New("not implemented in fake")
}
func (f *fakeRepo) UpsertOAuthUser(_ context.Context, _, _, _, _ string) (*User, error) {
	return nil, errors.New("not implemented in fake")
}
func (f *fakeRepo) CreateOrgForUser(_ context.Context, _ uuid.UUID, _ OrgInput) (*Organization, error) {
	return nil, errors.New("not implemented in fake")
}
func (f *fakeRepo) UpdateOrg(_ context.Context, _ uuid.UUID, _ string, _ float32, _ *float32) error {
	return nil
}
func (f *fakeRepo) ListMembers(_ context.Context, _ uuid.UUID) ([]Member, error) { return nil, nil }
func (f *fakeRepo) InviteMember(_ context.Context, _ uuid.UUID, _, _, _, _ string) (*User, error) {
	return nil, errors.New("not implemented in fake")
}
func (f *fakeRepo) UpdateMemberRole(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }
func (f *fakeRepo) ListAPIKeys(_ context.Context, _ uuid.UUID) ([]APIKey, error)       { return nil, nil }
func (f *fakeRepo) CreateAPIKey(_ context.Context, _ *APIKey) error                    { return nil }
func (f *fakeRepo) RevokeAPIKey(_ context.Context, _, _ uuid.UUID) error               { return nil }
func (f *fakeRepo) RemoveMember(_ context.Context, _, _ uuid.UUID) error { return nil }
func (f *fakeRepo) DeleteUser(_ context.Context, _ uuid.UUID) error      { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestService(t *testing.T, repo RepositoryI) (*Service, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewService(repo, rc, "test-jwt-secret"), mr
}

func mustHashPassword(t *testing.T, pw string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(h)
}

// ── slugify ───────────────────────────────────────────────────────────────────

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"acme", "acme"},
		{"ACME", "acme"},
		{"Acme Corp", "acme-corp"},
		{"Acme  Corp", "acme--corp"},
		{"Acme!@#Corp", "acmecorp"},
		{"acme-corp", "acme-corp"},
		{"acme_corp", "acme-corp"},
		{"acme123", "acme123"},
		{"", ""},
		{"日本語", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := slugify(tc.in)
			if got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestService_Login(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	goodUser := &User{
		ID:           userID,
		OrgID:        orgID,
		Email:        "alice@example.com",
		PasswordHash: mustHashPassword(t, "correct-pw"),
		Role:         "admin",
		Name:         "Alice",
		AuthMethod:   "password",
	}
	goodOrg := &Organization{ID: orgID, Name: "Acme", Slug: "acme", Plan: "free"}

	tests := []struct {
		name      string
		email     string
		password  string
		setup     func(*fakeRepo)
		wantErr   string
		wantToken bool
	}{
		{
			name:    "user not found",
			email:   "ghost@example.com",
			password: "anything",
			setup:   func(*fakeRepo) {},
			wantErr: "invalid credentials",
		},
		{
			name:    "oauth user has empty password hash",
			email:   "oauth@example.com",
			password: "anything",
			setup: func(r *fakeRepo) {
				u := *goodUser
				u.Email = "oauth@example.com"
				u.PasswordHash = ""
				u.AuthMethod = "oauth"
				r.userByEmail[u.Email] = &u
			},
			wantErr: "invalid credentials",
		},
		{
			name:    "wrong password",
			email:   goodUser.Email,
			password: "wrong-pw",
			setup: func(r *fakeRepo) {
				r.userByEmail[goodUser.Email] = goodUser
				r.orgs[orgID] = goodOrg
			},
			wantErr: "invalid credentials",
		},
		{
			name:    "user has no org (uuid.Nil)",
			email:   goodUser.Email,
			password: "correct-pw",
			setup: func(r *fakeRepo) {
				u := *goodUser
				u.OrgID = uuid.Nil
				r.userByEmail[u.Email] = &u
			},
			wantErr: "user has no organization",
		},
		{
			name:    "db error on user lookup",
			email:   goodUser.Email,
			password: "correct-pw",
			setup: func(r *fakeRepo) {
				r.userByEmailErr = errors.New("connection reset")
			},
			wantErr: "user lookup: connection reset",
		},
		{
			name:    "db error on org lookup",
			email:   goodUser.Email,
			password: "correct-pw",
			setup: func(r *fakeRepo) {
				r.userByEmail[goodUser.Email] = goodUser
				r.orgErr = errors.New("connection reset")
			},
			wantErr: "org lookup: connection reset",
		},
		{
			name:      "success — returns signed JWT with correct claims",
			email:     goodUser.Email,
			password:  "correct-pw",
			setup: func(r *fakeRepo) {
				r.userByEmail[goodUser.Email] = goodUser
				r.orgs[orgID] = goodOrg
			},
			wantToken: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			tc.setup(repo)
			svc, mr := newTestService(t, repo)

			token, user, org, err := svc.Login(context.Background(), tc.email, tc.password)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Login() want error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Errorf("Login() error = %q, want %q", err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Login() unexpected error: %v", err)
			}
			if token == "" {
				t.Error("Login() returned empty token")
			}
			if user == nil || org == nil {
				t.Error("Login() returned nil user or org")
			}

			// Verify JWT contains correct claims.
			parsed, err := jwt.Parse(token, func(tok *jwt.Token) (interface{}, error) {
				return []byte("test-jwt-secret"), nil
			})
			if err != nil || !parsed.Valid {
				t.Fatalf("Login() issued invalid JWT: %v", err)
			}
			claims, _ := parsed.Claims.(jwt.MapClaims)
			if claims["sub"] != user.ID.String() {
				t.Errorf("JWT sub = %v, want %v", claims["sub"], user.ID.String())
			}
			if claims["org_id"] != org.ID.String() {
				t.Errorf("JWT org_id = %v, want %v", claims["org_id"], org.ID.String())
			}
			if claims["role"] != user.Role {
				t.Errorf("JWT role = %v, want %v", claims["role"], user.Role)
			}
			if _, ok := claims["jti"]; !ok {
				t.Error("JWT missing jti claim")
			}
			exp, _ := claims["exp"].(float64)
			if exp <= float64(time.Now().Unix()) {
				t.Error("JWT exp is in the past")
			}

			// Verify session was persisted in Redis.
			found := false
			for _, k := range mr.Keys() {
				if strings.HasPrefix(k, "session:") {
					found = true
				}
			}
			if !found {
				t.Error("Login() did not store session in Redis")
			}
		})
	}
}

// ── Logout ────────────────────────────────────────────────────────────────────

func TestService_Logout(t *testing.T) {
	repo := newFakeRepo()
	svc, mr := newTestService(t, repo)

	// Pre-populate a session key.
	jti := "test-jti-12345"
	mr.Set("session:"+jti, "{}") //nolint:errcheck

	if err := svc.Logout(context.Background(), jti); err != nil {
		t.Fatalf("Logout() error: %v", err)
	}
	if mr.Exists("session:" + jti) {
		t.Error("Logout() did not delete session key from Redis")
	}
}

// ── GetMe ─────────────────────────────────────────────────────────────────────

func TestService_GetMe(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	user := &User{ID: userID, OrgID: orgID, Email: "alice@example.com", Role: "admin", Name: "Alice", AuthMethod: "password"}
	org := &Organization{ID: orgID, Name: "Acme", Slug: "acme", Plan: "free"}

	tests := []struct {
		name    string
		userID  string
		orgID   string
		setup   func(*fakeRepo)
		wantOrg bool
		wantErr bool
	}{
		{
			name:    "invalid user uuid",
			userID:  "not-a-uuid",
			orgID:   orgID.String(),
			setup:   func(*fakeRepo) {},
			wantErr: true,
		},
		{
			name:   "empty orgID — OAuth user awaiting onboarding",
			userID: userID.String(),
			orgID:  "",
			setup: func(r *fakeRepo) {
				r.userByID[userID] = user
			},
			wantOrg: false,
		},
		{
			name:   "invalid org uuid",
			userID: userID.String(),
			orgID:  "not-a-uuid",
			setup: func(r *fakeRepo) {
				r.userByID[userID] = user
			},
			wantErr: true,
		},
		{
			name:   "user not found",
			userID: userID.String(),
			orgID:  orgID.String(),
			setup:  func(*fakeRepo) {},
			wantErr: true,
		},
		{
			name:   "success with org",
			userID: userID.String(),
			orgID:  orgID.String(),
			setup: func(r *fakeRepo) {
				r.userByID[userID] = user
				r.orgs[orgID] = org
			},
			wantOrg: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			tc.setup(repo)
			svc, _ := newTestService(t, repo)

			result, err := svc.GetMe(context.Background(), tc.userID, tc.orgID)

			if tc.wantErr {
				if err == nil {
					t.Error("GetMe() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetMe() unexpected error: %v", err)
			}
			if result.User == nil {
				t.Error("GetMe() returned nil user")
			}
			if tc.wantOrg && result.Org == nil {
				t.Error("GetMe() expected org, got nil")
			}
			if !tc.wantOrg && result.Org != nil {
				t.Errorf("GetMe() expected nil org, got %+v", result.Org)
			}
		})
	}
}

// ── bcrypt 72-byte truncation ─────────────────────────────────────────────────

// TestService_Login_BcryptTruncation documents the known bcrypt behaviour:
// the algorithm silently truncates input at 72 bytes, so a password longer than
// 72 bytes is indistinguishable from its first 72 bytes. Any future migration
// to a pre-hash strategy should update or remove these assertions.
func TestService_Login_BcryptTruncation(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	base := strings.Repeat("A", 72) // exactly 72 bytes (all ASCII)
	hash := mustHashPassword(t, base)

	user := &User{
		ID:           userID,
		OrgID:        orgID,
		Email:        "bcrypt@example.com",
		PasswordHash: hash,
		Role:         "admin",
		Name:         "BcryptTest",
		AuthMethod:   "password",
	}
	org := &Organization{ID: orgID, Name: "Acme", Slug: "acme", Plan: "free"}

	setup := func() *fakeRepo {
		r := newFakeRepo()
		r.userByEmail[user.Email] = user
		r.orgs[orgID] = org
		return r
	}

	t.Run("72-byte password matches its own hash", func(t *testing.T) {
		svc, _ := newTestService(t, setup())
		_, _, _, err := svc.Login(context.Background(), user.Email, base)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("73-byte password matches 72-byte hash (bcrypt truncation at 72)", func(t *testing.T) {
		svc, _ := newTestService(t, setup())
		_, _, _, err := svc.Login(context.Background(), user.Email, base+"X")
		if err != nil {
			t.Errorf("73-byte password should match (bcrypt truncates): %v", err)
		}
	})

	t.Run("completely different password does not match", func(t *testing.T) {
		svc, _ := newTestService(t, setup())
		_, _, _, err := svc.Login(context.Background(), user.Email, "wrong")
		if err == nil {
			t.Error("expected error for wrong password, got nil")
		}
	})
}
