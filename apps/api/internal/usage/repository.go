package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles SQL for the usage domain.
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// UsageAggregate mirrors a row from the usage_aggregates table.
type UsageAggregate struct {
	ID                    uuid.UUID
	OrgID                 string
	PeriodStart           time.Time
	PeriodEnd             time.Time
	Granularity           string
	TotalRequests         int
	SuccessfulRequests    int
	FailedRequests        int
	CacheHits             int
	FallbackUsed          int
	TotalPromptTokens     int64
	TotalCompletionTokens int64
	ModelDistribution     map[string]int
	TaskDistribution      map[string]int
	AvgLatencyMs          float64
	P95LatencyMs          float64
	TotalEnergyKwh        float64
	TotalCO2eGrams        float64
	TotalCostUSD          float64
}

// AggregateRow is the output of AggregateHour — one row per org.
type AggregateRow struct {
	OrgID                 string
	TotalRequests         int
	SuccessfulRequests    int
	FailedRequests        int
	CacheHits             int
	FallbackUsed          int
	TotalPromptTokens     int64
	TotalCompletionTokens int64
	ModelDistribution     map[string]int
	TaskDistribution      map[string]int
	AvgLatencyMs          float64
	P95LatencyMs          float64
	TotalEnergyKwh        float64
	TotalCO2eGrams        float64
	TotalCostUSD          float64
}

// RequestDetail is a full request record with joined energy and carbon data.
type RequestDetail struct {
	ID                  string    `json:"id"`
	OrgID               string    `json:"org_id"`
	RequestID           string    `json:"request_id"`
	PromptOriginal      string    `json:"prompt_original"`
	PromptOptimized     string    `json:"prompt_optimized,omitempty"`
	TaskType            string    `json:"task_type"`
	Complexity          int       `json:"complexity"`
	ModelSelected       string    `json:"model_selected"`
	ModelFallback       string    `json:"model_fallback,omitempty"`
	RoutingScore        float64   `json:"routing_score"`
	RoutingConfidence   float64   `json:"routing_confidence"`
	UsedFallback        bool      `json:"used_fallback"`
	CacheHit            bool      `json:"cache_hit"`
	ResponseText        string    `json:"response_text,omitempty"`
	FinishReason        string    `json:"finish_reason,omitempty"`
	PromptTokens        int       `json:"prompt_tokens"`
	CompletionTokens    int       `json:"completion_tokens"`
	TotalTokens         int       `json:"total_tokens"`
	LatencyMs           int       `json:"latency_ms"`
	Status              string    `json:"status"`
	ErrorMessage        string    `json:"error_message,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	// From energy_measurements (nullable — not all requests have measurements)
	EnergyKwh           *float64  `json:"energy_kwh,omitempty"`
	GPUPowerWatts       *float64  `json:"gpu_power_watts,omitempty"`
	// From carbon_estimates (nullable)
	CO2eGrams           *float64  `json:"co2e_grams,omitempty"`
	GridRegion          *string   `json:"grid_region,omitempty"`
	GridCarbonIntensity *float64  `json:"grid_carbon_intensity,omitempty"`
	GPT4EquivalentCO2   *float64  `json:"gpt4_equivalent_co2,omitempty"`
	SavingsPercent      *float64  `json:"savings_percent,omitempty"`
}

// RequestFilter holds optional query-param filters for ListRequests.
type RequestFilter struct {
	Model    string
	TaskType string
	Status   string
	Limit    int
	Offset   int
}

// UsageSummary is an aggregated summary over a date range.
type UsageSummary struct {
	TotalRequests  int64   `json:"total_requests"`
	TotalTokens    int64   `json:"total_tokens"`
	TotalEnergyKwh float64 `json:"total_energy_kwh"`
	TotalCO2eGrams float64 `json:"total_co2e_grams"`
	TotalCostUSD   float64 `json:"total_cost_usd"`
	CacheHitRate   float64 `json:"cache_hit_rate"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
}

