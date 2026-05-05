package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository executes all SQL for the auth domain.
// Business logic lives in Service; this layer only handles data access.
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// User represents a row from the users table.
// OrgID is uuid.Nil when org_id IS NULL (OAuth users awaiting onboarding).
// PasswordHash is "" for OAuth-only users.
type User struct {
	ID           uuid.UUID
	OrgID        uuid.UUID // uuid.Nil ↔ NULL in DB (uses COALESCE in all SELECTs)
	Email        string
	PasswordHash string // "" for OAuth users
	Role         string
	Name         string
	AuthMethod   string // 'password' | 'oauth' | 'both'
	CreatedAt    time.Time
}

// Organization represents a row from the organizations table.
type Organization struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug"`
	Plan              string    `json:"plan"`
	MaxRequestsPerMin int       `json:"max_requests_per_min"`
	MaxRequestsPerDay int       `json:"max_requests_per_day"`
	QualityThreshold  float32   `json:"quality_threshold"`
	EnergyBudgetKwh   *float32  `json:"energy_budget_kwh"`
	CreatedAt         time.Time `json:"created_at"`
}

// Member is the public representation of a user within an org.
type Member struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKey represents a row from the api_keys table.
type APIKey struct {
	ID                uuid.UUID
	OrgID             uuid.UUID
	CreatedBy         uuid.UUID
	Name              string
	KeyHash           string
	KeyPrefix         string
	Scopes            []string
	RateLimitOverride *int
	LastUsedAt        *time.Time
	ExpiresAt         *time.Time
	RevokedAt         *time.Time
	CreatedAt         time.Time
}

// OrgInput carries fields for creating a new organization.
type OrgInput struct {
	Name string
	Slug string
	Plan string
}

// UserInput carries fields for creating a new user.
type UserInput struct {
	Email        string
	PasswordHash string
	Role         string
	Name         string
}

// userCols is the SELECT column list for the users table.
// COALESCE(org_id, uuid_nil) makes org_id always scannable into uuid.UUID;
// callers check uuid.Nil to detect "no org".
const userCols = `
	id,
	COALESCE(org_id, '00000000-0000-0000-0000-000000000000'::uuid) AS org_id,
	email,
	COALESCE(password_hash, '') AS password_hash,
	role,
	COALESCE(name, '') AS name,
	auth_method,
	created_at`

// userColsAliased is the same but with a table alias — used in JOINs.
const userColsAliased = `
	u.id,
	COALESCE(u.org_id, '00000000-0000-0000-0000-000000000000'::uuid) AS org_id,
	u.email,
	COALESCE(u.password_hash, '') AS password_hash,
	u.role,
	COALESCE(u.name, '') AS name,
	u.auth_method,
	u.created_at`

func scanUser(row pgx.Row, u *User) error {
	return row.Scan(
		&u.ID, &u.OrgID, &u.Email, &u.PasswordHash,
		&u.Role, &u.Name, &u.AuthMethod, &u.CreatedAt,
	)
}

// FindUserByEmail returns a user by email, or pgx.ErrNoRows if not found.
func (r *Repository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	if err := scanUser(r.db.QueryRow(ctx,
		`SELECT`+userCols+`
		 FROM users WHERE email = $1 AND revoked_at IS NULL`,
		email,
	), u); err != nil {
		return nil, err
	}
	return u, nil
}

// FindUserByID returns a user by primary key.
func (r *Repository) FindUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	u := &User{}
	if err := scanUser(r.db.QueryRow(ctx,
		`SELECT`+userCols+`
		 FROM users WHERE id = $1 AND revoked_at IS NULL`,
		userID,
	), u); err != nil {
		return nil, err
	}
	return u, nil
}

// FindOrgByID returns an organization by its UUID.
func (r *Repository) FindOrgByID(ctx context.Context, orgID uuid.UUID) (*Organization, error) {
	o := &Organization{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, slug, plan, max_requests_per_min, max_requests_per_day,
		        quality_threshold, energy_budget_kwh, created_at
		 FROM organizations WHERE id = $1`,
		orgID,
	).Scan(
		&o.ID, &o.Name, &o.Slug, &o.Plan,
		&o.MaxRequestsPerMin, &o.MaxRequestsPerDay,
		&o.QualityThreshold, &o.EnergyBudgetKwh,
		&o.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return o, nil
}

// FindAPIKeyByPrefix returns the API key record matching a given prefix.
func (r *Repository) FindAPIKeyByPrefix(ctx context.Context, prefix string) (*APIKey, error) {
	k := &APIKey{}
	err := r.db.QueryRow(ctx,
		`SELECT id, org_id, created_by, name, key_hash, key_prefix, scopes,
		        rate_limit_override, last_used_at, expires_at, revoked_at, created_at
		 FROM api_keys
		 WHERE key_prefix = $1 AND revoked_at IS NULL`,
		prefix,
	).Scan(
		&k.ID, &k.OrgID, &k.CreatedBy, &k.Name, &k.KeyHash, &k.KeyPrefix,
		&k.Scopes, &k.RateLimitOverride, &k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt,
		&k.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return k, nil
}

// TouchAPIKeyLastUsed updates the last_used_at timestamp for an API key.
func (r *Repository) TouchAPIKeyLastUsed(ctx context.Context, keyID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE api_keys SET last_used_at = now() WHERE id = $1`,
		keyID,
	)
	return err
}

