package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// All metrics follow the naming convention: ecollm_{subsystem}_{metric_name}_{unit}
// as defined in AGENT_KNOWLEDGE_BASE.md Layer 7.

var (
	// Request metrics
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ecollm_requests_total",
		Help: "Total requests by model, task_type, and status",
	}, []string{"model", "task_type", "status"})

	RequestLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ecollm_request_latency_seconds",
		Help:    "Request latency distribution in seconds",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
	}, []string{"model", "task_type"})

	// Energy metrics
	EnergyPerRequest = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ecollm_energy_kwh_per_request",
		Help:    "Energy consumption per request in kWh",
		Buckets: []float64{0.000001, 0.00001, 0.0001, 0.001},
	}, []string{"model"})

	CO2ePerRequest = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ecollm_co2e_grams_per_request",
		Help:    "CO2 equivalent emissions per request in grams",
		Buckets: []float64{0.1, 0.5, 1, 5, 10, 50},
	}, []string{"model"})

	// Cache metrics
	CacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ecollm_cache_hits_total",
		Help: "Cache hits by cache_type",
	}, []string{"cache_type"})

	CacheMisses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ecollm_cache_misses_total",
		Help: "Cache misses by cache_type",
	}, []string{"cache_type"})

	// Routing metrics
	RoutingDecisions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ecollm_routing_decisions_total",
		Help: "Routing decisions by model and task_type",
	}, []string{"model", "task_type"})

	RoutingScore = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ecollm_routing_score",
		Help:    "Routing score distribution",
		Buckets: prometheus.LinearBuckets(0, 0.1, 11),
	}, []string{"model"})

	FallbacksUsed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ecollm_fallbacks_used_total",
		Help: "Fallbacks triggered by primary_model",
	}, []string{"primary_model", "fallback_model"})

	// Cost metrics
	CostPerRequest = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ecollm_cost_usd_per_request",
		Help:    "Cost per request in USD",
		Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01},
	}, []string{"model", "org_id"})

	// Model health
	ModelHealthStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ecollm_model_health_status",
		Help: "Model health status (1=healthy, 0=unhealthy)",
	}, []string{"model"})

	// Inference throughput
	TokensPerSecond = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ecollm_tokens_per_second",
		Help: "Current tokens per second throughput by model",
	}, []string{"model"})
)

// RequestResult carries the data needed to record a single completed request.
type RequestResult struct {
	Model      string
	OrgID      string
	TaskType   string
	Status     string // "success" | "error" | "fallback"
	LatencyMs  int
	EnergyKwh  float64
	CO2eGrams  float64
	CostUSD    float64
	UsedFallback  bool
	FallbackModel string
}

// RecordRequest updates all relevant Prometheus metrics for one completed request.
func RecordRequest(r RequestResult) {
	status := r.Status
	if status == "" {
		status = "success"
	}

	RequestsTotal.WithLabelValues(r.Model, r.TaskType, status).Inc()
	RequestLatency.WithLabelValues(r.Model, r.TaskType).Observe(float64(r.LatencyMs) / 1000.0)
	EnergyPerRequest.WithLabelValues(r.Model).Observe(r.EnergyKwh)
	CO2ePerRequest.WithLabelValues(r.Model).Observe(r.CO2eGrams)
	RoutingDecisions.WithLabelValues(r.Model, r.TaskType).Inc()
	CostPerRequest.WithLabelValues(r.Model, r.OrgID).Observe(r.CostUSD)

	if r.UsedFallback && r.FallbackModel != "" {
		FallbacksUsed.WithLabelValues(r.Model, r.FallbackModel).Inc()
	}
}
