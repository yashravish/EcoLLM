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
type User struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	Name         string
	CreatedAt    time.Time
}

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

type Member struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

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

type OrgInput struct {
	Name string
	Slug string
	Plan string
}

type UserInput struct {
	Email        string
	PasswordHash string
	Role         string
	Name         string
}

// userCols is the SELECT column list for the users table.
// COALESCE on org_id and password_hash is preserved for backward-compatibility
// with legacy rows that may have NULLs from a prior schema version.
const userCols = `
	id,
	COALESCE(org_id, '00000000-0000-0000-0000-000000000000'::uuid) AS org_id,
	email,
	COALESCE(password_hash, '') AS password_hash,
	role,
	COALESCE(name, '') AS name,
	created_at`

func scanUser(row pgx.Row, u *User) error {
	return row.Scan(
		&u.ID, &u.OrgID, &u.Email, &u.PasswordHash,
		&u.Role, &u.Name, &u.CreatedAt,
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

func (r *Repository) TouchAPIKeyLastUsed(ctx context.Context, keyID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE api_keys SET last_used_at = now() WHERE id = $1`,
		keyID,
	)
	return err
}

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
// If a soft-deleted row already exists for this email, the row is revived
// (UPDATE clears revoked_at) and pointed at the new org — this lets users
// re-register with the same email after deleting their account. Hard-delete
// isn't an option because audit_logs.user_id has no ON DELETE CASCADE.
func (r *Repository) CreateOrgAndUser(ctx context.Context, org OrgInput, user UserInput, key *APIKey) (*Organization, *User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var existingID uuid.UUID
	reviveErr := tx.QueryRow(ctx,
		`SELECT id FROM users WHERE email = $1 AND revoked_at IS NOT NULL`,
		user.Email,
	).Scan(&existingID)
	if reviveErr != nil && !errors.Is(reviveErr, pgx.ErrNoRows) {
		return nil, nil, fmt.Errorf("check existing user: %w", reviveErr)
	}
	reviving := reviveErr == nil

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

	var u User
	if reviving {
		if err := tx.QueryRow(ctx,
			`UPDATE users
			   SET org_id        = $1,
			       password_hash = $2,
			       role          = $3,
			       name          = $4,
			       revoked_at    = NULL,
			       updated_at    = now()
			 WHERE id = $5
			 RETURNING id, org_id, email, COALESCE(password_hash,''), role, COALESCE(name,''), created_at`,
			orgID, user.PasswordHash, user.Role, user.Name, existingID,
		).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("revive user: %w", err)
		}
	} else {
		userID := uuid.New()
		if err := tx.QueryRow(ctx,
			`INSERT INTO users (id, org_id, email, password_hash, role, name)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id, org_id, email, COALESCE(password_hash,''), role, COALESCE(name,''), created_at`,
			userID, orgID, user.Email, user.PasswordHash, user.Role, user.Name,
		).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("create user: %w", err)
		}
	}

	key.OrgID = orgID
	key.CreatedBy = u.ID
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

func (r *Repository) InviteMember(ctx context.Context, orgID uuid.UUID, email, passwordHash, name, role string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (id, org_id, email, password_hash, role, name)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, org_id, email, COALESCE(password_hash,''), role, COALESCE(name,''), created_at`,
		uuid.New(), orgID, email, passwordHash, role, name,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

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

// DeleteUser soft-deletes a user and revokes every API key they created.
// Hard-delete is blocked by api_keys.created_by FK (no ON DELETE CASCADE);
// soft-delete also preserves the audit trail. The user's org is left intact.
func (r *Repository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE api_keys SET revoked_at = now()
		 WHERE created_by = $1 AND revoked_at IS NULL`,
		userID,
	); err != nil {
		return fmt.Errorf("revoke api keys: %w", err)
	}

	tag, err := tx.Exec(ctx,
		`UPDATE users SET revoked_at = now()
		 WHERE id = $1 AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("revoke user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return tx.Commit(ctx)
}
