-- Migration: 005_create_model_registry
-- The model_registry is the source of truth for available models and their
-- energy, latency, and quality baselines used by the routing engine.

CREATE TABLE IF NOT EXISTS model_registry (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  VARCHAR(100) UNIQUE NOT NULL,   -- phi_3, mistral_7b, llama_13b, llama_70b
    display_name          VARCHAR(255) NOT NULL,
    model_id              VARCHAR(255) NOT NULL,          -- HuggingFace model ID
    runtime               VARCHAR(50)  NOT NULL,          -- vllm, tgi, ollama
    quantization          VARCHAR(20),                    -- awq, gptq, gguf, none
    max_context_len       INT          NOT NULL,

    -- Hardware
    gpu_type              VARCHAR(50)  NOT NULL,
    gpu_count             INT          NOT NULL DEFAULT 1,

    -- Performance baselines (from benchmark suite)
    latency_p50_ms        INT          NOT NULL,
    latency_p95_ms        INT          NOT NULL,
    tokens_per_second     REAL         NOT NULL,

    -- Energy baselines (from MLPerf + measured; see AGENT_KNOWLEDGE_BASE Layer 6)
    energy_per_token_kwh  REAL         NOT NULL,
    idle_power_watts      REAL         NOT NULL,
    inference_power_watts REAL         NOT NULL,

    -- Quality baselines (0–1 scale)
    quality_benchmark     REAL         NOT NULL,
    quality_tasks         JSONB        NOT NULL DEFAULT '{}',

    -- Cost
    cost_per_request_usd  REAL         NOT NULL,

    -- Status
    status                VARCHAR(20)  NOT NULL DEFAULT 'active',   -- active, warming, draining, disabled
    endpoint_url          VARCHAR(500) NOT NULL,
    health_status         VARCHAR(20)  NOT NULL DEFAULT 'unknown',  -- healthy, unhealthy, unknown
    last_health_check     TIMESTAMPTZ,
    version               VARCHAR(50),

    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Routing config: which models serve which task types
CREATE TABLE IF NOT EXISTS model_routes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type   VARCHAR(50) NOT NULL,
    model_name  VARCHAR(100) NOT NULL REFERENCES model_registry(name),
    priority    INT          NOT NULL DEFAULT 0,    -- lower = higher priority
    is_fallback BOOLEAN      NOT NULL DEFAULT false,
    enabled     BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_model_routes_task ON model_routes(task_type, priority);
