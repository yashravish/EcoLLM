// Generated TypeScript types from openapi.yaml
// Do not edit manually — run `make generate-types` to regenerate.

export interface ChatCompletionRequest {
  messages: Message[];
  max_tokens?: number;
  temperature?: number;
  stream?: boolean;
  ecollm?: EcoLLMRequestOptions;
}

export interface Message {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

export interface EcoLLMRequestOptions {
  prefer?: 'efficiency' | 'speed' | 'quality';
  max_latency_ms?: number;
  min_quality?: number;
  include_metadata?: boolean;
}

export interface ChatCompletionResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: Choice[];
  usage: Usage;
  ecollm: Metadata;
}

export interface Choice {
  index: number;
  message: Message;
  finish_reason: string;
}

export interface Usage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface Metadata {
  route: RouteMetadata;
  energy: EnergyMetadata;
  cost: CostMetadata;
  performance: PerformanceMetadata;
}

export interface RouteMetadata {
  task_type: string;
  complexity: number;
  model_selected: string;
  fallback_model?: string;
  routing_score: number;
  confidence: number;
  used_fallback: boolean;
}

export interface EnergyMetadata {
  total_energy_kwh: number;
  co2e_grams: number;
  grid_region: string;
  grid_carbon_intensity?: number;
}

export interface CostMetadata {
  total_cost_usd: number;
  savings_vs_gpt4_percent: number;
}

export interface PerformanceMetadata {
  latency_ms: number;
  time_to_first_token_ms?: number;
}
