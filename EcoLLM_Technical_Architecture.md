# EcoLLM Technical Architecture Document

**Version:** 1.0  
**Status:** Implementation-Ready  
**Last Updated:** May 2026  
**Classification:** Engineering Internal

---

## 1. Executive Summary

EcoLLM is an open-source LLM inference platform that routes each user request to the smallest, fastest, lowest-energy model capable of completing the task. It exposes an OpenAI-compatible API, tracks carbon emissions per request, and provides full cost/energy transparency to customers.

**Why this architecture exists:** Every architectural decision in this document is filtered through a single constraint hierarchy: (1) minimize energy per request, (2) minimize cost per request, (3) maximize quality for the task, (4) minimize latency. This ordering is non-negotiable. If a design improves quality but increases energy consumption without justification, it is rejected.

**What the system does:**

- Accepts LLM inference requests via a REST API (OpenAI-compatible format)
- Optimizes prompts for clarity and brevity (reducing token count = reducing energy)
- Classifies task complexity using lightweight heuristics
- Routes to the smallest capable model (Phi-3 3.8B → Mistral 7B → Llama 13B → Llama 70B)
- Tracks GPU power draw, inference time, and grid carbon intensity per request
- Returns responses with embedded energy/cost/carbon metadata
- Exposes a dashboard for usage analytics, carbon tracking, and model routing insights

**Core architectural principles:**

1. **Efficiency is a hard constraint, not a metric.** The system does not log energy for reporting—it uses energy data to make routing decisions.
2. **Smallest viable model wins.** The router defaults to the smallest model and only escalates when quality thresholds are unmet.
3. **Open-source only.** No proprietary model dependencies. All models are self-hosted with quantized weights.
4. **Measure everything.** Every request produces a telemetry record: latency, tokens, energy, CO2e, cost, model used, routing confidence.
5. **Production-first.** This is not a research prototype. The system is designed for 99.5% uptime, <1s p95 latency, and horizontal scaling.

---

## 2. Product Goals

| Goal | Description | Success Metric |
|------|-------------|----------------|
| **Fast responses** | Sub-second p95 latency for simple/medium tasks | p95 < 800ms for Phi-3/Mistral routes |
| **Low-cost inference** | 70–80% cheaper than OpenAI for equivalent tasks | Avg cost/request < $0.001 |
| **Open-source models** | No proprietary model dependencies | 100% of inference on self-hosted open weights |
| **Carbon-aware routing** | Energy is a first-class routing signal | Energy weighted 40% in routing score |
| **Minimal compute waste** | No redundant generations, aggressive caching | Cache hit rate > 15% |
| **Production-ready API** | OpenAI-compatible, rate-limited, authenticated | 99.5% uptime SLA |
| **Developer-friendly** | Drop-in replacement for OpenAI SDK | < 30 min integration time |
| **Transparent accounting** | Per-request cost/energy/CO2 in response metadata | 100% of responses include metadata |

---

## 3. Non-Goals

These are explicitly excluded from the architecture scope:

| Non-Goal | Reason |
|----------|--------|
| **Training foundation models** | We route to existing open-source models. Training is not our business. |
| **Competing with GPT-5/Claude on every task** | We optimize for the 70% of tasks that don't need frontier models. |
| **Long-context inference (>8K tokens) by default** | Long context is exponentially expensive. Customers should use RAG. |
| **Multimodal support in MVP** | Vision models are 50x more energy-intensive. Deferred to Phase 5+. |
| **Unbounded agent loops** | Agents create unpredictable compute usage. Not aligned with efficiency. |
| **Large-model usage unless necessary** | Llama 70B is fallback only (<5% of traffic). |
| **Fine-tuning service** | Adds complexity and compute. Deferred until $5M+ ARR. |
| **Streaming in MVP** | Streaming increases per-token overhead. Added in Phase 3. |
| **Self-hosted customer deployments** | Managed cloud only for MVP. On-prem deferred to enterprise tier. |

---

## 4. System Architecture Overview

### High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                         CLIENTS                                      │
│   SDK (Python/JS)  │  cURL  │  Dashboard UI  │  Admin Panel          │
└────────────────────┬─────────────────────────────────────────────────┘
                     │ HTTPS
┌────────────────────▼─────────────────────────────────────────────────┐
│                    API GATEWAY (Go / Fiber)                           │
│  ┌─────────┐ ┌──────────┐ ┌───────────┐ ┌────────────┐ ┌─────────┐ │
│  │  Auth   │ │ Rate     │ │ Request   │ │ Validation │ │ CORS    │ │
│  │  MW     │ │ Limiter  │ │ Tracing   │ │ MW         │ │ MW      │ │
│  └─────────┘ └──────────┘ └───────────┘ └────────────┘ └─────────┘ │
└────────────────────┬─────────────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────────────────────────┐
│                    CORE SERVICES (Go)                                 │
│                                                                      │
│  ┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐ │
│  │ Prompt Optimizer  │──▶│ Task Classifier  │──▶│  Model Router    │ │
│  │ (rule-based +     │   │ (heuristic +     │   │ (scoring engine) │ │
│  │  Phi-3 fallback)  │   │  keyword match)  │   │                  │ │
│  └──────────────────┘   └──────────────────┘   └────────┬─────────┘ │
│                                                          │           │
│  ┌──────────────────┐   ┌──────────────────┐            │           │
│  │ Energy Estimator │◀──│ Carbon Service   │            │           │
│  │ (GPU telemetry)  │   │ (grid intensity) │            │           │
│  └──────────────────┘   └──────────────────┘            │           │
└─────────────────────────────────────────────────────────┬───────────┘
                                                          │
