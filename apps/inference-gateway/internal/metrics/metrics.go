package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	GPUUtilization = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ecollm_gpu_utilization_percent",
		Help: "Current GPU utilization percentage per model.",
	}, []string{"model"})

	InferenceLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ecollm_inference_latency_ms",
		Help:    "Inference request latency in milliseconds.",
		Buckets: []float64{50, 100, 250, 500, 1000, 2500, 5000, 10000},
	}, []string{"model"})

	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ecollm_inference_requests_total",
		Help: "Total inference requests dispatched.",
	}, []string{"model", "status"})

	TokensGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ecollm_tokens_generated_total",
		Help: "Total completion tokens generated.",
	}, []string{"model"})

	// MeasuredEnergyWh tracks watt-hours from live DCGM/NVML telemetry.
	// When this counter is non-zero it proves real hardware measurement is active;
	// the API uses it to replace static power estimates with auditable numbers.
	MeasuredEnergyWh = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ecollm_measured_energy_wh_total",
		Help: "Cumulative GPU energy measured from DCGM telemetry in watt-hours.",
	}, []string{"model"})

	// TelemetrySource tracks how many requests used measured vs. estimated energy.
	TelemetrySource = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ecollm_energy_source_total",
		Help: "Count of requests by energy accounting source: nvml_measured or static_estimate.",
	}, []string{"model", "source"})
)