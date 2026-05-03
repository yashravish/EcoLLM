# EcoLLM — Agent Knowledge Base & Documentation Reference

**Purpose:** This file tells the coding agent which documentation, books, papers, and specs to reference when making architectural and implementation decisions. Every resource is mapped to a specific system layer. When in doubt about a decision, check the mapped resource for this layer first.

**How to use this file:** Place in the project root. When prompting your coding agent, reference this file explicitly: *"Refer to AGENT_KNOWLEDGE_BASE.md for the correct documentation sources before making this decision."*

---

## Core Principle (Non-Negotiable)

Every decision made by this agent must be filtered through this priority hierarchy:

```
1. Minimize energy per request         (40% weight in routing score)
2. Minimize cost per request           (30% weight)
3. Maximize quality for the task       (20% weight)
4. Minimize latency                    (10% weight)
```

If a decision improves quality or latency but increases energy consumption without justification — reject it and find an alternative. This is not a preference. It is a hard constraint.

---

## Layer 1: Inference Runtime

> Applies to: `apps/inference-gateway/`, `apps/prompt-optimizer/`, `model-configs/*.yaml`, vLLM deployment manifests, quantization decisions, batching strategy, KV cache config.

### Primary References

| Priority | Resource | URL | Scope |
|----------|----------|-----|-------|
| 🔴 **REQUIRED** | vLLM Official Documentation | `https://docs.vllm.ai` | PagedAttention, continuous batching, quantization flags, OpenAI-compatible serving API, engine configuration. This is the inference bible. Read before touching any vLLM config. |
| 🔴 **REQUIRED** | vLLM Quantization Guide | `https://docs.vllm.ai/en/latest/features/quantization` | AWQ vs GPTQ vs FP8 decisions, Marlin kernels, per-model quantization config. Maps directly to `model-configs/*.yaml`. |
| 🟡 **IMPORTANT** | Inside vLLM: Anatomy of a High-Throughput LLM Inference System | `https://www.aleksagordic.com/blog/vllm` | Deep internals: scheduling, PagedAttention, continuous batching, speculative decoding, V1 engine architecture. Use when making batching or latency optimization decisions. |
| 🟡 **IMPORTANT** | AWQ Quantization Guide (Spheron, 2026) | `https://www.spheron.network/blog/awq-quantization-guide-llm-deployment` | AWQ vs GPTQ hardware matrix, Marlin kernel throughput gains, production deployment on L4/L40S/A100. Use when selecting quantization method per model. |
| 🟡 **IMPORTANT** | vLLM Quantization Benchmarks (Jarvis Labs) | `https://jarvislabs.ai/blog/vllm-quantization-complete-guide-benchmarks` | Real benchmarks: AWQ, GPTQ, Marlin, GGUF on H200. Perplexity vs throughput tradeoffs. Use when validating quantization quality assumptions. |
| 🟢 **REFERENCE** | GPTQModel vLLM Docs | `https://docs.vllm.ai/en/latest/features/quantization/gptqmodel/` | GPTQ-specific config, Marlin/Machete kernel flags, Ampere vs Hopper GPU targets. |
| 🟢 **REFERENCE** | GKE LLM Optimization Guide | `https://cloud.google.com/kubernetes-engine/docs/best-practices/machine-learning/inference/llm-optimization` | Quantized model serving on L4/L40S/A100 via vLLM on Kubernetes. Use for production GPU node config. |

### Quantization Decision Matrix (from docs above)

```
Target Hardware     → Recommended Format
──────────────────────────────────────────
L4, L40S, A100      → AWQ INT4 (Marlin kernel)
A100 80G fallback   → GPTQ INT4 (Machete kernel)
Consumer / CPU mix  → GGUF via llama.cpp (NOT for production vLLM)
Blackwell (B200)    → FP8 first, FP4 if throughput critical
```

### Model Runtime Config Files

