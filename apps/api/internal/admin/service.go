package admin

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles admin business logic: model registry CRUD and system metrics.
type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// Model is a typed row from the model_registry table.
type Model struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	DisplayName       string    `json:"display_name"`
	Runtime           string    `json:"runtime"`
	Quantization      string    `json:"quantization"`
	Status            string    `json:"status"`
	HealthStatus      string    `json:"health_status"`
	EndpointURL       string    `json:"endpoint_url"`
	EnergyPerTokenKwh float64   `json:"energy_per_token_kwh"`
	CostPerTokenUSD   float64   `json:"cost_per_token_usd"`
	QualityScore      float64   `json:"quality_score"`
	LatencyP50Ms      float64   `json:"latency_p50_ms"`
	LatencyP95Ms      float64   `json:"latency_p95_ms"`
	FailureRate       float64   `json:"failure_rate"`
	CreatedAt         time.Time `json:"created_at"`
}

// SystemMetrics is the payload for GET /admin/metrics.
type SystemMetrics struct {
	Requests24h    int64   `json:"requests_24h"`
	RequestsTotal  int64   `json:"requests_total"`
	EnergyKwh24h   float64 `json:"energy_kwh_24h"`
	CO2eGrams24h   float64 `json:"co2e_grams_24h"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P95LatencyMs   float64 `json:"p95_latency_ms"`
	ActiveOrgs     int64   `json:"active_orgs"`
	ActiveModels   int64   `json:"active_models"`
}

// CarbonDay is one day of aggregated carbon data.
type CarbonDay struct {
	Date           string  `json:"date"`
	TotalCO2eGrams float64 `json:"total_co2e_grams"`
	TotalEnergyKwh float64 `json:"total_energy_kwh"`
	TotalRequests  int64   `json:"total_requests"`
}

// RouteEntry is a model with its current routing state.
type RouteEntry struct {
	ModelID   string  `json:"model_id"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	QScore    float64 `json:"quality_score"`
	EnergyKwh float64 `json:"energy_per_token_kwh"`
	CostUSD   float64 `json:"cost_per_token_usd"`
	P95Ms     float64 `json:"latency_p95_ms"`
}

// CreateModelInput carries fields for a new model registry entry.
type CreateModelInput struct {
	Name              string  `json:"name"`
	DisplayName       string  `json:"display_name"`
	Runtime           string  `json:"runtime"`
	Quantization      string  `json:"quantization"`
	EndpointURL       string  `json:"endpoint_url"`
	EnergyPerTokenKwh float64 `json:"energy_per_token_kwh"`
	CostPerTokenUSD   float64 `json:"cost_per_token_usd"`
	QualityScore      float64 `json:"quality_score"`
	LatencyP50Ms      float64 `json:"latency_p50_ms"`
	LatencyP95Ms      float64 `json:"latency_p95_ms"`
}

// UpdateModelInput carries mutable model fields (all optional).
type UpdateModelInput struct {
	Status            *string  `json:"status"`
	HealthStatus      *string  `json:"health_status"`
	EndpointURL       *string  `json:"endpoint_url"`
	EnergyPerTokenKwh *float64 `json:"energy_per_token_kwh"`
	CostPerTokenUSD   *float64 `json:"cost_per_token_usd"`
	QualityScore      *float64 `json:"quality_score"`
	LatencyP50Ms      *float64 `json:"latency_p50_ms"`
	LatencyP95Ms      *float64 `json:"latency_p95_ms"`
}

