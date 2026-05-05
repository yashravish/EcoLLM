-- Migration: 001_create_organizations
-- Creates the organizations table. Organizations are the billing/auth tenant unit.

CREATE TABLE IF NOT EXISTS organizations (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  VARCHAR(255) NOT NULL,
    slug                  VARCHAR(100) UNIQUE NOT NULL,
    plan                  VARCHAR(50)  NOT NULL DEFAULT 'starter',  -- starter, growth, enterprise
    max_requests_per_min  INT          NOT NULL DEFAULT 60,
    max_requests_per_day  INT          NOT NULL DEFAULT 10000,
    quality_threshold     REAL         NOT NULL DEFAULT 0.70,
    energy_budget_kwh     REAL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);