// DailyBreakdown is a single day's aggregated usage.
type DailyBreakdown struct {
	Date         string  `json:"date"`
	Requests     int64   `json:"requests"`
	EnergyKwh    float64 `json:"energy_kwh"`
	CO2eGrams    float64 `json:"co2e_grams"`
	CostUSD      float64 `json:"cost_usd"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

// AggregateHour runs a CTE query over the requests table for the given window
// and returns one AggregateRow per org. Called exclusively by the background worker.
func (r *Repository) AggregateHour(ctx context.Context, periodStart, periodEnd time.Time) ([]AggregateRow, error) {
	const q = `
WITH base AS (
    SELECT
        r.org_id::text,
        COUNT(*)                                                  AS total_requests,
        COUNT(*) FILTER (WHERE r.status = 'completed')           AS successful_requests,
        COUNT(*) FILTER (WHERE r.status IN ('failed','timeout'))  AS failed_requests,
        COUNT(*) FILTER (WHERE r.cache_hit = true)               AS cache_hits,
        COUNT(*) FILTER (WHERE r.used_fallback = true)           AS fallback_used,
        COALESCE(SUM(r.prompt_tokens), 0)                        AS total_prompt_tokens,
        COALESCE(SUM(r.completion_tokens), 0)                    AS total_completion_tokens,
        COALESCE(AVG(r.latency_ms), 0)                           AS avg_latency_ms,
        COALESCE(
            PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY r.latency_ms), 0
        )                                                         AS p95_latency_ms
    FROM requests r
    WHERE r.created_at >= $1 AND r.created_at < $2
    GROUP BY r.org_id
),
model_dist AS (
    SELECT sub.org_id,
           COALESCE(jsonb_object_agg(sub.model_selected, sub.cnt), '{}') AS dist
    FROM (
        SELECT org_id::text, model_selected, COUNT(*) AS cnt
        FROM requests
        WHERE created_at >= $1 AND created_at < $2
        GROUP BY org_id, model_selected
    ) sub
    GROUP BY sub.org_id
),
task_dist AS (
    SELECT sub.org_id,
           COALESCE(jsonb_object_agg(sub.task_type, sub.cnt), '{}') AS dist
    FROM (
        SELECT org_id::text, task_type, COUNT(*) AS cnt
        FROM requests
        WHERE created_at >= $1 AND created_at < $2
        GROUP BY org_id, task_type
    ) sub
    GROUP BY sub.org_id
),
energy_agg AS (
    SELECT r.org_id::text, COALESCE(SUM(e.total_energy_kwh), 0) AS total_energy_kwh
    FROM requests r
    LEFT JOIN energy_measurements e ON e.request_id = r.id
    WHERE r.created_at >= $1 AND r.created_at < $2
    GROUP BY r.org_id
),
carbon_agg AS (
    SELECT r.org_id::text, COALESCE(SUM(c.co2e_grams), 0) AS total_co2e_grams
    FROM requests r
    LEFT JOIN carbon_estimates c ON c.request_id = r.id
    WHERE r.created_at >= $1 AND r.created_at < $2
    GROUP BY r.org_id
),
cost_agg AS (
    SELECT r.org_id::text, COALESCE(SUM(mr.cost_per_request_usd), 0) AS total_cost_usd
    FROM requests r
    LEFT JOIN model_registry mr ON mr.name = r.model_selected
    WHERE r.created_at >= $1 AND r.created_at < $2
    GROUP BY r.org_id
)
SELECT
    b.org_id,
    b.total_requests, b.successful_requests, b.failed_requests,
    b.cache_hits, b.fallback_used,
    b.total_prompt_tokens, b.total_completion_tokens,
    b.avg_latency_ms, b.p95_latency_ms,
    COALESCE(md.dist, '{}')::text AS model_distribution,
    COALESCE(td.dist, '{}')::text AS task_distribution,
    COALESCE(ea.total_energy_kwh, 0),
    COALESCE(ca.total_co2e_grams, 0),
    COALESCE(co.total_cost_usd, 0)
