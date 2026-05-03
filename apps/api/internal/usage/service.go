package usage

import (
	"context"
	"time"
)

// Service handles usage business logic.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// UsageResponse is the full payload returned by GET /v1/usage.
type UsageResponse struct {
	OrgID             string                 `json:"org_id"`
	Period            string                 `json:"period"`
	From              string                 `json:"from"`
	To                string                 `json:"to"`
	Summary           *UsageSummary          `json:"summary"`
	ModelDistribution map[string]int64       `json:"model_distribution"`
	DailyBreakdown    []DailyBreakdown       `json:"daily_breakdown,omitempty"`
	MonthlyBreakdown  []DailyBreakdown       `json:"monthly_breakdown,omitempty"`
}

// GetUsage builds the full usage response for an org.
func (s *Service) GetUsage(ctx context.Context, orgID, period string, from, to time.Time) (*UsageResponse, error) {
	summary, err := s.repo.GetUsageSummary(ctx, orgID, from, to)
	if err != nil {
		return nil, err
	}

	modelDist, err := s.repo.GetModelDistribution(ctx, orgID, from, to)
	if err != nil {
		return nil, err
	}

	resp := &UsageResponse{
		OrgID:             orgID,
		Period:            period,
		From:              from.Format("2006-01-02"),
		To:                to.Add(-24 * time.Hour).Format("2006-01-02"),
		Summary:           summary,
		ModelDistribution: modelDist,
	}

	breakdown, err := s.repo.GetDailyBreakdown(ctx, orgID, from, to)
	if err != nil {
		return nil, err
	}
	if period == "monthly" {
		resp.MonthlyBreakdown = breakdown
	} else {
		resp.DailyBreakdown = breakdown
	}

	return resp, nil
}

// CarbonResponse is the response shape for GET /v1/carbon.
type CarbonResponse struct {
	Period                  string             `json:"period"`
	TotalCO2eGrams          float64            `json:"total_co2e_grams"`
	TotalEnergyKwh          float64            `json:"total_energy_kwh"`
	GPT4EquivalentCO2eGrams float64            `json:"gpt4_equivalent_co2e_grams"`
	SavingsPercent          float64            `json:"savings_percent"`
	GridRegion              string             `json:"grid_region"`
	GridCarbonIntensity     float64            `json:"grid_carbon_intensity"`
	DailyBreakdown          []CarbonDailyItem  `json:"daily_breakdown"`
	ModelEnergyBreakdown    []ModelEnergyItem  `json:"model_energy_breakdown"`
}

// CarbonDailyItem is one entry in the daily_breakdown array.
type CarbonDailyItem struct {
	Date                    string  `json:"date"`
	CO2eGrams               float64 `json:"co2e_grams"`
	EnergyKwh               float64 `json:"energy_kwh"`
	GPT4EquivalentCO2eGrams float64 `json:"gpt4_equivalent_co2e_grams"`
}

// ModelEnergyItem is one entry in the model_energy_breakdown array.
type ModelEnergyItem struct {
	Model               string  `json:"model"`
	EnergyKwh           float64 `json:"energy_kwh"`
	CO2eGrams           float64 `json:"co2e_grams"`
	RequestCount        int64   `json:"request_count"`
	PercentageOfTraffic float64 `json:"percentage_of_traffic"`
}

// GetCarbon returns carbon and energy aggregates for an org for the given period.
// period: "daily" = last 24 h, "monthly" = last 30 days.
func (s *Service) GetCarbon(ctx context.Context, orgID, period string) (*CarbonResponse, error) {
	from := time.Now().AddDate(0, -1, 0) // 30 days default
	if period == "daily" {
		from = time.Now().Add(-24 * time.Hour)
	}

	summary, err := s.repo.GetCarbonSummary(ctx, orgID, from)
	if err != nil {
		return nil, err
	}

	dailyRows, err := s.repo.GetCarbonDailyBreakdown(ctx, orgID, from)
	if err != nil {
		return nil, err
	}

	modelRows, err := s.repo.GetModelCarbonBreakdown(ctx, orgID, from)
	if err != nil {
		return nil, err
	}

	savingsPct := 0.0
	if summary.GPT4EquivalentCO2eGrams > 0 {
		savingsPct = (summary.GPT4EquivalentCO2eGrams - summary.TotalCO2eGrams) / summary.GPT4EquivalentCO2eGrams * 100
		if savingsPct < 0 {
			savingsPct = 0
		}
	}

	daily := make([]CarbonDailyItem, len(dailyRows))
	for i, d := range dailyRows {
		daily[i] = CarbonDailyItem{
			Date:                    d.Date,
			CO2eGrams:               d.CO2eGrams,
			EnergyKwh:               d.EnergyKwh,
			GPT4EquivalentCO2eGrams: d.GPT4EquivalentCO2eGrams,
		}
	}

	var totalRequests int64
	for _, m := range modelRows {
		totalRequests += m.RequestCount
	}
	models := make([]ModelEnergyItem, len(modelRows))
	for i, m := range modelRows {
		pct := 0.0
		if totalRequests > 0 {
			pct = float64(m.RequestCount) / float64(totalRequests) * 100
		}
		models[i] = ModelEnergyItem{
			Model:               m.Model,
			EnergyKwh:           m.EnergyKwh,
			CO2eGrams:           m.CO2eGrams,
			RequestCount:        m.RequestCount,
			PercentageOfTraffic: pct,
		}
	}

	return &CarbonResponse{
		Period:                  period,
		TotalCO2eGrams:          summary.TotalCO2eGrams,
		TotalEnergyKwh:          summary.TotalEnergyKwh,
		GPT4EquivalentCO2eGrams: summary.GPT4EquivalentCO2eGrams,
		SavingsPercent:          savingsPct,
		GridRegion:              summary.GridRegion,
		GridCarbonIntensity:     summary.GridCarbonIntensity,
		DailyBreakdown:          daily,
		ModelEnergyBreakdown:    models,
	}, nil
}

// GetRequest returns a single request with energy/carbon detail.
func (s *Service) GetRequest(ctx context.Context, requestID, orgID string) (*RequestDetail, error) {
	return s.repo.FindRequestByID(ctx, requestID, orgID)
}

// ListRequestsResponse is the paginated response for GET /v1/requests.
type ListRequestsResponse struct {
	Requests  []RequestDetail `json:"requests"`
	Total     int64           `json:"total"`
	Page      int             `json:"page"`
	PerPage   int             `json:"per_page"`
}

// GetRequests returns a paginated list of requests for an org.
func (s *Service) GetRequests(ctx context.Context, orgID string, f RequestFilter) (*ListRequestsResponse, error) {
	total, err := s.repo.CountRequests(ctx, orgID, f)
	if err != nil {
		return nil, err
	}

	requests, err := s.repo.ListRequests(ctx, orgID, f)
	if err != nil {
		return nil, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	page := 1
	if limit > 0 {
		page = f.Offset/limit + 1
	}

	return &ListRequestsResponse{
		Requests: requests,
		Total:    total,
		Page:     page,
		PerPage:  limit,
	}, nil
}
