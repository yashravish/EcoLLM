package billing

import (
	"time"

	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// Handler exposes billing endpoints.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetBilling handles GET /v1/billing
// Query params: from (YYYY-MM-DD), to (YYYY-MM-DD)
func (h *Handler) GetBilling(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	var from, to time.Time
	if s := c.Query("from"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("from", "must be YYYY-MM-DD"))
		}
		from = t
	}
	if s := c.Query("to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("to", "must be YYYY-MM-DD"))
		}
		to = t.Add(24 * time.Hour) // inclusive end
	}

	resp, err := h.svc.GetBilling(c.UserContext(), orgID, from, to)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(resp)
}

// GetBillingEvent handles GET /v1/billing/:id
func (h *Handler) GetBillingEvent(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	eventID := c.Params("id")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("id", "required"))
	}

	ev, err := h.svc.GetBillingEvent(c.UserContext(), orgID, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apierror.ErrNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(ev)
}
