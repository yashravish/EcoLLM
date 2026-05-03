package carbon

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnergyRepository persists and retrieves energy measurements.
type EnergyRepository interface {
	InsertMeasurement(ctx context.Context, m *EnergyMeasurement) error
	GetByRequestID(ctx context.Context, requestID string) (*EnergyMeasurement, error)
}

// CarbonRepository persists and retrieves CO2e estimates.
type CarbonRepository interface {
	InsertEstimate(ctx context.Context, e *CarbonEstimate) error
	GetByRequestID(ctx context.Context, requestID string) (*CarbonEstimate, error)
}

// ── Energy ────────────────────────────────────────────────────────────────────

type pgEnergyRepository struct {
	pool *pgxpool.Pool
}

func NewEnergyRepository(pool *pgxpool.Pool) EnergyRepository {
	return &pgEnergyRepository{pool: pool}
}

func (r *pgEnergyRepository) InsertMeasurement(ctx context.Context, m *EnergyMeasurement) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO energy_measurements
			(request_id, org_id, model_name,
			 gpu_power_watts, inference_time_ms, batch_size,
			 inference_energy_wh, pue_multiplier, total_energy_wh, total_energy_kwh)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		m.RequestID, m.OrgID, m.ModelName,
		m.GPUPowerWatts, m.InferenceTimeMs, m.BatchSize,
		m.InferenceEnergyWh, m.PUEMultiplier, m.TotalEnergyWh, m.TotalEnergyKwh,
	)
	return err
}

func (r *pgEnergyRepository) GetByRequestID(ctx context.Context, requestID string) (*EnergyMeasurement, error) {
	var m EnergyMeasurement
	err := r.pool.QueryRow(ctx, `
		SELECT id, request_id, org_id, model_name,
		       gpu_power_watts, inference_time_ms, batch_size,
		       inference_energy_wh, pue_multiplier, total_energy_wh, total_energy_kwh,
		       created_at
		FROM energy_measurements
		WHERE request_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, requestID).Scan(
		&m.ID, &m.RequestID, &m.OrgID, &m.ModelName,
		&m.GPUPowerWatts, &m.InferenceTimeMs, &m.BatchSize,
		&m.InferenceEnergyWh, &m.PUEMultiplier, &m.TotalEnergyWh, &m.TotalEnergyKwh,
		&m.CreatedAt,
	)
	return &m, err
}

// ── Carbon ────────────────────────────────────────────────────────────────────

type pgCarbonRepository struct {
	pool *pgxpool.Pool
}

func NewCarbonRepository(pool *pgxpool.Pool) CarbonRepository {
	return &pgCarbonRepository{pool: pool}
}

func (r *pgCarbonRepository) InsertEstimate(ctx context.Context, e *CarbonEstimate) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO carbon_estimates
			(request_id, energy_measurement_id, grid_region,
			 grid_carbon_intensity, carbon_data_source, co2e_grams,
			 gpt4_equivalent_co2e, savings_percent)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		e.RequestID, optString(e.EnergyMeasurementID), e.GridRegion,
		e.GridCarbonIntensity, optString(e.CarbonDataSource), e.CO2eGrams,
		e.GPT4EquivalentCO2e, e.SavingsPercent,
	)
	return err
}

func (r *pgCarbonRepository) GetByRequestID(ctx context.Context, requestID string) (*CarbonEstimate, error) {
	var e CarbonEstimate
	err := r.pool.QueryRow(ctx, `
		SELECT id, request_id,
		       COALESCE(energy_measurement_id::text, ''),
		       grid_region, grid_carbon_intensity,
		       COALESCE(carbon_data_source, ''), co2e_grams,
		       COALESCE(gpt4_equivalent_co2e, 0), COALESCE(savings_percent, 0),
		       created_at
		FROM carbon_estimates
		WHERE request_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, requestID).Scan(
		&e.ID, &e.RequestID, &e.EnergyMeasurementID,
		&e.GridRegion, &e.GridCarbonIntensity,
		&e.CarbonDataSource, &e.CO2eGrams,
		&e.GPT4EquivalentCO2e, &e.SavingsPercent,
		&e.CreatedAt,
	)
	return &e, err
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
