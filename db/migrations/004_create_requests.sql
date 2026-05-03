-- Migration: 004_create_requests
-- The requests table is the central audit log for every inference request.
-- It grows fastest of all tables; partition by month at > 1M rows.

CREATE TABLE IF NOT EXISTS requests (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id              UUID         NOT NULL REFERENCES organizations(id),
    api_key_id          UUID         REFERENCES api_keys(id),
    request_id          VARCHAR(100) UNIQUE NOT NULL,    -- External request ID (eco-req-xxx)

    -- Input
    prompt_original     TEXT         NOT NULL,
    prompt_optimized    TEXT,
    messages_json       JSONB,

    -- Routing
    task_type           VARCHAR(50)  NOT NULL,           -- simple, medium, hard, specialized
    complexity          INT          NOT NULL,
    model_selected      VARCHAR(100) NOT NULL,
    model_fallback      VARCHAR(100),
    routing_score       REAL         NOT NULL,
    routing_confidence  REAL         NOT NULL,
    used_fallback       BOOLEAN      NOT NULL DEFAULT false,
    cache_hit           BOOLEAN      NOT NULL DEFAULT false,

    -- Response
    response_text       TEXT,
    finish_reason       VARCHAR(50),
    prompt_tokens       INT,
    completion_tokens   INT,
    total_tokens        INT,

    -- Performance
    latency_ms          INT          NOT NULL,
    time_to_first_token_ms INT,
    tokens_per_second   REAL,

    -- Status
    status              VARCHAR(20)  NOT NULL DEFAULT 'pending',  -- pending, completed, failed, timeout
    error_message       TEXT,

    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_requests_org_id       ON requests(org_id);
CREATE INDEX IF NOT EXISTS idx_requests_created_at   ON requests(created_at);
CREATE INDEX IF NOT EXISTS idx_requests_model        ON requests(model_selected);
CREATE INDEX IF NOT EXISTS idx_requests_status       ON requests(status);
-- Composite for per-org dashboards (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_requests_org_created  ON requests(org_id, created_at DESC);
