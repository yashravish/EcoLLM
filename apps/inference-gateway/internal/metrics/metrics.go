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
)