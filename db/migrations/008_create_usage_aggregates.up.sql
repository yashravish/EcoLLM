-- Migration: 008_create_usage_aggregates
-- Pre-aggregated usage statistics per org per time window.
-- Populated hourly by the background aggregation worker.
-- Dashboard queries hit this table, not the raw requests table.

CREATE TABLE IF NOT EXISTS usage_aggregates (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                   UUID        NOT NULL REFERENCES organizations(id),
    period_start             TIMESTAMPTZ NOT NULL,
    period_end               TIMESTAMPTZ NOT NULL,
    granularity              VARCHAR(20) NOT NULL,  -- hourly, daily, monthly

    -- Counts
    total_requests           INT         NOT NULL DEFAULT 0,
    successful_requests      INT         NOT NULL DEFAULT 0,
    failed_requests          INT         NOT NULL DEFAULT 0,
    cache_hits               INT         NOT NULL DEFAULT 0,
    fallback_used            INT         NOT NULL DEFAULT 0,

    -- Tokens
    total_prompt_tokens      BIGINT      NOT NULL DEFAULT 0,
    total_completion_tokens  BIGINT      NOT NULL DEFAULT 0,

    -- Model and task distribution (JSON for flexibility)
    model_distribution       JSONB       NOT NULL DEFAULT '{}',  -- {"phi_3": 500, "mistral_7b": 300}
    task_distribution        JSONB       NOT NULL DEFAULT '{}',  -- {"simple": 400, "medium": 350}

    -- Performance
    avg_latency_ms           REAL,
    p95_latency_ms           REAL,

    -- Energy and cost
    total_energy_kwh         REAL        NOT NULL DEFAULT 0,
    total_co2e_grams         REAL        NOT NULL DEFAULT 0,
    total_cost_usd           REAL        NOT NULL DEFAULT 0,

    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (org_id, period_start, granularity)
);

CREATE INDEX IF NOT EXISTS idx_usage_org_period ON usage_aggregates(org_id, period_start DESC);
