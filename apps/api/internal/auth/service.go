package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ecollm/api/pkg/hash"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	authCacheTTL = 5 * time.Minute
	sessionTTL   = 24 * time.Hour
)

// authCacheEntry is the value stored under the auth:{prefix} Redis key.
type authCacheEntry struct {
	OrgID     string   `json:"org_id"`
	Scopes    []string `json:"scopes"`
	RateLimit int      `json:"rate_limit"`
}

// Service handles authentication business logic.
type Service struct {
	repo      *Repository
	redis     *redis.Client
	jwtSecret []byte
}

func NewService(repo *Repository, redisClient *redis.Client, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		redis:     redisClient,
		jwtSecret: []byte(jwtSecret),
	}
}

// ValidateAPIKey checks Redis first (5-min TTL), then falls back to Postgres.
func (s *Service) ValidateAPIKey(ctx context.Context, rawKey string) (orgID string, scopes []string, err error) {
	if len(rawKey) < 10 {
		return "", nil, errors.New("key too short")
	}
	prefix := rawKey[:10]
	cacheKey := "auth:" + prefix

	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var entry authCacheEntry
		if json.Unmarshal([]byte(cached), &entry) == nil {
			return entry.OrgID, entry.Scopes, nil
		}
	}

	keyRecord, err := s.repo.FindAPIKeyByPrefix(ctx, prefix)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, errors.New("api key not found")
		}
		return "", nil, fmt.Errorf("db lookup: %w", err)
	}

	if keyRecord.ExpiresAt != nil && keyRecord.ExpiresAt.Before(time.Now()) {
		return "", nil, errors.New("api key expired")
	}

	if err := hash.CompareAPIKey(rawKey, keyRecord.KeyHash); err != nil {
		return "", nil, errors.New("invalid api key")
	}

	entry := authCacheEntry{
		OrgID:  keyRecord.OrgID.String(),
		Scopes: keyRecord.Scopes,
	}
	if keyRecord.RateLimitOverride != nil {
		entry.RateLimit = *keyRecord.RateLimitOverride
	}
	if b, jerr := json.Marshal(entry); jerr == nil {
		s.redis.Set(ctx, cacheKey, string(b), authCacheTTL)
	}

	go func() {
		bCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s.repo.TouchAPIKeyLastUsed(bCtx, keyRecord.ID)
	}()

	return keyRecord.OrgID.String(), keyRecord.Scopes, nil
}

// Login validates credentials and returns a signed JWT.
func (s *Service) Login(ctx context.Context, email, password string) (token string, user *User, org *Organization, err error) {
	u, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, nil, errors.New("invalid credentials")
		}
		return "", nil, nil, fmt.Errorf("user lookup: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", nil, nil, errors.New("invalid credentials")
	}

	org, err = s.repo.FindOrgByID(ctx, u.OrgID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("org lookup: %w", err)
	}

	signed, jti, err := s.issueJWT(u)
	if err != nil {
		return "", nil, nil, err
	}

	sessionData, _ := json.Marshal(map[string]string{
		"user_id": u.ID.String(),
		"org_id":  u.OrgID.String(),
		"role":    u.Role,
	})
	s.redis.Set(ctx, "session:"+jti, sessionData, sessionTTL)

	return signed, u, org, nil
}

// Logout invalidates the session token stored in Redis.
func (s *Service) Logout(ctx context.Context, jti string) error {
	return s.redis.Del(ctx, "session:"+jti).Err()
}

// RegisterInput carries the fields needed to create a new org + admin user.
type RegisterInput struct {
	OrgName  string
	Email    string
	Password string
	Name     string
}

// RegisterResponse is returned from Register.
type RegisterResponse struct {
	Token  string        `json:"token"`
	APIKey string        `json:"api_key"` // plaintext — shown once
	User   *User         `json:"user"`
	Org    *Organization `json:"org"`
}

// Register creates a new org, admin user, and initial API key in a transaction.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*RegisterResponse, error) {
	pwHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	plaintext, keyHash, keyPrefix, err := hash.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	apiKey := &APIKey{
		ID:        uuid.New(),
		Name:      "Default",
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Scopes:    []string{"inference:write", "usage:read"},
	}

	org, user, err := s.repo.CreateOrgAndUser(ctx,
		OrgInput{Name: in.OrgName, Slug: slugify(in.OrgName), Plan: "free"},
		UserInput{Email: in.Email, PasswordHash: string(pwHash), Role: "admin", Name: in.Name},
		apiKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create org and user: %w", err)
	}

	signed, jti, err := s.issueJWT(user)
	if err != nil {
		return nil, err
	}

	sessionData, _ := json.Marshal(map[string]string{
		"user_id": user.ID.String(),
		"org_id":  org.ID.String(),
		"role":    user.Role,
	})
	s.redis.Set(ctx, "session:"+jti, sessionData, sessionTTL)

	return &RegisterResponse{
		Token:  signed,
		APIKey: plaintext,
		User:   user,
		Org:    org,
	}, nil
}

// GetMeResult carries the full user + org objects returned by GET /me.
type GetMeResult struct {
	User *User
	Org  *Organization
}

// GetMe returns the full user and org objects for the authenticated caller.
func (s *Service) GetMe(ctx context.Context, userID, orgID string) (*GetMeResult, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, errors.New("invalid org id")
	}
	user, err := s.repo.FindUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	org, err := s.repo.FindOrgByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	return &GetMeResult{User: user, Org: org}, nil
}

