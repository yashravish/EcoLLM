package proxy

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// ModelEndpoints holds the base URL for each model's inference backend.
type ModelEndpoints struct {
	Phi3URL     string
	MistralURL  string
	Llama13BURL string
	Llama70BURL string
}

// Proxy forwards chat completion requests to the appropriate model backend
// and, when GPU metrics are available, injects a measured energy header so the
// API can use actual watt-hours instead of a static power estimate.
type Proxy struct {
	endpoints      ModelEndpoints
	client         *http.Client
	metricsClient  *http.Client // short-timeout client dedicated to /metrics scrapes
}

func New(endpoints ModelEndpoints) *Proxy {
	return &Proxy{
		endpoints: endpoints,
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		// 500 ms is enough for a local Prometheus scrape; if it times out the
		// energy header is simply omitted and the API falls back to static estimate.
		metricsClient: &http.Client{Timeout: 500 * time.Millisecond},
	}
}

// Forward returns a Fiber handler that proxies the request to the named model's
// backend, measures GPU power draw around the inference call, and injects
// X-Measured-Energy-Wh so the API layer can use real hardware telemetry.
func (p *Proxy) Forward(model string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		upstream := p.backendURL(model) + "/v1/chat/completions"
		metricsURL := p.backendURL(model) + "/metrics"

		req, err := http.NewRequestWithContext(c.Context(), http.MethodPost, upstream, bytes.NewReader(c.Body()))
		if err != nil {
			log.Error().Err(err).Str("model", model).Msg("failed to build upstream request")
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to build upstream request"})
		}
		req.Header.Set("Content-Type", "application/json")

		// Sample GPU power before inference. Returns 0 when DCGM/NVML is absent.
		powerBefore := p.samplePowerWatts(metricsURL)
		start := time.Now()

		resp, err := p.client.Do(req)
		if err != nil {
			log.Error().Err(err).Str("model", model).Msg("upstream request failed")
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "upstream inference failed"})
		}
		defer resp.Body.Close()

		durationMs := time.Since(start).Milliseconds()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to read upstream response"})
		}

		// Sample power again post-inference and compute measured energy.
		powerAfter := p.samplePowerWatts(metricsURL)

		log.Debug().
			Str("model", model).
			Int("status", resp.StatusCode).
			Int64("latency_ms", durationMs).
			Float64("power_before_w", powerBefore).
			Float64("power_after_w", powerAfter).
			Msg("upstream inference complete")

		c.Status(resp.StatusCode)
		c.Set("Content-Type", resp.Header.Get("Content-Type"))

		// Inject measured energy header when GPU telemetry is available.
		if wh := p.computeMeasuredEnergyWh(powerBefore, powerAfter, durationMs); wh > 0 {
			c.Set("X-Measured-Energy-Wh", strconv.FormatFloat(wh, 'f', 9, 64))
		}

		return c.Send(body)
	}
}

// ListModels proxies a GET /models request to the named model's backend.
func (p *Proxy) ListModels(c *fiber.Ctx) error {
	model := c.Params("model")
	upstream := p.backendURL(model) + "/v1/models"

	req, err := http.NewRequestWithContext(c.Context(), http.MethodGet, upstream, nil)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to build request"})
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "upstream unavailable"})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Status(resp.StatusCode)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func (p *Proxy) backendURL(model string) string {
	switch model {
	case "phi3":
		return p.endpoints.Phi3URL
	case "mistral":
		return p.endpoints.MistralURL
	case "llama13b":
		return p.endpoints.Llama13BURL
	case "llama70b":
		return p.endpoints.Llama70BURL
	default:
		return p.endpoints.Phi3URL
	}
}

// samplePowerWatts queries the vLLM /metrics endpoint for the DCGM gauge
// nv_gpu_power_usage_watts. Returns 0 when the metric is absent or the
// endpoint is unreachable (non-GPU host, dev environment).
func (p *Proxy) samplePowerWatts(metricsURL string) float64 {
	req, err := http.NewRequest(http.MethodGet, metricsURL, nil)
	if err != nil {
		return 0
	}
	resp, err := p.metricsClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return 0
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0
	}
	return parsePrometheusGauge(string(body), "nv_gpu_power_usage_watts")
}

// computeMeasuredEnergyWh calculates watt-hours for the inference call using
// the average of two power samples and applying the Green Grid PUE factor (1.3).
// Returns 0 when both power samples are zero (no GPU telemetry available).
func (p *Proxy) computeMeasuredEnergyWh(powerBefore, powerAfter float64, durationMs int64) float64 {
	if powerBefore == 0 && powerAfter == 0 {
		return 0
	}
	avgPower := powerBefore + powerAfter
	count := 0.0
	if powerBefore > 0 {
		count++
	}
	if powerAfter > 0 {
		count++
	}
	avgPower /= count

	const pue = 1.3
	durationHours := float64(durationMs) / 3_600_000.0
	return avgPower * durationHours * pue
}

// parsePrometheusGauge extracts the numeric value of the first line whose name
// matches metricName from a Prometheus text-format metrics body.
//
// Handles both labelled ("metric{k=v} 45.3") and unlabelled ("metric 45.3") lines.
func parsePrometheusGauge(body, metricName string) float64 {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, metricName) {
			continue
		}
		// The numeric value is always the last whitespace-separated field.
		// Optional timestamp is ignored.
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if v, err := strconv.ParseFloat(fields[len(fields)-1], 64); err == nil {
			return v
		}
	}
	return 0
}
