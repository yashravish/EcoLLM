# EcoLLM Routing Algorithm

## Overview

Every inference request is routed through a three-stage pipeline:

```
Request → Prompt Optimizer → Task Classifier → Model Scorer → Model Selector → Inference
```

---

## Stage 1: Prompt Optimization

**File:** `apps/api/internal/prompt/optimizer.go`

Rule-based rewriting applied to every request before classification:

1. Remove redundant whitespace and punctuation
2. Collapse repetitive phrasing
3. Strip boilerplate instructions that vLLM handles natively
4. If the resulting prompt is still verbose (>512 tokens), optionally call the Phi-3 Python sidecar for semantic compression

Token reduction lowers energy consumption directly: fewer tokens → less GPU time → lower CO2e.

---

## Stage 2: Task Classification

**File:** `apps/api/internal/router/classifier.go`

Assigns a `TaskType` and `complexity` score (0–1) using keyword heuristics:

| TaskType | Examples | Typical model |
|----------|----------|---------------|
| `trivial` | "What is 2+2?", "Format this JSON" | phi_3 |
| `simple` | Short Q&A, greetings, list formatting | phi_3 |
| `medium` | Summarisation, translation, simple code | mistral_7b |
| `complex` | Multi-step reasoning, long-form writing | llama_13b |
| `expert` | PhD-level analysis, complex code generation | llama_70b |

**Complexity signals:**
- Token count of prompt
- Presence of reasoning keywords ("explain why", "compare and contrast")
- Nested instruction depth
- Code language detected (Python > Bash for complexity)

---

## Stage 3: Model Scoring

**File:** `apps/api/internal/router/scorer.go`

Each candidate model receives a score based on four weighted dimensions:

```
score = (energy_score × 0.40)
      + (cost_score   × 0.30)
      + (quality_score × 0.20)
      + (latency_score × 0.10)
```

**Architecture constraint:** `energy_weight` must be ≥ 0.35 and cannot be reduced below this floor. This is a hard constraint, not a soft preference.

### Dimension calculation

| Dimension | Calculation |
|-----------|-------------|
| `energy_score` | `1 - (model_energy_kwh / max_energy_kwh_in_pool)` |
| `cost_score` | `1 - (model_cost_per_req / max_cost_in_pool)` |
| `quality_score` | `model.quality_benchmark` (0–1, from model registry) |
| `latency_score` | `1 - (model.latency_p95_ms / max_latency_ms_in_pool)` |

---

## Stage 4: Model Selection

**File:** `apps/api/internal/router/selector.go`

1. Filter candidates: remove models whose `quality_benchmark < request.min_quality`
2. Filter candidates: remove models whose `latency_p95_ms > request.max_latency_ms`
3. Select model with highest score
4. If no model passes filters → use largest model (llama_70b) as emergency fallback
5. Set `Fallback` = next-best model (used by inference gateway on error)

**Default routing:** Phi-3 handles ~70% of traffic, Mistral ~20%, Llama-13B ~9%, Llama-70B <1%.

---

## Fallback Chain

If the selected model returns an error or low-confidence response:

```
phi_3 → mistral_7b → llama_13b → llama_70b → error
```

Fallback is controlled by `apps/api/internal/inference/fallback.go`.
