package billing

import (
	"context"
	"testing"
	"time"
)

// ── CalculateBill ──────────────────────────────────────────────────────────────

func TestCalculateBill(t *testing.T) {
	tests := []struct {
		name     string
		co2e     float64
		requests int
		want     float64
	}{
		{"zero", 0, 0, 0},
		{"co2e only", 1000, 0, 1.0},        // 1000g × $0.001
		{"requests only", 0, 10000, 1.0},    // 10000 × $0.0001
		{"both", 500, 5000, 1.0},            // 0.5 + 0.5
		{"small request", 1, 1, 0.0011},     // 0.001 + 0.0001
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateBill(tc.co2e, tc.requests)
			if !approxEqual(got, tc.want, 1e-9) {
				t.Errorf("CalculateBill(%.2f, %d) = %.10f, want %.10f", tc.co2e, tc.requests, got, tc.want)
			}
		})
	}
}

// ── ApplyVolumeDiscount ────────────────────────────────────────────────────────

func TestApplyVolumeDiscount(t *testing.T) {
	tests := []struct {
		monthly  int64
		wantPct  float64
	}{
		{0, 0},
		{999_999, 0},
		{1_000_000, 15},
		{5_000_000, 15},
		{9_999_999, 15},
		{10_000_000, 25},
		{100_000_000, 25},
	}

	for _, tc := range tests {
		got := ApplyVolumeDiscount(tc.monthly)
		if got != tc.wantPct {
			t.Errorf("ApplyVolumeDiscount(%d) = %.1f, want %.1f", tc.monthly, got, tc.wantPct)
		}
	}
}

// ── generateDailyBillingEvents ─────────────────────────────────────────────────

// mockUsageReader is a test double for UsageReader.
type mockUsageReader struct {
	orgs         []string
	usageByOrg   map[string]*UsageSummaryDTO
	monthlyCounts map[string]int64
}

func (m *mockUsageReader) ListOrgsWithUsage(_ context.Context, _ time.Time) ([]string, error) {
	return m.orgs, nil
}

func (m *mockUsageReader) GetUsageForBilling(_ context.Context, orgID string, _ time.Time) (*UsageSummaryDTO, error) {
	if u, ok := m.usageByOrg[orgID]; ok {
		return u, nil
	}
	return &UsageSummaryDTO{}, nil
}

func (m *mockUsageReader) GetMonthlyRequestCount(_ context.Context, orgID string, _ time.Time) (int64, error) {
	return m.monthlyCounts[orgID], nil
}

// mockBillingRepo captures inserted events.
type mockBillingRepo struct {
	inserted []*BillingEvent
}

func (m *mockBillingRepo) Insert(_ context.Context, ev *BillingEvent) error {
	m.inserted = append(m.inserted, ev)
	return nil
}

func TestGenerateDailyBillingEvents_TwoOrgs(t *testing.T) {
	usageReader := &mockUsageReader{
		orgs: []string{
			"00000000-0000-0000-0000-000000000001",
			"00000000-0000-0000-0000-000000000002",
		},
		usageByOrg: map[string]*UsageSummaryDTO{
			"00000000-0000-0000-0000-000000000001": {
				TotalRequests:  1000,
				TotalTokens:    50000,
				TotalEnergyKwh: 0.5,
				TotalCO2eGrams: 200,
				TotalCostUSD:   0.3,
			},
			"00000000-0000-0000-0000-000000000002": {
				TotalRequests:  500,
				TotalTokens:    25000,
				TotalEnergyKwh: 0.25,
				TotalCO2eGrams: 100,
				TotalCostUSD:   0.15,
			},
		},
		monthlyCounts: map[string]int64{
			"00000000-0000-0000-0000-000000000001": 2_000_000, // → 15% discount
			"00000000-0000-0000-0000-000000000002": 500_000,   // → 0% discount
		},
	}

	repo := &mockBillingRepo{}

	if err := generateDailyBillingEvents(repo, usageReader); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.inserted) != 2 {
		t.Fatalf("expected 2 billing events, got %d", len(repo.inserted))
	}

	// Org 1: co2e=200g, requests=1000 → subtotal = 200*0.001 + 1000*0.0001 = 0.2 + 0.1 = 0.3
	// discount 15% → total = 0.3 * 0.85 = 0.255
	ev1 := findEvent(repo.inserted, "00000000-0000-0000-0000-000000000001")
	if ev1 == nil {
		t.Fatal("billing event for org1 not found")
	}
	if !approxEqual(ev1.SubtotalUSD, 0.3, 1e-9) {
		t.Errorf("org1 subtotal: got %.6f, want 0.300000", ev1.SubtotalUSD)
	}
	if !approxEqual(ev1.DiscountPercent, 15, 1e-9) {
		t.Errorf("org1 discount: got %.1f, want 15", ev1.DiscountPercent)
	}
	if !approxEqual(ev1.TotalUSD, 0.255, 1e-9) {
		t.Errorf("org1 total: got %.6f, want 0.255000", ev1.TotalUSD)
	}

	// Org 2: co2e=100g, requests=500 → subtotal = 0.1 + 0.05 = 0.15
	// discount 0% → total = 0.15
	ev2 := findEvent(repo.inserted, "00000000-0000-0000-0000-000000000002")
	if ev2 == nil {
		t.Fatal("billing event for org2 not found")
	}
	if !approxEqual(ev2.SubtotalUSD, 0.15, 1e-9) {
		t.Errorf("org2 subtotal: got %.6f, want 0.150000", ev2.SubtotalUSD)
	}
	if ev2.DiscountPercent != 0 {
		t.Errorf("org2 discount: got %.1f, want 0", ev2.DiscountPercent)
	}
}

func TestGenerateDailyBillingEvents_SkipsZeroRequests(t *testing.T) {
	usageReader := &mockUsageReader{
		orgs: []string{"00000000-0000-0000-0000-000000000001"},
		usageByOrg: map[string]*UsageSummaryDTO{
			"00000000-0000-0000-0000-000000000001": {TotalRequests: 0},
		},
		monthlyCounts: map[string]int64{},
	}
	repo := &mockBillingRepo{}

	if err := generateDailyBillingEvents(repo, usageReader); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.inserted) != 0 {
		t.Errorf("expected 0 events for org with zero requests, got %d", len(repo.inserted))
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func approxEqual(a, b, epsilon float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= epsilon
}

func findEvent(events []*BillingEvent, orgID string) *BillingEvent {
	for _, e := range events {
		if e.OrgID.String() == orgID {
			return e
		}
	}
	return nil
}