FROM base b
LEFT JOIN model_dist md ON md.org_id = b.org_id
LEFT JOIN task_dist  td ON td.org_id = b.org_id
LEFT JOIN energy_agg ea ON ea.org_id = b.org_id
LEFT JOIN carbon_agg ca ON ca.org_id = b.org_id
LEFT JOIN cost_agg   co ON co.org_id = b.org_id`

	rows, err := r.db.Query(ctx, q, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AggregateRow
	for rows.Next() {
		var row AggregateRow
		var modelDistJSON, taskDistJSON string
		err := rows.Scan(
			&row.OrgID,
			&row.TotalRequests, &row.SuccessfulRequests, &row.FailedRequests,
			&row.CacheHits, &row.FallbackUsed,
			&row.TotalPromptTokens, &row.TotalCompletionTokens,
			&row.AvgLatencyMs, &row.P95LatencyMs,
			&modelDistJSON, &taskDistJSON,
			&row.TotalEnergyKwh, &row.TotalCO2eGrams, &row.TotalCostUSD,
		)
		if err != nil {
			return nil, fmt.Errorf("scan aggregate row: %w", err)
		}
		if err := json.Unmarshal([]byte(modelDistJSON), &row.ModelDistribution); err != nil {
			return nil, fmt.Errorf("unmarshal model_distribution for org %s: %w", row.OrgID, err)
		}
		if err := json.Unmarshal([]byte(taskDistJSON), &row.TaskDistribution); err != nil {
			return nil, fmt.Errorf("unmarshal task_distribution for org %s: %w", row.OrgID, err)
		}
		if row.ModelDistribution == nil {
			row.ModelDistribution = map[string]int{}
		}
		if row.TaskDistribution == nil {
			row.TaskDistribution = map[string]int{}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// UpsertAggregate inserts or updates a usage aggregate row (all columns).
func (r *Repository) UpsertAggregate(ctx context.Context, ua *UsageAggregate) error {
	modelDistJSON, _ := json.Marshal(ua.ModelDistribution)
	taskDistJSON, _ := json.Marshal(ua.TaskDistribution)

	_, err := r.db.Exec(ctx, `