┌─────────────────────────────────────────────────────────▼───────────┐
│                 INFERENCE GATEWAY (Go)                               │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐               │
│  │ Phi-3   │  │ Mistral │  │ Llama   │  │ Llama   │               │
│  │ 3.8B    │  │ 7B      │  │ 13B     │  │ 70B     │               │
│  │ (vLLM)  │  │ (vLLM)  │  │ (vLLM)  │  │ (vLLM)  │               │
│  │ L4 GPU  │  │ L4 GPU  │  │ L40S    │  │ A100    │               │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘               │
└─────────────────────────────────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────────────────────────┐
│                    DATA LAYER                                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │ PostgreSQL   │  │ Redis        │  │ Prometheus + Grafana     │  │
│  │ (metadata,   │  │ (cache,      │  │ (metrics, dashboards,    │  │
│  │  billing,    │  │  rate limits, │  │  alerts)                 │  │
│  │  audit logs) │  │  sessions)   │  │                          │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────────────────────────────┐
│                    FRONTEND (Next.js)                                 │
│  Dashboard │ Playground │ API Keys │ Usage/Billing │ Carbon Report   │
└──────────────────────────────────────────────────────────────────────┘
```

### Service Inventory

| Service | Language | Responsibility | Runs On |
|---------|----------|---------------|---------|
| **api-gateway** | Go (Fiber) | HTTP routing, auth, rate limiting, request orchestration | CPU node |
| **prompt-optimizer** | Go + Python sidecar | Prompt rewriting (rules in Go, Phi-3 fallback in Python) | CPU node |
| **task-classifier** | Go | Classify task complexity via heuristics | CPU node (in-process) |
| **model-router** | Go | Score models, select optimal route | CPU node (in-process) |
| **inference-gateway** | Go | Proxy requests to model runtimes, handle retries/fallback | CPU node |
| **carbon-service** | Go | Grid carbon intensity lookup, CO2e calculation | CPU node |
| **model-runtime** | Python (vLLM) | Run inference on GPU | GPU node |
| **web** | Next.js/TypeScript | Dashboard, playground, admin panel | CDN + edge |
| **postgres** | PostgreSQL 16 | Persistent storage | Managed DB |
| **redis** | Redis 7 | Caching, rate limiting, sessions | Managed cache |

### Why These Technology Choices

| Decision | Choice | Justification |
|----------|--------|---------------|
| **API Language** | Go (Fiber) | Low memory footprint, fast HTTP, excellent concurrency. Go binary uses ~10MB RAM vs ~50MB for Node.js. Fiber benchmarks at 100K+ req/s. |
| **Inference Runtime** | vLLM | Best throughput for batched inference. PagedAttention reduces GPU memory waste. Production-proven. |
| **Database** | PostgreSQL 16 | ACID compliance for billing/audit. JSON support for flexible metadata. Mature ecosystem. |
| **Cache** | Redis 7 | Sub-millisecond lookups. Built-in TTL. Rate limiting primitives. Pub/sub for invalidation. |
| **Frontend** | Next.js 14+ | Server components reduce client JS. App router for clean layouts. React ecosystem for components. |
| **Metrics** | Prometheus + Grafana | Industry standard. PromQL for custom energy queries. Grafana for visual dashboards. |
| **Tracing** | OpenTelemetry | Vendor-neutral. Correlates traces across Go services and Python inference. |

---

## 5. Recommended Monorepo File Structure

```
ecollm/
├── apps/
│   ├── api/                          # Go API gateway + core services
│   │   ├── cmd/
│   │   │   └── server/
│   │   │       └── main.go           # Entry point
│   │   ├── internal/
│   │   │   ├── auth/                 # Authentication & API key validation
│   │   │   │   ├── handler.go
│   │   │   │   ├── service.go
│   │   │   │   ├── repository.go
│   │   │   │   └── middleware.go
│   │   │   ├── chat/                 # Chat completions handler
│   │   │   │   ├── handler.go
│   │   │   │   ├── service.go
│   │   │   │   └── types.go
│   │   │   ├── router/               # Model routing engine
│   │   │   │   ├── classifier.go     # Task classification
│   │   │   │   ├── scorer.go         # Model scoring function
│   │   │   │   ├── selector.go       # Final model selection
│   │   │   │   └── types.go
│   │   │   ├── prompt/               # Prompt optimization
│   │   │   │   ├── optimizer.go      # Rule-based rewriting
│   │   │   │   ├── rules.go          # Optimization rules
│   │   │   │   └── types.go
│   │   │   ├── inference/            # Inference gateway
│   │   │   │   ├── gateway.go        # Route to model runtimes
│   │   │   │   ├── client.go         # HTTP client for vLLM
│   │   │   │   ├── fallback.go       # Fallback logic
│   │   │   │   └── types.go
│   │   │   ├── carbon/               # Energy & carbon estimation
│   │   │   │   ├── estimator.go      # CO2e calculation
│   │   │   │   ├── grid.go           # Grid carbon intensity data
│   │   │   │   └── types.go
│   │   │   ├── usage/                # Usage tracking & aggregation
│   │   │   │   ├── handler.go
│   │   │   │   ├── service.go
│   │   │   │   └── repository.go
│   │   │   ├── admin/                # Admin endpoints
│   │   │   │   ├── handler.go
│   │   │   │   └── service.go
│   │   │   ├── billing/              # Billing events & invoicing
│   │   │   │   ├── handler.go
│   │   │   │   ├── service.go
│   │   │   │   └── repository.go
│   │   │   ├── middleware/           # Shared middleware
│   │   │   │   ├── ratelimit.go
│   │   │   │   ├── cors.go
│   │   │   │   ├── requestid.go
│   │   │   │   ├── logging.go
│   │   │   │   └── recovery.go
│   │   │   ├── config/              # Configuration loading
│   │   │   │   └── config.go
│   │   │   ├── database/            # Database connection & migrations
│   │   │   │   ├── postgres.go
│   │   │   │   └── redis.go
│   │   │   └── telemetry/           # OpenTelemetry setup
│   │   │       ├── tracer.go
│   │   │       └── metrics.go
│   │   ├── pkg/                     # Shared utilities (exported)
│   │   │   ├── apierror/            # Standardized API errors
│   │   │   │   └── errors.go
│   │   │   ├── validator/           # Input validation
│   │   │   │   └── validator.go
│   │   │   └── hash/               # API key hashing
│   │   │       └── hash.go
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── Dockerfile
│   │
│   ├── inference-gateway/            # Lightweight Go proxy for model runtimes
│   │   ├── cmd/
│   │   │   └── gateway/
│   │   │       └── main.go
│   │   ├── internal/
│   │   │   ├── proxy/               # HTTP proxy to vLLM instances
│   │   │   ├── health/              # Model health checks
│   │   │   ├── pool/                # Connection pooling
│   │   │   └── metrics/             # GPU telemetry collection
│   │   ├── go.mod
│   │   └── Dockerfile
│   │
│   ├── carbon-service/               # Carbon intensity data service
│   │   ├── cmd/
│   │   │   └── carbon/
│   │   │       └── main.go
│   │   ├── internal/
│   │   │   ├── grid/                # Grid carbon intensity API client
│   │   │   ├── calculator/          # CO2e calculation engine
│   │   │   └── cache/              # Cache grid data (changes hourly)
│   │   ├── go.mod
│   │   └── Dockerfile
│   │
│   ├── prompt-optimizer/             # Python sidecar for Phi-3 fallback
│   │   ├── src/
│   │   │   ├── optimizer.py         # Phi-3 prompt refinement
│   │   │   ├── server.py            # FastAPI server
│   │   │   └── config.py
│   │   ├── requirements.txt
│   │   └── Dockerfile
│   │
│   └── web/                          # Next.js frontend
│       ├── src/
│       │   ├── app/                  # App router pages
│       │   │   ├── (auth)/
│       │   │   │   ├── login/
│       │   │   │   └── register/
│       │   │   ├── (dashboard)/
│       │   │   │   ├── overview/
│       │   │   │   ├── requests/
│       │   │   │   ├── models/
│       │   │   │   ├── carbon/
│       │   │   │   ├── usage/
│       │   │   │   ├── billing/
│       │   │   │   ├── api-keys/
│       │   │   │   ├── playground/
│       │   │   │   └── settings/
│       │   │   ├── (admin)/
│       │   │   │   ├── models/
│       │   │   │   ├── routes/
│       │   │   │   └── metrics/
│       │   │   ├── layout.tsx
│       │   │   └── page.tsx
│       │   ├── components/
│       │   │   ├── ui/              # shadcn/ui primitives
│       │   │   ├── dashboard/       # Dashboard-specific components
│       │   │   ├── carbon/          # Carbon tracking components
│       │   │   ├── models/          # Model management components
│       │   │   ├── playground/      # Playground components
│       │   │   └── shared/          # Layout, nav, footer
│       │   ├── lib/
│       │   │   ├── api.ts           # API client (fetch wrapper)
│       │   │   ├── hooks/           # Custom hooks
│       │   │   ├── utils.ts         # Utility functions
│       │   │   └── constants.ts
│       │   ├── stores/              # Zustand stores (minimal)
│       │   │   └── playground.ts
│       │   └── types/               # Shared TypeScript types
│       │       ├── api.ts
│       │       ├── models.ts
│       │       └── carbon.ts
│       ├── public/
│       ├── tailwind.config.ts
│       ├── next.config.ts
│       ├── tsconfig.json
│       ├── package.json
│       └── Dockerfile
│
├── packages/
│   ├── contracts/                    # Shared API contracts (OpenAPI spec)
│   │   ├── openapi.yaml
│   │   └── types/
│   │       ├── request.go
│   │       ├── response.go
│   │       └── index.ts             # Generated TS types from OpenAPI
│   └── ui/                          # Shared UI component library (if needed)
│       └── package.json
│
├── infra/
│   ├── docker/
│   │   ├── docker-compose.yml        # Local development
│   │   ├── docker-compose.gpu.yml    # GPU inference local
│   │   └── docker-compose.test.yml   # Test environment
│   ├── k8s/
│   │   ├── base/                    # Base Kubernetes manifests
│   │   │   ├── api-deployment.yaml
│   │   │   ├── inference-gateway.yaml
│   │   │   ├── carbon-service.yaml
│   │   │   ├── web-deployment.yaml
│   │   │   ├── postgres.yaml
│   │   │   ├── redis.yaml
│   │   │   └── namespace.yaml
│   │   ├── overlays/
│   │   │   ├── staging/
│   │   │   └── production/
│   │   └── gpu/
│   │       ├── vllm-phi3.yaml
│   │       ├── vllm-mistral7b.yaml
│   │       ├── vllm-llama13b.yaml
│   │       └── vllm-llama70b.yaml
│   └── terraform/                   # Cloud infrastructure (if needed)
│       ├── main.tf
│       └── variables.tf
│
├── db/
│   ├── migrations/
│   │   ├── 001_create_users.sql
│   │   ├── 002_create_organizations.sql
│   │   ├── 003_create_api_keys.sql
│   │   ├── 004_create_requests.sql
│   │   ├── 005_create_model_registry.sql
│   │   ├── 006_create_inference_results.sql
│   │   ├── 007_create_energy_measurements.sql
│   │   ├── 008_create_billing_events.sql
│   │   ├── 009_create_audit_logs.sql
│   │   └── 010_create_feedback.sql
│   ├── seeds/
│   │   ├── models.sql               # Default model registry entries
│   │   └── grid_data.sql            # Baseline grid carbon data
│   └── schema.sql                   # Full schema reference
│
├── model-configs/
│   ├── phi3-3.8b-q4.yaml
│   ├── mistral-7b-q4.yaml
│   ├── llama-13b-q4.yaml
│   └── llama-70b-q8.yaml
│
├── observability/
│   ├── prometheus/
│   │   └── prometheus.yml
│   ├── grafana/
│   │   ├── dashboards/
│   │   │   ├── overview.json
│   │   │   ├── routing.json
│   │   │   ├── energy.json
│   │   │   ├── latency.json
│   │   │   └── cost.json
│   │   └── datasources.yml
│   └── otel/
│       └── collector.yaml
│
├── scripts/
│   ├── setup.sh                     # Local dev setup
│   ├── migrate.sh                   # Run database migrations
│   ├── seed.sh                      # Seed database
│   ├── benchmark.sh                 # Run model benchmarks
│   └── generate-types.sh            # Generate TS types from OpenAPI
│
├── docs/
│   ├── architecture.md              # This document
│   ├── api-reference.md
│   ├── deployment.md
│   ├── routing-algorithm.md
│   └── carbon-methodology.md
│
├── tests/
│   ├── integration/
│   │   ├── api_test.go
│   │   ├── routing_test.go
│   │   └── inference_test.go
│   ├── load/
│   │   └── k6/
│   │       └── load_test.js
│   └── e2e/
│       └── playwright/
│           ├── dashboard.spec.ts
│           └── playground.spec.ts
│
├── .github/
│   └── workflows/
│       ├── ci.yml                   # Lint, test, build
│       ├── deploy-staging.yml
│       └── deploy-production.yml
│
├── .env.example
├── Makefile
├── README.md
└── LICENSE
```

### Folder Purpose Summary

| Folder | Purpose |
|--------|---------|
| `apps/api` | Go API gateway — all HTTP handling, routing logic, prompt optimization, auth |
| `apps/inference-gateway` | Lightweight Go proxy that manages connections to vLLM model runtimes |
| `apps/carbon-service` | Standalone Go service for grid carbon intensity lookups and CO2e calculation |
| `apps/prompt-optimizer` | Python FastAPI sidecar — only used for Phi-3 fallback prompt refinement |
| `apps/web` | Next.js dashboard, playground, admin panel |
| `packages/contracts` | OpenAPI specification and generated types shared between Go and TypeScript |
| `infra/` | Docker Compose (local), Kubernetes manifests (staging/prod), Terraform (cloud) |
| `db/` | SQL migrations, seeds, schema reference |
| `model-configs/` | YAML configs per model (quantization, max tokens, batch size, energy baseline) |
| `observability/` | Prometheus scrape configs, Grafana dashboard JSON, OpenTelemetry collector config |
| `scripts/` | Developer scripts for setup, migration, benchmarking |
| `docs/` | Architecture docs, API reference, deployment guides |
| `tests/` | Integration tests, load tests (k6), E2E tests (Playwright) |

---

## 6. Backend Architecture in Go

### 6.1 Service Boundaries

The Go backend is a **modular monolith** — a single binary with clear internal package boundaries. This is intentional:

- **Why not microservices:** At MVP scale (50–200 customers), microservices add operational overhead without benefit. Network hops between services increase latency and energy.
- **Why not a monolithic blob:** Internal packages enforce separation of concerns. When scale demands it, any package can be extracted into a standalone service.
- **Exception:** The inference gateway and carbon service are separate binaries because they have different deployment requirements (GPU nodes for inference, separate scaling for carbon lookups).

### 6.2 Package Structure

```
apps/api/
├── cmd/
│   └── server/
│       └── main.go                 # Wire up dependencies, start server
│
├── internal/                       # Private packages (not importable)
│   ├── config/
│   │   └── config.go              # Env-based config with defaults
│   │
│   ├── database/
│   │   ├── postgres.go            # Connection pool, health checks
│   │   └── redis.go               # Redis client, connection pool
│   │
│   ├── middleware/
│   │   ├── auth.go                # API key extraction + validation
│   │   ├── ratelimit.go           # Redis-backed sliding window rate limiter
│   │   ├── requestid.go           # Generate + propagate X-Request-ID
│   │   ├── logging.go             # Structured request/response logging
│   │   ├── cors.go                # CORS headers
│   │   ├── recovery.go            # Panic recovery
│   │   └── maxbody.go             # Request body size limits
│   │
│   ├── auth/
│   │   ├── handler.go             # POST /auth/login, POST /auth/logout, GET /me
│   │   ├── service.go             # Login logic, session management
│   │   ├── repository.go          # User + API key queries
│   │   └── middleware.go          # JWT validation middleware
│   │
│   ├── chat/
│   │   ├── handler.go             # POST /v1/chat/completions
│   │   ├── service.go             # Orchestrate: optimize → classify → route → infer → track
│   │   └── types.go               # OpenAI-compatible request/response types
│   │
│   ├── router/
│   │   ├── classifier.go          # Task classification (heuristic + keyword)
│   │   ├── scorer.go              # Model scoring function (energy 40%, cost 30%, quality 20%, latency 10%)
│   │   ├── selector.go            # Select best model, apply constraints, get fallback
│   │   ├── registry.go            # In-memory model registry (loaded from DB on startup)
│   │   └── types.go               # RouteDecision, ModelCandidate, TaskType
│   │
│   ├── prompt/
│   │   ├── optimizer.go           # Rule-based prompt optimization (Go-native)
│   │   ├── rules.go               # Optimization rule definitions
│   │   ├── phi3_client.go         # HTTP client to Python Phi-3 sidecar (fallback only)
│   │   └── types.go
│   │
│   ├── inference/
│   │   ├── gateway.go             # Route to correct vLLM instance, handle timeouts
│   │   ├── client.go              # HTTP client for vLLM /v1/completions endpoint
│   │   ├── fallback.go            # Retry with larger model on failure/low confidence
│   │   ├── pool.go                # Connection pool management per model
│   │   └── types.go
│   │
│   ├── carbon/
│   │   ├── estimator.go           # Calculate energy_kwh and co2e_grams per request
│   │   ├── grid.go                # Fetch grid carbon intensity (cached, hourly refresh)
│   │   └── types.go
│   │
│   ├── usage/
│   │   ├── handler.go             # GET /v1/usage, GET /v1/requests/{id}
│   │   ├── service.go             # Aggregate usage by org/time/model
│   │   ├── repository.go          # Query usage_aggregates, requests
│   │   └── worker.go              # Background worker: aggregate usage hourly
│   │
│   ├── admin/
│   │   ├── handler.go             # GET/POST/PATCH /admin/models, GET /admin/metrics
│   │   └── service.go             # Model registry CRUD, system metrics
│   │
│   ├── billing/
│   │   ├── handler.go             # GET /v1/billing
│   │   ├── service.go             # Calculate billing from usage aggregates
│   │   ├── repository.go          # billing_events table
│   │   └── worker.go              # Background: generate daily billing events
│   │
│   └── telemetry/
│       ├── tracer.go              # OpenTelemetry tracer setup
│       ├── metrics.go             # Prometheus metric definitions
│       └── logger.go              # Structured logger (zerolog)
│
├── pkg/                            # Exported packages
│   ├── apierror/
│   │   └── errors.go             # Standardized error responses
│   ├── validator/
│   │   └── validator.go          # Input validation helpers
│   └── hash/
│       └── hash.go               # bcrypt-based API key hashing
│
├── go.mod
├── go.sum
└── Dockerfile
```

### 6.3 Layered Architecture

Each domain package follows a strict three-layer pattern:

```
Handler (HTTP) → Service (Business Logic) → Repository (Data Access)
```

**Handler Layer:**
- Parses HTTP request (body, params, headers)
- Validates input using `pkg/validator`
- Calls service layer
- Returns standardized JSON response
- Never contains business logic

**Service Layer:**
- Contains all business logic
- Orchestrates calls to other services
- Handles errors and retries
- Never touches HTTP or SQL directly

**Repository Layer:**
- Executes SQL queries via `pgx` (PostgreSQL driver)
- Returns domain types (not `sql.Row`)
- Handles connection pooling
- Never contains business logic

### 6.4 Key Design Patterns

**Dependency Injection (Constructor-Based):**

```go
// main.go — wire up dependencies
func main() {
    cfg := config.Load()
    
    // Data layer
    pgPool := database.NewPostgresPool(cfg.DatabaseURL)
    redisClient := database.NewRedisClient(cfg.RedisURL)
    
    // Repositories
    userRepo := auth.NewRepository(pgPool)
    requestRepo := chat.NewRequestRepository(pgPool)
    usageRepo := usage.NewRepository(pgPool)
    modelRepo := router.NewModelRepository(pgPool)
    
    // Services
    authService := auth.NewService(userRepo, redisClient, cfg.JWTSecret)
    carbonEstimator := carbon.NewEstimator(cfg.GridRegion)
    promptOptimizer := prompt.NewOptimizer(cfg.Phi3SidecarURL)
    taskClassifier := router.NewClassifier()
    modelScorer := router.NewScorer(modelRepo, carbonEstimator)
    modelSelector := router.NewSelector(modelScorer)
    inferenceGateway := inference.NewGateway(cfg.InferenceEndpoints)
    
    chatService := chat.NewService(
        promptOptimizer,
        taskClassifier,
        modelSelector,
        inferenceGateway,
        carbonEstimator,
        requestRepo,
    )
    
    usageService := usage.NewService(usageRepo)
    adminService := admin.NewService(modelRepo)
    
    // Handlers
    authHandler := auth.NewHandler(authService)
    chatHandler := chat.NewHandler(chatService)
    usageHandler := usage.NewHandler(usageService)
    adminHandler := admin.NewHandler(adminService)
    
    // Router setup
    app := fiber.New(fiber.Config{
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 30 * time.Second,
        BodyLimit:    1 * 1024 * 1024, // 1MB max body
    })
    
    // Global middleware
    app.Use(middleware.RequestID())
    app.Use(middleware.Logging(cfg.LogLevel))
    app.Use(middleware.Recovery())
    app.Use(middleware.CORS(cfg.AllowedOrigins))
    
    // Public API routes
    api := app.Group("/v1", middleware.Auth(authService))
    api.Post("/chat/completions", chatHandler.CreateCompletion)
    api.Post("/completions", chatHandler.CreateCompletion)
    api.Post("/route/preview", chatHandler.PreviewRoute)
    api.Get("/models", chatHandler.ListModels)
    api.Get("/usage", usageHandler.GetUsage)
    api.Get("/requests/:id", usageHandler.GetRequest)
    
    // Auth routes
    authGroup := app.Group("/auth")
    authGroup.Post("/login", authHandler.Login)
    authGroup.Post("/logout", authHandler.Logout)
    authGroup.Get("/me", middleware.Auth(authService), authHandler.Me)
    
    // Admin routes
    adminGroup := app.Group("/admin", middleware.Auth(authService), middleware.RequireRole("admin"))
    adminGroup.Get("/metrics", adminHandler.GetMetrics)
    adminGroup.Get("/models", adminHandler.ListModels)
    adminGroup.Post("/models", adminHandler.CreateModel)
    adminGroup.Patch("/models/:id", adminHandler.UpdateModel)
    adminGroup.Get("/routes", adminHandler.GetRoutes)
    adminGroup.Get("/carbon", adminHandler.GetCarbonMetrics)
    
    // Background workers
    go usage.StartAggregationWorker(usageRepo, 1*time.Hour)
    go billing.StartBillingWorker(billingRepo, 24*time.Hour)
    
    // Start server
    log.Fatal(app.Listen(":" + cfg.Port))
}
```

**Configuration Management (Environment Variables):**

```go
// internal/config/config.go
type Config struct {
    Port               string        `env:"PORT" default:"8080"`
    DatabaseURL        string        `env:"DATABASE_URL" required:"true"`
    RedisURL           string        `env:"REDIS_URL" required:"true"`
    JWTSecret          string        `env:"JWT_SECRET" required:"true"`
    LogLevel           string        `env:"LOG_LEVEL" default:"info"`
    AllowedOrigins     []string      `env:"ALLOWED_ORIGINS" default:"http://localhost:3000"`
    Phi3SidecarURL     string        `env:"PHI3_SIDECAR_URL" default:"http://prompt-optimizer:8081"`
    GridRegion         string        `env:"GRID_REGION" default:"US-EAST"`
    RateLimitPerMinute int           `env:"RATE_LIMIT_PER_MIN" default:"60"`
    RequestTimeout     time.Duration `env:"REQUEST_TIMEOUT" default:"30s"`
    InferenceEndpoints map[string]string // Loaded from model-configs/
}
```

**Structured Logging (zerolog):**

```go
// Every log entry includes: request_id, org_id, model, latency, energy
log.Info().
    Str("request_id", requestID).
    Str("org_id", orgID).
    Str("model", routeDecision.Model).
    Int64("latency_ms", latencyMs).
    Float64("energy_kwh", energyKwh).
    Float64("co2e_grams", co2eGrams).
    Float64("cost_usd", costUSD).
    Msg("inference completed")
