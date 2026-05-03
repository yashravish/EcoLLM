package usage

import (
	"strconv"
	"time"

	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// Handler exposes the usage and request-history endpoints.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetUsage handles GET /v1/usage
// Query params: period (daily|monthly), from (YYYY-MM-DD), to (YYYY-MM-DD)
func (h *Handler) GetUsage(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	period := c.Query("period", "daily")
	if period != "daily" && period != "monthly" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("period", "must be 'daily' or 'monthly'"))
	}

	fromStr := c.Query("from", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	toStr := c.Query("to", time.Now().Format("2006-01-02"))

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("from", "must be YYYY-MM-DD"))
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("to", "must be YYYY-MM-DD"))
	}
	to = to.Add(24 * time.Hour) // make to exclusive (include the full to-day)

	resp, err := h.svc.GetUsage(c.UserContext(), orgID, period, from, to)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(resp)
}

// GetRequests handles GET /v1/requests
// Query params: limit (default 20, max 100), offset, model, task_type, status
func (h *Handler) GetRequests(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	f := RequestFilter{
		Model:    c.Query("model"),
		TaskType: c.Query("task_type"),
		Status:   c.Query("status"),
		Limit:    limit,
		Offset:   offset,
	}

	result, err := h.svc.GetRequests(c.UserContext(), orgID, f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(result)
}

// GetCarbon handles GET /v1/carbon
// Query param: period (daily|monthly, default monthly)
func (h *Handler) GetCarbon(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	period := c.Query("period", "monthly")
	if period != "daily" && period != "monthly" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("period", "must be 'daily' or 'monthly'"))
	}

	resp, err := h.svc.GetCarbon(c.UserContext(), orgID, period)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(resp)
}

// GetRequest handles GET /v1/requests/:id — returns full request detail with
// joined energy and carbon data.
func (h *Handler) GetRequest(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	requestID := c.Params("id")
	if requestID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("id", "required"))
	}

	detail, err := h.svc.GetRequest(c.UserContext(), requestID, orgID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apierror.ErrNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(detail)
}
