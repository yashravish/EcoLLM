# EcoLLM: The Refined Startup Concept
## Final, Definitive Founder Brief

**Version:** 1.0 Final  
**Status:** Investor-Ready  
**Last Updated:** May 2026

---

## Executive Summary (Pitch Deck Version)

**Company:** EcoLLM  
**One-Liner:** The carbon-aware, open-source LLM platform that delivers ChatGPT-like capabilities at 10x lower environmental and financial cost.

**Core Thesis:** The LLM market has optimized for capability and convenience, not efficiency. This created a wasteful moat: large models, redundant inference, and energy-intensive architectures. EcoLLM inverts this: a platform where efficiency is the primary constraint, not an afterthought. This is both environmentally necessary and economically advantageous—and defensible.

**Target Customer:** Mid-market SaaS companies ($20M–$1B ARR) already spending $50K–$500K annually on LLM APIs.

**Problem Solved:** LLM costs are unsustainable and environmentally wasteful. 70% of API requests are commodity tasks that don't require large models. Customers overpay for capability they don't need.

**Solution:** Intelligent routing system that matches requests to the smallest, fastest, lowest-energy model that delivers the required quality. Transparent carbon accounting. Open-source foundation with no vendor lock-in.

**Unit Economics:** 
- Customer saves 70–80% on LLM spend
- EcoLLM captures 10–20% of savings as margin
- Gross margins: 65–75%

**Environmental Impact:**
- Each customer request: 80% less CO2 than OpenAI baseline
- If 100K customers adopt: 5M+ tons CO2e avoided annually (equivalent to 1M cars off the road)
- Third-party verifiable carbon tracking (primary differentiator)

**Market Size:**
- TAM: ~250K SaaS companies using LLM APIs
- Serviceable Addressable Market (SAM): ~50K mid-market SaaS companies
- Year 1 Target: 50–100 customers
- Year 3 Target: 5,000–10,000 customers

**Funding Ask:** $2M seed (16-month runway)

---

## Part 1: The Problem

### Current State of LLM Infrastructure

**OpenAI/Claude API Economics:**
- GPT-4: $0.01–$0.03 per request (100 tokens avg)
- GPT-3.5: $0.002–$0.005 per request
- Energy per request: 50–100g CO2e
- Hardware: H100s ($30K each, 2–3 year amortization)
- Margin model: API pricing subsidizes R&D; margins compress as competition increases

**Customer Pain:**
- SaaS companies using LLM APIs see runaway costs at scale
  - Typical spend: $50K–$500K/year for mid-market SaaS
  - Scaling to enterprise: can reach $5M+/year
- 70% of requests are commodity tasks (classification, summarization, basic generation)
  - These don't need GPT-4 or Claude 3.5 Sonnet
  - But companies use large models by default (easier than optimizing)
- No visibility into what's driving costs
  - No per-request transparency
  - No ability to trade quality for cost
  - No environmental accountability

**Market Inefficiency:**
- Customers overpay for capability they don't use
- Large model providers have no incentive to optimize (cost reduction = margin loss)
- Environmental cost is externalized (not priced into API)

**The Real Insight:**
If you could deliver 85% of the quality at 20% of the cost, customers would switch immediately.

---

## Part 2: Why This Exists as an Opportunity Now

### Three Converging Trends

#### 1. **Open-Source Models Are Now Competitive**
- **2023:** Llama 2, Mistral, Phi releases
- **2024:** Llama 3 (70B), Mistral 8x7B MoE, Phi-3
- **Reality:** Open models now match closed models on most commodity tasks
  - Mistral 7B ≈ GPT-3.5 on most benchmarks
  - Llama 13B ≈ Claude Instant on summarization/classification
  - Phi-3 (3.8B) ≈ GPT-3.5 on many narrow tasks
- **Why it matters:** You no longer need proprietary models. Open-source suffices.