```

**Error Handling (Standardized API Errors):**

```go
// pkg/apierror/errors.go
type APIError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Type    string `json:"type"`
    TraceID string `json:"trace_id,omitempty"`
}

var (
    ErrUnauthorized    = &APIError{Code: 401, Message: "Invalid API key", Type: "auth_error"}
    ErrRateLimited     = &APIError{Code: 429, Message: "Rate limit exceeded", Type: "rate_limit_error"}
    ErrInvalidRequest  = &APIError{Code: 400, Message: "Invalid request body", Type: "validation_error"}
    ErrModelUnavailable = &APIError{Code: 503, Message: "No model available", Type: "service_error"}
    ErrInferenceFailed = &APIError{Code: 502, Message: "Inference failed", Type: "inference_error"}
)
```

**Rate Limiting (Redis Sliding Window):**

```go
// internal/middleware/ratelimit.go
func RateLimit(redis *redis.Client, limit int, window time.Duration) fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := fmt.Sprintf("rl:%s", c.Get("X-API-Key"))
        
        count, err := redis.Incr(ctx, key).Result()
        if err != nil {
            return c.Next() // Fail open (don't block on Redis failure)
        }
        
        if count == 1 {
            redis.Expire(ctx, key, window)
        }
        
        c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
        c.Set("X-RateLimit-Remaining", strconv.Itoa(max(0, limit-int(count))))
        
        if int(count) > limit {
            return c.Status(429).JSON(apierror.ErrRateLimited)
        }
        
        return c.Next()
    }
}
```

**Request Tracing (OpenTelemetry):**

Every request gets a trace ID propagated through all services:

```go
// internal/middleware/requestid.go
func RequestID() fiber.Handler {
    return func(c *fiber.Ctx) error {
        id := c.Get("X-Request-ID")
        if id == "" {
            id = uuid.NewString()
        }
        c.Set("X-Request-ID", id)
        c.Locals("request_id", id)
        
        // Start OpenTelemetry span
        ctx, span := tracer.Start(c.UserContext(), "http.request",
            trace.WithAttributes(
                attribute.String("http.method", c.Method()),
                attribute.String("http.path", c.Path()),
                attribute.String("request.id", id),
            ),
        )
        defer span.End()
        c.SetUserContext(ctx)
        
        return c.Next()
    }
}
```

---

## 7. AI / LLM Routing Architecture

This is the core IP of EcoLLM. The routing system is a deterministic pipeline that transforms a user request into a model selection decision in <10ms.

### 7.1 Routing Pipeline

```
User Prompt
    │
    ▼
┌─────────────────────────┐
│  1. Prompt Normalization │  Trim whitespace, normalize encoding, detect language
│     Cost: <0.1ms         │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  2. Prompt Optimization  │  Rule-based rewriting → Phi-3 fallback if confidence < 0.7
│     Cost: 0.5–50ms       │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  3. Cache Check          │  SHA-256 hash of normalized prompt → Redis lookup
│     Cost: <1ms           │  If hit: return cached response (skip inference entirely)
└────────────┬────────────┘
             │ (cache miss)
┌────────────▼────────────┐
│  4. Task Classification  │  Keyword matching + length heuristic → task_type + complexity
│     Cost: <0.5ms         │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  5. Safety Check         │  Blocklist scan, content policy check
│     Cost: <0.5ms         │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  6. Model Scoring        │  Score each candidate model on energy/cost/quality/latency
│     Cost: <0.5ms         │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  7. Constraint Check     │  Apply customer min_quality, max_latency, max_cost thresholds
│     Cost: <0.1ms         │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  8. Energy Estimation    │  Predict energy_kwh and co2e for selected model
│     Cost: <0.5ms         │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  9. Inference Dispatch   │  Send to vLLM instance, handle timeout/retry/fallback
│     Cost: 80–3000ms      │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│ 10. Response Assembly    │  Attach metadata: model, energy, cost, co2e, trace_id
│     Cost: <0.5ms         │
└─────────────────────────┘
```

**Total routing overhead (excluding inference): <15ms.** This is critical — routing must never become a bottleneck.

### 7.2 Prompt Normalization

```go
func NormalizePrompt(raw string) string {
    // 1. Trim whitespace
    normalized := strings.TrimSpace(raw)
    // 2. Normalize unicode
    normalized = norm.NFC.String(normalized)
    // 3. Collapse multiple newlines
    normalized = regexp.MustCompile(`\n{3,}`).ReplaceAllString(normalized, "\n\n")
    // 4. Remove null bytes
    normalized = strings.ReplaceAll(normalized, "\x00", "")
    return normalized
}
```

### 7.3 Prompt Optimization

**Rule-based optimizer (Go-native, fast path):**

```go
type OptimizationRule struct {
    Name     string
    Applies  func(prompt string) bool
    Optimize func(prompt string) string
}

var rules = []OptimizationRule{
    {
        Name:    "add_specificity_to_code_requests",
        Applies: func(p string) bool { return containsAny(p, "write code", "code for", "script to") },
        Optimize: func(p string) string {
            if !contains(p, "language") && !containsAny(p, "python", "go", "javascript", "rust") {
                p += "\nPlease specify the programming language and include comments."
            }
            return p
        },
    },
    {
        Name:    "add_format_guidance",
        Applies: func(p string) bool { return containsAny(p, "explain", "describe", "what is") },
        Optimize: func(p string) string {
            if len(p) < 50 {
                p += "\nProvide a concise, structured answer."
            }
            return p
        },
    },
    {
        Name:    "truncate_excessive_context",
        Applies: func(p string) bool { return len(p) > 4000 },
        Optimize: func(p string) string {
            // Trim to most relevant 4000 chars (keep first + last sections)
            if len(p) > 4000 {
                return p[:2000] + "\n...\n" + p[len(p)-2000:]
            }
            return p
        },
    },
}
```

**Phi-3 fallback (Python sidecar, used <10% of requests):**

Only invoked when the rule-based optimizer has low confidence (e.g., ambiguous multi-part request). The sidecar is a FastAPI server running Phi-3 3.8B quantized.

```python
# apps/prompt-optimizer/src/server.py
from fastapi import FastAPI
from vllm import LLM

app = FastAPI()
model = LLM("microsoft/Phi-3-mini-4k-instruct", quantization="awq", max_model_len=2048)

@app.post("/optimize")
async def optimize(request: dict):
    prompt = request["prompt"]
    system = (
        "Rewrite this user prompt to be clearer and more specific. "
        "Do not change the intent. Keep it concise. Return only the rewritten prompt."
    )
    result = model.generate(
        [f"<|system|>{system}<|end|><|user|>{prompt}<|end|><|assistant|>"],
        max_tokens=512
    )
    return {"optimized_prompt": result[0].outputs[0].text.strip()}
```

### 7.4 Task Classification

```go
type TaskType string

const (
    TaskSimple      TaskType = "simple"       // FAQ, classification, extraction
    TaskMedium      TaskType = "medium"       // Writing, basic coding, summarization
    TaskHard        TaskType = "hard"         // Complex reasoning, debugging, analysis
    TaskSpecialized TaskType = "specialized"  // Domain-specific, long-form
)

type ClassificationResult struct {
    TaskType   TaskType
    Complexity int      // 1–10
    Confidence float64  // 0–1
    Signals    []string // Which heuristics fired
}

func Classify(prompt string) ClassificationResult {
    complexity := 1
    signals := []string{}

    // Length heuristic
    if len(prompt) > 500 {
        complexity += 2
        signals = append(signals, "long_prompt")
    }
    if len(prompt) > 1500 {
        complexity += 2
        signals = append(signals, "very_long_prompt")
    }

    // Keyword-based complexity scoring
    hardKeywords := map[string]int{
        "debug":        3, "architect":   3, "design system": 3,
        "reason about": 3, "analyze":     2, "compare":       2,
        "optimize":     2, "refactor":    2,
    }
    mediumKeywords := map[string]int{
        "write":     1, "code":      1, "generate": 1,
        "summarize": 1, "translate": 1, "explain":  1,
    }
    simpleKeywords := map[string]int{
        "classify": 0, "extract": 0, "detect": 0,
        "list":     0, "find":    0, "what is": 0,
    }

    lower := strings.ToLower(prompt)
    for keyword, weight := range hardKeywords {
        if strings.Contains(lower, keyword) {
            complexity += weight
            signals = append(signals, "hard:"+keyword)
        }
    }
    for keyword, weight := range mediumKeywords {
        if strings.Contains(lower, keyword) {
            complexity += weight
            signals = append(signals, "medium:"+keyword)
        }
    }
    for keyword := range simpleKeywords {
        if strings.Contains(lower, keyword) {
            signals = append(signals, "simple:"+keyword)
        }
    }

    // Multi-step detection
    if strings.Contains(lower, "step by step") || strings.Contains(lower, "first") && strings.Contains(lower, "then") {
        complexity += 2
        signals = append(signals, "multi_step")
    }

    // Cap at 10
    if complexity > 10 {
        complexity = 10
    }

    // Map to task type
    var taskType TaskType
    switch {
    case complexity <= 3:
        taskType = TaskSimple
    case complexity <= 6:
        taskType = TaskMedium
    case complexity <= 9:
        taskType = TaskHard
    default:
        taskType = TaskSpecialized
    }

    confidence := 0.8 // Base confidence for heuristic
    if len(signals) >= 3 {
        confidence = 0.9
    }

    return ClassificationResult{
        TaskType:   taskType,
        Complexity: complexity,
        Confidence: confidence,
        Signals:    signals,
    }
}
```

### 7.5 Model Scoring Function

This is the core algorithm. **Energy is weighted 40% — this is non-negotiable.**

```go
type ModelCandidate struct {
    Name             string
    Size             string
    Quantization     string
    LatencyP95Ms     int
    EnergyKwh        float64
    CostUSD          float64
    QualityBenchmark float64  // 0–1 scale, from benchmark suite
    FailureRate      float64  // Historical failure rate (0–1)
}

type ScoringWeights struct {
    Energy  float64 // 0.40 (NON-NEGOTIABLE MINIMUM)
    Cost    float64 // 0.30
    Quality float64 // 0.20
    Latency float64 // 0.10
}

var DefaultWeights = ScoringWeights{
    Energy:  0.40,
    Cost:    0.30,
    Quality: 0.20,
    Latency: 0.10,
}

// Reference maximums for normalization
const (
    MaxEnergyKwh = 0.001   // Llama 70B worst case
    MaxCostUSD   = 0.01    // Llama 70B worst case
    MaxLatencyMs = 5000    // 5 second ceiling
    MaxRisk      = 0.10    // 10% failure rate ceiling
)

func ScoreModel(candidate ModelCandidate, taskType TaskType, weights ScoringWeights) float64 {
    // Normalize to 0–1 (higher = better)
    energyScore := 1.0 - (candidate.EnergyKwh / MaxEnergyKwh)
    costScore := 1.0 - (candidate.CostUSD / MaxCostUSD)
    qualityScore := candidate.QualityBenchmark
    latencyScore := 1.0 - (float64(candidate.LatencyP95Ms) / MaxLatencyMs)

    // Clamp to [0, 1]
    energyScore = clamp(energyScore, 0, 1)
    costScore = clamp(costScore, 0, 1)
    qualityScore = clamp(qualityScore, 0, 1)
    latencyScore = clamp(latencyScore, 0, 1)

    // Apply task-specific quality threshold penalty
    minQuality := getMinQuality(taskType)
    if qualityScore < minQuality {
        qualityScore *= 0.5 // Heavy penalty, not disqualified
    }

    // Calculate composite score
    score := weights.Energy*energyScore +
        weights.Cost*costScore +
        weights.Quality*qualityScore +
        weights.Latency*latencyScore

    // Risk penalty (subtract failure risk)
    riskPenalty := candidate.FailureRate / MaxRisk * 0.05
    score -= riskPenalty

    return score
}

func getMinQuality(taskType TaskType) float64 {
    switch taskType {
    case TaskSimple:
        return 0.60
    case TaskMedium:
        return 0.75
    case TaskHard:
        return 0.85
    case TaskSpecialized:
        return 0.90
    default:
        return 0.70
    }
}
```

**Scoring formula (text representation):**

```
route_score =
    0.40 × (1 - energy_kwh / max_energy)        # ENERGY (primary)
  + 0.30 × (1 - cost_usd / max_cost)            # COST
  + 0.20 × quality_benchmark                     # QUALITY
  + 0.10 × (1 - latency_ms / max_latency)       # LATENCY
  - 0.05 × (failure_rate / max_risk)             # RISK PENALTY
```

### 7.6 Model Selection

```go
var modelCandidates = map[TaskType][]string{
    TaskSimple:      {"phi_3", "mistral_7b"},
    TaskMedium:      {"mistral_7b", "llama_13b"},
    TaskHard:        {"llama_13b", "llama_70b"},
    TaskSpecialized: {"llama_70b"},
}

var fallbackChain = map[string]string{
    "phi_3":      "mistral_7b",
    "mistral_7b": "llama_13b",
    "llama_13b":  "llama_70b",
    "llama_70b":  "", // No fallback — return error
}

type RouteDecision struct {
    Model           string             `json:"model"`
    Fallback        string             `json:"fallback_model"`
    Score           float64            `json:"score"`
    TaskType        TaskType           `json:"task_type"`
    Complexity      int                `json:"complexity"`
    EstimatedEnergy float64            `json:"estimated_energy_kwh"`
    EstimatedCO2    float64            `json:"estimated_co2e_grams"`
    EstimatedCost   float64            `json:"estimated_cost_usd"`
    Confidence      float64            `json:"confidence"`
}

