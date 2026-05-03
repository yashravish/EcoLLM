package telemetry

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordRequest_IncrementsRequestsTotal(t *testing.T) {
	before := testutil.ToFloat64(RequestsTotal.WithLabelValues("phi_3", "simple", "success"))
	RecordRequest(RequestResult{
		Model:     "phi_3",
		OrgID:     "org-test",
		TaskType:  "simple",
		Status:    "success",
		LatencyMs: 80,
		EnergyKwh: 1e-6,
		CO2eGrams: 0.45,
		CostUSD:   0.0001,
	})
	after := testutil.ToFloat64(RequestsTotal.WithLabelValues("phi_3", "simple", "success"))
	if after-before != 1 {
		t.Errorf("RequestsTotal increment = %.0f, want 1", after-before)
	}
}

func TestRecordRequest_RecordsLatency(t *testing.T) {
	// Verify the histogram accepts the observation without panicking.
	// testutil.CollectAndCount confirms the metric is registered.
	RecordRequest(RequestResult{
		Model:     "mistral_7b",
		OrgID:     "org-latency",
		TaskType:  "hard",
		Status:    "success",
		LatencyMs: 400,
		EnergyKwh: 5e-6,
		CO2eGrams: 2.25,
		CostUSD:   0.0005,
	})
	count := testutil.CollectAndCount(RequestLatency)
	if count == 0 {
		t.Error("RequestLatency histogram should have observations")
	}
}

func TestRecordRequest_FallbackIncrementsFallbacksUsed(t *testing.T) {
	before := testutil.ToFloat64(FallbacksUsed.WithLabelValues("phi_3", "mistral_7b"))
	RecordRequest(RequestResult{
		Model:         "phi_3",
		OrgID:         "org-fallback",
		TaskType:      "simple",
		Status:        "fallback",
		LatencyMs:     300,
		EnergyKwh:     5e-6,
		CO2eGrams:     2.25,
		CostUSD:       0.0005,
		UsedFallback:  true,
		FallbackModel: "mistral_7b",
	})
	after := testutil.ToFloat64(FallbacksUsed.WithLabelValues("phi_3", "mistral_7b"))
	if after-before != 1 {
		t.Errorf("FallbacksUsed increment = %.0f, want 1", after-before)
	}
}

func TestRecordRequest_NoFallback_DoesNotIncrementFallbacksUsed(t *testing.T) {
	before := testutil.ToFloat64(FallbacksUsed.WithLabelValues("llama_13b", "llama_70b"))
	RecordRequest(RequestResult{
		Model:        "llama_13b",
		OrgID:        "org-no-fallback",
		TaskType:     "hard",
		Status:       "success",
		LatencyMs:    800,
		EnergyKwh:    1e-5,
		CO2eGrams:    4.5,
		CostUSD:      0.001,
		UsedFallback: false,
	})
	after := testutil.ToFloat64(FallbacksUsed.WithLabelValues("llama_13b", "llama_70b"))
	if after != before {
		t.Errorf("FallbacksUsed should not increment when UsedFallback=false, got +%.0f", after-before)
	}
}

func TestRecordRequest_DefaultStatusIsSuccess(t *testing.T) {
	before := testutil.ToFloat64(RequestsTotal.WithLabelValues("phi_3", "simple", "success"))
	RecordRequest(RequestResult{
		Model:     "phi_3",
		OrgID:     "org-default",
		TaskType:  "simple",
		LatencyMs: 80,
		EnergyKwh: 1e-6,
	})
	after := testutil.ToFloat64(RequestsTotal.WithLabelValues("phi_3", "simple", "success"))
	if after-before != 1 {
		t.Errorf("empty Status should default to 'success', got increment=%.0f", after-before)
	}
}
