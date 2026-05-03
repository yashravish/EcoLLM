package audit

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Repository writes immutable audit log entries.
// Never update or delete rows — audit_logs is an append-only table.
type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Entry carries the fields for a single audit log write.
type Entry struct {
	OrgID        *uuid.UUID
	UserID       *uuid.UUID
	APIKeyID     *uuid.UUID
	Action       string // api_key.created, api_key.revoked, user.login, model.created …
	ResourceType string // api_key, user, organization, model
	ResourceID   *uuid.UUID
	IPAddress    string
	UserAgent    string
	Success      bool
	ErrorMessage string
}

// Write inserts an audit log entry. Errors are logged and swallowed so that
// a logging failure never blocks the primary request path.
func (r *Repository) Write(ctx context.Context, e *Entry) {
	_, err := r.db.Exec(ctx,
		`INSERT INTO audit_logs
		    (org_id, user_id, api_key_id, action, resource_type, resource_id,
		     ip_address, user_agent, success, error_message)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		e.OrgID, e.UserID, e.APIKeyID,
		e.Action, nullStr(e.ResourceType), e.ResourceID,
		nullStr(e.IPAddress), nullStr(e.UserAgent),
		e.Success, nullStr(e.ErrorMessage),
	)
	if err != nil {
		log.Warn().Err(err).Str("action", e.Action).Msg("audit log write failed")
	}
}

// WriteAsync fires Write in a background goroutine so callers are never blocked.
func (r *Repository) WriteAsync(e *Entry) {
	go func() {
		r.Write(context.Background(), e)
	}()
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