func SelectModel(classification ClassificationResult, customerConstraints *Constraints) RouteDecision {
    candidates := modelCandidates[classification.TaskType]
    
    bestModel := ""
    bestScore := -1.0
    
    for _, name := range candidates {
        candidate := getModelCandidate(name)
        score := ScoreModel(candidate, classification.TaskType, DefaultWeights)
        
        // Apply customer constraints
        if customerConstraints != nil {
            if customerConstraints.MaxLatencyMs > 0 && candidate.LatencyP95Ms > customerConstraints.MaxLatencyMs {
                continue // Skip this model
            }
            if customerConstraints.MinQuality > 0 && candidate.QualityBenchmark < customerConstraints.MinQuality {
                continue
            }
            if customerConstraints.MaxCostUSD > 0 && candidate.CostUSD > customerConstraints.MaxCostUSD {
                continue
            }
        }
        
        if score > bestScore {
            bestScore = score
            bestModel = name
        }
    }
    
    if bestModel == "" {
        // All candidates filtered out — use smallest available
        bestModel = candidates[0]
    }
    
    selected := getModelCandidate(bestModel)
    
    return RouteDecision{
        Model:           bestModel,
        Fallback:        fallbackChain[bestModel],
        Score:           bestScore,
        TaskType:        classification.TaskType,
        Complexity:      classification.Complexity,
        EstimatedEnergy: selected.EnergyKwh,
        EstimatedCO2:    energyToCO2(selected.EnergyKwh),
        EstimatedCost:   selected.CostUSD,
        Confidence:      classification.Confidence,
    }
}
```

### 7.7 Fallback Logic

```go
func InferWithFallback(ctx context.Context, decision RouteDecision, prompt string) (*InferenceResult, error) {
    // Try primary model
    result, err := inferenceGateway.Infer(ctx, decision.Model, prompt)
    
    if err == nil && result.FinishReason == "stop" {
        return result, nil
    }
    
    // Primary failed — try fallback
    if decision.Fallback != "" {
        log.Warn().
            Str("primary", decision.Model).
            Str("fallback", decision.Fallback).
            Err(err).
            Msg("primary model failed, trying fallback")
        
        result, err = inferenceGateway.Infer(ctx, decision.Fallback, prompt)
        if err == nil {
            result.UsedFallback = true
            return result, nil
        }
    }
    
    return nil, fmt.Errorf("all models failed: %w", err)
}
```

### 7.8 Response Metadata

Every response includes routing and environmental metadata:

```json
{
  "id": "eco-req-abc123",
  "object": "chat.completion",
  "model": "mistral-7b-q4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Here is your answer..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 45,
    "completion_tokens": 120,
    "total_tokens": 165
  },
  "ecollm": {
    "route": {
      "task_type": "medium",
      "complexity": 4,
      "model_selected": "mistral-7b-q4",
      "fallback_model": "llama-13b-q4",
      "routing_score": 0.87,
      "confidence": 0.85,
      "used_fallback": false
    },
    "energy": {
      "inference_energy_kwh": 0.000008,
      "total_energy_kwh": 0.0000104,
      "co2e_grams": 4.68,
      "grid_carbon_intensity": 450,
      "grid_region": "US-EAST"
    },
    "cost": {
      "inference_cost_usd": 0.00035,
      "total_cost_usd": 0.0005,
      "savings_vs_gpt4_percent": 85
    },
    "performance": {
      "latency_ms": 380,
      "time_to_first_token_ms": 95,
      "tokens_per_second": 316
    }
  }
}
```

---

## 8. Model Runtime Strategy

### 8.1 Runtime Comparison

| Runtime | Throughput | Memory Efficiency | Batching | Production Readiness | Best For |
|---------|-----------|-------------------|----------|---------------------|----------|
| **vLLM** | Excellent (PagedAttention) | Excellent | Continuous batching | Production-proven | Multi-model serving, high throughput |
| **llama.cpp** | Good (CPU + GPU) | Good (GGUF quants) | Limited | Stable | Edge/local, CPU inference |
| **Ollama** | Good (wraps llama.cpp) | Good | Limited | Stable | Developer experience, local dev |
| **TGI** | Very good | Good | Continuous batching | Production-proven | HuggingFace ecosystem |

### 8.2 Recommendation

**MVP:** vLLM for all models.

**Justification:**
- PagedAttention reduces GPU memory waste by 30–50% (directly reduces energy)
- Continuous batching maximizes GPU utilization (amortizes energy across requests)
- OpenAI-compatible API out of the box (no custom client needed)
- Supports GPTQ/AWQ quantization natively
- Active development, largest community

**Local development:** Ollama (simpler setup, developer-friendly). Swap to vLLM for staging/prod.

### 8.3 Model Configuration

```yaml
# model-configs/phi3-3.8b-q4.yaml
name: phi-3-3.8b
display_name: "Phi-3 Mini 3.8B"
runtime: vllm
model_id: "microsoft/Phi-3-mini-4k-instruct"
quantization: awq    # AWQ 4-bit
max_model_len: 4096
gpu_memory_utilization: 0.85
max_num_seqs: 64      # Max concurrent sequences
tensor_parallel_size: 1
dtype: float16
gpu_type: l4
gpu_count: 1
# Energy baselines (measured)
energy_per_token_kwh: 0.00000002
warmup_energy_kwh: 0.001
idle_power_watts: 15
inference_power_watts: 35
# Performance baselines
latency_p50_ms: 50
latency_p95_ms: 80
tokens_per_second: 450
# Quality baselines (from benchmark suite)
quality_benchmark: 0.65
quality_tasks:
  classification: 0.78
  summarization: 0.62
  code_generation: 0.55
  creative_writing: 0.48
  reasoning: 0.42

---
# model-configs/mistral-7b-q4.yaml
name: mistral-7b
display_name: "Mistral 7B Instruct"
runtime: vllm
model_id: "mistralai/Mistral-7B-Instruct-v0.3"
quantization: awq
max_model_len: 8192
gpu_memory_utilization: 0.90
max_num_seqs: 32
tensor_parallel_size: 1
dtype: float16
gpu_type: l4
gpu_count: 1
energy_per_token_kwh: 0.00000006
warmup_energy_kwh: 0.002
idle_power_watts: 20
inference_power_watts: 45
latency_p50_ms: 250
latency_p95_ms: 400
tokens_per_second: 180
quality_benchmark: 0.85
quality_tasks:
  classification: 0.90
  summarization: 0.85
  code_generation: 0.82
  creative_writing: 0.78
  reasoning: 0.72

---
# model-configs/llama-13b-q4.yaml
name: llama-13b
display_name: "Llama 3 13B"
runtime: vllm
model_id: "meta-llama/Llama-3-13B-Instruct"
quantization: awq
max_model_len: 8192
gpu_memory_utilization: 0.90
max_num_seqs: 16
tensor_parallel_size: 1
dtype: float16
gpu_type: l40s
gpu_count: 1
energy_per_token_kwh: 0.0000001
warmup_energy_kwh: 0.004
idle_power_watts: 25
inference_power_watts: 48
latency_p50_ms: 500
latency_p95_ms: 800
tokens_per_second: 95
quality_benchmark: 0.92
quality_tasks:
  classification: 0.94
  summarization: 0.91
  code_generation: 0.90
  creative_writing: 0.88
  reasoning: 0.85

---
# model-configs/llama-70b-q8.yaml
name: llama-70b
display_name: "Llama 3 70B"
runtime: vllm
model_id: "meta-llama/Llama-3-70B-Instruct"
quantization: gptq   # 8-bit for quality retention
max_model_len: 8192
gpu_memory_utilization: 0.95
max_num_seqs: 8
tensor_parallel_size: 4
dtype: float16
gpu_type: a100
gpu_count: 4
energy_per_token_kwh: 0.0000008
warmup_energy_kwh: 0.02
idle_power_watts: 200
inference_power_watts: 250
latency_p50_ms: 2000
latency_p95_ms: 3000
tokens_per_second: 35
quality_benchmark: 0.98
quality_tasks:
  classification: 0.97
  summarization: 0.96
  code_generation: 0.95
  creative_writing: 0.94
  reasoning: 0.93
```

### 8.4 Operational Strategy

**Cold-start prevention:**
- All models pre-loaded on startup. No lazy loading.
- Health checks every 30s to ensure models are warm.
- If a model crashes, auto-restart + alert.

**Model warm pools:**
- Phi-3 and Mistral 7B: always running (>95% of traffic)
- Llama 13B: always running (fallback)
- Llama 70B: scale-to-zero with 30s warm-up budget (used <5% of traffic)

**Batching strategy:**
- vLLM continuous batching handles this natively
- Max batch size per model configured in YAML
- Larger batches = better GPU utilization = lower energy per request

**Model versioning:**
- Each model config includes a `version` field
- New model versions deployed alongside old (blue-green)
- A/B testing: 10% traffic to new version, compare quality/latency/energy
- Benchmark results stored in `model_benchmarks` table

---

## 9. Database Schema

### 9.1 PostgreSQL Tables

```sql
-- ============================================================
-- USERS & ORGANIZATIONS
-- ============================================================

CREATE TABLE organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(100) UNIQUE NOT NULL,
    plan            VARCHAR(50) NOT NULL DEFAULT 'starter',  -- starter, growth, enterprise
    max_requests_per_min  INT NOT NULL DEFAULT 60,
    max_requests_per_day  INT NOT NULL DEFAULT 10000,
    quality_threshold     REAL NOT NULL DEFAULT 0.70,  -- min quality score for routing
    energy_budget_kwh     REAL,  -- optional daily energy cap
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(50) NOT NULL DEFAULT 'member',  -- admin, member, viewer
    name            VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_org_id ON users(org_id);
CREATE INDEX idx_users_email ON users(email);

CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by      UUID NOT NULL REFERENCES users(id),
    name            VARCHAR(255) NOT NULL,
    key_hash        VARCHAR(255) NOT NULL,  -- bcrypt hash of API key
    key_prefix      VARCHAR(10) NOT NULL,   -- First 8 chars for identification (e.g., "eco-sk-ab")
    scopes          TEXT[] NOT NULL DEFAULT ARRAY['inference'],  -- inference, admin, billing
    rate_limit_override  INT,               -- Override org-level rate limit
    last_used_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_api_keys_org_id ON api_keys(org_id);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);

-- ============================================================
-- REQUESTS & INFERENCE
-- ============================================================

CREATE TABLE requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    api_key_id      UUID REFERENCES api_keys(id),
    request_id      VARCHAR(100) UNIQUE NOT NULL,  -- External request ID (eco-req-xxx)
    
    -- Input
    prompt_original TEXT NOT NULL,
    prompt_optimized TEXT,
    messages_json   JSONB,             -- Full messages array for chat completions
    
    -- Routing
    task_type       VARCHAR(50) NOT NULL,  -- simple, medium, hard, specialized
    complexity      INT NOT NULL,
    model_selected  VARCHAR(100) NOT NULL,
    model_fallback  VARCHAR(100),
    routing_score   REAL NOT NULL,
    routing_confidence REAL NOT NULL,
    used_fallback   BOOLEAN NOT NULL DEFAULT false,
    cache_hit       BOOLEAN NOT NULL DEFAULT false,
    
    -- Response
    response_text   TEXT,
    finish_reason   VARCHAR(50),
    prompt_tokens   INT,
    completion_tokens INT,
    total_tokens    INT,
    
    -- Performance
    latency_ms      INT NOT NULL,
    time_to_first_token_ms INT,
    tokens_per_second REAL,
    
    -- Status
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, completed, failed, timeout
    error_message   TEXT,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_requests_org_id ON requests(org_id);
CREATE INDEX idx_requests_created_at ON requests(created_at);
CREATE INDEX idx_requests_model ON requests(model_selected);
CREATE INDEX idx_requests_status ON requests(status);
CREATE INDEX idx_requests_org_created ON requests(org_id, created_at DESC);

-- Partition by month for performance (requests table will grow fast)
-- CREATE TABLE requests_2026_05 PARTITION OF requests
--     FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE prompt_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id      UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    version         INT NOT NULL DEFAULT 1,
    prompt_text     TEXT NOT NULL,
    optimization_type VARCHAR(50),  -- original, rule_based, phi3_refined
    token_count     INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_prompt_versions_request ON prompt_versions(request_id);

-- ============================================================
-- MODEL REGISTRY
-- ============================================================

CREATE TABLE model_registry (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) UNIQUE NOT NULL,  -- phi_3, mistral_7b, llama_13b, llama_70b
    display_name    VARCHAR(255) NOT NULL,
    model_id        VARCHAR(255) NOT NULL,         -- HuggingFace model ID
    runtime         VARCHAR(50) NOT NULL,          -- vllm, tgi, ollama
    quantization    VARCHAR(20),                   -- awq, gptq, gguf, none
    max_context_len INT NOT NULL,
    
    -- Hardware
    gpu_type        VARCHAR(50) NOT NULL,
    gpu_count       INT NOT NULL DEFAULT 1,
    
    -- Performance baselines
    latency_p50_ms  INT NOT NULL,
    latency_p95_ms  INT NOT NULL,
    tokens_per_second REAL NOT NULL,
    
    -- Energy baselines
    energy_per_token_kwh REAL NOT NULL,
    idle_power_watts     REAL NOT NULL,
    inference_power_watts REAL NOT NULL,
    
    -- Quality baselines (JSON for task-specific scores)
    quality_benchmark REAL NOT NULL,
    quality_tasks     JSONB NOT NULL DEFAULT '{}',
    
    -- Cost
    cost_per_request_usd REAL NOT NULL,
    
    -- Status
    status          VARCHAR(20) NOT NULL DEFAULT 'active',  -- active, warming, draining, disabled
    endpoint_url    VARCHAR(500) NOT NULL,
    health_status   VARCHAR(20) NOT NULL DEFAULT 'unknown',  -- healthy, unhealthy, unknown
    last_health_check TIMESTAMPTZ,
    
    -- Versioning
    version         VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE model_routes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type       VARCHAR(50) NOT NULL,
    model_name      VARCHAR(100) NOT NULL REFERENCES model_registry(name),
    priority        INT NOT NULL DEFAULT 0,  -- Lower = higher priority
    is_fallback     BOOLEAN NOT NULL DEFAULT false,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_model_routes_task ON model_routes(task_type, priority);

-- ============================================================
-- ENERGY & CARBON
-- ============================================================

CREATE TABLE energy_measurements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id      UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    org_id          UUID NOT NULL REFERENCES organizations(id),
    model_name      VARCHAR(100) NOT NULL,
    
    -- GPU metrics
    gpu_power_watts REAL NOT NULL,
    inference_time_ms INT NOT NULL,
    batch_size      INT NOT NULL DEFAULT 1,
    
    -- Calculated energy
    inference_energy_wh REAL NOT NULL,
    pue_multiplier      REAL NOT NULL DEFAULT 1.3,  -- Power Usage Effectiveness
    total_energy_wh     REAL NOT NULL,
    total_energy_kwh    REAL NOT NULL,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_energy_request ON energy_measurements(request_id);
