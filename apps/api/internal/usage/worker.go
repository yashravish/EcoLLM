package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// StartAggregationWorker runs a background goroutine that aggregates per-request
// records into the usage_aggregates table on the given interval.
// It exits cleanly when ctx is cancelled.
// Call once from main.go: go usage.StartAggregationWorker(ctx, repo, time.Hour)
func StartAggregationWorker(ctx context.Context, repo *Repository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().Dur("interval", interval).Msg("usage aggregation worker started")

	for {
		select {
		case <-ticker.C:
			if err := aggregateHourly(repo); err != nil {
				log.Error().Err(err).Msg("usage aggregation failed")
			}
		case <-ctx.Done():
			log.Info().Msg("usage aggregation worker stopped")
			return
		}
	}
}

// aggregateHourly processes the previous complete hour.
// When called at 14:03 UTC it processes 13:00–14:00 UTC.
func aggregateHourly(repo *Repository) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now().UTC()
	periodEnd := now.Truncate(time.Hour)
	periodStart := periodEnd.Add(-time.Hour)

	start := time.Now()
	rows, err := repo.AggregateHour(ctx, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("aggregate query: %w", err)
	}

	for _, row := range rows {
		ua := &UsageAggregate{
			OrgID:                 row.OrgID,
			PeriodStart:           periodStart,
			PeriodEnd:             periodEnd,
			Granularity:           "hourly",
			TotalRequests:         row.TotalRequests,
			SuccessfulRequests:    row.SuccessfulRequests,
			FailedRequests:        row.FailedRequests,
			CacheHits:             row.CacheHits,
			FallbackUsed:          row.FallbackUsed,
			TotalPromptTokens:     row.TotalPromptTokens,
			TotalCompletionTokens: row.TotalCompletionTokens,
			ModelDistribution:     row.ModelDistribution,
			TaskDistribution:      row.TaskDistribution,
			AvgLatencyMs:          row.AvgLatencyMs,
			P95LatencyMs:          row.P95LatencyMs,
			TotalEnergyKwh:        row.TotalEnergyKwh,
			TotalCO2eGrams:        row.TotalCO2eGrams,
			TotalCostUSD:          row.TotalCostUSD,
		}
		if err := repo.UpsertAggregate(ctx, ua); err != nil {
			log.Error().Err(err).
				Str("org_id", row.OrgID).
				Msg("upsert aggregate failed")
		}
	}

	log.Info().
		Dur("duration", time.Since(start)).
		Int("orgs", len(rows)).
		Time("period_start", periodStart).
		Time("period_end", periodEnd).
		Msg("usage aggregation completed")

	return nil
}
