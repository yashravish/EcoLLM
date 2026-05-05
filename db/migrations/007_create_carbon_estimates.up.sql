-- Migration: 007_create_carbon_estimates
-- Stores CO2e estimates per request.
-- Formula: co2e_grams = energy_kwh * grid_carbon_intensity_gco2_per_kwh * 1000
-- Grid intensity sourced from Electricity Maps API (AGENT_KNOWLEDGE_BASE Layer 6).

CREATE TABLE IF NOT EXISTS carbon_estimates (
    id                      UUID   PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id              UUID   NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    energy_measurement_id   UUID   REFERENCES energy_measurements(id),

    -- Grid data (captured at request time; intensity changes hourly)
    grid_region             VARCHAR(50)  NOT NULL,
    grid_carbon_intensity   REAL         NOT NULL,  -- gCO2/kWh
    carbon_data_source      VARCHAR(100),

    -- Calculated carbon
    co2e_grams              REAL         NOT NULL,

    -- Comparison to GPT-4 baseline for transparency reporting
    gpt4_equivalent_co2e    REAL,
    savings_percent         REAL,

    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_carbon_request ON carbon_estimates(request_id);
