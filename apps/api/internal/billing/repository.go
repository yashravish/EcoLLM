package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles SQL for billing events.
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// BillingEvent mirrors a row from the billing_events table.
type BillingEvent struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	PeriodStart     time.Time
	PeriodEnd       time.Time
	TotalRequests   int
	TotalTokens     int64
	TotalEnergyKwh  float64
	TotalCO2eGrams  float64
	SubtotalUSD     float64
	DiscountPercent float64
	TotalUSD        float64
	Status          string
	InvoiceURL      string
	CreatedAt       time.Time
}

// Insert writes a new billing event.
func (r *Repository) Insert(ctx context.Context, ev *BillingEvent) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO billing_events (
			id, org_id, period_start, period_end,
			total_requests, total_tokens,
			total_energy_kwh, total_co2e_grams,
			subtotal_usd, discount_percent, total_usd, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		ev.ID, ev.OrgID, ev.PeriodStart, ev.PeriodEnd,
		ev.TotalRequests, ev.TotalTokens,
		ev.TotalEnergyKwh, ev.TotalCO2eGrams,
		ev.SubtotalUSD, ev.DiscountPercent, ev.TotalUSD, ev.Status,
	)
	return err
}

type billingScanner interface {
	Scan(dest ...any) error
}

const billingEventSelect = `SELECT id, org_id, period_start, period_end,
	        total_requests, total_tokens,
	        total_energy_kwh, total_co2e_grams,
	        subtotal_usd, discount_percent, total_usd,
	        status, COALESCE(invoice_url,''), created_at`

func scanBillingEvent(s billingScanner) (BillingEvent, error) {
	var e BillingEvent
	err := s.Scan(
		&e.ID, &e.OrgID, &e.PeriodStart, &e.PeriodEnd,
		&e.TotalRequests, &e.TotalTokens,
		&e.TotalEnergyKwh, &e.TotalCO2eGrams,
		&e.SubtotalUSD, &e.DiscountPercent, &e.TotalUSD,
		&e.Status, &e.InvoiceURL, &e.CreatedAt,
	)
	return e, err
}

// ListByOrgPeriod returns billing events for an org within [from, to).
func (r *Repository) ListByOrgPeriod(ctx context.Context, orgID string, from, to time.Time) ([]BillingEvent, error) {
	rows, err := r.db.Query(ctx,
		billingEventSelect+`
		 FROM billing_events
		 WHERE org_id = $1 AND period_start >= $2 AND period_start < $3
		 ORDER BY period_start DESC`,
		orgID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []BillingEvent
	for rows.Next() {
		e, err := scanBillingEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// FindByID returns a single billing event scoped to the org.
func (r *Repository) FindByID(ctx context.Context, orgID, eventID string) (*BillingEvent, error) {
	row := r.db.QueryRow(ctx,
		billingEventSelect+`
		 FROM billing_events
		 WHERE id = $1 AND org_id = $2`,
		eventID, orgID,
	)
	e, err := scanBillingEvent(row)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