// CreateAPIKey inserts a new API key record.
func (r *Repository) CreateAPIKey(ctx context.Context, k *APIKey) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO api_keys (id, org_id, created_by, name, key_hash, key_prefix, scopes, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		k.ID, k.OrgID, k.CreatedBy, k.Name, k.KeyHash, k.KeyPrefix, k.Scopes, k.ExpiresAt,
	)
	return err
}

// RevokeAPIKey sets revoked_at on an API key, preserving the audit trail.
func (r *Repository) RevokeAPIKey(ctx context.Context, keyID uuid.UUID, orgID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE api_keys SET revoked_at = now()
		 WHERE id = $1 AND org_id = $2 AND revoked_at IS NULL`,
		keyID, orgID,
	)
	return err
}

// CreateOrgAndUser creates org, user, and initial API key in a single transaction.
func (r *Repository) CreateOrgAndUser(ctx context.Context, org OrgInput, user UserInput, key *APIKey) (*Organization, *User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	orgID := uuid.New()
	var o Organization
	if err := tx.QueryRow(ctx,
		`INSERT INTO organizations (id, name, slug, plan, max_requests_per_min, max_requests_per_day, quality_threshold)
		 VALUES ($1, $2, $3, $4, 60, 10000, 0.7)
		 RETURNING id, name, slug, plan, max_requests_per_min, max_requests_per_day, quality_threshold, energy_budget_kwh, created_at`,
		orgID, org.Name, org.Slug, org.Plan,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.MaxRequestsPerMin, &o.MaxRequestsPerDay,
		&o.QualityThreshold, &o.EnergyBudgetKwh, &o.CreatedAt,
	); err != nil {
		return nil, nil, fmt.Errorf("create org: %w", err)
	}

	userID := uuid.New()
	var u User
	if err := tx.QueryRow(ctx,
		`INSERT INTO users (id, org_id, email, password_hash, role, name, auth_method)
		 VALUES ($1, $2, $3, $4, $5, $6, 'password')
		 RETURNING id, org_id, email, COALESCE(password_hash,''), role, COALESCE(name,''), auth_method, created_at`,
		userID, orgID, user.Email, user.PasswordHash, user.Role, user.Name,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.AuthMethod, &u.CreatedAt); err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	key.OrgID = orgID
	key.CreatedBy = userID
	if _, err := tx.Exec(ctx,
		`INSERT INTO api_keys (id, org_id, created_by, name, key_hash, key_prefix, scopes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		key.ID, key.OrgID, key.CreatedBy, key.Name, key.KeyHash, key.KeyPrefix, key.Scopes,
	); err != nil {
		return nil, nil, fmt.Errorf("create api key: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return &o, &u, nil
}

// UpdateOrg updates mutable org fields.
func (r *Repository) UpdateOrg(ctx context.Context, orgID uuid.UUID, name string, qualityThreshold float32, energyBudgetKwh *float32) error {
	_, err := r.db.Exec(ctx,
		`UPDATE organizations
		 SET name = COALESCE(NULLIF($2,''), name),
		     quality_threshold = $3,
		     energy_budget_kwh = $4
		 WHERE id = $1`,
		orgID, name, qualityThreshold, energyBudgetKwh,
	)
	return err
}

// ListMembers returns all active users in an org.
func (r *Repository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]Member, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, email, COALESCE(name,''), role, created_at
		 FROM users WHERE org_id = $1 AND revoked_at IS NULL
		 ORDER BY created_at ASC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.Name, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// InviteMember creates a new user in the org.
func (r *Repository) InviteMember(ctx context.Context, orgID uuid.UUID, email, passwordHash, name, role string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (id, org_id, email, password_hash, role, name, auth_method)
		 VALUES ($1, $2, $3, $4, $5, $6, 'password')
		 RETURNING id, org_id, email, COALESCE(password_hash,''), role, COALESCE(name,''), auth_method, created_at`,
		uuid.New(), orgID, email, passwordHash, role, name,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.AuthMethod, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateMemberRole changes a member's role within an org.
func (r *Repository) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET role = $3 WHERE id = $2 AND org_id = $1 AND revoked_at IS NULL`,
		orgID, userID, role,
	)
	return err
}

// RemoveMember soft-deletes a user from an org.
func (r *Repository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET revoked_at = now() WHERE id = $2 AND org_id = $1`,
		orgID, userID,
	)
	return err
}

// ListAPIKeys returns all active (non-revoked) API keys for an org.
func (r *Repository) ListAPIKeys(ctx context.Context, orgID uuid.UUID) ([]APIKey, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, created_by, name, key_hash, key_prefix, scopes,
		        rate_limit_override, last_used_at, expires_at, revoked_at, created_at
		 FROM api_keys
		 WHERE org_id = $1 AND revoked_at IS NULL
		 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(
			&k.ID, &k.OrgID, &k.CreatedBy, &k.Name, &k.KeyHash, &k.KeyPrefix,
			&k.Scopes, &k.RateLimitOverride, &k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt,
			&k.CreatedAt,
		); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// ── OAuth ─────────────────────────────────────────────────────────────────────

// UpsertOAuthUser implements three-way account-linking inside a single transaction:
//
//  1. oauth_accounts row found → return linked user (read-only path).
//  2. Email matches an existing user → insert oauth_accounts + set auth_method='both'.
//  3. No match → insert new user (org_id/password_hash NULL) + oauth_accounts row.
func (r *Repository) UpsertOAuthUser(ctx context.Context, provider, providerID, email, name string) (*User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Case 1 — existing OAuth link.
	var u User
	err = tx.QueryRow(ctx,
		`SELECT`+userColsAliased+`
		 FROM oauth_accounts oa
		 JOIN users u ON u.id = oa.user_id
		 WHERE oa.provider = $1 AND oa.provider_account_id = $2 AND u.revoked_at IS NULL`,
		provider, providerID,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.AuthMethod, &u.CreatedAt)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return &u, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lookup oauth account: %w", err)
	}

	// Case 2 — existing password user, same email → link.
	var u2 User
	err = tx.QueryRow(ctx,
		`SELECT`+userCols+`
		 FROM users WHERE email = $1 AND revoked_at IS NULL`,
		email,
	).Scan(&u2.ID, &u2.OrgID, &u2.Email, &u2.PasswordHash, &u2.Role, &u2.Name, &u2.AuthMethod, &u2.CreatedAt)
	if err == nil {
		if _, execErr := tx.Exec(ctx,
			`INSERT INTO oauth_accounts (user_id, provider, provider_account_id)
			 VALUES ($1, $2, $3)`,
			u2.ID, provider, providerID,
		); execErr != nil {
			return nil, fmt.Errorf("insert oauth account (link): %w", execErr)
		}
		if _, execErr := tx.Exec(ctx,
			`UPDATE users SET auth_method = 'both', updated_at = now() WHERE id = $1`,
			u2.ID,
		); execErr != nil {
			return nil, fmt.Errorf("update auth_method: %w", execErr)
		}
		u2.AuthMethod = "both"
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return &u2, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lookup user by email: %w", err)
	}

	// Case 3 — brand-new user.
	var u3 User
	if err := tx.QueryRow(ctx,
		`INSERT INTO users (id, email, name, role, auth_method)
		 VALUES ($1, $2, $3, 'member', 'oauth')
		 RETURNING
		   id,
		   COALESCE(org_id, '00000000-0000-0000-0000-000000000000'::uuid),
		   email,
		   COALESCE(password_hash, '') AS password_hash,
		   role,
		   COALESCE(name, '') AS name,
		   auth_method,
		   created_at`,
		uuid.New(), email, name,
	).Scan(&u3.ID, &u3.OrgID, &u3.Email, &u3.PasswordHash, &u3.Role, &u3.Name, &u3.AuthMethod, &u3.CreatedAt); err != nil {
		return nil, fmt.Errorf("create oauth user: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO oauth_accounts (user_id, provider, provider_account_id)
		 VALUES ($1, $2, $3)`,
		u3.ID, provider, providerID,
	); err != nil {
		return nil, fmt.Errorf("insert oauth account (new user): %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &u3, nil
}

// CreateOrgForUser creates a new org and sets the user's org_id in one transaction.
// Called by the onboarding endpoint for OAuth users with no org yet.
func (r *Repository) CreateOrgForUser(ctx context.Context, userID uuid.UUID, org OrgInput) (*Organization, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	orgID := uuid.New()
	var o Organization
	if err := tx.QueryRow(ctx,
		`INSERT INTO organizations (id, name, slug, plan, max_requests_per_min, max_requests_per_day, quality_threshold)
		 VALUES ($1, $2, $3, $4, 60, 10000, 0.7)
		 RETURNING id, name, slug, plan, max_requests_per_min, max_requests_per_day, quality_threshold, energy_budget_kwh, created_at`,
		orgID, org.Name, org.Slug, org.Plan,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.MaxRequestsPerMin, &o.MaxRequestsPerDay,
		&o.QualityThreshold, &o.EnergyBudgetKwh, &o.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE users SET org_id = $1, updated_at = now() WHERE id = $2`,
		orgID, userID,
	); err != nil {
		return nil, fmt.Errorf("set user org: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &o, nil
}