CREATE INDEX idx_energy_org ON energy_measurements(org_id, created_at);

CREATE TABLE carbon_estimates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id      UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    energy_measurement_id UUID REFERENCES energy_measurements(id),
    
    -- Grid data
    grid_region             VARCHAR(50) NOT NULL,
    grid_carbon_intensity   REAL NOT NULL,  -- gCO2/kWh
    carbon_data_source      VARCHAR(100),
    
    -- Calculated carbon
    co2e_grams              REAL NOT NULL,
    
    -- Comparison
    gpt4_equivalent_co2e   REAL,
    savings_percent        REAL,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_carbon_request ON carbon_estimates(request_id);

-- ============================================================
-- USAGE & BILLING
-- ============================================================

CREATE TABLE usage_aggregates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    granularity     VARCHAR(20) NOT NULL,  -- hourly, daily, monthly
    
    -- Counts
    total_requests      INT NOT NULL DEFAULT 0,
    successful_requests INT NOT NULL DEFAULT 0,
    failed_requests     INT NOT NULL DEFAULT 0,
    cache_hits          INT NOT NULL DEFAULT 0,
    fallback_used       INT NOT NULL DEFAULT 0,
    
    -- Tokens
    total_prompt_tokens     BIGINT NOT NULL DEFAULT 0,
    total_completion_tokens BIGINT NOT NULL DEFAULT 0,
    
    -- Model distribution
    model_distribution JSONB NOT NULL DEFAULT '{}',  -- {"phi_3": 500, "mistral_7b": 300}
    task_distribution  JSONB NOT NULL DEFAULT '{}',  -- {"simple": 400, "medium": 350}
    
    -- Performance
    avg_latency_ms      REAL,
    p95_latency_ms      REAL,
    
    -- Energy & Cost
    total_energy_kwh    REAL NOT NULL DEFAULT 0,
    total_co2e_grams    REAL NOT NULL DEFAULT 0,
    total_cost_usd      REAL NOT NULL DEFAULT 0,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    UNIQUE(org_id, period_start, granularity)
);
CREATE INDEX idx_usage_org_period ON usage_aggregates(org_id, period_start DESC);

CREATE TABLE billing_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    
    -- Breakdown
    total_requests      INT NOT NULL,
    total_tokens        BIGINT NOT NULL,
    total_energy_kwh    REAL NOT NULL,
    total_co2e_grams    REAL NOT NULL,
    
    -- Pricing
    subtotal_usd        REAL NOT NULL,
    discount_percent    REAL NOT NULL DEFAULT 0,
    total_usd           REAL NOT NULL,
    
    -- Status
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, invoiced, paid, overdue
    invoice_url         VARCHAR(500),
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_billing_org ON billing_events(org_id, period_start DESC);

-- ============================================================
-- AUDIT & FEEDBACK
-- ============================================================

CREATE TABLE rate_limits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    api_key_id      UUID REFERENCES api_keys(id),
    limit_type      VARCHAR(50) NOT NULL,  -- per_minute, per_day, per_month
    max_value       INT NOT NULL,
    current_value   INT NOT NULL DEFAULT 0,
    window_start    TIMESTAMPTZ NOT NULL,
    window_end      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID REFERENCES organizations(id),
    user_id         UUID REFERENCES users(id),
    action          VARCHAR(100) NOT NULL,  -- api_key.created, model.updated, user.login
    resource_type   VARCHAR(50),
    resource_id     UUID,
    details         JSONB,
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_org ON audit_logs(org_id, created_at DESC);
CREATE INDEX idx_audit_action ON audit_logs(action);

CREATE TABLE feedback_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id      UUID NOT NULL REFERENCES requests(id),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    rating          INT CHECK (rating >= 1 AND rating <= 5),
    feedback_type   VARCHAR(50),   -- quality, speed, relevance, wrong_model
    comment         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_feedback_request ON feedback_events(request_id);
```

### 9.2 What Goes in Redis (Not Postgres)

| Data | Redis Structure | TTL | Purpose |
|------|----------------|-----|---------|
| **Rate limit counters** | `rl:{api_key}` → INT | 60s (per-minute window) | Sliding window rate limiting |
| **API key validation cache** | `auth:{key_prefix}` → JSON | 5 min | Avoid DB lookup on every request |
| **Prompt response cache** | `cache:{sha256(prompt)}` → JSON | 1 hour | Skip inference for repeated prompts |
| **Model health status** | `health:{model_name}` → JSON | 30s | Fast health check without DB |
| **Org usage counters** | `usage:{org_id}:{date}` → INT | 24 hours | Real-time usage tracking |
| **Session tokens** | `session:{token}` → JSON | 24 hours | User session state |
| **Grid carbon intensity** | `grid:{region}` → JSON | 1 hour | Cached grid data (changes hourly) |
| **Request dedup** | `dedup:{request_hash}` → 1 | 5s | Prevent duplicate submissions |
| **Prompt optimization cache** | `opt:{sha256(prompt)}` → TEXT | 30 min | Cache optimized prompts |

**Key design pattern:** `{namespace}:{identifier}` — keeps keys organized and scannable.

**TTL strategy:** Aggressive TTLs (minutes, not hours) for most caches. Energy savings from cache hits > cost of occasional cache misses.

---

## 10. API Endpoints

### 10.1 Public API

#### POST `/v1/chat/completions`

**Purpose:** Primary inference endpoint. OpenAI-compatible format.

**Auth:** API key required (`Authorization: Bearer eco-sk-...`)

**Rate limit:** Org-level (default 60/min)

**Request body:**
```json
{
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Explain quicksort in Python."}
  ],
  "max_tokens": 512,
  "temperature": 0.7,
  "ecollm": {
    "prefer": "efficiency",
    "max_latency_ms": 2000,
    "min_quality": 0.75,
    "include_metadata": true
  }
}
```

**Response body:**
```json
{
  "id": "eco-req-abc123",
  "object": "chat.completion",
  "created": 1717200000,
  "model": "mistral-7b-q4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Here's a Python implementation of QuickSort..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 28,
    "completion_tokens": 150,
    "total_tokens": 178
  },
  "ecollm": {
    "route": {
      "task_type": "medium",
      "complexity": 5,
      "model_selected": "mistral-7b-q4",
      "routing_score": 0.87,
      "confidence": 0.85
    },
    "energy": {
      "total_energy_kwh": 0.0000104,
      "co2e_grams": 4.68,
      "grid_region": "US-EAST"
    },
    "cost": {
      "total_cost_usd": 0.0005
    },
    "performance": {
      "latency_ms": 380
    }
  }
}
```

**Validation rules:**
- `messages` required, at least 1 message
- Each message must have `role` (system/user/assistant) and `content` (non-empty string)
- `max_tokens` optional, default 512, max 4096
- `temperature` optional, default 0.7, range [0, 2]
- Total input tokens must not exceed model's max context

---

#### POST `/v1/completions`

**Purpose:** Legacy completions endpoint. Same routing logic, different format.

**Request body:**
```json
{
  "prompt": "Explain quicksort in Python.",
  "max_tokens": 512,
  "temperature": 0.7
}
```

**Response:** Same structure as chat completions but with `text` instead of `message`.

---

#### POST `/v1/route/preview`

**Purpose:** Preview routing decision without running inference. Useful for debugging and cost estimation.

**Auth:** API key required

**Request body:**
```json
{
  "messages": [
    {"role": "user", "content": "Debug this Python code..."}
  ]
}
```

**Response body:**
```json
{
  "route": {
    "task_type": "hard",
    "complexity": 7,
    "model_selected": "llama-13b-q4",
    "fallback_model": "llama-70b-q8",
    "routing_score": 0.82,
    "candidates": [
      {"model": "llama-13b-q4", "score": 0.82, "energy_kwh": 0.00015, "cost_usd": 0.001},
      {"model": "llama-70b-q8", "score": 0.61, "energy_kwh": 0.0008, "cost_usd": 0.005}
    ]
  },
  "estimated_energy_kwh": 0.00015,
  "estimated_co2e_grams": 6.75,
  "estimated_cost_usd": 0.001,
  "estimated_latency_ms": 800
}
```

---

#### GET `/v1/models`

**Purpose:** List available models and their capabilities.

**Auth:** API key required

**Response body:**
```json
{
  "models": [
    {
      "id": "phi-3-3.8b-q4",
      "name": "Phi-3 Mini 3.8B",
      "tasks": ["classification", "extraction", "simple_qa"],
      "max_context": 4096,
      "quality_benchmark": 0.65,
      "latency_p95_ms": 80,
      "energy_per_request_kwh": 0.00001,
      "status": "active"
    },
    {
      "id": "mistral-7b-q4",
      "name": "Mistral 7B Instruct",
      "tasks": ["writing", "coding", "summarization", "reasoning"],
      "max_context": 8192,
      "quality_benchmark": 0.85,
      "latency_p95_ms": 400,
      "energy_per_request_kwh": 0.00008,
      "status": "active"
    }
  ]
}
```

---

#### GET `/v1/usage`

**Purpose:** Get usage statistics for the authenticated organization.

**Auth:** API key required

**Query params:** `period=daily|monthly`, `from=2026-05-01`, `to=2026-05-31`

**Response body:**
```json
{
  "org_id": "org-abc123",
  "period": "daily",
  "from": "2026-05-01",
  "to": "2026-05-31",
  "summary": {
    "total_requests": 45230,
    "total_tokens": 8500000,
    "total_energy_kwh": 0.45,
    "total_co2e_grams": 202.5,
    "total_cost_usd": 22.50,
    "cache_hit_rate": 0.18,
    "avg_latency_ms": 320
  },
  "model_distribution": {
    "phi-3-3.8b": 27138,
    "mistral-7b": 13569,
    "llama-13b": 4070,
    "llama-70b": 453
  },
  "daily_breakdown": [
    {
      "date": "2026-05-01",
      "requests": 1500,
      "energy_kwh": 0.015,
      "co2e_grams": 6.75,
      "cost_usd": 0.75
    }
  ]
}
```

---

#### GET `/v1/requests/{id}`

**Purpose:** Get details of a specific request including routing and energy metadata.

**Auth:** API key required. Only returns requests from the authenticated org.

**Response:** Full request record as stored in the `requests` table, plus joined energy/carbon data.

---

### 10.2 Auth / Organization

#### POST `/auth/login`

**Request:** `{"email": "user@example.com", "password": "..."}`

**Response:** `{"token": "jwt-token-here", "user": {...}, "org": {...}}`

**Validation:** Email format, password min 8 chars.

---

#### POST `/auth/logout`

**Auth:** JWT required

**Effect:** Invalidates session token in Redis.

---

#### GET `/me`

**Auth:** JWT required

**Response:** Current user profile + org details.

---

#### POST `/api-keys`

**Auth:** JWT required, role = admin

**Request:** `{"name": "Production Key", "scopes": ["inference"], "expires_in_days": 90}`

**Response:** `{"id": "...", "key": "eco-sk-abc123...", "key_prefix": "eco-sk-ab"}` — key shown only once.

---

#### DELETE `/api-keys/{id}`

**Auth:** JWT required, role = admin

**Effect:** Revokes API key (sets `revoked_at`). Does not delete from DB (audit trail).

---

### 10.3 Admin / Dashboard

#### GET `/admin/metrics`

**Auth:** JWT required, role = admin

**Response:** System-wide metrics: total requests, avg latency, cache hit rate, model distribution, energy totals.

---

#### GET `/admin/models`

**Auth:** JWT required, role = admin

**Response:** Full model registry including health status, endpoints, benchmarks.

---

#### POST `/admin/models`

**Auth:** JWT required, role = admin

**Request:** Model configuration (name, endpoint, baselines, etc.)

**Validation:** Endpoint must respond to health check. Baselines must be populated.

---

#### PATCH `/admin/models/{id}`

**Auth:** JWT required, role = admin

**Request:** Partial update (status, endpoint, baselines).

**Use case:** Disable a model, update benchmarks, change endpoint.

---

#### GET `/admin/routes`

**Auth:** JWT required, role = admin

**Response:** Current routing configuration: which models serve which task types, priority, fallback chains.

---

#### GET `/admin/carbon`

**Auth:** JWT required, role = admin

**Response:** Aggregated carbon metrics: total CO2e saved, grid region breakdown, model-level energy, comparison vs GPT-4 baseline.

---

## 11. Data Flow

### 11.1 Full Request Lifecycle

1. **Client sends request** → HTTPS POST to `/v1/chat/completions` with API key
2. **API Gateway receives** → Extracts API key, generates request ID, starts OTel span
3. **Auth middleware** → Validates API key against Redis cache (or DB fallback), loads org config
4. **Rate limiter** → Checks Redis counter `rl:{api_key}`, rejects if over limit
5. **Request validation** → Validates body schema (messages, max_tokens, temperature)
6. **Request logged** → Inserts row into `requests` table (status: pending)
7. **Prompt optimizer** → Rule-based rewriting in Go; Phi-3 sidecar if confidence < 0.7
8. **Cache check** → SHA-256 hash of normalized prompt → Redis lookup; if hit, return cached response
9. **Task classifier** → Keyword + heuristic analysis → task_type + complexity score
10. **Model scorer** → Score all candidate models (energy 40%, cost 30%, quality 20%, latency 10%)
11. **Constraint enforcement** → Apply org quality threshold, latency SLA, energy budget
12. **Energy estimator** → Predict energy_kwh and co2e for selected model
13. **Inference gateway** → HTTP POST to vLLM `/v1/completions` endpoint for selected model
14. **Response received** → Parse vLLM response, extract tokens, finish_reason
15. **Fallback check** → If inference failed or quality low, retry with fallback model
16. **Energy measurement** → Calculate actual energy from GPU telemetry + inference time
17. **Carbon calculation** → Multiply energy by grid carbon intensity
18. **Response assembly** → Build OpenAI-compatible response + EcoLLM metadata
19. **Request updated** → Update `requests` row (status: completed, latency, tokens, model, etc.)
20. **Energy record** → Insert into `energy_measurements` + `carbon_estimates`
21. **Cache write** → Store response in Redis for future cache hits
22. **Usage counter** → Increment Redis counters `usage:{org_id}:{date}`
23. **Response returned** → JSON response to client with full metadata
24. **Background** → Hourly worker aggregates usage into `usage_aggregates`

### 11.2 Sequence Diagram

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as API Gateway
    participant Auth as Auth MW
    participant RL as Rate Limiter
    participant PO as Prompt Optimizer
    participant Cache as Redis Cache
    participant TC as Task Classifier
    participant MR as Model Router
    participant CE as Carbon Estimator
    participant IG as Inference Gateway
    participant RT as vLLM Runtime
    participant DB as PostgreSQL
    
    C->>GW: POST /v1/chat/completions
    GW->>Auth: Validate API key
    Auth->>Cache: Check auth cache
    Cache-->>Auth: Org config
    Auth-->>GW: Authorized
    
    GW->>RL: Check rate limit
    RL->>Cache: INCR rl:{key}
    Cache-->>RL: count=42/60
    RL-->>GW: Allowed
    
    GW->>DB: Log request (status: pending)
    GW->>PO: Optimize prompt
    PO-->>GW: Optimized prompt
    
    GW->>Cache: Check prompt cache
    
    alt Cache Hit
        Cache-->>GW: Cached response
        GW->>DB: Update request (cache_hit: true)
        GW-->>C: Return cached response
    else Cache Miss
        GW->>TC: Classify task
        TC-->>GW: task_type=medium, complexity=5
        
        GW->>MR: Score models
        MR->>CE: Get energy estimates
        CE-->>MR: Energy per model
        MR-->>GW: RouteDecision (mistral-7b)
        
        GW->>IG: Infer(mistral-7b, prompt)
        IG->>RT: POST /v1/completions
        RT-->>IG: Generated text
        IG-->>GW: InferenceResult
        
        GW->>CE: Calculate actual CO2e
        CE-->>GW: co2e=4.68g
        
        GW->>DB: Update request (completed)
        GW->>DB: Insert energy_measurement
        GW->>DB: Insert carbon_estimate
        GW->>Cache: Cache response
        GW->>Cache: INCR usage counter
        
        GW-->>C: Response + metadata
    end
```

