package admin

import (
	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// Handler exposes admin endpoints for model registry management and system metrics.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetMetrics handles GET /admin/metrics
func (h *Handler) GetMetrics(c *fiber.Ctx) error {
	metrics, err := h.svc.GetSystemMetrics(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}
	return c.JSON(metrics)
}

// ListModels handles GET /admin/models
func (h *Handler) ListModels(c *fiber.Ctx) error {
	models, err := h.svc.ListModels(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}
	return c.JSON(fiber.Map{"models": models})
}

// CreateModel handles POST /admin/models
func (h *Handler) CreateModel(c *fiber.Ctx) error {
	var in CreateModelInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}
	if in.Name == "" || in.Runtime == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("name/runtime", "required"))
	}

	model, err := h.svc.CreateModel(c.UserContext(), in)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}
	return c.Status(fiber.StatusCreated).JSON(model)
}

// UpdateModel handles PATCH /admin/models/:id
func (h *Handler) UpdateModel(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("id", "required"))
	}

	var in UpdateModelInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}

	model, err := h.svc.UpdateModel(c.UserContext(), id, in)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apierror.ErrNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}
	return c.JSON(model)
}

// GetRoutes handles GET /admin/routes
func (h *Handler) GetRoutes(c *fiber.Ctx) error {
	routes, err := h.svc.GetRoutes(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}
	return c.JSON(fiber.Map{"routes": routes})
}

// UpdateRoutes handles PATCH /admin/routes — enables or disables models in the routing pool.
func (h *Handler) UpdateRoutes(c *fiber.Ctx) error {
	var body struct {
		Updates []UpdateRouteInput `json:"updates"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}
	if len(body.Updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("updates", "required"))
	}

	if err := h.svc.UpdateRoutes(c.UserContext(), body.Updates); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// GetCarbonMetrics handles GET /admin/carbon
func (h *Handler) GetCarbonMetrics(c *fiber.Ctx) error {
	days, err := h.svc.GetCarbonMetrics(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}
	return c.JSON(fiber.Map{"days": days})
}