INSERT INTO usage_aggregates (
    org_id, period_start, period_end, granularity,
    total_requests, successful_requests, failed_requests,
    cache_hits, fallback_used,
    total_prompt_tokens, total_completion_tokens,
    model_distribution, task_distribution,
    avg_latency_ms, p95_latency_ms,
    total_energy_kwh, total_co2e_grams, total_cost_usd
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
ON CONFLICT (org_id, period_start, granularity) DO UPDATE SET
    total_requests          = EXCLUDED.total_requests,
    successful_requests     = EXCLUDED.successful_requests,
    failed_requests         = EXCLUDED.failed_requests,
    cache_hits              = EXCLUDED.cache_hits,
    fallback_used           = EXCLUDED.fallback_used,
    total_prompt_tokens     = EXCLUDED.total_prompt_tokens,
    total_completion_tokens = EXCLUDED.total_completion_tokens,
    model_distribution      = EXCLUDED.model_distribution,
    task_distribution       = EXCLUDED.task_distribution,
    avg_latency_ms          = EXCLUDED.avg_latency_ms,
    p95_latency_ms          = EXCLUDED.p95_latency_ms,
    total_energy_kwh        = EXCLUDED.total_energy_kwh,
    total_co2e_grams        = EXCLUDED.total_co2e_grams,
    total_cost_usd          = EXCLUDED.total_cost_usd`,
		ua.OrgID, ua.PeriodStart, ua.PeriodEnd, ua.Granularity,
		ua.TotalRequests, ua.SuccessfulRequests, ua.FailedRequests,
		ua.CacheHits, ua.FallbackUsed,
		ua.TotalPromptTokens, ua.TotalCompletionTokens,
		modelDistJSON, taskDistJSON,
		ua.AvgLatencyMs, ua.P95LatencyMs,
		ua.TotalEnergyKwh, ua.TotalCO2eGrams, ua.TotalCostUSD,
	)
	return err
}

// GetUsageSummary returns aggregated stats for an org over a date range,
// querying requests directly for real-time accuracy.
func (r *Repository) GetUsageSummary(ctx context.Context, orgID string, from, to time.Time) (*UsageSummary, error) {
	row := r.db.QueryRow(ctx, `
SELECT
    COUNT(*)                                                 AS total_requests,
    COALESCE(SUM(r.total_tokens), 0)                         AS total_tokens,
    COALESCE(SUM(e.total_energy_kwh), 0)                     AS total_energy_kwh,
    COALESCE(SUM(c.co2e_grams), 0)                           AS total_co2e_grams,
    COALESCE(SUM(mr.cost_per_request_usd), 0)                AS total_cost_usd,
    CASE WHEN COUNT(*) > 0
         THEN COUNT(*) FILTER (WHERE r.cache_hit)::float / COUNT(*)
         ELSE 0 END                                          AS cache_hit_rate,
    COALESCE(AVG(r.latency_ms), 0)                           AS avg_latency_ms
FROM requests r
LEFT JOIN energy_measurements e ON e.request_id = r.id
LEFT JOIN carbon_estimates c    ON c.request_id = r.id
LEFT JOIN model_registry mr     ON mr.name = r.model_selected
WHERE r.org_id = $1 AND r.created_at >= $2 AND r.created_at < $3`,
		orgID, from, to,
	)

	s := &UsageSummary{}
	return s, row.Scan(
		&s.TotalRequests, &s.TotalTokens,
		&s.TotalEnergyKwh, &s.TotalCO2eGrams, &s.TotalCostUSD,
		&s.CacheHitRate, &s.AvgLatencyMs,
	)
}

// GetDailyBreakdown returns per-day stats for an org within a date range.
func (r *Repository) GetDailyBreakdown(ctx context.Context, orgID string, from, to time.Time) ([]DailyBreakdown, error) {
	rows, err := r.db.Query(ctx, `
SELECT
    TO_CHAR(DATE_TRUNC('day', r.created_at), 'YYYY-MM-DD') AS date,
    COUNT(*)                                               AS requests,
    COALESCE(SUM(e.total_energy_kwh), 0)                   AS energy_kwh,
    COALESCE(SUM(c.co2e_grams), 0)                         AS co2e_grams,
    COALESCE(SUM(mr.cost_per_request_usd), 0)              AS cost_usd,
    COALESCE(AVG(r.latency_ms), 0)                         AS avg_latency_ms
FROM requests r
LEFT JOIN energy_measurements e ON e.request_id = r.id
LEFT JOIN carbon_estimates c    ON c.request_id = r.id
LEFT JOIN model_registry mr     ON mr.name = r.model_selected
WHERE r.org_id = $1 AND r.created_at >= $2 AND r.created_at < $3
GROUP BY DATE_TRUNC('day', r.created_at)
ORDER BY DATE_TRUNC('day', r.created_at) ASC`,
		orgID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdown []DailyBreakdown
	for rows.Next() {
		var d DailyBreakdown
		if err := rows.Scan(&d.Date, &d.Requests, &d.EnergyKwh, &d.CO2eGrams, &d.CostUSD, &d.AvgLatencyMs); err != nil {
			return nil, err
		}
		breakdown = append(breakdown, d)
	}
	return breakdown, rows.Err()
}

// GetModelDistribution returns model usage counts for an org in a date range.
func (r *Repository) GetModelDistribution(ctx context.Context, orgID string, from, to time.Time) (map[string]int64, error) {
	rows, err := r.db.Query(ctx, `
SELECT model_selected, COUNT(*) AS cnt
FROM requests
WHERE org_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY model_selected`,
		orgID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := make(map[string]int64)
	for rows.Next() {
		var model string
		var cnt int64
		if err := rows.Scan(&model, &cnt); err != nil {
			return nil, err
		}
		dist[model] = cnt
	}
	return dist, rows.Err()
}

type requestScanner interface {
	Scan(dest ...any) error
}

const requestDetailSelect = `
SELECT
    r.id::text, r.org_id::text, r.request_id,
    r.prompt_original, COALESCE(r.prompt_optimized,''),
    r.task_type, r.complexity,
    r.model_selected, COALESCE(r.model_fallback,''),
    r.routing_score, r.routing_confidence,
    r.used_fallback, r.cache_hit,
    COALESCE(r.response_text,''), COALESCE(r.finish_reason,''),
    COALESCE(r.prompt_tokens,0), COALESCE(r.completion_tokens,0), COALESCE(r.total_tokens,0),
    r.latency_ms, r.status, COALESCE(r.error_message,''),
    r.created_at,
    e.total_energy_kwh, e.gpu_power_watts,
    c.co2e_grams, c.grid_region, c.grid_carbon_intensity,
    c.gpt4_equivalent_co2e, c.savings_percent
FROM requests r
LEFT JOIN energy_measurements e ON e.request_id = r.id
LEFT JOIN carbon_estimates c    ON c.request_id = r.id`

func scanRequestDetail(s requestScanner) (RequestDetail, error) {
	var d RequestDetail
	err := s.Scan(
		&d.ID, &d.OrgID, &d.RequestID,
		&d.PromptOriginal, &d.PromptOptimized,
		&d.TaskType, &d.Complexity,
		&d.ModelSelected, &d.ModelFallback,
		&d.RoutingScore, &d.RoutingConfidence,
		&d.UsedFallback, &d.CacheHit,
		&d.ResponseText, &d.FinishReason,
		&d.PromptTokens, &d.CompletionTokens, &d.TotalTokens,
		&d.LatencyMs, &d.Status, &d.ErrorMessage,
		&d.CreatedAt,
		&d.EnergyKwh, &d.GPUPowerWatts,
		&d.CO2eGrams, &d.GridRegion, &d.GridCarbonIntensity,
		&d.GPT4EquivalentCO2, &d.SavingsPercent,
	)
	return d, err
}

// FindRequestByID returns a full request record with joined energy/carbon data.
// Enforces org isolation: request must belong to orgID.
func (r *Repository) FindRequestByID(ctx context.Context, requestID, orgID string) (*RequestDetail, error) {
	row := r.db.QueryRow(ctx,
		requestDetailSelect+` WHERE r.request_id = $1 AND r.org_id = $2`,
		requestID, orgID,
	)
	d, err := scanRequestDetail(row)
	return &d, err
}

// CarbonSummaryRow holds aggregate carbon totals for an org in a time window.
type CarbonSummaryRow struct {
	TotalCO2eGrams          float64
	TotalEnergyKwh          float64
	GPT4EquivalentCO2eGrams float64
	GridRegion              string
	GridCarbonIntensity     float64
}

// CarbonDayRow is per-day carbon data for the daily_breakdown array.
type CarbonDayRow struct {
	Date                    string
	CO2eGrams               float64
	EnergyKwh               float64
	GPT4EquivalentCO2eGrams float64
}

// ModelCarbonRow is per-model energy/carbon data for model_energy_breakdown.
type ModelCarbonRow struct {
	Model        string
	EnergyKwh    float64
	CO2eGrams    float64
	RequestCount int64
}

// GetCarbonSummary returns aggregate carbon totals for an org from `from` to now.
func (r *Repository) GetCarbonSummary(ctx context.Context, orgID string, from time.Time) (*CarbonSummaryRow, error) {
	row := r.db.QueryRow(ctx, `
SELECT
    COALESCE(SUM(c.co2e_grams), 0)            AS total_co2e_grams,
    COALESCE(SUM(e.total_energy_kwh), 0)       AS total_energy_kwh,
    COALESCE(SUM(c.gpt4_equivalent_co2e), 0)   AS gpt4_equivalent_co2e_grams,
    COALESCE(MAX(c.grid_region), 'US-EAST')    AS grid_region,
    COALESCE(AVG(c.grid_carbon_intensity), 400) AS grid_carbon_intensity
FROM requests r
LEFT JOIN energy_measurements e ON e.request_id = r.id
LEFT JOIN carbon_estimates    c ON c.request_id = r.id
WHERE r.org_id = $1 AND r.created_at >= $2`, orgID, from)

	s := &CarbonSummaryRow{}
	return s, row.Scan(
		&s.TotalCO2eGrams, &s.TotalEnergyKwh,
		&s.GPT4EquivalentCO2eGrams, &s.GridRegion, &s.GridCarbonIntensity,
	)
}

// GetCarbonDailyBreakdown returns per-day carbon totals for an org from `from` to now.
func (r *Repository) GetCarbonDailyBreakdown(ctx context.Context, orgID string, from time.Time) ([]CarbonDayRow, error) {
	rows, err := r.db.Query(ctx, `
SELECT
    TO_CHAR(DATE_TRUNC('day', r.created_at), 'YYYY-MM-DD') AS date,
    COALESCE(SUM(c.co2e_grams), 0)                          AS co2e_grams,
    COALESCE(SUM(e.total_energy_kwh), 0)                    AS energy_kwh,
    COALESCE(SUM(c.gpt4_equivalent_co2e), 0)                AS gpt4_equivalent_co2e_grams
FROM requests r
LEFT JOIN energy_measurements e ON e.request_id = r.id
LEFT JOIN carbon_estimates    c ON c.request_id = r.id
WHERE r.org_id = $1 AND r.created_at >= $2
GROUP BY DATE_TRUNC('day', r.created_at)
ORDER BY 1 ASC`, orgID, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CarbonDayRow
	for rows.Next() {
		var d CarbonDayRow
		if err := rows.Scan(&d.Date, &d.CO2eGrams, &d.EnergyKwh, &d.GPT4EquivalentCO2eGrams); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// GetModelCarbonBreakdown returns per-model energy/carbon totals for an org from `from` to now.
func (r *Repository) GetModelCarbonBreakdown(ctx context.Context, orgID string, from time.Time) ([]ModelCarbonRow, error) {
	rows, err := r.db.Query(ctx, `
SELECT
    r.model_selected,
    COALESCE(SUM(e.total_energy_kwh), 0) AS energy_kwh,
    COALESCE(SUM(c.co2e_grams), 0)        AS co2e_grams,
    COUNT(*)                              AS request_count
FROM requests r
LEFT JOIN energy_measurements e ON e.request_id = r.id
LEFT JOIN carbon_estimates    c ON c.request_id = r.id
WHERE r.org_id = $1 AND r.created_at >= $2 AND r.model_selected != ''
GROUP BY r.model_selected
ORDER BY energy_kwh DESC`, orgID, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ModelCarbonRow
	for rows.Next() {
		var m ModelCarbonRow
		if err := rows.Scan(&m.Model, &m.EnergyKwh, &m.CO2eGrams, &m.RequestCount); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// ListRequests returns a paginated list of requests for an org with optional filters.
func (r *Repository) ListRequests(ctx context.Context, orgID string, f RequestFilter) ([]RequestDetail, error) {
	conds := []string{"r.org_id = $1"}
	args := []interface{}{orgID}
	idx := 2

	if f.Model != "" {
		conds = append(conds, fmt.Sprintf("r.model_selected = $%d", idx))
		args = append(args, f.Model)
		idx++
	}
	if f.TaskType != "" {
		conds = append(conds, fmt.Sprintf("r.task_type = $%d", idx))
		args = append(args, f.TaskType)
		idx++
	}
	if f.Status != "" {
		conds = append(conds, fmt.Sprintf("r.status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	args = append(args, limit, f.Offset)

	q := fmt.Sprintf(requestDetailSelect+`
WHERE %s
ORDER BY r.created_at DESC
LIMIT $%d OFFSET $%d`,
		strings.Join(conds, " AND "), idx, idx+1,
	)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RequestDetail
	for rows.Next() {
		d, err := scanRequestDetail(rows)
		if err != nil {
			return nil, fmt.Errorf("scan request: %w", err)
		}
		results = append(results, d)
	}
	return results, rows.Err()
}

// CountRequests returns the total count matching the filters (used for pagination).
func (r *Repository) CountRequests(ctx context.Context, orgID string, f RequestFilter) (int64, error) {
	conds := []string{"org_id = $1"}
	args := []interface{}{orgID}
	idx := 2

	if f.Model != "" {
		conds = append(conds, fmt.Sprintf("model_selected = $%d", idx))
		args = append(args, f.Model)
		idx++
	}
	if f.TaskType != "" {
		conds = append(conds, fmt.Sprintf("task_type = $%d", idx))
		args = append(args, f.TaskType)
		idx++
	}
	if f.Status != "" {
		conds = append(conds, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
	q := fmt.Sprintf("SELECT COUNT(*) FROM requests WHERE %s", strings.Join(conds, " AND "))
	var count int64
	return count, r.db.QueryRow(ctx, q, args...).Scan(&count)
}

// GetUsageForBilling returns rolled-up totals for a given org and day from
// usage_aggregates. Used by the billing worker.
func (r *Repository) GetUsageForBilling(ctx context.Context, orgID string, day time.Time) (*UsageSummary, error) {
	dayStart := day.UTC().Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)

	row := r.db.QueryRow(ctx, `
SELECT
    COALESCE(SUM(total_requests), 0),
    COALESCE(SUM(total_prompt_tokens + total_completion_tokens), 0),
    COALESCE(SUM(total_energy_kwh), 0),
    COALESCE(SUM(total_co2e_grams), 0),
    COALESCE(SUM(total_cost_usd), 0)
FROM usage_aggregates
WHERE org_id = $1
  AND period_start >= $2 AND period_end <= $3
  AND granularity = 'hourly'`,
		orgID, dayStart, dayEnd,
	)

	s := &UsageSummary{}
	return s, row.Scan(
		&s.TotalRequests, &s.TotalTokens,
		&s.TotalEnergyKwh, &s.TotalCO2eGrams, &s.TotalCostUSD,
	)
}

// GetMonthlyRequestCount returns total requests for an org in the given calendar month.
func (r *Repository) GetMonthlyRequestCount(ctx context.Context, orgID string, month time.Time) (int64, error) {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	var count int64
	return count, r.db.QueryRow(ctx, `
SELECT COALESCE(SUM(total_requests), 0)
FROM usage_aggregates
WHERE org_id = $1
  AND period_start >= $2 AND period_end <= $3
  AND granularity = 'hourly'`,
		orgID, monthStart, monthEnd,
	).Scan(&count)
}

// ListOrgsWithUsage returns org IDs that have hourly aggregates for the given day.
func (r *Repository) ListOrgsWithUsage(ctx context.Context, day time.Time) ([]string, error) {
	dayStart := day.UTC().Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)

	rows, err := r.db.Query(ctx, `
SELECT DISTINCT org_id::text
FROM usage_aggregates
WHERE period_start >= $1 AND period_end <= $2 AND granularity = 'hourly'`,
		dayStart, dayEnd,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		orgs = append(orgs, id)
	}
	return orgs, rows.Err()
}
