package auth

import (
	"github.com/ecollm/api/internal/audit"
	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// Handler exposes auth and org-management HTTP endpoints.
type Handler struct {
	svc       *Service
	auditRepo *audit.Repository
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// WithAudit attaches an audit repository; call after NewHandler.
func (h *Handler) WithAudit(repo *audit.Repository) *Handler {
	h.auditRepo = repo
	return h
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string        `json:"token"`
	User  *userResponse `json:"user"`
	Org   *orgResponse  `json:"org"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type orgResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Plan string `json:"plan"`
}

// Login handles POST /auth/login.
func (h *Handler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("email/password", "required"))
	}

	token, user, org, err := h.svc.Login(c.UserContext(), req.Email, req.Password)
	if err != nil {
		if h.auditRepo != nil {
			h.auditRepo.WriteAsync(&audit.Entry{
				Action:       "user.login",
				ResourceType: "user",
				IPAddress:    c.IP(),
				UserAgent:    c.Get("User-Agent"),
				Success:      false,
				ErrorMessage: "invalid credentials",
			})
		}
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	if h.auditRepo != nil {
		uid := user.ID
		oid := user.OrgID
		h.auditRepo.WriteAsync(&audit.Entry{
			OrgID:        &oid,
			UserID:       &uid,
			Action:       "user.login",
			ResourceType: "user",
			ResourceID:   &uid,
			IPAddress:    c.IP(),
			UserAgent:    c.Get("User-Agent"),
			Success:      true,
		})
	}

	return c.Status(fiber.StatusOK).JSON(loginResponse{
		Token: token,
		User: &userResponse{
			ID:    user.ID.String(),
			Email: user.Email,
			Name:  user.Name,
			Role:  user.Role,
		},
		Org: &orgResponse{
			ID:   org.ID.String(),
			Name: org.Name,
			Slug: org.Slug,
			Plan: org.Plan,
		},
	})
}

// Logout handles POST /auth/logout.
func (h *Handler) Logout(c *fiber.Ctx) error {
	jti, _ := c.Locals("jti").(string)
	if jti != "" {
		h.svc.Logout(c.UserContext(), jti)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Me handles GET /auth/me and GET /me.
func (h *Handler) Me(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if orgID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	result, err := h.svc.GetMe(c.UserContext(), userID, orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(fiber.Map{
		"user": userResponse{
			ID:    result.User.ID.String(),
			Email: result.User.Email,
			Name:  result.User.Name,
			Role:  result.User.Role,
		},
		"org": orgResponse{
			ID:   result.Org.ID.String(),
			Name: result.Org.Name,
			Slug: result.Org.Slug,
			Plan: result.Org.Plan,
		},
	})
}

// Register handles POST /auth/register.
func (h *Handler) Register(c *fiber.Ctx) error {
	var req struct {
		OrgName  string `json:"org_name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}
	if req.OrgName == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("org_name/email/password", "required"))
	}
	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("password", "must be at least 8 characters"))
	}

	resp, err := h.svc.Register(c.UserContext(), RegisterInput{
		OrgName:  req.OrgName,
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"token":   resp.Token,
		"api_key": resp.APIKey,
		"user": userResponse{
			ID:    resp.User.ID.String(),
			Email: resp.User.Email,
			Name:  resp.User.Name,
			Role:  resp.User.Role,
		},
		"org": orgResponse{
			ID:   resp.Org.ID.String(),
			Name: resp.Org.Name,
			Slug: resp.Org.Slug,
			Plan: resp.Org.Plan,
		},
	})
}

// GetOrg handles GET /organizations/:id.
func (h *Handler) GetOrg(c *fiber.Ctx) error {
	orgID := c.Params("id")
	callerOrgID, _ := c.Locals("org_id").(string)
	if orgID != callerOrgID {
		return c.Status(fiber.StatusForbidden).JSON(apierror.ErrForbidden)
	}

	org, err := h.svc.GetOrg(c.UserContext(), orgID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apierror.ErrNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(org)
}

// UpdateOrg handles PATCH /organizations/:id.
func (h *Handler) UpdateOrg(c *fiber.Ctx) error {
	orgID := c.Params("id")
	callerOrgID, _ := c.Locals("org_id").(string)
	if orgID != callerOrgID {
		return c.Status(fiber.StatusForbidden).JSON(apierror.ErrForbidden)
	}

	var in UpdateOrgInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}

	org, err := h.svc.UpdateOrg(c.UserContext(), orgID, in)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(org)
}

