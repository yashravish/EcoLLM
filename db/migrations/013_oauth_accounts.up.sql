-- 013_oauth_accounts.up.sql
-- Relax constraints on users so OAuth users can be inserted without org/password.

ALTER TABLE users ALTER COLUMN org_id       DROP NOT NULL;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_method VARCHAR(20) NOT NULL DEFAULT 'password'
    CHECK (auth_method IN ('password', 'oauth', 'both'));

-- Backfill: existing rows are password-based.
UPDATE users SET auth_method = 'password' WHERE auth_method = 'password';

-- Account-linking table: one row per (provider, provider_account_id) pair.
CREATE TABLE IF NOT EXISTS oauth_accounts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider            VARCHAR(50) NOT NULL CHECK (provider IN ('github', 'google')),
    provider_account_id TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_account_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth_accounts_user_id ON oauth_accounts(user_id);