When writing or editing `model-configs/*.yaml`, always verify against:
- vLLM docs: `max_model_len`, `gpu_memory_utilization`, `tensor_parallel_size`, `quantization` flags
- AWQ guide: `group_size: 128` recommendation, dtype requirements (`float16` for AWQ)
- GKE guide: single L4 can serve a 13B AWQ model; two L4s for 70B AWQ

---

## Layer 2: LLM Routing Engine

> Applies to: `apps/api/internal/router/`, `classifier.go`, `scorer.go`, `selector.go`, routing algorithm weights, fallback chain logic, task classification, confidence scoring.

### Primary References

| Priority | Resource | URL / Source | Scope |
|----------|----------|-------------|-------|
| 🔴 **REQUIRED** | vLLM Semantic Router Project | `https://github.com/vllm-project/semantic-router` | Open-source routing layer for vLLM. Multi-factor routing logic, category-aware semantic caching, fleet energy-efficiency analysis. The closest existing production system to what we're building. |
| 🔴 **REQUIRED** | Workload-Router-Pool Architecture Vision Paper | `https://arxiv.org/pdf/2603.21354` | Maps workload characterization, routing strategy, and GPU pool architecture into a unified framework. Read before designing routing algorithm changes. |
| 🟡 **IMPORTANT** | RouteLLM Paper (Ong et al., 2024) | Search `arxiv.org` for "RouteLLM" | Trains classifiers to route between models based on query difficulty. Direct academic precursor to our scoring function. Use when improving task classifier accuracy. |
| 🟡 **IMPORTANT** | Universal Model Routing for Efficient LLM Inference | `https://arxiv.org/pdf/2502.08773` | Formally grounded routing over a dynamic model pool. Use when adding new models or deprecating old ones — routing must generalize across pool changes. |
| 🟡 **IMPORTANT** | AutoMix Paper (Aggarwal et al., 2023) | Search `arxiv.org` for "AutoMix LLM cascading" | Formulates cascading model selection as a POMDP. Provides theoretical basis for the fallback chain: `phi_3 → mistral_7b → llama_13b → llama_70b`. |
| 🟡 **IMPORTANT** | vLLM Signal-Driven Routing White Paper | `https://arxiv.org/pdf/2603.04444` | Deployment diversity, multi-turn statefulness, policy conflict detection. Use when adding customer-specific routing constraints. |

### Routing Scoring Function (Do Not Change Weights Without Justification)

```
route_score =
    0.40 × (1 - energy_kwh / max_energy)        # ENERGY — non-negotiable primary
  + 0.30 × (1 - cost_usd / max_cost)            # COST
  + 0.20 × quality_benchmark                     # QUALITY
  + 0.10 × (1 - latency_ms / max_latency)       # LATENCY
  - 0.05 × (failure_rate / max_risk)             # RISK PENALTY
```

Energy weight cannot go below 0.35 without explicit justification and review. The environmental constraint is the product's core differentiator.

### Model Candidate Pools (from `selector.go`)

```
TaskSimple      → [phi_3, mistral_7b]
TaskMedium      → [mistral_7b, llama_13b]
TaskHard        → [llama_13b, llama_70b]
TaskSpecialized → [llama_70b]
```

Fallback chain: `phi_3 → mistral_7b → llama_13b → llama_70b → error`

---

## Layer 3: Go Backend

> Applies to: `apps/api/`, `apps/inference-gateway/`, `apps/carbon-service/`, all Go packages, handler/service/repository pattern, middleware, concurrency, HTTP clients, error handling.

### Primary References

