package carbon

import "context"

const (
	// DefaultPUEMultiplier is the Power Usage Effectiveness factor per Green Grid standard.
	// Accounts for datacenter cooling and infrastructure overhead.
	DefaultPUEMultiplier = 1.3
)

// Per-model inference power draw in watts (from model-configs, architecture doc Section 8.3).
var modelPowerWatts = map[string]float64{
	"phi_3":      35,
	"mistral_7b": 45,
	"llama_13b":  48,
	"llama_70b":  250,
}

// Estimator calculates energy and CO2 estimates for inference requests.
// It uses published power baselines from model-configs and regional grid intensity.
type Estimator struct {
	gridRegion string
	gridSvc    *GridService // nil → static fallback
}

func NewEstimator(gridRegion string) *Estimator {
	return &Estimator{gridRegion: gridRegion}
}

// NewEstimatorWithGrid creates an Estimator backed by a live GridService.
// When the GridService is non-nil, GridIntensity() resolves via cache → API → static.
func NewEstimatorWithGrid(gridRegion string, gs *GridService) *Estimator {
	return &Estimator{gridRegion: gridRegion, gridSvc: gs}
}

// EstimateEnergy calculates total energy in kWh for a single request.
//
// Formula (from architecture spec Section 7, Layer 6 of knowledge base):
//
//	energy_kwh = (gpu_power_watts × inference_time_hours / batch_size) × pue_multiplier / 1000
func (e *Estimator) EstimateEnergy(modelName string, latencyMs, batchSize int) float64 {
	if batchSize <= 0 {
		batchSize = 1
	}

	powerWatts := modelPowerWatts[modelName]
	if powerWatts == 0 {
		powerWatts = 50 // safe default
	}

	inferenceTimeHours := float64(latencyMs) / 1000.0 / 3600.0
	rawEnergyWh := powerWatts * inferenceTimeHours
	amortizedWh := rawEnergyWh / float64(batchSize)
	totalWh := amortizedWh * DefaultPUEMultiplier
	return totalWh / 1000.0 // convert Wh → kWh
}

// EstimateCO2 converts energy in kWh to CO2e grams using regional grid intensity.
//
// Formula:
//
//	co2e_grams = energy_kwh × grid_carbon_intensity_gco2_per_kwh × 1000
func (e *Estimator) EstimateCO2(energyKwh float64) float64 {
	intensity, _ := e.GridIntensity()
	return energyKwh * intensity * 1000
}

// GridIntensity returns the current grid carbon intensity and region code.
// When a GridService is configured, it uses the live cache → API → static resolution.
func (e *Estimator) GridIntensity() (float64, string) {
	if e.gridSvc != nil {
		data := e.gridSvc.GetIntensity(context.Background(), e.gridRegion)
		return data.IntensityGCO2, data.Region
	}
	return GridIntensityForRegion(e.gridRegion), e.gridRegion
}

// GPUPowerForModel returns the inference GPU power draw in watts for a model.
// Returns 50 W for unknown models as a safe default.
func GPUPowerForModel(modelName string) float64 {
	if p, ok := modelPowerWatts[modelName]; ok {
		return p
	}
	return 50
}

// CalculateEnergy computes a full EnergyMeasurement from the given inputs.
//
// Formula (SCI Spec / Architecture Appendix B):
//
//	inferenceEnergyWh = (gpu_power_watts × inference_time_hours) / batch_size
//	totalEnergyWh     = inferenceEnergyWh × pue_multiplier
//	totalEnergyKwh    = totalEnergyWh / 1000
func CalculateEnergy(input EnergyInput) EnergyMeasurement {
	pue := input.PUEMultiplier
	if pue <= 0 {
		pue = DefaultPUEMultiplier
	}
	batch := input.BatchSize
	if batch <= 0 {
		batch = 1
	}

	power := GPUPowerForModel(input.ModelName)
	inferenceHours := float64(input.InferenceMs) / 3_600_000.0
	inferenceWh := power * inferenceHours / float64(batch)
	totalWh := inferenceWh * pue

	return EnergyMeasurement{
		ModelName:         input.ModelName,
		GPUPowerWatts:     power,
		InferenceTimeMs:   input.InferenceMs,
		BatchSize:         batch,
		InferenceEnergyWh: inferenceWh,
		PUEMultiplier:     pue,
		TotalEnergyWh:     totalWh,
		TotalEnergyKwh:    totalWh / 1000.0,
	}
}
