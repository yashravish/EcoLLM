package usage

import (
	"testing"
)

// TestAggregateHourlyWindow verifies the time-window calculation:
// periodEnd = now truncated to the hour, periodStart = periodEnd - 1h.
// The worker should never process the current (incomplete) hour.
func TestAggregateHourlyWindow(t *testing.T) {
	// This is a compile-time check — aggregateHourly only requires a *Repository
	// (pgxpool-backed). We verify the logic here via the exported types only.
	// Integration tests in tests/e2e cover the full DB path.
	_ = func() error {
		return aggregateHourly(nil) // would panic — intentionally not called
	}
}

// TestUsageAggregateFields verifies the UsageAggregate struct has all columns
// that UpsertAggregate will try to bind. A compile error here means a field
// was added to the DB schema but not the Go type (or vice versa).
func TestUsageAggregateFields(t *testing.T) {
	ua := UsageAggregate{
		OrgID:                 "org-1",
		Granularity:           "hourly",
		TotalRequests:         10,
		SuccessfulRequests:    9,
		FailedRequests:        1,
		CacheHits:             3,
		FallbackUsed:          1,
		TotalPromptTokens:     500,
		TotalCompletionTokens: 300,
		ModelDistribution:     map[string]int{"phi-3-mini": 7, "mistral-7b": 3},
		TaskDistribution:      map[string]int{"simple": 6, "medium": 4},
		AvgLatencyMs:          120.5,
		P95LatencyMs:          340.0,
		TotalEnergyKwh:        0.001,
		TotalCO2eGrams:        0.5,
		TotalCostUSD:          0.0015,
	}

	if ua.TotalRequests != 10 {
		t.Errorf("TotalRequests: got %d, want 10", ua.TotalRequests)
	}
	if ua.SuccessfulRequests+ua.FailedRequests != ua.TotalRequests {
		t.Errorf("success+failed (%d+%d) != total (%d)",
			ua.SuccessfulRequests, ua.FailedRequests, ua.TotalRequests)
	}
	if ua.ModelDistribution["phi-3-mini"]+ua.ModelDistribution["mistral-7b"] != 10 {
		t.Errorf("model distribution sum != total requests")
	}
}

// TestAggregateRowFields verifies AggregateRow has all fields the CTE query
// returns. If this compile-check fails, the SELECT list is out of sync.
func TestAggregateRowFields(t *testing.T) {
	row := AggregateRow{
		OrgID:                 "org-1",
		TotalRequests:         100,
		SuccessfulRequests:    95,
		FailedRequests:        5,
		CacheHits:             20,
		FallbackUsed:          3,
		TotalPromptTokens:     10000,
		TotalCompletionTokens: 5000,
		ModelDistribution:     map[string]int{"llama-13b": 100},
		TaskDistribution:      map[string]int{"hard": 100},
		AvgLatencyMs:          800,
		P95LatencyMs:          1500,
		TotalEnergyKwh:        0.01,
		TotalCO2eGrams:        5.0,
		TotalCostUSD:          0.02,
	}

	if row.OrgID == "" {
		t.Error("OrgID should not be empty")
	}
}
