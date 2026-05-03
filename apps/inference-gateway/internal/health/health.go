package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ecollm/inference-gateway/internal/proxy"
)

// Handler checks liveness and per-model readiness.
type Handler struct {
	endpoints proxy.ModelEndpoints
	client    *http.Client
}

func New(endpoints proxy.ModelEndpoints) *Handler {
	return &Handler{
		endpoints: endpoints,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Check returns 200 if the gateway process is alive.
func (h *Handler) Check(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok", "service": "inference-gateway"})
}

// CheckModel probes the named model's backend /health endpoint.
func (h *Handler) CheckModel(c *fiber.Ctx) error {
	model := c.Params("model")
	url := h.modelHealthURL(model)
	if url == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "unknown model"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"model": model, "status": "unhealthy"})
	}

	resp, err := h.client.Do(req)
	if err != nil || resp.StatusCode >= 500 {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"model": model, "status": "unhealthy"})
	}
	return c.JSON(fiber.Map{"model": model, "status": "healthy"})
}

func (h *Handler) modelHealthURL(model string) string {
	switch model {
	case "phi3":
		return h.endpoints.Phi3URL + "/health"
	case "mistral":
		return h.endpoints.MistralURL + "/health"
	case "llama13b":
		return h.endpoints.Llama13BURL + "/health"
	case "llama70b":
		return h.endpoints.Llama70BURL + "/health"
	default:
		return ""
	}
}
