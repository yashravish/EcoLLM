-- Migration: 011_create_feedback_events
-- Stores explicit thumbs-up/down feedback and optional free-text from API consumers.
-- Used to tune quality benchmarks and routing weights over time.

CREATE TABLE IF NOT EXISTS feedback_events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id      UUID        NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    org_id          UUID        NOT NULL REFERENCES organizations(id),

    -- Rating
    rating          SMALLINT    NOT NULL,   -- 1 (thumbs down) or 5 (thumbs up); 1-5 scale reserved
    comment         TEXT,

    -- Classification of what went wrong (populated by org if rating <= 2)
    issue_type      VARCHAR(50),  -- quality, latency, wrong_model, hallucination, other
    expected_output TEXT,

    -- Which model was actually used (denormalized for fast analytics)
    model_used      VARCHAR(100),
    task_type       VARCHAR(50),

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_feedback_request ON feedback_events(request_id);
CREATE INDEX IF NOT EXISTS idx_feedback_org     ON feedback_events(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_model   ON feedback_events(model_used, rating);
