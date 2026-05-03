# EcoLLM API Reference

**Base URL:** `https://api.ecollm.io`  
**Authentication:** `Authorization: Bearer <api_key>`

---

## Chat Completions

### POST /v1/chat/completions

OpenAI-compatible inference endpoint. EcoLLM automatically routes to the smallest capable model.

**Request**

```json
{
  "messages": [
    { "role": "user", "content": "Summarise this paragraph in one sentence." }
  ],
  "max_tokens": 256,
  "temperature": 0.7,
  "ecollm": {
    "prefer": "efficiency",
    "include_metadata": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `messages` | array | Conversation turns (required) |
| `max_tokens` | int | Maximum completion tokens |
| `temperature` | float | Sampling temperature (0–2) |
| `ecollm.prefer` | string | `efficiency` \| `speed` \| `quality` |
| `ecollm.max_latency_ms` | int | Abort if routing exceeds this latency |
| `ecollm.min_quality` | float | Minimum quality threshold (0–1) |
| `ecollm.include_metadata` | bool | Include energy/cost metadata in response |

**Response**

```json
{
  "id": "req_01j...",
  "object": "chat.completion",
  "created": 1714600000,
  "model": "phi_3",
  "choices": [{ "index": 0, "message": { "role": "assistant", "content": "..." }, "finish_reason": "stop" }],
  "usage": { "prompt_tokens": 42, "completion_tokens": 18, "total_tokens": 60 },
  "ecollm": {
    "route": { "task_type": "summarise", "complexity": 0.3, "model_selected": "phi_3", "routing_score": 0.87, "confidence": 0.91, "used_fallback": false },
    "energy": { "total_energy_kwh": 0.000042, "co2e_grams": 0.016, "grid_region": "US-EAST", "grid_carbon_intensity": 386 },
    "cost": { "total_cost_usd": 0.000016, "savings_vs_gpt4_percent": 97.2 },
    "performance": { "latency_ms": 312 }
  }
}
```

---

## Usage

### GET /v1/usage

Returns aggregated usage statistics for the authenticated organisation.

**Query parameters:** `period` (daily|monthly), `from` (ISO date), `to` (ISO date)

### GET /v1/requests

Lists inference requests with optional filters.

**Query parameters:** `limit`, `offset`, `model`, `status`, `task_type`

### GET /v1/requests/:id

Returns full detail for a single request including route trace.

---

## Carbon

### GET /v1/carbon

Returns carbon impact summary for the period.

---

## Billing

### GET /v1/billing

Lists billing events for the authenticated organisation.

### GET /v1/billing/:id

Returns a single billing event.

---

## Auth

### POST /auth/login

```json
{ "email": "user@example.com", "password": "••••••••" }
```

Returns `{ "token": "...", "user": {...}, "org": {...} }`.

### POST /auth/register

```json
{ "name": "Ada", "email": "ada@example.com", "password": "••••••••", "org_name": "Acme" }
```

### POST /auth/logout

Invalidates the current session token.

### GET /me

Returns the authenticated user and their organisation.

---

## API Keys

### GET /api-keys
### POST /api-keys
### DELETE /api-keys/:id

---

## Error format

All errors use a consistent envelope:

```json
{ "code": 422, "message": "max_tokens must be > 0", "type": "validation_error", "trace_id": "..." }
```
