// ===== Chat Completions =====

export interface ChatCompletionRequest {
  messages: Array<{ role: 'system' | 'user' | 'assistant'; content: string }>;
  max_tokens?: number;
  temperature?: number;
  ecollm?: {
    prefer?: 'efficiency' | 'speed' | 'quality';
    max_latency_ms?: number;
    min_quality?: number;
    include_metadata?: boolean;
  };
}

export interface ChatCompletionResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: Array<{
    index: number;
    message: { role: string; content: string };
    finish_reason: string;
  }>;
  usage: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  };
  ecollm: EcoLLMMetadata;
}

export interface EcoLLMMetadata {
  route: {
    task_type: string;
    complexity: number;
    model_selected: string;
    fallback_model?: string;
    routing_score: number;
    confidence: number;
    used_fallback: boolean;
  };
  energy: {
    total_energy_kwh: number;
    co2e_grams: number;
    grid_region: string;
    grid_carbon_intensity?: number;
  };
  cost: {
    total_cost_usd: number;
    savings_vs_gpt4_percent: number;
  };
  performance: {
    latency_ms: number;
    time_to_first_token_ms?: number;
  };
}

// ===== Usage =====

export interface UsageResponse {
  org_id: string;
  period: string;
  from: string;
  to: string;
  summary: {
    total_requests: number;
    total_tokens: number;
    total_energy_kwh: number;
    total_co2e_grams: number;
    total_cost_usd: number;
    cache_hit_rate: number;
    avg_latency_ms: number;
  };
  model_distribution: Record<string, number>;
  daily_breakdown: Array<{
    date: string;
    requests: number;
    energy_kwh: number;
    co2e_grams: number;
    cost_usd: number;
  }>;
}

// ===== Models =====

export interface Model {
  id: string;
  name: string;
  tasks: string[];
  max_context: number;
  quality_benchmark: number;
  latency_p95_ms: number;
  energy_per_request_kwh: number;
  status: string;
}

export interface ModelListResponse {
  models: Model[];
}

// ===== Requests =====

export interface RequestRecord {
  id: string;
  request_id: string;
  prompt_original: string;
  prompt_optimized?: string;
  task_type: string;
  complexity: number;
  model_selected: string;
  model_fallback?: string;
  routing_score: number;
  routing_confidence?: number;
  cache_hit: boolean;
  used_fallback: boolean;
  response_text?: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  latency_ms: number;
  status: string;
  energy_kwh?: number;
  co2e_grams?: number;
  cost_usd?: number;
  created_at: string;
}

export interface RequestListResponse {
  requests: RequestRecord[];
  total: number;
  page: number;
  per_page: number;
}

export interface RequestFilters {
  limit?: number;
  offset?: number;
  sort?: string;
  status?: string;
  model?: string;
  task_type?: string;
}

// ===== Carbon =====

export interface CarbonResponse {
  period: string;
  total_co2e_grams: number;
  total_energy_kwh: number;
  gpt4_equivalent_co2e_grams: number;
  savings_percent: number;
  grid_region: string;
  grid_carbon_intensity: number;
  daily_breakdown: Array<{
    date: string;
    co2e_grams: number;
    energy_kwh: number;
    gpt4_equivalent_co2e_grams: number;
  }>;
  model_energy_breakdown: Array<{
    model: string;
    energy_kwh: number;
    co2e_grams: number;
    request_count: number;
    percentage_of_traffic: number;
  }>;
}

// ===== API Keys =====

export interface ApiKey {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  last_used_at?: string;
  expires_at?: string;
  revoked_at?: string;
  created_at: string;
}

export interface CreateApiKeyRequest {
  name: string;
  scopes: string[];
  expires_in_days?: number;
}

export interface CreateApiKeyResponse {
  id: string;
  key: string;
  key_prefix: string;
  name: string;
  scopes: string[];
}

// ===== Auth =====

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: {
    id: string;
    email: string;
    name: string;
    role: string;
  };
  org: {
    id: string;
    name: string;
    slug: string;
    plan: string;
  };
}

export interface MeResponse {
  user: LoginResponse['user'];
  org: LoginResponse['org'];
}

// ===== Billing =====

export interface BillingEvent {
  id: string;
  org_id: string;
  period_start: string;
  period_end: string;
  total_requests: number;
  total_co2e_grams: number;
  base_cost_usd: number;
  discount_percent: number;
  final_cost_usd: number;
  created_at: string;
}

export interface BillingListResponse {
  events: BillingEvent[];
  total: number;
}

// ===== Register =====

export interface RegisterRequest {
  name: string;
  email: string;
  password: string;
  org_name: string;
}

export interface RegisterResponse {
  token: string;
  user: {
    id: string;
    email: string;
    name: string;
    role: string;
  };
  org: {
    id: string;
    name: string;
    slug: string;
    plan: string;
  };
}

// ===== Admin =====

export interface SystemMetrics {
  total_requests_today: number;
  active_models: number;
  avg_latency_ms: number;
  cache_hit_rate: number;
  gpu_utilization: Record<string, number>;
  total_co2e_today_grams: number;
}

export interface RouteConfig {
  energy_weight: number;
  cost_weight: number;
  quality_weight: number;
  latency_weight: number;
}

// ===== Errors =====

export interface ApiErrorResponse {
  code: number;
  message: string;
  type: string;
  trace_id?: string;
}
