-- Migration: 003_create_api_keys
-- API keys are the primary auth mechanism for the inference API.
-- Keys are stored as bcrypt hashes; only the prefix is stored in plaintext
-- for identification without exposing the full key.

CREATE TABLE IF NOT EXISTS api_keys (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id              UUID         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by          UUID         NOT NULL REFERENCES users(id),
    name                VARCHAR(255) NOT NULL,
    key_hash            VARCHAR(255) NOT NULL,    -- bcrypt hash of the full key
    key_prefix          VARCHAR(10)  NOT NULL,    -- first 10 chars for identification (e.g. "eco-sk-abc")
    scopes              TEXT[]       NOT NULL DEFAULT ARRAY['inference'],
    rate_limit_override INT,
    last_used_at        TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ,
    revoked_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_org_id     ON api_keys(org_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
-- Partial index: only active (non-revoked, non-expired) keys need fast lookup
CREATE INDEX IF NOT EXISTS idx_api_keys_active
    ON api_keys(key_prefix)
    WHERE revoked_at IS NULL;