---

## 12. Caching Strategy

### 12.1 Cache Layers

| Layer | Key Pattern | Value | TTL | Purpose |
|-------|------------|-------|-----|---------|
| **Prompt response cache** | `cache:{sha256(norm_prompt+model)}` | Full JSON response | 1 hour | Eliminate redundant inference (biggest energy saver) |
| **Prompt optimization cache** | `opt:{sha256(raw_prompt)}` | Optimized prompt text | 30 min | Skip re-optimization for same prompt |
| **API key auth cache** | `auth:{key_prefix}` | `{org_id, scopes, rate_limit}` | 5 min | Avoid DB query per request |
| **Model metadata cache** | `model:{name}` | Model config JSON | 10 min | Fast model lookup without DB |
| **Org config cache** | `org:{org_id}` | Org settings JSON | 5 min | Quality thresholds, rate limits |
| **Rate limit counter** | `rl:{api_key}:{window}` | INT counter | Window size (60s) | Sliding window rate limiting |
| **Request dedup** | `dedup:{sha256(body)}` | `1` | 5s | Prevent double-submit |
| **Grid carbon cache** | `grid:{region}` | Carbon intensity JSON | 1 hour | Grid data changes hourly |
| **Org usage counter** | `usage:{org_id}:{date}` | INT counter | 48 hours | Real-time usage display |
| **Model health** | `health:{model_name}` | `{status, last_check}` | 30s | Quick health check |

### 12.2 Cache Key Design

```
Namespace:Identifier[:Qualifier]

Examples:
  cache:a1b2c3d4e5f6...        (prompt response)
  auth:eco-sk-ab                (API key prefix)
  rl:eco-sk-ab:1717200000       (rate limit window)
  usage:org-abc:2026-05-02      (daily usage)
  grid:US-EAST                  (carbon intensity)
```

### 12.3 Semantic Cache (Deferred to Phase 4+)

**Not in MVP.** Semantic caching uses embeddings to find "similar enough" prompts. It adds:
- An embedding model (more compute, more energy)
- A vector store (pgvector or similar)
- Quality uncertainty (similar ≠ same)

**Justification for deferral:** Exact-match caching (SHA-256) captures 15–20% of requests at zero additional compute. Semantic caching may add 5–10% more hits but at the cost of running an embedding model per request — violating the efficiency constraint.

**When to add it:** When exact-match cache hit rate plateaus AND customers request it.

### 12.4 Invalidation Rules

- **Prompt cache:** Invalidated when model version changes (new model deployment)
- **Auth cache:** Invalidated on API key revocation or org config change
- **Model cache:** Invalidated on model registry update (admin action)
- **All caches:** Natural TTL expiration (no manual invalidation needed for most cases)

---

## 13. Frontend / UI Architecture

### 13.1 App Router Structure

```
apps/web/src/app/
├── (auth)/                           # Auth layout (no sidebar)
│   ├── login/
│   │   └── page.tsx
│   ├── register/
│   │   └── page.tsx
│   └── layout.tsx
│
├── (dashboard)/                      # Dashboard layout (sidebar + header)
│   ├── overview/
│   │   └── page.tsx                  # Summary: requests, cost, CO2, latency
│   ├── requests/
│   │   ├── page.tsx                  # Request log table with filters
│   │   └── [id]/
│   │       └── page.tsx              # Single request detail (routing trace)
│   ├── models/
│   │   └── page.tsx                  # Model performance comparison
│   ├── carbon/
│   │   └── page.tsx                  # Carbon dashboard: CO2 saved, grid data
│   ├── usage/
│   │   └── page.tsx                  # Usage analytics: by model, by task, over time
│   ├── billing/
│   │   └── page.tsx                  # Billing summary, invoices
│   ├── api-keys/
│   │   └── page.tsx                  # API key management (create, revoke, list)
│   ├── playground/
│   │   └── page.tsx                  # Interactive prompt testing + routing preview
│   ├── settings/
│   │   └── page.tsx                  # Org settings, quality thresholds, energy budget
│   └── layout.tsx                    # Sidebar + header layout
│
├── (admin)/                          # Admin layout (super-user only)
│   ├── models/
│   │   └── page.tsx                  # Model registry CRUD
│   ├── routes/
│   │   └── page.tsx                  # Routing config management
│   ├── metrics/
│   │   └── page.tsx                  # System-wide metrics
│   └── layout.tsx
│
├── layout.tsx                        # Root layout (providers, fonts, global styles)
├── page.tsx                          # Landing / redirect to dashboard
├── not-found.tsx
└── error.tsx
```

### 13.2 Frontend Folder Structure

```
apps/web/src/
├── app/                              # Pages (see above)
├── components/
│   ├── ui/                           # shadcn/ui primitives
│   │   ├── button.tsx
│   │   ├── input.tsx
│   │   ├── textarea.tsx
│   │   ├── select.tsx
│   │   ├── dialog.tsx
│   │   ├── toast.tsx
│   │   ├── tabs.tsx
│   │   ├── table.tsx
│   │   ├── badge.tsx
│   │   ├── card.tsx
│   │   ├── skeleton.tsx
│   │   ├── tooltip.tsx
│   │   └── dropdown-menu.tsx
│   │
│   ├── dashboard/                    # Dashboard-specific
│   │   ├── metric-card.tsx           # Single metric display (value + trend + icon)
│   │   ├── chart-card.tsx            # Chart wrapper with title + period selector
│   │   ├── usage-chart.tsx           # Line/bar chart for usage over time
│   │   ├── model-distribution.tsx    # Pie/donut chart of model usage
│   │   ├── request-log-table.tsx     # Paginated request log with filters
│   │   ├── route-trace-timeline.tsx  # Visual timeline of routing steps
│   │   └── overview-grid.tsx         # Grid of metric cards
│   │
│   ├── carbon/                       # Carbon tracking
│   │   ├── carbon-badge.tsx          # Inline CO2e display (with color coding)
│   │   ├── carbon-chart.tsx          # CO2e over time
│   │   ├── carbon-comparison.tsx     # "vs GPT-4" comparison visual
│   │   ├── impact-summary.tsx        # "Equivalent to X trees planted"
│   │   └── grid-intensity-map.tsx    # Regional grid carbon intensity
│   │
│   ├── models/                       # Model management
│   │   ├── model-badge.tsx           # Model name with size/quant badge
│   │   ├── model-card.tsx            # Model overview card
│   │   ├── model-comparison.tsx      # Side-by-side model comparison table
│   │   └── model-health.tsx          # Health status indicator
│   │
│   ├── playground/                   # Playground
│   │   ├── prompt-input.tsx          # Textarea with send button
│   │   ├── response-panel.tsx        # Response display with metadata
│   │   ├── routing-preview.tsx       # Show routing decision before inference
│   │   ├── model-selector.tsx        # Override model selection
│   │   └── metadata-panel.tsx        # Energy, cost, latency display
│   │
│   ├── shared/                       # Layout & navigation
│   │   ├── sidebar.tsx               # Dashboard sidebar navigation
│   │   ├── header.tsx                # Top bar with user menu
│   │   ├── footer.tsx
│   │   ├── page-header.tsx           # Page title + breadcrumbs
│   │   ├── empty-state.tsx           # "No data yet" with CTA
│   │   ├── error-state.tsx           # Error display with retry
│   │   ├── loading-skeleton.tsx      # Skeleton loaders per section
│   │   ├── latency-badge.tsx         # Latency display with color coding
│   │   ├── cost-badge.tsx            # Cost display
│   │   └── api-key-card.tsx          # API key display with copy/revoke
│   │
│   └── providers/
│       ├── query-provider.tsx        # TanStack Query provider
│       ├── theme-provider.tsx        # Dark/light mode
│       └── toast-provider.tsx        # Toast notifications
│
├── lib/
│   ├── api.ts                        # API client (fetch wrapper with auth)
│   ├── hooks/
│   │   ├── use-requests.ts           # Fetch request logs
│   │   ├── use-usage.ts              # Fetch usage data
│   │   ├── use-models.ts             # Fetch model registry
│   │   ├── use-carbon.ts             # Fetch carbon data
│   │   ├── use-api-keys.ts           # CRUD API keys
│   │   └── use-auth.ts               # Login, logout, session
│   ├── utils.ts                      # formatCO2, formatCost, formatLatency
│   └── constants.ts                  # API base URL, default config
│
├── stores/
│   └── playground.ts                 # Zustand: playground state (prompt, model override, history)
│
├── types/
│   ├── api.ts                        # API request/response types
│   ├── models.ts                     # Model types
│   └── carbon.ts                     # Carbon/energy types
│
└── styles/
    └── globals.css                    # Tailwind base + custom CSS vars
```

### 13.3 Key Pages

**Overview Dashboard:**
- 4 metric cards: Total Requests (today), Avg Latency, Total CO2 Saved, Total Cost
- Usage chart (requests over last 7/30 days)
- Model distribution pie chart
- Recent requests table (last 10)

**Carbon Dashboard:**
- CO2e saved this month (large hero number)
- "Equivalent to..." translation (trees, cars, flights)
- CO2e over time chart
- Grid carbon intensity display
- Model-level energy breakdown
- Comparison vs GPT-4 baseline

**Playground:**
- Left panel: Prompt input textarea + model override selector + send button
- Right panel: Response text + metadata panel (routing decision, energy, cost, latency)
- Route preview button (shows routing decision without running inference)

**Request Detail Page:**
- Full request/response text
- Route trace timeline (visual steps: optimize → classify → route → infer)
- Energy breakdown
- Carbon calculation
- Model scoring details (all candidates with scores)

---

## 14. Reusable UI Components

### 14.1 Component Specifications

| Component | Purpose | Key Props | Accessibility | States |
|-----------|---------|-----------|---------------|--------|
| **Button** | Primary action trigger | `variant` (primary/secondary/ghost/destructive), `size`, `loading`, `disabled` | Focus ring, aria-label for icon-only, aria-busy when loading | default, hover, focus, active, disabled, loading |
| **Input** | Text input field | `type`, `label`, `error`, `hint`, `prefix`, `suffix` | Linked label via `htmlFor`, aria-describedby for errors, aria-invalid | default, focus, error, disabled |
| **Textarea** | Multi-line text input | `label`, `error`, `maxLength`, `rows` | Same as Input | default, focus, error, disabled |
| **Select** | Dropdown selection | `options`, `value`, `label`, `error`, `placeholder` | Keyboard navigation, aria-expanded, role=listbox | default, open, focus, disabled |
| **Modal** | Dialog overlay | `open`, `onClose`, `title`, `description` | Focus trap, Escape to close, aria-modal, return focus on close | open, closing animation |
| **Toast** | Notification message | `variant` (success/error/warning/info), `message`, `duration` | role=status for info, role=alert for errors, aria-live | entering, visible, exiting |
| **Tabs** | Content tab navigation | `tabs[]`, `activeTab`, `onChange` | role=tablist, aria-selected, keyboard arrows | default, active, disabled |
| **DataTable** | Sortable/paginated table | `columns`, `data`, `pagination`, `onSort`, `onFilter` | scope=col headers, aria-sort, keyboard navigation, caption | loading, empty, error, populated |
| **MetricCard** | Single KPI display | `label`, `value`, `trend` (up/down/flat), `icon`, `unit` | Descriptive aria-label combining label+value+trend | loading (skeleton), populated |
| **ChartCard** | Chart with controls | `title`, `period`, `onPeriodChange`, `children` | aria-label on chart, data table fallback for screen readers | loading, populated, error |
| **RequestLogTable** | Request log viewer | `requests[]`, `onRowClick`, `filters` | Clickable rows as links (not buttons), filter form labels | loading, empty, populated, filtering |
| **ModelBadge** | Model name + size tag | `name`, `size`, `quantization` | aria-label="Mistral 7B quantized 4-bit" | default |
| **CarbonBadge** | CO2e value display | `co2eGrams`, `showComparison` | aria-label with full context | green (<5g), yellow (5-20g), red (>20g) |
| **LatencyBadge** | Latency value display | `latencyMs` | aria-label="380 milliseconds" | green (<500ms), yellow (500-1500ms), red (>1500ms) |
| **CostBadge** | Cost value display | `costUsd` | aria-label="0.05 cents" | green (<$0.001), yellow, red |
| **RouteTraceTimeline** | Visual routing steps | `steps[]` with timing | Semantic list, each step as `<li>`, time labels | animated, static |
| **ApiKeyCard** | API key display | `key`, `name`, `lastUsed`, `onRevoke` | Copy button aria-label, confirm dialog for revoke | active, revoking, revoked |
| **UsageChart** | Time-series usage chart | `data`, `metric` (requests/energy/cost), `period` | SVG with `<title>`, data table fallback | loading, populated, empty |
| **EmptyState** | No data placeholder | `title`, `description`, `action` (CTA button) | Descriptive text, actionable CTA | default |
| **ErrorState** | Error display | `error`, `onRetry` | role=alert, retry button focus | default, retrying |
| **LoadingSkeleton** | Loading placeholder | `variant` (card/table/chart/text), `lines` | aria-busy=true, aria-label="Loading" | pulsing |

