# EcoLLM Carbon Methodology

## Overview

EcoLLM tracks the carbon footprint of every inference request. This document explains how energy consumption and CO2 emissions are measured and reported.

---

## Energy Measurement

**File:** `apps/api/internal/carbon/estimator.go`

Energy per request is estimated using:

```
energy_kwh = (gpu_tdp_watts × inference_time_seconds) / 3_600_000
```

Where:
- `gpu_tdp_watts` is the rated TDP of the GPU running the model (from `model-configs/*.yaml`)
- `inference_time_seconds` is measured end-to-end at the inference gateway

### GPU TDP values used

| Model | GPU | TDP |
|-------|-----|-----|
| Phi-3 3.8B | NVIDIA A10G | 150W |
| Mistral 7B | NVIDIA A10G | 150W |
| Llama 13B | NVIDIA L40S | 300W |
| Llama 70B | NVIDIA A100 (×2) | 400W × 2 |

These are conservative estimates. Actual draw is typically 60–80% of TDP at inference load.

---

## Carbon Intensity

**File:** `apps/api/internal/carbon/grid.go`

Grid carbon intensity (gCO2e/kWh) is fetched hourly from the Electricity Maps API for the deployment region. The value reflects the real-time marginal emissions factor — i.e., how carbon-intensive the electricity grid is at the time of inference.

Baseline regional values are seeded from `db/seeds/grid_data.sql`.

---

## CO2e Calculation

```
co2e_grams = energy_kwh × grid_carbon_intensity_gco2_per_kwh
```

This is a Scope 2 emissions figure (electricity consumption). It does not include hardware manufacturing (Scope 3).

---

## Billing Formula

Carbon tracking feeds directly into billing:

```
cost_usd = (co2e_grams × $0.001) + (request_count × $0.0001)
```

Volume discounts:
- ≥1M requests/month: 15% discount
- ≥10M requests/month: 25% discount

---

## GPT-4 Baseline Comparison

EcoLLM compares its CO2e against a GPT-4 Turbo baseline of **36 gCO2e per request** (derived from published estimates of OpenAI's data centre energy mix and average response length).

This is displayed in the dashboard as "CO2e saved vs GPT-4 equivalent."

---

## Limitations

1. GPU TDP-based estimation is an approximation. Actual per-request energy varies with batch size, prompt length, and GPU load.
2. Grid intensity is sampled hourly, not per-second. Short inference bursts may use stale intensity data.
3. Network transmission energy is not included.
4. Cooling overhead (PUE) is not modelled — typical data centre PUE of 1.2–1.4 would increase estimates by 20–40%.

A future version will integrate NVIDIA NVML for real-time per-GPU power readings.
