-- Migration: 006_create_energy_measurements
-- Stores per-request GPU energy measurements.
-- Formula: energy_kwh = (gpu_power_watts * inference_time_hours / batch_size) * pue / 1000
-- pue_multiplier default 1.3 per Green Grid standard (AGENT_KNOWLEDGE_BASE Layer 6).

CREATE TABLE IF NOT EXISTS energy_measurements (
    id                    UUID   PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id            UUID   NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    org_id                UUID   NOT NULL REFERENCES organizations(id),
    model_name            VARCHAR(100) NOT NULL,

    -- GPU metrics
    gpu_power_watts       REAL   NOT NULL,
    inference_time_ms     INT    NOT NULL,
    batch_size            INT    NOT NULL DEFAULT 1,

    -- Calculated energy
    inference_energy_wh   REAL   NOT NULL,
    pue_multiplier        REAL   NOT NULL DEFAULT 1.3,
    total_energy_wh       REAL   NOT NULL,
    total_energy_kwh      REAL   NOT NULL,

    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_energy_request ON energy_measurements(request_id);
CREATE INDEX IF NOT EXISTS idx_energy_org     ON energy_measurements(org_id, created_at);
