package chat

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
)

// Handler exposes the inference endpoints. Only request parsing and response
// serialization happen here; all orchestration is in Service.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// CreateCompletion handles POST /v1/chat/completions and POST /v1/completions.
// When the request body contains "stream": true, the response is an SSE stream
// in the OpenAI-compatible format so existing client SDKs work without changes.
func (h *Handler) CreateCompletion(c *fiber.Ctx) error {
	var req CompletionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}

	if err := validateCompletionRequest(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	if req.MaxTokens == 0 {
		req.MaxTokens = 512
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	orgID, _ := c.Locals("org_id").(string)
	requestID, _ := c.Locals("request_id").(string)

	if req.Stream {
		return h.handleStream(c, orgID, requestID, &req)
	}

	resp, err := h.svc.Complete(c.UserContext(), orgID, requestID, &req)
	if err != nil {
		if errors.Is(err, ErrDuplicateRequest) {
			return c.Status(fiber.StatusConflict).JSON(&apierror.APIError{
				Code:    409,
				Message: "Duplicate request submitted within 5 seconds",
				Type:    "validation_error",
			})
		}
		return c.Status(fiber.StatusBadGateway).JSON(apierror.ErrInferenceFailed)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// handleStream opens an SSE connection and writes StreamChunks as they arrive.
// Each chunk is serialised as "data: {json}\n\n"; the stream is terminated with
// "data: [DONE]\n\n" so clients following the OpenAI streaming spec work as-is.
func (h *Handler) handleStream(c *fiber.Ctx, orgID, requestID string, req *CompletionRequest) error {
	ch, err := h.svc.CompleteStream(c.UserContext(), orgID, requestID, req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(apierror.ErrInferenceFailed)
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // disable nginx proxy buffering

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for chunk := range ch {
			b, err := json.Marshal(chunk)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", b)
			w.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		w.Flush()
	})
	return nil
}

// PreviewRoute handles POST /v1/route/preview — returns the routing decision
// without running inference. Useful for cost estimation and debugging.
func (h *Handler) PreviewRoute(c *fiber.Ctx) error {
	var req CompletionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}
	if len(req.Messages) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("messages", "required"))
	}

	orgID, _ := c.Locals("org_id").(string)

	preview, err := h.svc.PreviewRoute(c.UserContext(), orgID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.Status(fiber.StatusOK).JSON(preview)
}

// ListModels handles GET /v1/models.
func (h *Handler) ListModels(c *fiber.Ctx) error {
	models, err := h.svc.ListModels(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"models": models})
}

func validateCompletionRequest(req *CompletionRequest) *apierror.APIError {
	if len(req.Messages) == 0 {
		return apierror.Validation("messages", "required, must contain at least one message")
	}
	for i, m := range req.Messages {
		if m.Role == "" || m.Content == "" {
			return apierror.Validation("messages", fmt.Sprintf("message[%d] must have role and content", i))
		}
		if m.Role != "system" && m.Role != "user" && m.Role != "assistant" {
			return apierror.Validation("messages", fmt.Sprintf("message[%d].role must be system|user|assistant", i))
		}
	}
	if req.MaxTokens < 0 || req.MaxTokens > 4096 {
		return apierror.Validation("max_tokens", "must be between 0 and 4096")
	}
	if req.Temperature < 0 || req.Temperature > 2 {
		return apierror.Validation("temperature", "must be between 0 and 2")
	}
	return nil
}
