package calculator

import (
	"context"

	"github.com/ecollm/carbon-service/internal/grid"
)

const pueMultiplier = 1.3 // Green Grid standard

// modelPowerWatts is peak GPU power draw per model (same values as API estimator).
var modelPowerWatts = map[string]float64{
	"phi_3":      35,
	"mistral_7b": 45,
	"llama_13b":  48,
	"llama_70b":  250,
}

// EstimateRequest is the JSON body for POST /v1/estimate.
type EstimateRequest struct {
	ModelName   string  `json:"model_name"`
	LatencyMs   int     `json:"latency_ms"`
	BatchSize   int     `json:"batch_size"`
	Region      string  `json:"region"`
}

// EstimateResult is returned by POST /v1/estimate.
type EstimateResult struct {
	EnergyKwh    float64 `json:"energy_kwh"`
	CO2eGrams    float64 `json:"co2e_grams"`
	Intensity    float64 `json:"grid_intensity_gco2_per_kwh"`
	IntensitySource string `json:"intensity_source"`
	// Savings vs GPT-4 baseline (≈0.0023 kWh/request, US-EAST = ~1.035 gCO2e)
	GPT4CO2eGrams float64 `json:"gpt4_equivalent_co2e_grams"`
	SavingsPercent float64 `json:"savings_percent"`
}

// Calculator estimates energy and CO2 for a completed inference request.
type Calculator struct {
	grid *grid.Client
}

func New(g *grid.Client) *Calculator {
	return &Calculator{grid: g}
}

func (c *Calculator) Estimate(ctx context.Context, req EstimateRequest) (EstimateResult, error) {
	if req.BatchSize <= 0 {
		req.BatchSize = 1
	}
	if req.Region == "" {
		req.Region = "US-EAST"
	}

	powerW := modelPowerWatts[req.ModelName]
	if powerW == 0 {
		powerW = 50 // conservative unknown-model fallback
	}

	inferenceTimeH := float64(req.LatencyMs) / 3_600_000.0
	energyKwh := (powerW * inferenceTimeH / float64(req.BatchSize)) * pueMultiplier / 1000.0

	intensity, source, err := c.grid.Intensity(ctx, req.Region)
	if err != nil {
		return EstimateResult{}, err
	}
	co2eGrams := energyKwh * intensity * 1000.0

	// GPT-4 baseline: ~0.0023 kWh/request at US-EAST intensity
	gpt4Energy := 0.0023
	gpt4CO2e := gpt4Energy * intensity * 1000.0
	savingsPct := 0.0
	if gpt4CO2e > 0 {
		savingsPct = (gpt4CO2e - co2eGrams) / gpt4CO2e * 100.0
	}

	return EstimateResult{
		EnergyKwh:       energyKwh,
		CO2eGrams:       co2eGrams,
		Intensity:       intensity,
		IntensitySource: source,
		GPT4CO2eGrams:   gpt4CO2e,
		SavingsPercent:  savingsPct,
	}, nil
}