### 14.2 Testing Considerations

Every component should have:
1. **Render test:** Renders without crashing with required props
2. **Props test:** Renders correctly with all prop variants
3. **Interaction test:** Click, keyboard, focus events work
4. **Accessibility test:** Passes axe-core automated checks
5. **Snapshot test:** Visual regression detection (optional)

---

## 15. Accessibility Requirements

### 15.1 WCAG 2.1 AA Compliance

| Requirement | Implementation |
|-------------|----------------|
| **Semantic HTML** | Use `<main>`, `<nav>`, `<section>`, `<article>`, `<header>`, `<footer>`. Tables use `<th scope="col">`. Forms use `<label>` linked to inputs. |
| **Keyboard navigation** | All interactive elements reachable via Tab. Modals trap focus. Escape closes overlays. Arrow keys navigate tabs, menus, selects. |
| **Focus states** | Visible focus ring on all interactive elements (2px solid, high contrast). Never `outline: none` without replacement. |
| **ARIA usage** | `aria-label` for icon-only buttons. `aria-expanded` for dropdowns. `aria-live="polite"` for dynamic content. `aria-busy` for loading states. `role="alert"` for errors. |
| **Color contrast** | Minimum 4.5:1 for normal text, 3:1 for large text. Badge colors (green/yellow/red) also have text labels, not color-only. |
| **Reduced motion** | Respect `prefers-reduced-motion`. Disable animations, transitions, and auto-scrolling. |
| **Screen reader labels** | Charts include `<title>` + `<desc>` in SVGs. Data tables have `<caption>`. Complex components have `aria-label` with full context. |
| **Form validation** | Inline error messages linked via `aria-describedby`. Invalid fields use `aria-invalid="true"`. Error summary at top of form for multi-field errors. |
| **Table accessibility** | `<caption>` for table description. `<th scope="col">` for headers. Sortable columns announce sort direction via `aria-sort`. |
| **Chart accessibility** | Every chart has a hidden data table equivalent (`<table>` with `sr-only` class). Charts use patterns (not just colors) for categories. |

### 15.2 Testing

- **Automated:** axe-core in component tests + Playwright
- **Manual:** Keyboard-only navigation test per page, screen reader testing (VoiceOver + NVDA)
- **CI:** axe accessibility checks in Playwright E2E suite

---

## 16. State Management

### 16.1 Server State (TanStack Query)

TanStack Query manages all server-fetched data: requests, usage, models, carbon, billing. This is the primary state management approach — 90%+ of state is server state.

```typescript
// lib/hooks/use-requests.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';

export function useRequests(filters: RequestFilters) {
  return useQuery({
    queryKey: ['requests', filters],
    queryFn: () => api.get('/v1/requests', { params: filters }),
    staleTime: 30 * 1000,       // 30 seconds (requests update frequently)
    refetchInterval: 60 * 1000, // Auto-refresh every minute
  });
}

export function useRequest(id: string) {
  return useQuery({
    queryKey: ['requests', id],
    queryFn: () => api.get(`/v1/requests/${id}`),
    staleTime: 5 * 60 * 1000, // 5 minutes (individual requests don't change)
  });
}

export function useUsage(period: string) {
  return useQuery({
    queryKey: ['usage', period],
    queryFn: () => api.get('/v1/usage', { params: { period } }),
    staleTime: 60 * 1000,
  });
}
```

### 16.2 Form State (React Hook Form + Zod)

```typescript
// Playground form example
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const playgroundSchema = z.object({
  prompt: z.string().min(1, 'Prompt required').max(10000),
  maxTokens: z.number().min(1).max(4096).default(512),
  temperature: z.number().min(0).max(2).default(0.7),
  modelOverride: z.string().optional(),
});

type PlaygroundForm = z.infer<typeof playgroundSchema>;

function PlaygroundPage() {
  const { register, handleSubmit, formState: { errors } } = useForm<PlaygroundForm>({
    resolver: zodResolver(playgroundSchema),
  });
  // ...
}
```

### 16.3 Client State (Zustand — Minimal)

Only used for playground state that persists across navigation but isn't server data:

```typescript
// stores/playground.ts
import { create } from 'zustand';

interface PlaygroundStore {
  history: Array<{ prompt: string; response: string; metadata: any }>;
  addToHistory: (entry: any) => void;
  clearHistory: () => void;
}

export const usePlaygroundStore = create<PlaygroundStore>((set) => ({
  history: [],
  addToHistory: (entry) => set((s) => ({ history: [...s.history.slice(-49), entry] })),
  clearHistory: () => set({ history: [] }),
}));
```

### 16.4 API Client

```typescript
// lib/api.ts
const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

class ApiClient {
  private token: string | null = null;

  setToken(token: string) { this.token = token; }
  clearToken() { this.token = null; }

  async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const res = await fetch(`${BASE_URL}${path}`, { ...options, headers });

    if (!res.ok) {
      const error = await res.json().catch(() => ({ message: 'Request failed' }));
      throw new ApiError(res.status, error.message, error.type);
    }

    return res.json();
  }

  get<T>(path: string, opts?: { params?: Record<string, any> }) {
    const url = opts?.params ? `${path}?${new URLSearchParams(opts.params)}` : path;
    return this.request<T>(url);
  }

  post<T>(path: string, body: any) {
    return this.request<T>(path, { method: 'POST', body: JSON.stringify(body) });
  }

  patch<T>(path: string, body: any) {
    return this.request<T>(path, { method: 'PATCH', body: JSON.stringify(body) });
  }

  delete<T>(path: string) {
    return this.request<T>(path, { method: 'DELETE' });
  }
}

export const api = new ApiClient();
```

---

## 17. Security Architecture

### 17.1 Authentication & Authorization

| Mechanism | Implementation | Details |
|-----------|----------------|---------|
| **API Key Auth** | `Authorization: Bearer eco-sk-...` | For API access. Key hashed with bcrypt before storage. Only prefix stored in plaintext for identification. |
| **JWT Sessions** | HTTP-only cookie + Redis session | For dashboard. Short-lived JWT (15 min) + refresh token (7 days). Stored in Redis for revocation. |
| **RBAC** | `admin`, `member`, `viewer` roles | Admin: full access. Member: inference + read. Viewer: read-only dashboard. |
| **Organization Isolation** | Every query filtered by `org_id` | Users can only access their own org's data. Enforced at repository layer, not just middleware. |

### 17.2 API Key Security

```go
// Generating API keys
func GenerateAPIKey() (plaintext string, hash string, prefix string) {
    // Generate 32 random bytes
    raw := make([]byte, 32)
    crypto_rand.Read(raw)
    
    // Encode as base62 with prefix
    plaintext = "eco-sk-" + base62.Encode(raw)
    
    // Hash for storage
    hashBytes, _ := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
    hash = string(hashBytes)
    
    // Store prefix for identification (first 10 chars)
    prefix = plaintext[:10]
    
    return plaintext, hash, prefix
}
```

### 17.3 Input Validation & Safety

| Threat | Mitigation |
|--------|-----------|
| **Prompt injection** | Content policy check before routing. Blocklist of known injection patterns. Log suspicious prompts for review. Do NOT attempt to sanitize — classify and reject. |
| **Request body size** | 1MB max body limit (Fiber config). Reject oversized requests before parsing. |
| **Token limits** | Enforce max_tokens (4096 ceiling). Count input tokens before inference. Reject if input + max_output > model context. |
| **SQL injection** | Parameterized queries only (pgx). No string concatenation in SQL. |
| **Rate limiting** | Redis sliding window per API key. Org-level daily caps. IP-level emergency throttle. |
| **CORS** | Strict origin whitelist. No wildcard `*` in production. |
| **Secrets** | Environment variables only. Never in code, config files, or logs. Use secrets manager in production (Vault or cloud-native). |
| **Audit logging** | All admin actions logged to `audit_logs` table with user ID, IP, timestamp. |
| **Model output safety** | Post-inference content filter (blocklist-based). Log and flag harmful outputs. |
| **Abuse prevention** | Monitor for: unusually high request rates, prompt patterns suggesting scraping, API key sharing (multiple IPs). Alert + auto-throttle. |

---

## 18. Observability

### 18.1 Structured Logging

All services use **zerolog** (Go) with JSON output. Every log entry includes:

```json
{
  "level": "info",
  "time": "2026-05-02T14:32:15.123Z",
  "request_id": "eco-req-abc123",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "org_id": "org-def456",
  "service": "api-gateway",
  "msg": "inference completed",
  "model": "mistral-7b-q4",
  "task_type": "medium",
  "latency_ms": 380,
  "energy_kwh": 0.0000104,
  "co2e_grams": 4.68,
  "cost_usd": 0.0005,
  "cache_hit": false,
  "tokens": 178
}
```

### 18.2 Prometheus Metrics

```go
// internal/telemetry/metrics.go

var (
    // Request metrics
    RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "ecollm_requests_total",
        Help: "Total requests by model, task_type, status",
    }, []string{"model", "task_type", "status"})

    RequestLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "ecollm_request_latency_seconds",
        Help:    "Request latency distribution",
        Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
    }, []string{"model", "task_type"})

    // Energy metrics
    EnergyPerRequest = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "ecollm_energy_kwh_per_request",
        Help:    "Energy consumption per request in kWh",
        Buckets: []float64{0.000001, 0.00001, 0.0001, 0.001},
    }, []string{"model"})

    CO2ePerRequest = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "ecollm_co2e_grams_per_request",
        Help:    "CO2 equivalent emissions per request in grams",
        Buckets: []float64{0.1, 0.5, 1, 5, 10, 50},
    }, []string{"model"})

    // Cache metrics
    CacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "ecollm_cache_hits_total",
        Help: "Cache hits by cache type",
    }, []string{"cache_type"})

    CacheMisses = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "ecollm_cache_misses_total",
        Help: "Cache misses by cache type",
    }, []string{"cache_type"})

    // Routing metrics
    RoutingDecisions = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "ecollm_routing_decisions_total",
        Help: "Routing decisions by model and task type",
    }, []string{"model", "task_type"})

    RoutingScore = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "ecollm_routing_score",
        Help:    "Routing score distribution",
        Buckets: prometheus.LinearBuckets(0, 0.1, 11),
    }, []string{"model"})

    FallbacksUsed = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "ecollm_fallbacks_used_total",
        Help: "Fallbacks triggered by primary model",
    }, []string{"primary_model", "fallback_model"})

    // Cost metrics
    CostPerRequest = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "ecollm_cost_usd_per_request",
        Help:    "Cost per request in USD",
        Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01},
    }, []string{"model", "org_id"})

    // Model health
    ModelHealthStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "ecollm_model_health_status",
        Help: "Model health (1=healthy, 0=unhealthy)",
    }, []string{"model"})

    // Inference metrics
    TokensPerSecond = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "ecollm_tokens_per_second",
        Help: "Current tokens per second throughput",
    }, []string{"model"})
)
```

### 18.3 Grafana Dashboards

| Dashboard | Panels | Purpose |
|-----------|--------|---------|
| **Overview** | Request rate, error rate, p95 latency, total CO2e, model distribution | Quick health check |
| **Routing** | Model selection frequency, routing scores, fallback rate, task type distribution | Routing algorithm performance |
| **Energy** | Energy per request (by model), CO2e over time, grid intensity, cumulative savings vs GPT-4 | Environmental tracking |
| **Latency** | p50/p95/p99 by model, time-to-first-token, inference time breakdown | Performance monitoring |
| **Cost** | Cost per request (by model), cost per org, daily/monthly totals, margin analysis | Financial monitoring |
| **Cache** | Hit rate, hit/miss by cache type, cache size, TTL effectiveness | Cache optimization |

### 18.4 OpenTelemetry Spans

Each request produces a trace with these spans:

```
[http.request] ─── 400ms total
  ├── [auth.validate] ─── 0.5ms
  ├── [ratelimit.check] ─── 0.3ms
  ├── [prompt.optimize] ─── 2ms
  ├── [cache.check] ─── 0.8ms
  ├── [task.classify] ─── 0.2ms
  ├── [router.score] ─── 0.3ms
  ├── [carbon.estimate] ─── 0.2ms
  ├── [inference.dispatch] ─── 380ms
  │     └── [vllm.generate] ─── 375ms
  ├── [response.assemble] ─── 0.3ms
  └── [telemetry.record] ─── 1ms
```

---

## 19. Testing Strategy

### 19.1 Go Backend Tests

| Test Type | Scope | Tool | Example |
|-----------|-------|------|---------|
| **Unit tests** | Single function/method | `go test` | Scoring function returns correct scores for known inputs |
| **Integration tests** | Service + DB/Redis | `go test` + testcontainers | Chat service creates request, routes, stores result |
| **API contract tests** | HTTP endpoints | `go test` + httptest | POST /v1/chat/completions returns OpenAI-compatible response |
| **Routing evaluation** | Classifier + scorer accuracy | Custom test harness | 100 labeled prompts → verify correct model selection >90% |

```go
// Example: scoring function unit test
func TestScoreModel_PrefersSmallModel(t *testing.T) {
    phi3 := ModelCandidate{
        Name: "phi_3", EnergyKwh: 0.00001, CostUSD: 0.0001,
        QualityBenchmark: 0.65, LatencyP95Ms: 80, FailureRate: 0.01,
    }
    mistral := ModelCandidate{
        Name: "mistral_7b", EnergyKwh: 0.00008, CostUSD: 0.0005,
        QualityBenchmark: 0.85, LatencyP95Ms: 400, FailureRate: 0.02,
    }
    
    phi3Score := ScoreModel(phi3, TaskSimple, DefaultWeights)
    mistralScore := ScoreModel(mistral, TaskSimple, DefaultWeights)
    
    // For simple tasks, Phi-3 should score higher (lower energy wins)
    assert.Greater(t, phi3Score, mistralScore,
        "Phi-3 should score higher than Mistral for simple tasks")
}
```

### 19.2 Frontend Tests

| Test Type | Tool | Scope |
|-----------|------|-------|
| **Component tests** | Vitest + React Testing Library | Individual component rendering, interaction, accessibility |
| **E2E tests** | Playwright | Full user flows: login → send prompt → view result → check carbon |
| **Accessibility tests** | axe-core (Playwright plugin) | Automated WCAG checks on every page |

### 19.3 Load Testing

