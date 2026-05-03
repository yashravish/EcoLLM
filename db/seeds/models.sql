-- Seed: model_registry and model_routes
-- Run after migrations to load the four standard model candidates.

INSERT INTO model_registry (
    name, display_name, model_id, runtime, quantization, max_context_len,
    gpu_type, gpu_count,
    latency_p50_ms, latency_p95_ms, tokens_per_second,
    energy_per_token_kwh, idle_power_watts, inference_power_watts,
    quality_benchmark, quality_tasks,
    cost_per_request_usd,
    status, endpoint_url, health_status
) VALUES
(
    'phi_3', 'Phi-3 Mini 4K Instruct', 'microsoft/Phi-3-mini-4k-instruct',
    'ollama', 'gguf', 4096,
    'NVIDIA A10G', 1,
    180, 350, 55.0,
    0.0000002, 20, 35,
    0.72, '{"simple":0.80,"medium":0.65,"hard":0.45,"specialized":0.30}',
    0.0001,
    'active', 'http://inference-gateway:8090/phi3', 'unknown'
),
(
    'mistral_7b', 'Mistral 7B Instruct v0.2', 'mistralai/Mistral-7B-Instruct-v0.2',
    'vllm', 'awq', 8192,
    'NVIDIA A10G', 1,
    350, 700, 38.0,
    0.0000004, 30, 45,
    0.81, '{"simple":0.85,"medium":0.82,"hard":0.62,"specialized":0.50}',
    0.0003,
    'active', 'http://inference-gateway:8090/mistral', 'unknown'
),
(
    'llama_13b', 'Llama 2 13B Chat', 'meta-llama/Llama-2-13b-chat-hf',
    'vllm', 'gptq', 4096,
    'NVIDIA A100 40GB', 1,
    600, 1400, 22.0,
    0.0000008, 35, 48,
    0.86, '{"simple":0.88,"medium":0.87,"hard":0.82,"specialized":0.70}',
    0.0007,
    'active', 'http://inference-gateway:8090/llama13b', 'unknown'
),
(
    'llama_70b', 'Llama 2 70B Chat', 'meta-llama/Llama-2-70b-chat-hf',
    'vllm', 'awq', 4096,
    'NVIDIA A100 80GB', 2,
    1800, 4200, 8.0,
    0.000004, 180, 250,
    0.93, '{"simple":0.93,"medium":0.93,"hard":0.92,"specialized":0.91}',
    0.003,
    'active', 'http://inference-gateway:8090/llama70b', 'unknown'
)
ON CONFLICT (name) DO UPDATE SET
    display_name          = EXCLUDED.display_name,
    latency_p50_ms        = EXCLUDED.latency_p50_ms,
    latency_p95_ms        = EXCLUDED.latency_p95_ms,
    energy_per_token_kwh  = EXCLUDED.energy_per_token_kwh,
    inference_power_watts = EXCLUDED.inference_power_watts,
    quality_benchmark     = EXCLUDED.quality_benchmark,
    quality_tasks         = EXCLUDED.quality_tasks,
    cost_per_request_usd  = EXCLUDED.cost_per_request_usd,
    updated_at            = now();

-- Routing table: lower priority = higher precedence
INSERT INTO model_routes (task_type, model_name, priority, is_fallback, enabled) VALUES
    ('simple',      'phi_3',       0, false, true),
    ('simple',      'mistral_7b',  1, true,  true),
    ('medium',      'mistral_7b',  0, false, true),
    ('medium',      'llama_13b',   1, true,  true),
    ('hard',        'llama_13b',   0, false, true),
    ('hard',        'llama_70b',   1, true,  true),
    ('specialized', 'llama_70b',   0, false, true)
ON CONFLICT DO NOTHING;

-- Demo organization (development only — never run in production)
INSERT INTO organizations (name, slug, plan, max_requests_per_min, max_requests_per_day, quality_threshold, energy_budget_kwh)
VALUES ('Demo Org', 'demo', 'free', 60, 1000, 0.70, 1.0)
ON CONFLICT (slug) DO NOTHING;
