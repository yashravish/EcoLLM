package proxy

import (
	"bytes"
	"io"
	"net/http"
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

// Proxy forwards chat completion requests to the appropriate model backend.
type Proxy struct {
	endpoints ModelEndpoints
	client    *http.Client
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
	}
}

// Forward returns a Fiber handler that proxies the request to the named model's backend.
func (p *Proxy) Forward(model string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		upstream := p.backendURL(model) + "/v1/chat/completions"

		req, err := http.NewRequestWithContext(c.Context(), http.MethodPost, upstream, bytes.NewReader(c.Body()))
		if err != nil {
			log.Error().Err(err).Str("model", model).Msg("failed to build upstream request")
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to build upstream request"})
		}
		req.Header.Set("Content-Type", "application/json")

		start := time.Now()
		resp, err := p.client.Do(req)
		if err != nil {
			log.Error().Err(err).Str("model", model).Msg("upstream request failed")
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "upstream inference failed"})
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to read upstream response"})
		}

		log.Debug().
			Str("model", model).
			Int("status", resp.StatusCode).
			Int64("latency_ms", time.Since(start).Milliseconds()).
			Msg("upstream inference complete")

		c.Status(resp.StatusCode)
		c.Set("Content-Type", resp.Header.Get("Content-Type"))
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
