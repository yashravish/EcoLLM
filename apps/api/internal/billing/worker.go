package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// BillingWriter is the minimal interface the worker needs from the billing repo.
type BillingWriter interface {
	Insert(ctx context.Context, ev *BillingEvent) error
}

// StartBillingWorker runs a background goroutine that generates daily billing
// events. Fires on the given interval (typically 24h).
// It exits cleanly when ctx is cancelled.
// Call once from main.go: go billing.StartBillingWorker(ctx, repo, usageRepo, 24*time.Hour)
func StartBillingWorker(ctx context.Context, repo BillingWriter, usageRepo UsageReader, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().Dur("interval", interval).Msg("billing worker started")

	for {
		select {
		case <-ticker.C:
			if err := generateDailyBillingEvents(repo, usageRepo); err != nil {
				log.Error().Err(err).Msg("billing event generation failed")
			}
		case <-ctx.Done():
			log.Info().Msg("billing worker stopped")
			return
		}
	}
}

// UsageReader is the minimal interface the billing worker needs from the usage package.
// Defined here to avoid an import cycle.
type UsageReader interface {
	GetUsageForBilling(ctx context.Context, orgID string, day time.Time) (*UsageSummaryDTO, error)
	GetMonthlyRequestCount(ctx context.Context, orgID string, month time.Time) (int64, error)
	ListOrgsWithUsage(ctx context.Context, day time.Time) ([]string, error)
}

// UsageSummaryDTO carries the rolled-up daily totals the billing worker needs.
type UsageSummaryDTO struct {
	TotalRequests  int64
	TotalTokens    int64
	TotalEnergyKwh float64
	TotalCO2eGrams float64
	TotalCostUSD   float64
}

// generateDailyBillingEvents creates one billing_event per org for the previous
// calendar day using hourly usage_aggregates as the source.
func generateDailyBillingEvents(repo BillingWriter, usageRepo UsageReader) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	start := time.Now()

	orgs, err := usageRepo.ListOrgsWithUsage(ctx, yesterday)
	if err != nil {
		return fmt.Errorf("list orgs: %w", err)
	}
	if len(orgs) == 0 {
		log.Debug().Time("day", yesterday).Msg("no orgs with usage, skipping billing")
		return nil
	}

	dayStart := yesterday.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)

	var succeeded int
	for _, orgID := range orgs {
		usage, err := usageRepo.GetUsageForBilling(ctx, orgID, yesterday)
		if err != nil {
			log.Error().Err(err).Str("org_id", orgID).Msg("get usage for billing failed")
			continue
		}
		if usage.TotalRequests == 0 {
			continue
		}

		monthlyCount, err := usageRepo.GetMonthlyRequestCount(ctx, orgID, yesterday)
		if err != nil {
			log.Warn().Err(err).Str("org_id", orgID).Msg("monthly count failed, using 0")
		}

		subtotal := CalculateBill(usage.TotalCO2eGrams, int(usage.TotalRequests))
		discount := ApplyVolumeDiscount(monthlyCount)
		total := subtotal * (1 - discount/100)

		orgUUID, err := uuid.Parse(orgID)
		if err != nil {
			log.Error().Err(err).Str("org_id", orgID).Msg("invalid org UUID, skipping")
			continue
		}

		ev := &BillingEvent{
			ID:              uuid.New(),
			OrgID:           orgUUID,
			PeriodStart:     dayStart,
			PeriodEnd:       dayEnd,
			TotalRequests:   int(usage.TotalRequests),
			TotalTokens:     usage.TotalTokens,
			TotalEnergyKwh:  usage.TotalEnergyKwh,
			TotalCO2eGrams:  usage.TotalCO2eGrams,
			SubtotalUSD:     subtotal,
			DiscountPercent: discount,
			TotalUSD:        total,
			Status:          "pending",
		}
		if err := repo.Insert(ctx, ev); err != nil {
			log.Error().Err(err).Str("org_id", orgID).Msg("insert billing event failed")
			continue
		}
		succeeded++
	}

	log.Info().
		Dur("duration", time.Since(start)).
		Int("orgs_billed", succeeded).
		Int("orgs_total", len(orgs)).
		Time("day", yesterday).
		Msg("daily billing events generated")

	return nil
}

// CalculateBill computes the USD charge for a given amount of CO2e and requests.
//
//	Price = (co2e_grams × $0.001) + ($0.0001 × requests)
func CalculateBill(totalCO2eGrams float64, totalRequests int) float64 {
	co2Cost := totalCO2eGrams * 0.001
	overheadCost := float64(totalRequests) * 0.0001
	return co2Cost + overheadCost
}

// ApplyVolumeDiscount returns the discount percentage (0, 15, or 25) based on
// the total monthly request count.
//
//	≥ 10M requests/month → 25% discount
//	≥  1M requests/month → 15% discount
//	<  1M requests/month →  0% discount
func ApplyVolumeDiscount(monthlyRequests int64) float64 {
	switch {
	case monthlyRequests >= 10_000_000:
		return 25.0
	case monthlyRequests >= 1_000_000:
		return 15.0
	default:
		return 0.0
	}
}