| Priority | Resource | Author | Scope |
|----------|----------|--------|-------|
| 🔴 **REQUIRED** | Let's Go | Alex Edwards (`lets-go.alexedwards.net`) | Production web applications in Go. Authentication, middleware, database integration, security, performance. Maps directly to `apps/api/internal/` handler and middleware layers. |
| 🔴 **REQUIRED** | Let's Go Further | Alex Edwards (`lets-go.alexedwards.net`) | Advanced: JSON APIs, rate limiting, graceful shutdown, deployment. Covers patterns used in `apps/api/cmd/server/main.go`. |
| 🟡 **IMPORTANT** | Learning Go (2nd ed.) | Jon Bodner (O'Reilly) | Idiomatic Go. Error handling, interfaces, generics, concurrency patterns. Use this as the style guide for all Go code. |
| 🟡 **IMPORTANT** | Concurrency in Go | Katherine Cox-Buday (O'Reilly) | Goroutines, channels, sync primitives, concurrency patterns. Required for inference gateway concurrent model requests, fallback logic, and background workers. |
| 🟡 **IMPORTANT** | Network Programming with Go | Adam Woodbeck (No Starch Press) | HTTP connection pooling, secure clients, production-grade network code. Use for `apps/inference-gateway/internal/pool/`. |
| 🟢 **REFERENCE** | Fiber v2 Documentation | `https://docs.gofiber.io` | Middleware signatures, context handling, route groups, config. All HTTP routing decisions. |
| 🟢 **REFERENCE** | pgx v5 Documentation | `https://github.com/jackc/pgx` | PostgreSQL driver. Parameterized queries, connection pooling, batch operations. All repository layer code. |
| 🟢 **REFERENCE** | zerolog Documentation | `https://github.com/rs/zerolog` | Structured JSON logging. Use for all log entries. Never use `fmt.Println` in production code. |
| 🟢 **REFERENCE** | golang-migrate Documentation | `https://github.com/golang-migrate/migrate` | Database migration patterns. All files under `db/migrations/`. |

### Go Architecture Rules (Enforce These)

```
Handler layer   → HTTP parsing, validation, call service, return JSON. No business logic.
Service layer   → Business logic, orchestration. No HTTP, no raw SQL.
Repository layer → SQL queries via pgx only. No business logic.

Error pattern   → Return typed errors (apierror package), never panic in handlers.
Config pattern  → Environment variables only, loaded at startup via config.Load().
Logging pattern → zerolog structured JSON, always include request_id and org_id fields.
```

---

## Layer 4: PostgreSQL + Redis

> Applies to: `db/migrations/`, `db/schema.sql`, all repository layer SQL, Redis cache key design, rate limiter implementation, session management.

### Primary References

| Priority | Resource | URL | Scope |
|----------|----------|-----|-------|
| 🔴 **REQUIRED** | PostgreSQL 16 Official Documentation | `https://postgresql.org/docs/16` | Index strategies, JSONB operators, partitioning, `gen_random_uuid()`, `TIMESTAMPTZ`, query planning. Read the Indexing and JSONB sections before writing any schema migrations. |
| 🟡 **IMPORTANT** | The Art of PostgreSQL | Dimitri Fontaine (`https://theartofpostgresql.com`) | Window functions, CTEs, advanced aggregation. Required for `usage_aggregates` worker queries and billing report generation. |
| 🟡 **IMPORTANT** | Redis 7 Commands Reference | `https://redis.io/commands` | Exact syntax for `INCR`, `EXPIRE`, `SETNX`, `SETEX`, sorted sets for sliding window rate limiting. |
| 🟢 **REFERENCE** | Redis Best Practices Guide | `https://redis.io/docs/management/optimization` | Key naming conventions, TTL strategy, memory optimization. Maps to caching strategy in architecture doc. |

### Redis Key Namespace Reference

```
cache:{sha256(normalized_prompt)}          → Full JSON response       TTL: 1 hour
opt:{sha256(raw_prompt)}                   → Optimized prompt text    TTL: 30 min
auth:{key_prefix}                          → Org config JSON          TTL: 5 min
model:{model_name}                         → Model config JSON        TTL: 10 min
org:{org_id}                               → Org settings JSON        TTL: 5 min
rl:{api_key}:{window_start}                → INT rate limit counter   TTL: 60s
dedup:{sha256(request_body)}               → "1"                      TTL: 5s
grid:{region}                              → Carbon intensity JSON    TTL: 1 hour
usage:{org_id}:{date}                      → INT usage counter        TTL: 48 hours
health:{model_name}                        → Health status JSON       TTL: 30s
session:{token}                            → User session JSON        TTL: 24 hours
```

### Database Partition Strategy

The `requests` table will grow fastest. Partition by month at >1M rows:
```sql
-- Add to migration when requests table exceeds 1M rows
CREATE TABLE requests_y2026_m06 PARTITION OF requests
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
```

---

## Layer 5: Next.js Frontend

> Applies to: `apps/web/`, all TypeScript/React code, component design, routing, state management, API client, form validation, accessibility.

### Primary References

| Priority | Resource | URL | Scope |
|----------|----------|-----|-------|
| 🔴 **REQUIRED** | Next.js 14+ App Router Documentation | `https://nextjs.org/docs/app` | App router conventions, Server Components, layout nesting, route groups. The `(dashboard)/`, `(auth)/`, `(admin)/` folder structure depends on this. |
| 🔴 **REQUIRED** | WCAG 2.1 Quick Reference | `https://www.w3.org/WAI/WCAG21/quickref` | The actual accessibility standard. Use when making any UI decision involving interaction, color, keyboard, or ARIA. |
| 🟡 **IMPORTANT** | TanStack Query v5 Documentation | `https://tanstack.com/query/latest` | `useQuery`, `useMutation`, `queryKey` design, stale time, `refetchInterval`. All data fetching hooks in `lib/hooks/`. |
| 🟡 **IMPORTANT** | shadcn/ui Documentation | `https://ui.shadcn.com` | Component primitives, CSS variable customization, installation patterns. All components in `components/ui/`. |
| 🟡 **IMPORTANT** | Zod Documentation | `https://zod.dev` | Schema validation. All form schemas and API response validation. |
| 🟡 **IMPORTANT** | React Hook Form Documentation | `https://react-hook-form.com` | Form state management. Use with Zod via `@hookform/resolvers/zod`. |
| 🟢 **REFERENCE** | Zustand Documentation | `https://zustand-demo.pmnd.rs` | Minimal global client state. Only use for playground history state. Do not use for server data (use TanStack Query). |
| 🟢 **REFERENCE** | Accessibility for Everyone | Laura Kalbag (A Book Apart) | Conceptual foundation for accessibility decisions when WCAG rules don't cover an edge case. |
| 🟢 **REFERENCE** | Tailwind CSS v3 Documentation | `https://tailwindcss.com/docs` | Utility classes only. No custom CSS unless CSS variables for theming. |

### State Management Rules (Enforce These)

```
Server data (requests, usage, models, billing) → TanStack Query ONLY
Form state                                     → React Hook Form + Zod ONLY
Global client state                            → Zustand ONLY (use sparingly)
No useState for server data                    → Always a TanStack Query violation
```

### Component Accessibility Checklist

Every interactive component must have:
- [ ] Visible focus ring (never `outline: none` without replacement)
- [ ] `aria-label` for icon-only buttons
- [ ] `aria-invalid` + `aria-describedby` for form errors
- [ ] Keyboard navigation (Tab, Enter, Escape, Arrow keys where applicable)
- [ ] Color contrast minimum 4.5:1 for normal text, 3:1 for large text
- [ ] `prefers-reduced-motion` respected for all animations

---

## Layer 6: Carbon & Energy Accounting

> Applies to: `apps/carbon-service/`, `apps/api/internal/carbon/`, energy estimation formulas, CO2e calculations, grid data integration, environmental reporting.

**This layer is the primary product differentiator. Treat it with the same rigor as the routing engine.**

### Primary References

| Priority | Resource | URL | Scope |
|----------|----------|-----|-------|
| 🔴 **REQUIRED** | Software Carbon Intensity (SCI) Specification | `https://greensoftware.foundation/articles/software-carbon-intensity` | The formal methodology for calculating CO2e from software systems. Makes our carbon accounting third-party verifiable. Use this spec for all carbon calculation decisions. |
| 🔴 **REQUIRED** | Principles of Green Software Engineering | `https://principles.green` (free) | Energy proportionality, carbon intensity, demand shaping, hardware efficiency. All 8 principles apply directly to EcoLLM. Read before making any architectural decision about energy. |
| 🟡 **IMPORTANT** | Electricity Maps API Documentation | `https://static.electricitymaps.com/api/docs/index.html` | Real-time and historical grid carbon intensity by region. This is the data source for `apps/carbon-service/internal/grid/`. |
| 🟡 **IMPORTANT** | MLPerf Energy Benchmarks | `https://mlcommons.org/benchmarks` | Published energy baselines per model on standard hardware. Use as starting values for `energy_per_token_kwh` in `model-configs/*.yaml`. Cross-reference against our measured values. |
| 🟢 **REFERENCE** | Green Grid PUE Standards | `https://www.thegreengrid.org` | Power Usage Effectiveness methodology. Source for `pue_multiplier: 1.3` in energy calculations. |
| 🟢 **REFERENCE** | EU AI Act — Energy Transparency Requirements | `https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX:32024R1689` | Regulatory requirement for energy reporting. Our carbon dashboard must meet this standard for EU customers. |

### Energy Calculation Formula (Do Not Change Without Review)

```
energy_kwh = (gpu_power_watts × inference_time_hours / batch_size) × pue_multiplier / 1000

co2e_grams = energy_kwh × grid_carbon_intensity_gco2_per_kwh × 1000

Defaults:
  pue_multiplier              = 1.3  (Green Grid standard)
  grid_carbon_intensity       = 450  gCO2/kWh (US average fallback)
  Max error margin to publish = ±10%
```

### Regional Carbon Intensity Reference

```
Region          Intensity (gCO2/kWh)   Source
──────────────────────────────────────────────
US Average      450                    EPA eGRID
US West Coast   200–280                Electricity Maps (high renewables)
EU Average      300                    Electricity Maps
Germany         350–400                Electricity Maps
Norway          30–50                  Electricity Maps (hydro-dominant)
Coal regions    800+                   Electricity Maps
```

---

## Layer 7: Observability & DevOps

> Applies to: `observability/`, `.github/workflows/`, `infra/k8s/`, `infra/docker/`, Prometheus metrics definitions, Grafana dashboard JSON, OpenTelemetry setup, CI/CD pipeline config.

### Primary References

| Priority | Resource | URL | Scope |
|----------|----------|-----|-------|
| 🔴 **REQUIRED** | Prometheus Documentation + PromQL Reference | `https://prometheus.io/docs` | Metric types (`Counter`, `Gauge`, `Histogram`, `Summary`), PromQL syntax, alerting rules. All metric definitions in `apps/api/internal/telemetry/metrics.go`. |
| 🔴 **REQUIRED** | OpenTelemetry Go SDK Documentation | `https://opentelemetry.io/docs/languages/go` | Tracer setup, span creation, attribute naming conventions. Tracing middleware and span definitions. |
| 🟡 **IMPORTANT** | Site Reliability Engineering (Google SRE Book) | `https://sre.google/sre-book/table-of-contents` (free online) | SLOs, error budgets, alerting philosophy, on-call design. Chapters 4 (SLOs), 6 (Monitoring), 13 (On-Call) are highest priority. |
| 🟡 **IMPORTANT** | Kubernetes GPU Scheduling Documentation | `https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus` | Node selectors, tolerations, resource limits for GPU nodes. All manifests under `infra/k8s/gpu/`. |
| 🟡 **IMPORTANT** | Grafana Dashboard JSON Model Docs | `https://grafana.com/docs/grafana/latest/dashboards/json-model` | Panel configuration, datasource references, variable templating. For generating `observability/grafana/dashboards/*.json`. |
| 🟢 **REFERENCE** | GitHub Actions Documentation | `https://docs.github.com/en/actions` | Workflow syntax, job dependencies, secrets, service containers. For `.github/workflows/ci.yml` and deploy workflows. |
| 🟢 **REFERENCE** | Docker Compose v3.9 Specification | `https://docs.docker.com/compose/compose-file/compose-file-v3` | Service definitions, volume mounts, environment variables, GPU runtime config. For `infra/docker/docker-compose.yml`. |

### Prometheus Metric Naming Convention

```
ecollm_{subsystem}_{metric_name}_{unit}

Examples:
  ecollm_requests_total                  (counter)
  ecollm_request_latency_seconds         (histogram)
  ecollm_energy_kwh_per_request          (histogram)
  ecollm_co2e_grams_per_request          (histogram)
  ecollm_cache_hits_total                (counter, label: cache_type)
  ecollm_routing_decisions_total         (counter, labels: model, task_type)
  ecollm_model_health_status             (gauge, label: model)
  ecollm_tokens_per_second               (gauge, label: model)
  ecollm_cost_usd_per_request            (histogram, labels: model, org_id)
```

### SLO Targets

```
Availability:      99.5% uptime (monthly)
Latency p95:       < 1000ms (simple/medium tasks)
Latency p95:       < 3000ms (hard tasks)
Error rate:        < 1% (5xx errors)
Cache hit rate:    > 10% (target 15–20%)
Routing accuracy:  > 85% correct model selection
```

---

## Quick Reference: Which Docs for Which Task

Use this table when starting any implementation task:

| Task | Primary Docs to Reference |
|------|--------------------------|
| Writing a vLLM model config YAML | vLLM docs, AWQ guide, GKE LLM guide |
| Changing routing weights | RouteLLM paper, WRP architecture paper, routing scoring function above |
| Writing a Go handler | Let's Go, Fiber v2 docs |
| Writing a Go service | Learning Go (Bodner), Let's Go Further |
| Writing concurrent Go code | Concurrency in Go (Cox-Buday) |
| Writing a repository (SQL) | pgx docs, PostgreSQL 16 docs |
| Designing a Redis cache key | Redis commands reference, key namespace table above |
| Writing a Next.js page | Next.js App Router docs |
| Writing a React component | shadcn/ui docs, WCAG 2.1 Quick Reference |
| Writing a form | React Hook Form docs, Zod docs |
| Writing data fetching hooks | TanStack Query v5 docs |
| Writing energy calculations | SCI spec, Principles of Green Software, energy formula above |
| Writing carbon estimates | Electricity Maps API docs, SCI spec |
| Writing Prometheus metrics | Prometheus docs, metric naming convention above |
| Writing OpenTelemetry spans | OTel Go SDK docs |
| Writing Kubernetes manifests | Kubernetes GPU docs, GKE LLM guide |
| Writing GitHub Actions workflows | GitHub Actions docs |
| Making an accessibility decision | WCAG 2.1 Quick Ref, accessibility checklist above |
| Writing database migrations | PostgreSQL 16 docs, golang-migrate docs |

---

## Three Load-Bearing Resources

If the agent can only reference three resources for the entire project, use these:

1. **`https://docs.vllm.ai`** — The entire inference layer depends on correct vLLM configuration. Every model serving, quantization, and batching decision must be grounded here.

2. **"Let's Go" + "Let's Go Further" by Alex Edwards** — The entire Go API backend pattern (handler/service/repository, middleware, auth, rate limiting) is built on this foundation.

3. **Green Software Foundation SCI Spec (`https://greensoftware.foundation/articles/software-carbon-intensity`)** — The carbon accounting methodology that makes our environmental claims credible and defensible. This is the brand promise.

---

## Related Project Files

| File | Purpose |
|------|---------|
| `docs/EcoLLM_Startup_Concept_Final.md` | Business model, target users, unit economics, GTM strategy, competitive analysis |
| `docs/EcoLLM_Technical_Architecture.md` | Full technical architecture: 23 sections, schema, API specs, routing algorithm, MVP plan |
| `AGENT_KNOWLEDGE_BASE.md` | This file — documentation sources and agent decision rules |
| `model-configs/*.yaml` | Per-model runtime configuration (quantization, energy baselines, quality benchmarks) |
| `db/schema.sql` | Full PostgreSQL schema reference |
| `observability/prometheus/prometheus.yml` | Prometheus scrape configuration |

---

*Last updated: May 2026 — Update this file when new dependencies, tools, or reference papers are adopted.*