// GetSystemMetrics returns system-wide operational metrics.
func (s *Service) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	var m SystemMetrics

	// 24-hour window from usage_aggregates
	err := s.db.QueryRow(ctx,
		`SELECT
		    COALESCE(SUM(total_requests), 0),
		    COALESCE(SUM(total_energy_kwh), 0),
		    COALESCE(SUM(total_co2e_grams), 0),
		    COALESCE(AVG(avg_latency_ms), 0),
		    COALESCE(AVG(p95_latency_ms), 0)
		 FROM usage_aggregates
		 WHERE period_start >= NOW() - INTERVAL '24 hours'`,
	).Scan(&m.Requests24h, &m.EnergyKwh24h, &m.CO2eGrams24h, &m.AvgLatencyMs, &m.P95LatencyMs)
	if err != nil {
		return nil, err
	}

	// All-time totals
	if err := s.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_requests), 0) FROM usage_aggregates`,
	).Scan(&m.RequestsTotal); err != nil {
		return nil, err
	}

	// Active org and model counts
	if err := s.db.QueryRow(ctx,
		`SELECT
		    (SELECT COUNT(*) FROM organizations),
		    (SELECT COUNT(*) FROM model_registry WHERE status = 'active')`,
	).Scan(&m.ActiveOrgs, &m.ActiveModels); err != nil {
		return nil, err
	}

	return &m, nil
}

// ListModels returns all models from the registry.
func (s *Service) ListModels(ctx context.Context) ([]Model, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, display_name, runtime, quantization, status, health_status,
		        COALESCE(endpoint_url,''), COALESCE(energy_per_token_kwh,0),
		        COALESCE(cost_per_token_usd,0), COALESCE(quality_score,0),
		        COALESCE(latency_p50_ms,0), COALESCE(latency_p95_ms,0),
		        COALESCE(failure_rate,0), created_at
		 FROM model_registry ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []Model
	for rows.Next() {
		var m Model
		if err := rows.Scan(
			&m.ID, &m.Name, &m.DisplayName, &m.Runtime, &m.Quantization,
			&m.Status, &m.HealthStatus, &m.EndpointURL,
			&m.EnergyPerTokenKwh, &m.CostPerTokenUSD, &m.QualityScore,
			&m.LatencyP50Ms, &m.LatencyP95Ms, &m.FailureRate, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

// CreateModel inserts a new model registry entry.
func (s *Service) CreateModel(ctx context.Context, in CreateModelInput) (*Model, error) {
	id := uuid.New()
	var m Model
	err := s.db.QueryRow(ctx,
		`INSERT INTO model_registry
		    (id, name, display_name, runtime, quantization, status, health_status,
		     endpoint_url, energy_per_token_kwh, cost_per_token_usd, quality_score,
		     latency_p50_ms, latency_p95_ms)
		 VALUES ($1,$2,$3,$4,$5,'inactive','unknown',$6,$7,$8,$9,$10,$11)
		 RETURNING id, name, display_name, runtime, quantization, status, health_status,
		           COALESCE(endpoint_url,''), COALESCE(energy_per_token_kwh,0),
		           COALESCE(cost_per_token_usd,0), COALESCE(quality_score,0),
		           COALESCE(latency_p50_ms,0), COALESCE(latency_p95_ms,0),
		           COALESCE(failure_rate,0), created_at`,
		id, in.Name, in.DisplayName, in.Runtime, in.Quantization,
		in.EndpointURL, in.EnergyPerTokenKwh, in.CostPerTokenUSD, in.QualityScore,
		in.LatencyP50Ms, in.LatencyP95Ms,
	).Scan(
		&m.ID, &m.Name, &m.DisplayName, &m.Runtime, &m.Quantization,
		&m.Status, &m.HealthStatus, &m.EndpointURL,
		&m.EnergyPerTokenKwh, &m.CostPerTokenUSD, &m.QualityScore,
		&m.LatencyP50Ms, &m.LatencyP95Ms, &m.FailureRate, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateModel applies a partial update to a model registry row.
func (s *Service) UpdateModel(ctx context.Context, modelID string, in UpdateModelInput) (*Model, error) {
	_, err := s.db.Exec(ctx,
		`UPDATE model_registry SET
		    status             = COALESCE($2, status),
		    health_status      = COALESCE($3, health_status),
		    endpoint_url       = COALESCE($4, endpoint_url),
		    energy_per_token_kwh = COALESCE($5, energy_per_token_kwh),
		    cost_per_token_usd   = COALESCE($6, cost_per_token_usd),
		    quality_score        = COALESCE($7, quality_score),
		    latency_p50_ms       = COALESCE($8, latency_p50_ms),
		    latency_p95_ms       = COALESCE($9, latency_p95_ms)
		 WHERE id = $1`,
		modelID,
		in.Status, in.HealthStatus, in.EndpointURL,
		in.EnergyPerTokenKwh, in.CostPerTokenUSD, in.QualityScore,
		in.LatencyP50Ms, in.LatencyP95Ms,
	)
	if err != nil {
		return nil, err
	}

	var m Model
	err = s.db.QueryRow(ctx,
		`SELECT id, name, display_name, runtime, quantization, status, health_status,
		        COALESCE(endpoint_url,''), COALESCE(energy_per_token_kwh,0),
		        COALESCE(cost_per_token_usd,0), COALESCE(quality_score,0),
		        COALESCE(latency_p50_ms,0), COALESCE(latency_p95_ms,0),
		        COALESCE(failure_rate,0), created_at
		 FROM model_registry WHERE id = $1`,
		modelID,
	).Scan(
		&m.ID, &m.Name, &m.DisplayName, &m.Runtime, &m.Quantization,
		&m.Status, &m.HealthStatus, &m.EndpointURL,
		&m.EnergyPerTokenKwh, &m.CostPerTokenUSD, &m.QualityScore,
		&m.LatencyP50Ms, &m.LatencyP95Ms, &m.FailureRate, &m.CreatedAt,
	)
	return &m, err
}

// GetRoutes returns active models and their current routing state.
func (s *Service) GetRoutes(ctx context.Context) ([]RouteEntry, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, status,
		        COALESCE(quality_score,0), COALESCE(energy_per_token_kwh,0),
		        COALESCE(cost_per_token_usd,0), COALESCE(latency_p95_ms,0)
		 FROM model_registry
		 ORDER BY status DESC, quality_score DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []RouteEntry
	for rows.Next() {
		var e RouteEntry
		if err := rows.Scan(&e.ModelID, &e.Name, &e.Status, &e.QScore, &e.EnergyKwh, &e.CostUSD, &e.P95Ms); err != nil {
			return nil, err
		}
		routes = append(routes, e)
	}
	return routes, rows.Err()
}

// UpdateRouteInput sets a model's active/inactive routing state.
type UpdateRouteInput struct {
	ModelID string `json:"model_id"`
	Status  string `json:"status"` // "active" or "inactive"
}

// UpdateRoutes enables or disables a model in the routing pool.
func (s *Service) UpdateRoutes(ctx context.Context, updates []UpdateRouteInput) error {
	for _, u := range updates {
		if u.Status != "active" && u.Status != "inactive" {
			continue
		}
		if _, err := s.db.Exec(ctx,
			`UPDATE model_registry SET status = $2 WHERE id = $1`,
			u.ModelID, u.Status,
		); err != nil {
			return err
		}
	}
	return nil
}

// GetCarbonMetrics returns daily CO2e and energy totals for the past 30 days.
func (s *Service) GetCarbonMetrics(ctx context.Context) ([]CarbonDay, error) {
	rows, err := s.db.Query(ctx,
		`SELECT
		    DATE_TRUNC('day', period_start)::date::text AS day,
		    COALESCE(SUM(total_co2e_grams), 0),
		    COALESCE(SUM(total_energy_kwh), 0),
		    COALESCE(SUM(total_requests), 0)
		 FROM usage_aggregates
		 WHERE period_start >= NOW() - INTERVAL '30 days'
		 GROUP BY DATE_TRUNC('day', period_start)
		 ORDER BY day DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var days []CarbonDay
	for rows.Next() {
		var d CarbonDay
		if err := rows.Scan(&d.Date, &d.TotalCO2eGrams, &d.TotalEnergyKwh, &d.TotalRequests); err != nil {
			return nil, err
		}
		days = append(days, d)
	}
	return days, rows.Err()
}
