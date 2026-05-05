-- Migration: 012_add_revoked_at_to_users
-- Adds the revoked_at column to users for soft-delete (RemoveMember).

ALTER TABLE users ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;
