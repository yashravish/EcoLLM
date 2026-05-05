-- Migration: 010_create_audit_logs
-- Immutable audit trail for security-sensitive actions.
-- Never DELETE or UPDATE rows in this table.

CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID        REFERENCES organizations(id),
    user_id       UUID        REFERENCES users(id),
    api_key_id    UUID        REFERENCES api_keys(id),

    action        VARCHAR(100) NOT NULL,  -- api_key.created, api_key.revoked, user.login, user.logout, org.updated, model.created
    resource_type VARCHAR(50),            -- api_key, user, organization, model
    resource_id   UUID,

    -- Request context
    ip_address    INET,
    user_agent    VARCHAR(500),
    request_id    UUID,

    -- Change payload (before/after for mutations; null for reads)
    old_value     JSONB,
    new_value     JSONB,

    -- Outcome
    success       BOOLEAN     NOT NULL DEFAULT true,
    error_message VARCHAR(500),

    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_org        ON audit_logs(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_user       ON audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action     ON audit_logs(action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_resource   ON audit_logs(resource_type, resource_id);
