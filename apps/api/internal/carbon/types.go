package carbon

import "time"

// EnergyInput carries the inputs for CalculateEnergy.
// PUEMultiplier of 0 means use DefaultPUEMultiplier.
type EnergyInput struct {
	ModelName     string
	InferenceMs   int64
	BatchSize     int
	PUEMultiplier float64
}

// EnergyMeasurement is the detailed record of energy for one inference request.
// Matches the energy_measurements table schema (migration 006).
type EnergyMeasurement struct {
	ID                string
	RequestID         string
	OrgID             string
	ModelName         string
	GPUPowerWatts     float64
	InferenceTimeMs   int64
	BatchSize         int
	InferenceEnergyWh float64
	PUEMultiplier     float64
	TotalEnergyWh     float64
	TotalEnergyKwh    float64
	CreatedAt         time.Time
}

// CarbonEstimate holds the CO2e estimate derived from an EnergyMeasurement.
// Matches the carbon_estimates table schema (migration 007).
type CarbonEstimate struct {
	ID                  string
	RequestID           string
	EnergyMeasurementID string // empty → NULL in DB
	GridRegion          string
	GridCarbonIntensity float64 // gCO2/kWh
	CarbonDataSource    string  // "electricity_maps" | "static_fallback"
	CO2eGrams           float64
	GPT4EquivalentCO2e  float64
	SavingsPercent      float64
	CreatedAt           time.Time
}

// GridData is a snapshot of grid intensity used for CO2 calculations.
type GridData struct {
	Region        string
	IntensityGCO2 float64 // gCO2/kWh
	DataSource    string  // "electricity_maps" | "static"
	UpdatedAt     time.Time
	ExpiresAt     time.Time
}