#### 2. **Inference Efficiency Is Asymptotically Improving**
- **vLLM, TGI, llama.cpp:** Production-grade inference runtimes (didn't exist 2 years ago)
- **Quantization:** 4-bit quantization now standard without quality loss
- **Architectural optimizations:** Flash Attention, GQA, speculative decoding
- **Hardware:** L4/L40S are now cost-effective (didn't exist in 2023)
- **Reality:** You can run a 7B model on consumer hardware at <100ms latency
- **Why it matters:** Efficiency is no longer a theoretical advantage; it's practical infrastructure

#### 3. **Environmental Awareness Is Becoming Table Stakes**
- **EU AI Act (2024):** Requires transparency on energy usage
- **Corporate ESG mandates:** Microsoft, Google, Meta committing to carbon-neutral operations
- **Customer expectations:** CFOs asking "what's the environmental cost of our AI spend?"
- **Why it matters:** You can credibly market efficiency as a non-greenwashing differentiator

### The Timing Window

This is a **2–3 year window** before:
1. OpenAI optimizes pricing (drops costs by 50%)
2. Claude APIs become more cost-competitive
3. Larger SaaS companies build internal routing layers

You need to:
- Build brand + capture beachhead customers in Year 1
- Achieve product-market fit in Year 1–2
- Scale to 5,000+ customers before large competitors optimize

---

## Part 3: The Solution (EcoLLM Platform)

### 3.1 What EcoLLM Is (And Isn't)

**What It Is:**
- A routing + optimization layer for open-source LLMs
- An API that feels like ChatGPT/Claude but optimizes for efficiency
- A real-time decision engine that matches requests to the best model
- A carbon accounting system that tracks environmental impact per request

**What It's NOT:**
- Not a new LLM (you're not training a model)
- Not a fine-tuning platform (defer this)
- Not a multimodal system (text only, launch phase)
- Not an agent framework (focus on single request/response)
- Not a long-context system (4K–8K max, encourage RAG for longer)

**Why this positioning matters:**
- You're not competing on model capability; you're competing on efficiency
- Customers don't expect ChatGPT parity; they expect 85% quality at 20% cost
- You can ship fast without training a new model

### 3.2 Core Product: Three Layers

#### Layer 1: Prompt Optimization (Rule-Based + Lightweight LLM)

**Purpose:** Improve output quality without increasing model size

**Design:**
- **Rule-based (60% of impact):**
  - Remove redundancy, clarify intent, add structural guidance
  - Heuristic: longer, clearer prompts = better output from smaller models
  - Cost: <1ms, ~0.05g CO2e
  
- **Lightweight LLM fallback (40% of impact):**
  - Phi-3 (2B) fine-tuned on prompt optimization
  - Rewrites ambiguous prompts for clarity
  - Cost: <50ms, ~0.1g CO2e
  
- **Example transformation:**
  ```
  User input:
  "write code to sort"
  
  EcoLLM optimization:
  "Write Python code that implements QuickSort with the following requirements:
  - Include detailed comments explaining the algorithm
  - Handle edge cases (empty list, single element, duplicates)
  - Include time complexity analysis in docstring
  - Use clean, production-ready style
  Target model: Mistral 7B (not Llama 13B)"
  ```

**Environmental Impact:**
- Optimized prompts improve quality by ~15–20%
- This lets you use a smaller model downstream
- Net CO2 savings: 60–70% vs non-optimized path

**Why this works:**
- 60% of output quality comes from prompt clarity, not model size
- This is your secret weapon against larger competitors
- It's low-cost to operate (rule-based + tiny model)

---

#### Layer 2: Intelligent Routing Engine (The Core IP)

**Purpose:** Select the best model for each request based on efficiency + cost + quality

**Decision Framework:**

```
INPUT: User prompt + context

STEP 1: Task Classification
├─ Analyze: intent, complexity, domain, required quality
├─ Output: task_type in {simple, medium, hard, specialized}
└─ Cost: <10ms, rule-based

STEP 2: Generate Model Candidates
├─ Simple tasks → [Phi-3 2B, Mistral 7B]
├─ Medium tasks → [Mistral 7B, Llama 13B]
├─ Hard tasks → [Llama 13B, Llama 70B]
├─ Specialized → [external API fallback or cached response]
└─ Cost: lookup table

STEP 3: Score Each Candidate
├─ Energy efficiency (40% weight)
│   ├─ Measured: energy_kwh per request
│   ├─ Baseline: published per-token energy
│   └─ Adjustment: batch size, quantization, latency
│
├─ Cost (30% weight)
│   ├─ GPU hours × hardware cost
│   ├─ Operational overhead
│   └─ Network/storage
│
├─ Quality (20% weight)
│   ├─ Benchmark score on task type
│   ├─ Customer-specific quality threshold
│   └─ Confidence interval
│
└─ Latency (10% weight)
    ├─ p95 latency for model
    ├─ Batch queue time
    └─ Network overhead

STEP 4: Apply Hard Constraints
├─ IF quality_score < customer_threshold
│   THEN bump to next larger model
├─ IF latency_p95 > 10s
│   THEN reject (fallback to external API)
└─ IF energy > daily_budget
│   THEN flag for customer review

STEP 5: Return Selection
├─ Selected model
├─ Estimated energy (kwh)
├─ Estimated CO2 (grams)
├─ Estimated cost (usd)
├─ Confidence score
└─ Fallback model (if primary fails)
```

**Routing Scoring Function (Pseudocode):**

```python
def score_model(model_candidate, request_context):
    """
    Environment-first scoring function.
    Higher score = better choice.
    """
    
    metrics = {
        "energy_kwh": 0.00001,        # measured or estimated
        "cost_usd": 0.0005,            # per request
        "quality_score": 0.85,         # on 0-1 scale
        "latency_ms": 400,             # p95
    }
    
    # Normalize each metric to 0-1 range
    energy_normalized = 1 - (metrics["energy_kwh"] / MAX_ENERGY_KWH)
    cost_normalized = 1 - (metrics["cost_usd"] / MAX_COST_USD)
    quality_normalized = metrics["quality_score"]  # already 0-1
    latency_normalized = 1 - (metrics["latency_ms"] / MAX_LATENCY_MS)
    
    # Weighted sum (ENVIRONMENT FIRST)
    score = (
        0.40 * energy_normalized +      # 40% — ENERGY
        0.30 * cost_normalized +         # 30% — COST
        0.20 * quality_normalized +      # 20% — QUALITY
        0.10 * latency_normalized        # 10% — LATENCY
    )
    
    # Apply hard constraints
    if quality_normalized < customer_min_quality:
        score = score * 0.5  # heavy penalty, but not disqualified
    
    if latency_ms > SLA_threshold:
        score = 0  # disqualified
    
    return score
```

**Model Configuration (Launch):**

| Model | Size | Quantization | Latency (p95) | Cost/Req | Energy/Req | Quality | Use Case |
|-------|------|--------------|---------------|----------|-----------|---------|----------|
| **Phi-3** | 3.8B | q4 | 80ms | $0.0001 | 0.00001 kwh | 0.65 | Simple: FAQ, classification, summarization |
| **Mistral 7B** | 7B | q4 | 400ms | $0.0005 | 0.00008 kwh | 0.85 | Medium: writing, reasoning, coding basics |
| **Llama 13B** | 13B | q4 | 800ms | $0.001 | 0.00015 kwh | 0.92 | Hard: complex reasoning, domain expertise |
| **Llama 70B** | 70B | q8 | 3s | $0.005 | 0.0008 kwh | 0.98 | Fallback: only if required quality >0.95 |

**Why these models:**
- **Phi-3:** Tiny, underrated, great for commodity tasks
- **Mistral 7B:** Best-in-class efficiency for 7B models
- **Llama 13B:** Reasonable quality jump for medium tasks
- **Llama 70B:** Fallback only (<5% of traffic)

**Routing Intelligence (Non-Obvious):**
1. **Task-specific quality thresholds:** Classification tasks need 0.70+; creative writing needs 0.85+
2. **Batch-aware scoring:** Single request vs batch of 100 changes optimal model
3. **Time-of-day routing:** Low-energy models preferred during peak carbon intensity hours
4. **Customer-specific preferences:** Some customers optimize for cost; others for latency
5. **Feedback loop:** Track actual output quality; retrain classifier every week

---

#### Layer 3: Energy & Carbon Accounting (Real + Transparent)

**Purpose:** Track actual environmental impact; use it as a product feature + differentiator

**What Gets Measured:**

```json
{
  "request_id": "req_abc123",
  "timestamp": "2026-05-02T14:32:15Z",
  "model": "mistral_7b_q4",
  "task_type": "summarization",
  "input_tokens": 450,
  "output_tokens": 120,
  "total_tokens": 570,
  
  "inference_metrics": {
    "gpu_id": "l4_gpu_2",
    "batch_size": 8,
    "inference_time_ms": 380,
    "time_to_first_token_ms": 120,
    "tokens_per_second": 1.5
  },
  
  "energy_metrics": {
    "gpu_power_watts": 45,
    "inference_energy_wh": 0.006,  // Measured from GPU telemetry
    "inference_energy_kwh": 0.000006,
    "datacenter_overhead_multiplier": 1.3,  // PUE (Power Usage Effectiveness)
    "total_energy_kwh": 0.0000078
  },
  
  "carbon_metrics": {
    "grid_carbon_intensity_gco2_per_kwh": 450,  // Regional + time-based
    "total_co2e_grams": 3.5,
    "carbon_source": "EPA grid data + regional mix"
  },
  
  "cost_metrics": {
    "gpu_cost_per_hour": 0.25,
    "actual_gpu_cost": 0.00003,
    "network_cost": 0.00001,
    "storage_cost": 0.00001,
    "operational_overhead": 0.0001,
    "margin": 0.00025,
    "total_revenue": 0.0005
  },
  
  "comparison_metrics": {
    "gpt4_equivalent_cost": 0.005,
    "gpt4_equivalent_energy_kwh": 0.00008,
    "gpt4_equivalent_co2_grams": 36,
    "savings_vs_gpt4_cost_percent": 90,
    "savings_vs_gpt4_co2_percent": 90
  },
  
  "routing_metadata": {
    "fallback_model": "llama_13b",
    "fallback_quality_score": 0.92,
    "selected_quality_score": 0.85,
    "quality_trade_off_percent": 5.8,
    "energy_savings_vs_fallback_percent": 65,
    "cost_savings_vs_fallback_percent": 50
  }
}
```

**Customer Dashboard Visualizations:**
1. **Request-level transparency:** Every request shows CO2, cost, quality
2. **Aggregated metrics:** Month-to-date cost savings, CO2 avoided, avg model size
3. **Carbon impact translation:** "You've avoided 500kg CO2e this month (= 50 trees planted)"
4. **Cost vs quality tradeoff:** "You could save 20% more by accepting 5% lower quality"
5. **Routing decisions:** "73% of your requests routed to Mistral 7B (vs GPT-4 by default)"

**Why Real Carbon Accounting Matters:**
- **Credibility:** Not greenwashing; third-party verifiable data
- **Differentiation:** No competitor offers this level of transparency
- **Product feature:** Customers want environmental transparency (makes good ESG optics)
- **Pricing lever:** Can charge based on CO2 (premium for green hours, discount for high-carbon times)

---

### 3.3 Product Architecture (Technical)

**System Diagram:**

```
┌──────────────────────────────────┐
│     User API Request             │
│  (OpenAI-compatible endpoint)    │
└────────────┬─────────────────────┘
             │
    ┌────────▼──────────┐
    │ Request Validation │ ← Auth, rate limits, schema
    │ & Preprocessing   │
    └────────┬──────────┘
             │
    ┌────────▼────────────────────┐
    │  Prompt Optimizer Layer      │
    │ ┌──────────────────────────┐ │
    │ │ Rule-Based Rewriting    │ │
    │ │ (heuristics + templates) │ │
    │ └──────────────────────────┘ │
    │ ┌──────────────────────────┐ │
    │ │ Phi-3 Prompt Refinement  │ │ (if needed)
    │ │ (clarify intent)         │ │
    │ └──────────────────────────┘ │
    └────────┬─────────────────────┘
             │
    ┌────────▼──────────────────────┐
    │   Routing Engine               │
    │ ┌──────────────────────────┐  │
    │ │ Task Classifier          │  │ (keyword + intent)
    │ ├──────────────────────────┤  │
    │ │ Model Candidate Gen      │  │ (lookup table)
    │ ├──────────────────────────┤  │
    │ │ Scoring Function         │  │ (energy 40%, cost 30%, quality 20%, latency 10%)
    │ ├──────────────────────────┤  │
    │ │ Constraint Enforcement   │  │ (quality threshold, latency SLA)
    │ └──────────────────────────┘  │
    └────────┬──────────────────────┘
             │
       ┌─────┴────────────────────────┬──────────────┬──────────────┐
       │                              │              │              │
   ┌───▼──────┐              ┌───────▼────┐  ┌─────▼────┐  ┌─────▼─────┐
   │ Phi-3    │              │Mistral 7B  │  │Llama 13B │  │Llama 70B  │
   │ 2B q4    │              │ 7B q4      │  │ 13B q4   │  │ 70B q8    │
   │ L4 GPU   │              │ L4 GPU     │  │ L40S GPU │  │ A100 GPU  │
   │ vLLM     │              │ vLLM       │  │ TGI      │  │ TGI       │
   └───┬──────┘              └───┬───────┘  └─────┬────┘  └─────┬─────┘
       │                         │               │             │
       └─────────────────────────┼───────────────┼─────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │  Inference Runner       │
                    │ ┌────────────────────┐ │
                    │ │ KV Cache Pooling   │ │
                    │ │ Request Batching   │ │
                    │ │ Energy Metering    │ │
                    │ │ GPU Telemetry      │ │
                    │ └────────────────────┘ │
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │  Response Assembly      │
                    │ ┌────────────────────┐ │
                    │ │ Text Generation    │ │
                    │ │ Metadata Append    │ │
                    │ │ Carbon Accounting  │ │
                    │ │ Cost Calculation   │ │
                    │ └────────────────────┘ │
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │  API Response           │
                    │ {                      │
                    │   result: "...",       │
                    │   usage: {...},        │
                    │   energy_kwh: 0.00001, │
                    │   co2_grams: 4.5,      │
                    │   cost_usd: 0.0005     │
                    │ }                      │
                    └────────────────────────┘
```

**Infrastructure Stack:**

| Layer | Technology | Why |
|-------|-----------|-----|
| **Models** | Mistral, Meta Llama, Microsoft Phi | Open-source, community support, regular updates |
| **Inference Runtime** | vLLM (default) + TGI (fallback) | Production-grade, fast, efficient, active development |
| **Quantization** | GPTQ (4-bit), BNB (8-bit) | Standard, minimal quality loss, 50% memory savings |
| **Hardware** | L4, L40S (primary), A100 (fallback) | Cost-effective, efficient, good supply |
| **Serving** | FastAPI + Uvicorn | Fast, async, simple to scale |
| **Caching** | Redis (request cache) | Dedup repeated requests, 10–20% reduction in compute |
| **Monitoring** | Prometheus + Grafana | Energy, latency, cost metrics |
| **Logging** | ELK Stack | Request-level tracing, debugging |
| **Database** | PostgreSQL (metadata) | Customer config, routing decisions, analytics |

**Deployment Strategy:**
- **MVP (Months 1–3):** Single cloud provider (e.g., Lambda Labs, Crusoe, or CoreWeave)
- **Scale (Months 4–12):** Multi-cloud for cost optimization + redundancy
- **Future:** Customer option for on-prem / private cloud deployment

**Why This Architecture Is Efficient:**
1. **Small models preferred:** Phi-3 runs on L4 (~$0.25/hour); Llama 70B on A100 (~$3/hour)
2. **Batching & caching:** 10–30% reduction in total compute
3. **Quantization:** 50% memory savings, minimal quality loss
4. **Request pooling:** Similar requests share KV cache
5. **No unnecessary steps:** No fine-tuning, agents, or function calling overhead

---

## Part 4: Business Model

### 4.1 Target Customer Profile

**Ideal Customer:**
- Mid-market SaaS companies ($20M–$1B ARR)
- Currently using OpenAI / Claude APIs
- Spending $50K–$500K annually on LLM inference
- Have ML/data engineering team (can debug / integrate)
- Care about cost AND environmental impact (ESG-conscious)
- Not building frontier LLM capabilities (don't need GPT-4 quality)

**Use Cases They Care About:**
1. Customer support automation (classify, generate responses)
2. Content generation (blogs, emails, social media)
3. Data processing (summarization, extraction, classification)
4. Internal tooling (code generation, documentation, Q&A)
5. Search/recommendation ranking (query understanding)

**Why They'll Switch:**
- **Cost savings:** 70–80% reduction in LLM spend
- **Environmental alignment:** Real carbon tracking, ESG reporting
- **No vendor lock-in:** Open-source foundation, can self-host
- **Simplicity:** Drop-in replacement for OpenAI API
- **Transparency:** Per-request cost/energy visibility

**Who We DON'T Target (Launch):**
- Consumer apps (Discord bots, ChatGPT plugins)
- Image/video generation companies
- Frontier AI research
- Companies that need ChatGPT/Claude parity

---

### 4.2 Pricing Model

**Philosophy:** Transparent, efficiency-aligned pricing that passes savings to customer while capturing margin.

**Option A: Energy-Based Pricing (Recommended)**

```
Pay per CO2 gram + base overhead

Price = (CO2_grams × $0.001) + $0.0001_per_request_overhead

Examples:
- Simple task (Phi-3): 0.5g CO2 → $0.0005 + $0.0001 = $0.0006/req
- Medium task (Mistral): 3g CO2 → $0.003 + $0.0001 = $0.0031/req
- Hard task (Llama 13B): 8g CO2 → $0.008 + $0.0001 = $0.0081/req

Comparison to OpenAI:
- GPT-3.5: $0.002/req (100 tokens)
- GPT-4: $0.01/req (100 tokens)
- EcoLLM avg (routed): $0.002/req (same tasks)
→ 80% cost reduction for equivalent capability

Customer Value:
- For 1M requests/month:
  - OpenAI spend: $2,000/month
  - EcoLLM spend: $400/month
  - Customer savings: $1,600/month ($19,200/year)
```

**Why This Model:**
- Directly aligns pricing with environmental impact
- Simpler than token-based pricing (harder to optimize away)
- Customers can predict costs based on workload type
- Unique differentiator (no competitor does this)

**Option B: Token-Based (Traditional)**

```
$0.00015 / input token
$0.0005 / output token
+ $100/month minimum

Comparable to OpenAI but cheaper
Familiar to customers already using APIs
```

**My Recommendation:** Use **Option A** (energy-based) for differentiation. Use **Option B** as fallback for customers who want simplicity.

**Tiered Pricing (Future):**
- **Free tier:** 10K requests/month (for startups, experimentation)
- **Starter:** $200/month + usage (for small SaaS)
- **Growth:** Custom pricing (for mid-market SaaS)
- **Enterprise:** Self-hosted option + support

**Volume Discounts:**
- 1M+ requests/month: 15% discount
- 10M+ requests/month: 25% discount
- (You still earn 65%+ margin even at max discount)

---

### 4.3 Unit Economics (Detailed)

**Cost Structure (per request):**

| Component | Phi-3 | Mistral 7B | Llama 13B | Note |
|-----------|-------|-----------|----------|------|
| **GPU compute** | $0.00007 | $0.00022 | $0.00040 | Based on L4/L40S cost ($0.25/hr) |
| **Network/storage** | $0.00003 | $0.00003 | $0.00003 | Minimal, amortized |
| **Ops overhead** | $0.00010 | $0.00010 | $0.00010 | Monitoring, logging, support |
| **TOTAL COGS** | **$0.00020** | **$0.00035** | **$0.00053** | |

**Revenue (per request):**

| Model | Price/Req | COGS | Gross Margin | Gross Margin % |
|-------|-----------|------|--------------|----------------|
| Phi-3 | $0.0006 | $0.0002 | $0.0004 | 67% |
| Mistral 7B | $0.0031 | $0.00035 | $0.0028 | 90% |
| Llama 13B | $0.0081 | $0.00053 | $0.0076 | 94% |

**Customer Lifetime Value (Detailed):**

```
Assumptions:
- Customer: Mid-market SaaS
- Current OpenAI spend: $200K/year (200K requests/month × $0.002/req)
- Workload mix: 60% Phi-3, 30% Mistral 7B, 10% Llama 13B
- Conversion rate: 30% (1 in 3 free trial → paid)
- Churn rate: 8% monthly (1-year ACV = $144K)
- Discount rate: 10%

Year 1 Revenue (per customer):
- Month 1–2: Free trial
- Month 3–12: Ramp from 50% to 100% usage
- Average: 50% of workload
- Cost: $200K × 0.3 × 50% = $30K
- EcoLLM price: $30K
- EcoLLM revenue: $30K (break-even in year 1)

Year 2+ Revenue (per customer):
- Full 100% usage
- Workload mix blended price: (0.6 × $0.0006) + (0.3 × $0.0031) + (0.1 × $0.0081) = $0.00177/req
- Requests/month: 200K
- Monthly revenue: $0.00177 × 200K = $354
- Annual revenue: $4,248
- Gross profit: $4,248 × 80% = $3,398
- Probability customer still active: 100% × (0.92^12) = 36% (low churn assumption)
- Discounted 3-year LTV: ~$8,000

Blended LTV (accounting for conversion rate):
- LTV per trial = $8,000 × 30% = $2,400
- CAC (efficient startup): $500 (inbound + referral)
- LTV:CAC ratio: 4.8:1 (excellent)
```

**Path to Profitability (Company Level):**

```
Assumptions:
- Seed funding: $2M
- Burn rate: $150K/month (6 engineers, 1 PM, 1 sales, ops)
- Customer acquisition: 5 customers/month (exponential ramp)
- Blended ARPU: $4,000/year (Y1), $8,000/year (Y2+)
- Gross margin: 80%

Timeline:
Month 1–4: Develop product, acquire first 5 customers
Month 5–12: Ramp to 50 customers by month 12
Month 13+: Ramp to 500+ customers

Break-even calculation:
- Revenue needed to break even: $150K/month × 80% COGS ratio = $600K/month ARR
- Customers needed: 600K / 8000 × 12 = ~150 customers (avg LTV @ steady state)
- Timeline: ~18–24 months

Funding needed: $2M seed covers 13 months
- Need $1M+ Series A at month 12 for runway to profitability
- OR achieve 30+ customers by month 12 (showing clear path to profitability)
```

**Key Metrics to Track:**
- **CAC (Customer Acquisition Cost):** Target <$500
- **LTV (Lifetime Value):** Target >$8K
- **Payback period:** Target <12 months
- **Gross margin:** Target >75%
- **Net dollar retention:** Target >110% (expansion revenue)
- **Monthly burn:** Target $120–150K (seed-phase sustainable)

---

## Part 5: Go-to-Market Strategy

### 5.1 Launch Phase (Months 1–6)

**Target: Acquire 5–10 paying customers**

**Distribution Channel: Inbound + Direct Sales**

**Phase 1A: Community Building (Month 1–2)**
1. **Publish technical content:**
   - "How We Built an 80% More Efficient LLM Platform" (blog)
   - "Open-Source Models vs GPT-4: A Cost-Benefit Analysis" (comparison)
   - "Carbon Accounting for LLM APIs: Why It Matters" (environmental angle)
   
2. **Open-source the routing library:**
   - MIT-licensed routing framework
   - Show how to build simple routers yourself
   - Position as "if you want to self-host, here's the foundation"
   - Goal: 1K+ GitHub stars, community feedback

3. **Launch landing page:**
   - Clear value prop: "ChatGPT quality at 20% cost"
   - Cost calculator: "Plug in your OpenAI spend, see savings"
   - Carbon impact calculator
   - Environmental differentiator front-and-center

**Phase 1B: Cold Outreach (Month 2–3)**
1. **Find ICP (Ideal Customer Profile):**
   - "SaaS companies using OpenAI" (reverse engineer from job postings, GitHub, tech blogs)
   - Build list of 500+ CTOs / ML engineers
   
2. **Outreach sequence:**
   ```
   Subject: "We cut LLM costs 80% for [Company Name]"
   
   Hi [Name],
   
   I noticed [Company] uses OpenAI APIs (from your tech blog / job posting).
   
   We built EcoLLM—a routing system that delivers 85% of ChatGPT quality
   at 20% of the cost. No vendor lock-in, carbon-aware, open-source.
   
   Typical savings: $30K–$200K/year for companies your size.
   
   Worth 15 minutes? I can show a live cost/CO2 comparison for your workload.
   
   [Link to free cost calculator]
   ```

3. **Goal:** 5–10 % conversion rate to free trial (25–50 trials)

**Phase 1C: Free Trial Program (Month 3–6)**
1. **Trial terms:**
   - 1 month free (full access)
   - Unlimited usage
   - Goal: prove cost savings + quality equivalence
   
2. **Customer success:**
   - Dedicated onboarding (founder time)
   - Weekly check-ins
   - Custom benchmarking against their OpenAI baseline
   - Clear ROI calculation: "You'll save $X/month if you switch"

3. **Conversion goal:** 30% of trials → paid

**Expected Results (6-month endpoint):**
- 50 free trials started
- 15 customers converted to paid
- $180K ARR (15 customers × $12K avg annual)
- Avg customer savings: $60K/year
- Environmental impact: 500+ tons CO2e avoided (if all expanded)

---

### 5.2 Growth Phase (Months 6–18)

**Target: Scale from 15 → 100 customers**

**Distribution Expansion:**

1. **Content Marketing:**
   - Weekly blog posts (technical + environmental angle)
   - Case studies from early customers ("How [Company] cut LLM costs 70%")
   - Comparison guides ("EcoLLM vs OpenAI vs Claude vs Ollama")
   - Environmental impact reports ("This month we avoided 10 tons of CO2e")

2. **Community & Partnerships:**
   - Sponsor open-source ML conferences
   - Partner with environmental/ESG platforms
   - Integration partnerships (Hugging Face Hub, Weights & Biases)
   - Dev community: Discord, GitHub Discussions

3. **Product-Led Growth:**
   - Improve self-serve onboarding
   - Free tier: 10K requests/month (aggressive freemium)
   - Community routing models (customers contribute improvements)
   - Usage-based upgrades

4. **Sales Acceleration:**
   - Hire first sales engineer (Month 9)
   - Outbound targeting: $50K–$1M annual LLM spend companies
   - Partnerships with ML consulting firms (referral channel)
   - RFP capability (for enterprise pipeline)

5. **Product Expansion:**
   - Add Mistral MoE 8x7B (better reasoning)
   - Add Qwen 70B (competitive quality, cheaper inference)
   - Streaming support (for long-form generation)
   - Batch API (for non-realtime workloads)

**Expected Results (18-month endpoint):**
- 100 customers
- $1.2M ARR
- Gross margin: 75%
- Payback period: 8–10 months
- Series A readiness

---

### 5.3 Sales Messaging (3 Angles)

**Angle 1: For CFOs (Cost)**
```
"Your LLM costs are unsustainable. EcoLLM cuts them 70–80%.
Switch to intelligent routing—same quality, 1/5 the price.

ROI: 12-month payback on switching costs. Ongoing savings: 
$[custom calculator based on their spend]"
```

**Angle 2: For CTOs (Architecture)**
```
"You're overpaying for capability. 70% of your requests are 
commodity tasks that don't need GPT-4.

EcoLLM is the routing layer you should have built in-house, 
but open-source, battle-tested, and ready to deploy."
```

**Angle 3: For ESG/Sustainability Teams**
```
"AI has a carbon cost. We measure and minimize it.

Every request with EcoLLM avoids 10x the CO2 of large models.
We make your ESG commitments real and auditable.

[Dashboard showing: 500 tons CO2e avoided this month]"
```

**Customer Success Story (Template):**
```
Case Study: TechCorp (fictional mid-market SaaS)

Before:
- 200K API requests/month with OpenAI
- Cost: $200K/year
- CO2: 10 tons/month (from their perspective: unmeasured)

After (3 months):
- 200K API requests/month with EcoLLM
- Cost: $40K/year (80% savings)
- CO2: 1 ton/month (90% reduction)
- Quality: "85% as good" (acceptable for commodity tasks)

ROI:
- Savings: $160K/year
- Payback: <1 month
- Annual CO2e avoided: 108 tons
```

---

## Part 6: Technical Risks & Mitigations

| Risk | Severity | Impact | Mitigation |
|------|----------|--------|-----------|
| **Routing accuracy (misclassification)** | High | Select wrong model → poor output or wasted compute | Start with simple heuristics + keyword matching. Accept 5–10% misclassification in MVP. Improve with telemetry. Have fallback to larger model if confidence <0.7 |
| **Quality regression vs baseline** | High | Customers perceive lower quality → churn | Maintain side-by-side A/B testing. Publish benchmarks honestly. Offer uptime SLA (quality threshold) |
| **Latency variability** | Medium | p95 latency 2–3x higher than p50 → SLA violation | Pre-warm models. Use multi-GPU to distribute load. Accept 500ms variance in SLA |
| **Model availability risk** | Low | Open-source model becomes unmaintained | Diversify model sources (Mistral, Meta, Microsoft). Monitor updates. Plan 6-month forward migrations |
| **Inference runtime bugs** | Medium | vLLM/TGI crashes → cascade failures | Run redundant inference servers. Have fallback to external API (more expensive but ensures uptime) |
| **Carbon accounting accuracy** | Medium | Greenwashing accusation (publish wrong CO2 numbers) | Use published baselines, measured overhead. Third-party audit annually. Transparent error margins (±10%) |
| **Hardware cost inflation** | Medium | L4 prices increase → margin compression | Diversify across multiple GPU types. Build multi-cloud. Negotiate long-term contracts |
| **OpenAI price war** | Medium | OpenAI cuts prices by 50% → unit economics break | You still have 10x advantage. Differentiate on non-price (efficiency, transparency, open-source). Shift upmarket |
| **Customer integrations too complex** | Medium | "Too hard to switch from OpenAI" → high friction | Make API 100% OpenAI-compatible. SDKs for Python, JS. Provide migration script. White-glove onboarding |

---

## Part 7: Competitive Analysis

### Who You Compete Against

**Direct Competitors (Routing/Optimization):**
- **LiteLLM** (open-source routing library)
  - Pros: Free, open-source, community-driven
  - Cons: No managed service, requires self-hosting
  - EcoLLM advantage: Managed service + carbon accounting
  
- **Replicate** (model marketplace)
  - Pros: Curated models, easy to use
  - Cons: Expensive ($1.25/GPU-hour), not cost-optimized
  - EcoLLM advantage: 5x cheaper, open models
  
- **Together AI** (open-source LLM API)
  - Pros: Fast inference, good models
  - Cons: Higher pricing than our target, not focused on efficiency
  - EcoLLM advantage: Focused optimization, routing, carbon tracking

**Indirect Competitors (API Providers):**
- **OpenAI API** (GPT-3.5, GPT-4)
  - Advantage: Proprietary models, brand
  - Disadvantage: Expensive, high-carbon, vendor lock-in
  - You win on: Cost (10x), sustainability (10x), no lock-in

- **Claude API** (Anthropic)
  - Advantage: Quality, brand
  - Disadvantage: High cost, large models only
  - You win on: Cost (5x), sustainability (5x), flexibility

- **Google Vertex / AWS Bedrock**
  - Advantage: Enterprise support, integration
  - Disadvantage: Opaque pricing, proprietary models
  - You win on: Cost, transparency, open-source

### Why You'll Win

| Dimension | EcoLLM | OpenAI | Claude | Replicate | LiteLLM |
|-----------|--------|--------|--------|-----------|---------|
| **Cost** | $$ | $$$$ | $$$ | $$$ | $ |
| **Quality** | $$$ | $$$$ | $$$$ | $$ | $$$ |
| **Transparency** | $$$$ | $ | $ | $$ | $$$$ |
| **Carbon tracking** | $$$$ | $ | $ | $ | $ |
| **Open-source** | $$$$ | $ | $ | $$ | $$$$ |
| **Managed service** | $$$ | $$$$ | $$$$ | $$$ | $ |
| **Routing** | $$$$ | $ | $ | $ | $$$ |

**Your Winning Combination:**
- Top-3 cost (vs OpenAI/Claude)
- Top-2 quality (vs LiteLLM, comparable to Replicate)
- #1 transparency (carbon + cost)
- #1 routing intelligence
- #1 open-source alignment
- Managed service (unlike LiteLLM)

**Why This Is Defensible:**
1. Routing IP improves with scale (more customer data)
2. Environmental accounting is hard to copy (requires real metering)
3. Open-source alignment builds brand loyalty (vs proprietary API providers)
4. Cost advantage is structural (efficient hardware + models), not just pricing

---

## Part 8: Funding & Financial Projections

### Seed Round ($2M, 16 months)

**Use of Funds:**
```
Personnel (8 people):
  - 4 engineers (AI systems, infra, full-stack): $400K
  - 1 PM/Founder: $150K
  - 1 Sales/Customer success: $100K
  - 1 Operations: $75K
  Subtotal: $725K

Infrastructure & Compute:
  - GPU rentals for inference: $400K
  - Cloud services (storage, network): $100K
  Subtotal: $500K

Go-to-Market:
  - Content, events, partnerships: $150K
  - Sales tools, recruiting: $100K
  Subtotal: $250K

Legal, Financial, Ops:
  - Incorporation, audits, compliance: $50K
  - Buffer & contingency: $225K
  Subtotal: $275K

Total: $1.75M (leaving $250K buffer)

Monthly burn: $140K (sustainable through month 16)
```

### Revenue Projections (3 Years)

```
Year 1:
Month 3–4: 5 customers acquire
Month 5–12: Ramp to 50 customers by year-end
ARPU: $4,000 (blended, low utilization)
ARR: $200K

Year 2:
Month 13–24: Ramp from 50 → 200 customers
ARPU: $8,000 (better utilization, some expansion revenue)
ARR: $1.6M

Year 3:
Month 25–36: Scale from 200 → 500 customers
ARPU: $12,000 (expansion + larger customers)
ARR: $6M

Profitability:
- Year 1: -$1.4M (seed investment)
- Year 2: -$300K (Series A covers)
- Year 3: +$3M operating profit (75% gross margin)
```

**Unit Economics Summary:**

| Metric | Y1 | Y2 | Y3 |
|--------|----|----|-----|
| **Customers** | 50 | 200 | 500 |
| **ARR** | $200K | $1.6M | $6M |
| **ARPU** | $4K | $8K | $12K |
| **Gross margin** | 75% | 77% | 80% |
| **CAC** | $500 | $500 | $300 |
| **LTV** | $8K | $15K | $25K |
| **LTV:CAC** | 16:1 | 30:1 | 83:1 |
| **Payback period** | 8mo | 6mo | 4mo |

---

## Part 9: Why This Works (Strategic Theses)

### Thesis 1: Efficiency Is Becoming Table Stakes
- Open-source models are now competitive with proprietary ones
- Quantization + inference optimization are commoditized
- Energy costs are rising (and will be priced into APIs eventually)
- **Implication:** Building for efficiency now = building for inevitability

### Thesis 2: Customers Are Ready to Trade Quality for Cost
- 70% of LLM use cases don't need GPT-4 quality
- Most companies would accept 85% quality for 80% cost reduction
- Customers already accept trade-offs in other domains (S3 storage tiers, instance types)
- **Implication:** "Good enough, cheap, transparent" beats "perfect, expensive, black-box"

### Thesis 3: Environmental Accountability Is Real
- EU AI Act mandates energy transparency
- Corporate ESG commitments require carbon tracking
- Customers want to publish real numbers (not greenwashing)
- **Implication:** Carbon tracking as a feature is defensible, differentiating, and valuable

### Thesis 4: Open-Source Models Will Dominate Commodity Tasks
- Mistral 7B ≈ GPT-3.5 on most benchmarks
- Llama 13B ≈ Claude Instant on many tasks
- Open-source improves 10x faster than proprietary (community + scale)
- **Implication:** You don't need proprietary models; you need smart routing

### Thesis 5: This Is a Sustainable Business Model
- Gross margins 75%+ (unlike SaaS at 60–70%)
- Unit economics improve with scale (routing gets smarter)
- LTV:CAC >10:1 by Year 2 (extremely healthy)
- No R&D cost to train models (use open-source)
- **Implication:** Path to profitability is clear; capital-efficient scaling

---

## Part 10: Success Metrics & Key Milestones

### Year 1 Milestones

| Milestone | Timeline | Success Criteria |
|-----------|----------|------------------|
| **MVP shipped** | Month 3 | API live, 3 models, basic routing |
| **First customer paid** | Month 4 | 1 paying customer, positive feedback |
| **10 customers** | Month 6 | $40K ARR, 50%+ satisfaction score |
| **50 customers** | Month 12 | $200K ARR, 4.5/5 NPS, <10% churn |
| **Series A ready** | Month 12–13 | Path to $1M ARR clear, $1M+ runway |

### Key Performance Indicators (KPIs)

**Product Metrics:**
- **P95 latency:** <1s (goal)
- **Model misclassification rate:** <5% (acceptable)
- **Output quality satisfaction:** >4.2/5 (customer survey)
- **Uptime:** >99.5% (SLA)
- **Average CO2 per request:** <10g (vs 50g GPT-4 baseline)

**Business Metrics:**
- **Monthly customer growth rate:** 15–20%
- **Net dollar retention:** >110% (expansion revenue)
- **Churn rate:** <8% monthly
- **CAC payback period:** <12 months
- **Gross margin:** >75%

**Environmental Metrics:**
- **Monthly CO2e avoided:** 100+ tons (by Month 6)
- **Customer ESG impact reports generated:** 1+ per month
- **Transparency audit score:** >90% (third-party)

---

## Part 11: The Elevator Pitch

**30-second version:**
```
"EcoLLM is the carbon-aware, open-source LLM platform 
that delivers ChatGPT-like capabilities at 10x lower cost 
and 90% less environmental impact.

We route requests to the smallest, fastest, most efficient 
model that can complete the task—so you pay for what you need, 
not what's popular.

Think: AWS for LLMs (cost optimization + transparency) 
not OpenAI (one-size-fits-all)."
```

**2-minute version:**
```
We're building EcoLLM, a platform that solves two problems:

1. Cost: Mid-market SaaS companies spend $50K–$500K/year 
   on OpenAI/Claude APIs. 70% of that is wasted on 
   capability they don't use. We cut costs 70–80% 
   by intelligently routing to the right model.

2. Environment: LLM inference is energy-intensive and 
   invisible. We measure CO2 per request, expose it to 
   customers, and optimize routing for both cost AND carbon.

The insight: Open-source models (Mistral, Llama, Phi) now 
match proprietary models on commodity tasks. Quantization + 
routing gets you 85% of GPT-4 quality at 20% of the cost.

Target: Mid-market SaaS (CTOs, CFOs, ESG teams). TAM is ~50K 
companies. We'll launch with 5–10 customers and aim for 500 
customers by Year 3.

Unit economics are strong: 75% gross margins, LTV:CAC >10:1, 
path to profitability in Year 3. We're raising $2M to get 
to $1M ARR and Series A readiness."
```

---

## Part 12: Final Founder Mandates

These are **non-negotiable** principles that guide every decision:

### Mandate 1: Efficiency Drives All Decisions
- Architecture choice? Pick the one that uses least energy.
- Feature request? Does it increase compute? If yes, reject or defer.
- Model selection? Smaller model always wins unless quality gap is >10%.
- This is not negotiable. This is your brand.

### Mandate 2: Real Transparency > Marketing
- Publish actual numbers: Cost, latency, quality, CO2
- Don't hide failures; publish benchmarks honestly
- Make carbon tracking third-party verifiable
- If you can't measure it, don't claim it

### Mandate 3: Open-Source Foundation
- Never lock customers into proprietary models
- Publish core IP (routing framework) as open-source
- Build community, not walled garden
- Your competitive advantage is the service/optimization, not the models

### Mandate 4: Unit Economics Always
- Know your CAC, LTV, and payback period every month
- If unit economics don't work, no amount of growth helps
- Prioritize profitable customers over vanity metrics
- If a customer costs more to acquire than their LTV, say no

### Mandate 5: Ship Fast, Iterate on Data
- MVP by Month 3. Don't over-engineer.
- Get customers in Month 4. Don't perfect first.
- Iterate on real usage data, not assumptions.
- Move from 10 customers → 50 customers → 100 customers. Learn at each step.

### Mandate 6: Environmental Accountability
- This is your brand promise. Don't break it for short-term gains.
- If a feature improves revenue but increases carbon per request, reject it.
- Publish annual environmental impact report (transparency builds trust).
- Make the hard trade-offs: sometimes efficiency wins over convenience.

---

## Conclusion

**EcoLLM is a venture-scale business.** It solves a real problem (cost + environmental impact) for a defined market (mid-market SaaS). It's technically feasible with existing open-source models and infrastructure. The unit economics are strong. The competitive differentiation is defensible.

**What makes it work:**
1. **Problem is real:** Customers spend $1B+/year on LLM APIs and want to cut costs
2. **Solution is simple:** Route to smaller models using open-source
3. **Timing is right:** Open models are competitive, infrastructure is ready, environmental awareness is rising
4. **Team can build it:** This doesn't require training new models; it requires systems engineering, which is solvable
5. **Market is large:** 50K+ companies could benefit; $5B+ TAM

**Funding ask:** $2M seed  
**Timeline to Series A:** 12–16 months  
**Path to profitability:** Year 3  
**Exit potential:** Acq-hire (by OpenAI, Anthropic, major cloud provider), or independent $500M+ SaaS company

**This is fundable. Build it.**

---

## Appendices

### Appendix A: Detailed Routing Algorithm Pseudocode

```python
class EcoLLMRouter:
    """
    Environment-first routing engine for open-source LLMs.
    Core IP of EcoLLM platform.
    """
    
    def __init__(self):
        self.models = {
            "phi_3": {
                "size": "3.8B",
                "quantization": "q4",
                "latency_ms": 80,
                "energy_kwh": 0.00001,
                "cost_usd": 0.0001,
                "quality_benchmark": 0.65,
            },
            "mistral_7b": {
                "size": "7B",
                "quantization": "q4",
                "latency_ms": 400,
                "energy_kwh": 0.00008,
                "cost_usd": 0.0005,
                "quality_benchmark": 0.85,
            },
            "llama_13b": {
                "size": "13B",
                "quantization": "q4",
                "latency_ms": 800,
                "energy_kwh": 0.00015,
                "cost_usd": 0.001,
                "quality_benchmark": 0.92,
            },
            "llama_70b": {
                "size": "70B",
                "quantization": "q8",
                "latency_ms": 3000,
                "energy_kwh": 0.0008,
                "cost_usd": 0.005,
                "quality_benchmark": 0.98,
            },
        }
        
        self.task_classifier = TaskClassifier()
        self.telemetry = TelemetryTracker()
    
    def route(self, prompt: str, customer_id: str, constraints: dict = None) -> RoutingDecision:
        """
        Main routing function.
        
        Args:
            prompt: User prompt
            customer_id: Customer identifier (for personalization)
            constraints: Optional {min_quality, max_latency, max_energy, cost_budget}
        
        Returns:
            RoutingDecision {model, energy_kwh, cost_usd, quality_score, confidence}
        """
        
        # Step 1: Optimize prompt
        optimized_prompt = self.optimize_prompt(prompt)
        
        # Step 2: Classify task
        task_type, complexity = self.task_classifier.classify(optimized_prompt)
        
        # Step 3: Generate candidates
        candidates = self.generate_candidates(task_type, complexity)
        
        # Step 4: Score each candidate
        scores = {}
        for model_name in candidates:
            score = self.score_model(
                model_name, 
                task_type, 
                customer_constraints=constraints
            )
            scores[model_name] = score
        
        # Step 5: Select best model
        best_model = max(scores, key=scores.get)
        
        # Step 6: Apply hard constraints
        if self.violates_constraints(best_model, constraints):
            best_model = self.find_constrained_alternative(
                best_model, 
                constraints
            )
        
        # Step 7: Get fallback
        fallback_model = self.get_fallback_model(best_model)
        
        # Step 8: Return decision
        decision = RoutingDecision(
            model=best_model,
            fallback=fallback_model,
            energy_kwh=self.models[best_model]["energy_kwh"],
            cost_usd=self.models[best_model]["cost_usd"],
            quality_score=self.models[best_model]["quality_benchmark"],
            confidence=scores[best_model],
            telemetry_id=self.telemetry.generate_id(),
        )
        
        # Log for analytics
        self.telemetry.log_routing_decision(customer_id, decision)
        
        return decision
    
    def optimize_prompt(self, prompt: str) -> str:
        """
        Improve prompt clarity without changing intent.
        
        Rule-based optimizations:
        1. Remove redundancy
        2. Add structural guidance (e.g., "provide steps", "be concise")
        3. Clarify intent
        
        Fallback to Phi-3 if confidence < 0.7
        """
        # Rule-based (fast path)
        optimized = self._apply_prompt_rules(prompt)
        
        # Confidence check
        confidence = self._estimate_optimization_confidence(prompt)
        
        if confidence < 0.7:
            # Use Phi-3 to refine
            optimized = self._refine_with_phi3(prompt)
        
        return optimized
    
    def score_model(self, model_name: str, task_type: str, customer_constraints: dict = None) -> float:
        """
        Score a model candidate.
        
        Weights:
        - 40% energy (PRIMARY CONSTRAINT)
        - 30% cost
        - 20% quality
        - 10% latency
        
        Returns: score in [0, 1]
        """
        
        model = self.models[model_name]
        
        # Normalize each metric
        energy_score = 1 - (model["energy_kwh"] / 0.001)  # max 0.001 kwh
        cost_score = 1 - (model["cost_usd"] / 0.01)        # max 0.01 usd
        quality_score = model["quality_benchmark"]         # 0-1
        latency_score = 1 - (model["latency_ms"] / 5000)   # max 5s
        
        # Apply task-specific quality threshold
        quality_threshold = self._get_quality_threshold(task_type)
        if quality_score < quality_threshold:
            quality_score = quality_score * 0.5  # penalty
        
        # Weighted sum
        score = (
            0.40 * energy_score +
            0.30 * cost_score +
            0.20 * quality_score +
            0.10 * latency_score
        )
        
        # Apply customer preferences
        if customer_constraints:
            if customer_constraints.get("prefer_speed"):
                score += 0.05 * latency_score
                score -= 0.05 * energy_score
            if customer_constraints.get("prefer_cost"):
                score += 0.05 * cost_score
            if customer_constraints.get("prefer_quality"):
                score += 0.05 * quality_score
        
        return score
    
    def generate_candidates(self, task_type: str, complexity: int) -> list:
        """
        Generate candidate models based on task type.
        
        - Simple (complexity 1–3): [Phi-3, Mistral 7B]
        - Medium (complexity 4–6): [Mistral 7B, Llama 13B]
        - Hard (complexity 7–9): [Llama 13B, Llama 70B]
        - Specialized (complexity 10): [Llama 70B, fallback to external API]
        """
        
        if complexity <= 3:
            return ["phi_3", "mistral_7b"]
        elif complexity <= 6:
            return ["mistral_7b", "llama_13b"]
        elif complexity <= 9:
            return ["llama_13b", "llama_70b"]
        else:
            return ["llama_70b"]  # fallback to external if needed
    
    def violates_constraints(self, model_name: str, constraints: dict) -> bool:
        """Check if model violates customer constraints."""
        if not constraints:
            return False
        
        model = self.models[model_name]
        
        if constraints.get("min_quality") and model["quality_benchmark"] < constraints["min_quality"]:
            return True
        
        if constraints.get("max_latency") and model["latency_ms"] > constraints["max_latency"]:
            return True
        
        if constraints.get("max_energy") and model["energy_kwh"] > constraints["max_energy"]:
            return True
        
        if constraints.get("max_cost") and model["cost_usd"] > constraints["max_cost"]:
            return True
        
        return False
    
    def get_fallback_model(self, primary_model: str) -> str:
        """Get fallback model (larger, if primary fails)."""
        fallback_chain = {
            "phi_3": "mistral_7b",
            "mistral_7b": "llama_13b",
            "llama_13b": "llama_70b",
            "llama_70b": "openai_gpt4",  # external API as last resort
        }
        return fallback_chain.get(primary_model)


class TaskClassifier:
    """Classify incoming requests to task types."""
    
    def classify(self, prompt: str) -> tuple:
        """
        Returns: (task_type, complexity_score)
        
        task_type: str in {simple, medium, hard, specialized}
        complexity_score: int in [1–10]
        """
        
        # Keyword-based heuristics
        complexity = self._heuristic_complexity(prompt)
        task_type = self._heuristic_task_type(prompt)
        
        # Confidence check
        confidence = self._estimate_confidence(prompt)
        
        if confidence < 0.6:
            # Use language model for disambiguation
            # (optional: use Phi-3 for uncertain cases)
            task_type, complexity = self._classify_with_lm(prompt)
        
        return task_type, complexity
    
    def _heuristic_complexity(self, prompt: str) -> int:
        """Fast heuristic: estimate complexity (1–10)."""
        
        score = 1
        
        # Length heuristic
        if len(prompt) > 500:
            score += 2
        if len(prompt) > 1000:
            score += 2
        
        # Keywords heuristic
        hard_keywords = ["reasoning", "explain", "debug", "design", "architecture"]
        for keyword in hard_keywords:
            if keyword.lower() in prompt.lower():
                score += 3
        
        medium_keywords = ["code", "write", "generate", "summarize"]
        for keyword in medium_keywords:
            if keyword.lower() in prompt.lower():
                score += 1
        
        return min(score, 10)
    
    def _heuristic_task_type(self, prompt: str) -> str:
        """Classify task type."""
        
        simple_keywords = ["classify", "match", "extract", "detect", "find"]
        medium_keywords = ["write", "code", "generate", "explain"]
        hard_keywords = ["design", "architect", "reason about", "debug complex"]
        
        for keyword in simple_keywords:
            if keyword.lower() in prompt.lower():
                return "simple"
        
        for keyword in medium_keywords:
            if keyword.lower() in prompt.lower():
                return "medium"
        
        for keyword in hard_keywords:
            if keyword.lower() in prompt.lower():
                return "hard"
        
        return "medium"  # default


class TelemetryTracker:
    """Track routing decisions and outcomes for continuous improvement."""
    
    def log_routing_decision(self, customer_id: str, decision: RoutingDecision):
        """Log a routing decision for later analysis."""
        # Store in database for analytics
        # Used to: improve routing, track ROI, monitor quality
        pass
    
    def generate_id(self) -> str:
        """Generate unique telemetry ID for this request."""
        return f"eco_{uuid.uuid4()}"
```

### Appendix B: Energy Calculation Formula

```python
def calculate_energy_per_request(
    model_name: str,
    input_tokens: int,
    output_tokens: int,
    batch_size: int,
    gpu_type: str,
    duration_ms: float,
) -> float:
    """
    Calculate energy consumption for a single request.
    
    Energy = (GPU Power Draw × Inference Time) / Batch Size
    
    Returns: energy in kwh
    """
    
    # GPU power draw (watts)
    gpu_power_map = {
        "l4": 35,          # Low power
        "l40s": 48,        # Medium power
        "a100": 250,       # High power
        "h100": 350,       # Very high power
    }
    
    gpu_power_watts = gpu_power_map.get(gpu_type, 50)
    
    # Inference time
    inference_time_hours = duration_ms / 1000 / 3600
    
    # Raw energy
    raw_energy_wh = gpu_power_watts * inference_time_hours
    
    # Amortize across batch
    amortized_energy_wh = raw_energy_wh / batch_size
    
    # Add datacenter overhead (PUE = 1.3)
    total_energy_wh = amortized_energy_wh * 1.3
    
    # Convert to kwh
    total_energy_kwh = total_energy_wh / 1000
    
    return total_energy_kwh


def calculate_co2e_per_request(
    energy_kwh: float,
    grid_carbon_intensity_gco2_per_kwh: int = 450,
) -> float:
    """
    Calculate CO2 equivalent for a request.
    
    CO2e = Energy (kwh) × Grid Carbon Intensity (g CO2 / kwh)
    
    Grid carbon intensity varies by region and time:
    - US avg: 450 g CO2 / kwh
    - Europe avg: 300 g CO2 / kwh (more renewable)
    - Coal-heavy regions: 800+ g CO2 / kwh
    
    Returns: CO2e in grams
    """
    
    co2e_grams = energy_kwh * grid_carbon_intensity_gco2_per_kwh * 1000
    return co2e_grams
```

### Appendix C: Customer Acquisition Script

```
Subject: "We cut LLM costs 80% for [Company Name]"

Hi [Name],

I noticed [Company] uses OpenAI APIs (from your [job posting / tech blog / GitHub]).

We built EcoLLM—a routing system that delivers 85% of ChatGPT quality at 
20% of the cost. No vendor lock-in, carbon-aware, open-source foundation.

Typical ROI:
- Companies your size save $30K–$200K/year on LLM spend
- Process: We analyze your current workload, benchmark quality, show savings
- Timeline: 2-week trial, no credit card required

Worth 15 minutes? I can show a live cost/CO2 comparison for your workload.

[Link to cost calculator] 
or reply with your current monthly OpenAI spend, and I'll calculate 
your custom savings.

Thanks,
[Name]
EcoLLM

---

Follow-up (if no response in 3 days):

Hi [Name],

Quick follow-up—I shared a tool to calculate your LLM cost savings.

Most companies are surprised: 70–80% cost reduction is realistic for 
commodity tasks (classification, summarization, basic generation).

If you get 5 min this week, curious to see your numbers.

[Link to calculator]

Best,
[Name]

---

If customer clicks calculator or replies:

Great—here's what I'm seeing in your usage:

[Custom analysis]

Next step: 2-week free trial of the full platform. We'll:
1. Route your actual requests (side-by-side with OpenAI)
2. Show you per-request cost savings + quality metrics
3. Measure carbon impact

Takes 30 min to set up. Let me know when you'd like to start.

[Calendly link]
```

---

**END OF DOCUMENT**

This is your founder brief. Print it. Memorize the key theses. Build against it.

Questions? Build. Don't overthink.

Good luck.
