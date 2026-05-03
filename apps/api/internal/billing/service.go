package billing

import (
	"context"
	"time"
)

// Service handles billing business logic.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// BillingResponse is the payload returned by GET /v1/billing.
type BillingResponse struct {
	OrgID       string         `json:"org_id"`
	From        string         `json:"from"`
	To          string         `json:"to"`
	TotalUSD    float64        `json:"total_usd"`
	Events      []BillingEvent `json:"events"`
}

// GetBilling returns billing events for an org within an optional date range.
// If from/to are zero, defaults to the last 12 months.
func (s *Service) GetBilling(ctx context.Context, orgID string, from, to time.Time) (*BillingResponse, error) {
	if from.IsZero() {
		from = time.Now().UTC().AddDate(0, -12, 0).Truncate(24 * time.Hour)
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}

	events, err := s.repo.ListByOrgPeriod(ctx, orgID, from, to)
	if err != nil {
		return nil, err
	}

	var total float64
	for _, e := range events {
		total += e.TotalUSD
	}

	return &BillingResponse{
		OrgID:    orgID,
		From:     from.Format("2006-01-02"),
		To:       to.Format("2006-01-02"),
		TotalUSD: total,
		Events:   events,
	}, nil
}

// GetBillingEvent returns a single billing event by ID, scoped to the org.
func (s *Service) GetBillingEvent(ctx context.Context, orgID, eventID string) (*BillingEvent, error) {
	return s.repo.FindByID(ctx, orgID, eventID)
}