```javascript
// tests/load/k6/load_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '1m', target: 50 },   // Ramp up
    { duration: '5m', target: 50 },   // Sustain
    { duration: '1m', target: 200 },  // Spike
    { duration: '2m', target: 200 },  // Sustain spike
    { duration: '1m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'],  // p95 < 1s
    http_req_failed: ['rate<0.01'],     // <1% error rate
  },
};

export default function () {
  const res = http.post(
    'http://localhost:8080/v1/chat/completions',
    JSON.stringify({
      messages: [{ role: 'user', content: 'What is quicksort?' }],
      max_tokens: 256,
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer eco-sk-test-key',
      },
    }
  );

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has ecollm metadata': (r) => JSON.parse(r.body).ecollm !== undefined,
    'latency < 1s': (r) => r.timings.duration < 1000,
  });

  sleep(1);
}
```

### 19.4 Routing Evaluation

Maintain a labeled test set of 200+ prompts with expected model selections:

```json
[
  {"prompt": "What is the capital of France?", "expected_task": "simple", "expected_model": "phi_3"},
  {"prompt": "Write a Python function to merge two sorted arrays", "expected_task": "medium", "expected_model": "mistral_7b"},
  {"prompt": "Design a distributed caching system for a social network with 100M users", "expected_task": "hard", "expected_model": "llama_13b"}
]
```

Run weekly. Track routing accuracy over time. Alert if accuracy drops below 85%.

---

## 20. Deployment Architecture

### 20.1 Local Development (Docker Compose)

```yaml
# infra/docker/docker-compose.yml
version: "3.9"
services:
  api:
    build: ../../apps/api
    ports: ["8080:8080"]
    environment:
      DATABASE_URL: postgres://ecollm:ecollm@postgres:5432/ecollm
      REDIS_URL: redis://redis:6379
      JWT_SECRET: dev-secret-change-me
      PHI3_SIDECAR_URL: http://prompt-optimizer:8081
      LOG_LEVEL: debug
    depends_on: [postgres, redis]

  web:
    build: ../../apps/web
    ports: ["3000:3000"]
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080

  prompt-optimizer:
    build: ../../apps/prompt-optimizer
    ports: ["8081:8081"]

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ecollm
      POSTGRES_PASSWORD: ecollm
      POSTGRES_DB: ecollm
    ports: ["5432:5432"]
    volumes: ["pgdata:/var/lib/postgresql/data"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  prometheus:
    image: prom/prometheus
    ports: ["9090:9090"]
    volumes: ["../../observability/prometheus:/etc/prometheus"]

  grafana:
    image: grafana/grafana
    ports: ["3001:3000"]
    volumes: ["../../observability/grafana:/etc/grafana/provisioning"]

volumes:
  pgdata:
```

For GPU inference locally, use a separate compose file:

```yaml
# infra/docker/docker-compose.gpu.yml (extends base)
services:
  vllm-mistral:
    image: vllm/vllm-openai:latest
    runtime: nvidia
    deploy:
      resources:
        reservations:
          devices:
            - capabilities: [gpu]
    command: >
      --model mistralai/Mistral-7B-Instruct-v0.3
      --quantization awq
      --max-model-len 4096
      --port 8000
    ports: ["8000:8000"]
```

### 20.2 CI/CD Pipeline (GitHub Actions)

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]

jobs:
  lint-and-test-go:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env: { POSTGRES_PASSWORD: test }
        ports: ["5432:5432"]
      redis:
        image: redis:7-alpine
        ports: ["6379:6379"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - run: cd apps/api && go vet ./...
      - run: cd apps/api && go test -race -coverprofile=coverage.out ./...
      - run: cd apps/api && go build -o /dev/null ./cmd/server

  lint-and-test-web:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: "20" }
      - run: cd apps/web && npm ci
      - run: cd apps/web && npm run lint
      - run: cd apps/web && npm run test
      - run: cd apps/web && npm run build

  e2e:
    runs-on: ubuntu-latest
    needs: [lint-and-test-go, lint-and-test-web]
    steps:
      - uses: actions/checkout@v4
      - run: docker compose -f infra/docker/docker-compose.yml up -d
      - run: npx playwright test
```

### 20.3 Production Deployment

**Environment strategy:** Local → Staging → Production

| Environment | Infrastructure | Models | Purpose |
|-------------|---------------|--------|---------|
| **Local** | Docker Compose, no GPU | Ollama (mock models) | Development |
| **Staging** | Kubernetes (2 CPU nodes, 1 L4 GPU) | Mistral 7B only | Integration testing |
| **Production** | Kubernetes (4+ CPU nodes, GPU pool) | All 4 models | Customer traffic |

**GPU node strategy:**
- L4 nodes: Phi-3 + Mistral 7B (always on, 95% of traffic)
- L40S nodes: Llama 13B (always on, fallback)
- A100 nodes: Llama 70B (scale-to-zero, <5% of traffic)

**Autoscaling:**
- CPU nodes: HPA based on request rate (scale at 70% CPU)
- GPU nodes: Custom metrics — scale Llama 70B nodes based on queue depth
- Never over-provision GPU (that contradicts the environmental mission)

**Database migrations:**
- `golang-migrate` for schema versioning
- Run before deployment (in CI pipeline)
- Forward-only (no down migrations in production)

---

## 21. MVP Build Plan

### Phase 1: Foundation (Weeks 1–4)

**Features:**
- Go API server with Fiber
- POST `/v1/chat/completions` (OpenAI-compatible)
- API key auth (generate, validate, revoke)
- Single model runtime (Mistral 7B via vLLM)
- Request logging to PostgreSQL
- Basic rate limiting (Redis)
- Docker Compose local setup

**Deliverables:**
- Working API that accepts prompts and returns completions
- API key management
- Request history in database

**Risks:**
- vLLM setup complexity → mitigate with Ollama fallback for dev
- Go + vLLM integration → simple HTTP client, well-documented API

**Acceptance criteria:**
- API responds to chat completion requests with <2s latency
- API keys can be created and validated
- Requests are logged with latency and token counts

---

### Phase 2: Routing + Optimization (Weeks 5–8)

**Features:**
- Task classifier (keyword + heuristic)
- Model scoring function (energy 40%, cost 30%, quality 20%, latency 10%)
- 3-model router: Phi-3, Mistral 7B, Llama 13B
- Rule-based prompt optimizer
- Redis response caching
- Fallback logic (retry with larger model)
- Next.js dashboard: overview + request log

**Deliverables:**
- Routing engine selects optimal model per request
- Prompt optimization improves output quality
- Cache reduces redundant inference
- Dashboard shows request history and model distribution

**Risks:**
- Routing accuracy too low → accept 85% accuracy, iterate with telemetry
- Cache hit rate too low → acceptable at 10%+ for MVP

**Acceptance criteria:**
- Router correctly classifies >85% of test prompts
- Prompt optimization reduces token count by 10%+ on average
- Cache hit rate > 10% on repeated workloads
- Dashboard loads and displays request data

---

### Phase 3: Energy + Carbon (Weeks 9–12)

**Features:**
- Energy estimator (GPU power × time / batch size)
- Carbon calculator (energy × grid intensity)
- Per-request energy/CO2 metadata in API response
- Carbon dashboard (CO2 saved, grid data, comparison vs GPT-4)
- Usage analytics (by model, task type, time period)
- API key scoping and management UI
- Prometheus + Grafana setup
- OpenTelemetry tracing

**Deliverables:**
- Every response includes energy and carbon metadata
- Carbon dashboard shows environmental impact
- Observability stack operational

**Risks:**
- Energy estimates inaccurate → use published baselines with ±10% error margin
- Grid carbon data unavailable → fallback to US average (450 gCO2/kWh)

**Acceptance criteria:**
- Every API response includes `ecollm.energy` and `ecollm.cost` metadata
- Carbon dashboard renders with real data
- Prometheus metrics visible in Grafana

---

### Phase 4: Production Hardening (Weeks 13–16)

**Features:**
- Billing events and usage aggregation
- Organization management (multi-tenant)
- Admin panel: model registry CRUD, system metrics
- Playground page (interactive prompt testing)
- Security hardening (CORS, input validation, content filtering)
- Kubernetes deployment manifests
- CI/CD pipeline (GitHub Actions)
- Load testing (k6)
- Documentation (API reference, deployment guide)

**Deliverables:**
- Production-ready platform deployable to Kubernetes
- Billing system tracks usage per org
- Admin can manage models and view system metrics
- CI/CD deploys on merge to main

**Risks:**
- Multi-tenant isolation bugs → enforce org_id at repository layer
- Kubernetes GPU scheduling → use node selectors and tolerations

**Acceptance criteria:**
- Platform handles 100 req/s sustained (load test)
- Multi-org data isolation verified (security test)
- CI/CD deploys to staging automatically
- API documentation complete and accurate

---

## 22. Engineering Tradeoffs

| Decision | Choice | Alternative | Why |
|----------|--------|-------------|-----|
| **Go vs Python (backend)** | Go | Python (FastAPI) | Go: 10x lower memory, 5x faster HTTP, better concurrency. Python only for ML sidecar. |
| **REST vs gRPC** | REST | gRPC | REST: simpler client integration, OpenAI compatibility. gRPC adds complexity without benefit at MVP scale. Internal services can switch later. |
| **PostgreSQL vs Event Store** | PostgreSQL | EventStoreDB | PostgreSQL: mature, simple, sufficient for request logging. Event sourcing adds unnecessary complexity. |
| **Redis caching vs Semantic caching** | Redis (exact match) | pgvector + embeddings | Exact match: zero compute cost, 15%+ hit rate. Semantic: requires embedding model (more energy). Deferred. |
| **vLLM vs llama.cpp** | vLLM | llama.cpp | vLLM: better batching, higher throughput, PagedAttention. llama.cpp better for edge/CPU — not our use case. |
| **Small models vs Large models** | Default small | Default large | Core thesis: 70% of tasks don't need large models. Quality trade-off (85% vs 95%) saves 80% energy. |
| **Latency vs Quality** | Accept higher latency for small models | Use large models for speed on easy tasks | Small model at 400ms + 0.00008 kWh beats large model at 200ms + 0.0008 kWh. Energy wins. |
| **Cost vs Accuracy** | Optimize for cost | Optimize for accuracy | Target market (mid-market SaaS) cares more about cost than frontier accuracy. Quality threshold ensures minimum standard. |
| **Energy efficiency vs UX** | Efficiency first | UX first | This is the brand promise. We don't add streaming (higher overhead) or long context (exponential cost) in MVP. |
| **Modular monolith vs Microservices** | Modular monolith | Microservices | At <200 customers, microservices add latency (network hops) and operational cost. Extract when scale demands. |
| **Fiber vs Chi vs Gin** | Fiber | Chi, Gin | Fiber: fastest benchmarks, Express-like API familiar to JS developers, good middleware ecosystem. |

---

## 23. Final Recommended Architecture

### Exact Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| **API Server** | Go + Fiber | Go 1.22+, Fiber v2 |
| **Database** | PostgreSQL | 16 |
| **Cache** | Redis | 7 |
| **Inference Runtime** | vLLM | Latest |
| **Models** | Phi-3 3.8B (AWQ), Mistral 7B (AWQ), Llama 13B (AWQ), Llama 70B (GPTQ) | Latest weights |
| **Frontend** | Next.js + TypeScript + Tailwind + shadcn/ui | Next.js 14+ |
| **State Management** | TanStack Query (server), Zustand (minimal client) | Latest |
| **Metrics** | Prometheus + Grafana | Latest |
| **Tracing** | OpenTelemetry | Latest |
| **Logging** | zerolog (Go), structured JSON | Latest |
| **CI/CD** | GitHub Actions | N/A |
| **Containers** | Docker + Docker Compose (local), Kubernetes (prod) | N/A |

### Services to Build

1. **`api`** — Go modular monolith: HTTP gateway, auth, routing engine, prompt optimizer, usage tracking, billing, admin
2. **`inference-gateway`** — Go proxy to vLLM model runtimes with health checks and fallback
3. **`carbon-service`** — Go service for grid carbon intensity data and CO2e calculation
4. **`prompt-optimizer`** — Python FastAPI sidecar for Phi-3 prompt refinement (fallback only)
5. **`web`** — Next.js dashboard, playground, admin panel

### Deployment Model

- **Local:** Docker Compose (all services, Ollama for models)
- **Staging:** Kubernetes (2 CPU nodes, 1 GPU node, managed Postgres + Redis)
- **Production:** Kubernetes (4+ CPU nodes, GPU pool with L4/L40S/A100, managed Postgres + Redis)

### MVP Scope (16 weeks)

- OpenAI-compatible API with 3-model routing
- Rule-based prompt optimization
- Energy + carbon tracking per request
- Redis caching
- Dashboard with usage analytics and carbon reporting
- API key management
- Kubernetes-ready deployment

### What to Avoid

| Avoid | Why |
|-------|-----|
| Fine-tuning | Adds compute + complexity. Prompt optimization gets 80% of the way. |
| Multimodal | Vision models are 50x more expensive. Text only. |
| Streaming | Higher per-token overhead. Add in Phase 5. |
| Long context (>8K) | Exponential cost. Encourage RAG instead. |
| Agent frameworks | Unpredictable compute. Not aligned with efficiency. |
| Semantic caching | Requires embedding model (more energy). Exact-match first. |
| Microservices (at launch) | Network hops add latency and energy. Modular monolith first. |
| Custom model training | We're a routing platform, not a model lab. |
| Multiple cloud providers (at launch) | Operational complexity. Single provider, optimize later. |

---

## Appendix: Environment Variable Reference

```env
# API Server
PORT=8080
DATABASE_URL=postgres://ecollm:password@localhost:5432/ecollm
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-256-bit-secret
LOG_LEVEL=info                        # debug, info, warn, error
ALLOWED_ORIGINS=http://localhost:3000
REQUEST_TIMEOUT=30s
RATE_LIMIT_PER_MIN=60

# Prompt Optimizer Sidecar
PHI3_SIDECAR_URL=http://prompt-optimizer:8081

# Carbon Service
GRID_REGION=US-EAST
GRID_API_URL=https://api.electricitymap.org
GRID_API_KEY=your-api-key
DEFAULT_CARBON_INTENSITY=450          # gCO2/kWh fallback

# Inference Endpoints (per model)
INFERENCE_PHI3_URL=http://vllm-phi3:8000
INFERENCE_MISTRAL_URL=http://vllm-mistral:8000
INFERENCE_LLAMA13B_URL=http://vllm-llama13b:8000
INFERENCE_LLAMA70B_URL=http://vllm-llama70b:8000

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
PROMETHEUS_PORT=9090
GRAFANA_PORT=3001

# Feature Flags
ENABLE_PROMPT_OPTIMIZATION=true
ENABLE_CARBON_TRACKING=true
ENABLE_CACHE=true
ENABLE_FALLBACK=true
```

---

**END OF DOCUMENT**

This architecture is designed to be built in 16 weeks by a team of 4–6 engineers. Every decision prioritizes efficiency, sustainability, and production readiness. Build it.