// GetOrg returns org details for a given org ID string.
func (s *Service) GetOrg(ctx context.Context, orgID string) (*Organization, error) {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return nil, errors.New("invalid org id")
	}
	return s.repo.FindOrgByID(ctx, id)
}

// UpdateOrgInput carries mutable org fields.
type UpdateOrgInput struct {
	Name             string   `json:"name"`
	QualityThreshold float32  `json:"quality_threshold"`
	EnergyBudgetKwh  *float32 `json:"energy_budget_kwh"`
}

// UpdateOrg modifies org settings.
func (s *Service) UpdateOrg(ctx context.Context, orgID string, in UpdateOrgInput) (*Organization, error) {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return nil, errors.New("invalid org id")
	}
	if err := s.repo.UpdateOrg(ctx, id, in.Name, in.QualityThreshold, in.EnergyBudgetKwh); err != nil {
		return nil, err
	}
	return s.repo.FindOrgByID(ctx, id)
}

// ListMembers returns the members of an org.
func (s *Service) ListMembers(ctx context.Context, orgID string) ([]Member, error) {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return nil, errors.New("invalid org id")
	}
	return s.repo.ListMembers(ctx, id)
}

// InviteMemberInput carries fields for adding a new member.
type InviteMemberInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

// InviteMember adds a new user to the org.
func (s *Service) InviteMember(ctx context.Context, orgID string, in InviteMemberInput) (*Member, error) {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return nil, errors.New("invalid org id")
	}
	if in.Role == "" {
		in.Role = "member"
	}
	if in.Role != "admin" && in.Role != "member" && in.Role != "viewer" {
		return nil, errors.New("role must be admin, member, or viewer")
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.InviteMember(ctx, id, in.Email, string(pwHash), in.Name, in.Role)
	if err != nil {
		return nil, err
	}

	return &Member{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

// UpdateMemberRole changes a member's role within the org.
func (s *Service) UpdateMemberRole(ctx context.Context, orgID, userID, role string) error {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return errors.New("invalid org id")
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user id")
	}
	if role != "admin" && role != "member" && role != "viewer" {
		return errors.New("role must be admin, member, or viewer")
	}
	return s.repo.UpdateMemberRole(ctx, oid, uid, role)
}

// ListAPIKeys returns active keys for an org (key_hash is omitted in response).
func (s *Service) ListAPIKeys(ctx context.Context, orgID string) ([]APIKey, error) {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return nil, errors.New("invalid org id")
	}
	return s.repo.ListAPIKeys(ctx, id)
}

// CreateAPIKeyInput carries fields for a new API key.
type CreateAPIKeyInput struct {
	Name         string
	Scopes       []string
	ExpiresInDays int
}

// CreateAPIKeyResult is returned from CreateAPIKey; plaintext shown once.
type CreateAPIKeyResult struct {
	Key    APIKey
	Raw    string // plaintext — shown to user exactly once
}

// CreateAPIKey generates and persists a new API key for the org.
func (s *Service) CreateAPIKey(ctx context.Context, orgID, createdByUserID string, in CreateAPIKeyInput) (*CreateAPIKeyResult, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, errors.New("invalid org id")
	}
	uid, err := uuid.Parse(createdByUserID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}

	plaintext, keyHash, keyPrefix, err := hash.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	if len(in.Scopes) == 0 {
		in.Scopes = []string{"inference:write", "usage:read"}
	}

	k := &APIKey{
		ID:        uuid.New(),
		OrgID:     oid,
		CreatedBy: uid,
		Name:      in.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Scopes:    in.Scopes,
	}
	if in.ExpiresInDays > 0 {
		t := time.Now().Add(time.Duration(in.ExpiresInDays) * 24 * time.Hour)
		k.ExpiresAt = &t
	}

	if err := s.repo.CreateAPIKey(ctx, k); err != nil {
		return nil, fmt.Errorf("create key: %w", err)
	}
	return &CreateAPIKeyResult{Key: *k, Raw: plaintext}, nil
}

// RevokeAPIKey soft-deletes an API key, enforcing org ownership.
func (s *Service) RevokeAPIKey(ctx context.Context, orgID, keyID string) error {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return errors.New("invalid org id")
	}
	kid, err := uuid.Parse(keyID)
	if err != nil {
		return errors.New("invalid key id")
	}
	return s.repo.RevokeAPIKey(ctx, kid, oid)
}

// RemoveMember soft-deletes a user from an org.
func (s *Service) RemoveMember(ctx context.Context, orgID, userID string) error {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return errors.New("invalid org id")
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user id")
	}
	return s.repo.RemoveMember(ctx, oid, uid)
}

// issueJWT signs a JWT for a user and returns the signed string and jti.
func (s *Service) issueJWT(u *User) (signed, jti string, err error) {
	jti = uuid.NewString()
	claims := jwt.MapClaims{
		"sub":    u.ID.String(),
		"org_id": u.OrgID.String(),
		"role":   u.Role,
		"exp":    time.Now().Add(15 * time.Minute).Unix(),
		"iat":    time.Now().Unix(),
		"jti":    jti,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err = t.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, jti, nil
}

// slugify converts an org name to a URL-safe slug.
func slugify(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_':
			b.WriteRune('-')
		}
	}
	return b.String()
}