// ListMembers handles GET /organizations/:id/members.
func (h *Handler) ListMembers(c *fiber.Ctx) error {
	orgID := c.Params("id")
	callerOrgID, _ := c.Locals("org_id").(string)
	if orgID != callerOrgID {
		return c.Status(fiber.StatusForbidden).JSON(apierror.ErrForbidden)
	}

	members, err := h.svc.ListMembers(c.UserContext(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.JSON(fiber.Map{"org_id": orgID, "members": members})
}

// InviteMember handles POST /organizations/:id/members.
func (h *Handler) InviteMember(c *fiber.Ctx) error {
	orgID := c.Params("id")
	callerOrgID, _ := c.Locals("org_id").(string)
	if orgID != callerOrgID {
		return c.Status(fiber.StatusForbidden).JSON(apierror.ErrForbidden)
	}

	var in InviteMemberInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}
	if in.Email == "" || in.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("email/password", "required"))
	}

	member, err := h.svc.InviteMember(c.UserContext(), orgID, in)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(member)
}

// UpdateMemberRole handles PATCH /organizations/:id/members/:userID.
func (h *Handler) UpdateMemberRole(c *fiber.Ctx) error {
	orgID := c.Params("id")
	userID := c.Params("userID")
	callerOrgID, _ := c.Locals("org_id").(string)
	if orgID != callerOrgID {
		return c.Status(fiber.StatusForbidden).JSON(apierror.ErrForbidden)
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}

	if err := h.svc.UpdateMemberRole(c.UserContext(), orgID, userID, body.Role); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListAPIKeys handles GET /api-keys.
func (h *Handler) ListAPIKeys(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	keys, err := h.svc.ListAPIKeys(c.UserContext(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	type apiKeyResponse struct {
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		KeyPrefix  string   `json:"key_prefix"`
		Scopes     []string `json:"scopes"`
		LastUsedAt *string  `json:"last_used_at,omitempty"`
		ExpiresAt  *string  `json:"expires_at,omitempty"`
		CreatedAt  string   `json:"created_at"`
	}

	resp := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		r := apiKeyResponse{
			ID:        k.ID.String(),
			Name:      k.Name,
			KeyPrefix: k.KeyPrefix,
			Scopes:    k.Scopes,
			CreatedAt: k.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if k.LastUsedAt != nil {
			s := k.LastUsedAt.Format("2006-01-02T15:04:05Z")
			r.LastUsedAt = &s
		}
		if k.ExpiresAt != nil {
			s := k.ExpiresAt.Format("2006-01-02T15:04:05Z")
			r.ExpiresAt = &s
		}
		resp = append(resp, r)
	}
	return c.JSON(resp)
}

// CreateAPIKey handles POST /api-keys.
func (h *Handler) CreateAPIKey(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	userID, _ := c.Locals("user_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	var req struct {
		Name          string   `json:"name"`
		Scopes        []string `json:"scopes"`
		ExpiresInDays int      `json:"expires_in_days"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("name", "required"))
	}

	result, err := h.svc.CreateAPIKey(c.UserContext(), orgID, userID, CreateAPIKeyInput{
		Name:          req.Name,
		Scopes:        req.Scopes,
		ExpiresInDays: req.ExpiresInDays,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	if h.auditRepo != nil {
		keyID := result.Key.ID
		oid := result.Key.OrgID
		uid := result.Key.CreatedBy
		h.auditRepo.WriteAsync(&audit.Entry{
			OrgID:        &oid,
			UserID:       &uid,
			APIKeyID:     &keyID,
			Action:       "api_key.created",
			ResourceType: "api_key",
			ResourceID:   &keyID,
			IPAddress:    c.IP(),
			UserAgent:    c.Get("User-Agent"),
			Success:      true,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         result.Key.ID.String(),
		"key":        result.Raw,
		"key_prefix": result.Key.KeyPrefix,
		"name":       result.Key.Name,
		"scopes":     result.Key.Scopes,
		"created_at": result.Key.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// RevokeAPIKey handles DELETE /api-keys/:id.
func (h *Handler) RevokeAPIKey(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	keyID := c.Params("id")
	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("id", "required"))
	}

	if err := h.svc.RevokeAPIKey(c.UserContext(), orgID, keyID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	if h.auditRepo != nil {
		userID, _ := c.Locals("user_id").(string)
		h.auditRepo.WriteAsync(&audit.Entry{
			Action:       "api_key.revoked",
			ResourceType: "api_key",
			IPAddress:    c.IP(),
			UserAgent:    c.Get("User-Agent"),
			Success:      true,
		})
		_ = userID // present in locals but UUID parse not needed for async log
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveMember handles DELETE /organizations/:id/members/:userID.
func (h *Handler) RemoveMember(c *fiber.Ctx) error {
	orgID := c.Params("id")
	userID := c.Params("userID")
	callerOrgID, _ := c.Locals("org_id").(string)
	if orgID != callerOrgID {
		return c.Status(fiber.StatusForbidden).JSON(apierror.ErrForbidden)
	}

	if err := h.svc.RemoveMember(c.UserContext(), orgID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
