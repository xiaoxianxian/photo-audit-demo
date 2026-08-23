package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"audit-platform/internal/model"
	"audit-platform/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// responder provides common JSON response helpers for all handlers.
type responder struct{}

func (r *responder) ok(c *fiber.Ctx, data interface{}) error {
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "ok",
		"data":    data,
	})
}

func (r *responder) created(c *fiber.Ctx, data interface{}) error {
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"code":    0,
		"message": "created",
		"data":    data,
	})
}

func (r *responder) error(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	return c.JSON(fiber.Map{
		"code":    statusCode,
		"message": message,
		"data":    data,
	})
}

var resp = &responder{}
type Handlers struct {
	AuthHandler          *AuthHandler
	TenantHandler        *TenantHandler
	TeamHandler          *TeamHandler
	ContentHandler       *ContentHandler
	ReviewHandler        *ReviewHandler
	AppealHandler        *AppealHandler
	DashboardHandler     *DashboardHandler
	QualityAuditHandler  *QualityAuditHandler
	LiveWallHandler      *LiveWallHandler
	RuleHandler          *TenantRuleHandler
	LevelHandler         *TenantLevelHandler
	WordHandler          *TenantWordHandler
	AIConfigHandler      *AIConfigHandler
}

// NewHandlers creates all handler instances from the Services container.
func NewHandlers(svc *service.Services) *Handlers {
	return &Handlers{
		AuthHandler:         &AuthHandler{authService: svc.AuthService},
		TenantHandler:       &TenantHandler{tenantService: svc.TenantService},
		TeamHandler:         &TeamHandler{teamService: svc.TeamService},
		ContentHandler: func() *ContentHandler {
			h := NewContentHandlerWithStorage(svc.IngestionService, svc.ContentRepo, svc.MinIO)
			h.elementRepo = svc.ElementRepo
			// Wire video processor if available (attached to ingestion service).
			h.WithVideoProcessor(svc.IngestionService.VideoProcessor())
			return h
		}(),
		ReviewHandler:       NewReviewHandler(svc.ReviewService, svc.ElementRepo, svc.AuthService),
		AppealHandler:       NewAppealHandler(svc.AppealService),
		DashboardHandler:    NewDashboardHandler(svc.DashboardService),
		QualityAuditHandler: NewQualityAuditHandler(svc.QualityAuditService),
		LiveWallHandler: func() *LiveWallHandler {
			h := NewLiveWallHandler(svc.LiveWallService, svc.WsHub, svc.AuthService)
			h.WithScheduler(svc.StreamScheduler)
			return h
		}(),
		RuleHandler:         NewTenantRuleHandler(svc.RuleService),
		LevelHandler:        NewTenantLevelHandler(svc.LevelService),
		WordHandler:         NewTenantWordHandler(svc.WordService),
		AIConfigHandler:     NewAIConfigHandler(svc.AIConfigService),
	}
}

// --- AuthHandler ---

type AuthHandler struct {
	authService *service.AuthService
}

func (h *AuthHandler) error(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	return c.JSON(fiber.Map{
		"code":    statusCode,
		"message": message,
		"data":    data,
	})
}

func (h *AuthHandler) ok(c *fiber.Ctx, data interface{}) error {
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "ok",
		"data":    data,
	})
}

func (h *AuthHandler) created(c *fiber.Ctx, data interface{}) error {
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"code":    0,
		"message": "created",
		"data":    data,
	})
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return h.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		return h.error(c, http.StatusBadRequest, "username is required", nil)
	}
	if req.Password == "" {
		return h.error(c, http.StatusBadRequest, "password is required", nil)
	}

	resp, err := h.authService.Login(c.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			return h.error(c, http.StatusUnauthorized, "invalid username or password", nil)
		case errors.Is(err, service.ErrInvalidPassword):
			return h.error(c, http.StatusUnauthorized, "invalid username or password", nil)
		case errors.Is(err, service.ErrInactiveUser):
			return h.error(c, http.StatusForbidden, "account is disabled", nil)
		default:
			return h.error(c, http.StatusInternalServerError, "internal server error", nil)
		}
	}

	return h.ok(c, resp)
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return h.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	if strings.TrimSpace(req.Username) == "" {
		return h.error(c, http.StatusBadRequest, "username is required", nil)
	}
	if req.Password == "" {
		return h.error(c, http.StatusBadRequest, "password is required", nil)
	}

	user, err := h.authService.Register(c.Context(), req)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "invalid role"):
			return h.error(c, http.StatusBadRequest, "invalid role", nil)
		case strings.Contains(err.Error(), "password"):
			return h.error(c, http.StatusBadRequest, err.Error(), nil)
		case strings.Contains(err.Error(), "already exists"):
			return h.error(c, http.StatusConflict, err.Error(), nil)
		default:
			return h.error(c, http.StatusBadRequest, err.Error(), nil)
		}
	}

	return h.created(c, user)
}

// --- TenantHandler ---

type TenantHandler struct {
	tenantService *service.TenantService
}

// List handles GET /api/v1/tenants with ?page=&page_size= query params.
func (h *TenantHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	tenants, total, err := h.tenantService.List(c.Context(), page, pageSize)
	if err != nil {
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, fiber.Map{
		"tenants":   tenants,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Create handles POST /api/v1/tenants.
func (h *TenantHandler) Create(c *fiber.Ctx) error {
	var req model.CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	if strings.TrimSpace(req.Name) == "" {
		return resp.error(c, http.StatusBadRequest, "tenant name is required", nil)
	}
	if strings.TrimSpace(req.CountryCode) == "" {
		return resp.error(c, http.StatusBadRequest, "country code is required", nil)
	}

	// Extract the platform admin user who created this tenant.
	createdBy := uuid.Nil
	userID := c.Locals("user_id")
	if userID != nil {
		createdBy, _ = uuid.Parse(userID.(string))
	}

	tenant, err := h.tenantService.Create(c.Context(), req, createdBy)
	if err != nil {
		return resp.error(c, http.StatusBadRequest, err.Error(), nil)
	}

	return resp.created(c, tenant)
}

// GetByID handles GET /api/v1/tenants/:id.
func (h *TenantHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid tenant id", nil)
	}

	tenant, err := h.tenantService.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return resp.error(c, http.StatusNotFound, "tenant not found", nil)
		}
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, tenant)
}

// Update handles PUT /api/v1/tenants/:id.
func (h *TenantHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid tenant id", nil)
	}

	var req model.UpdateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	tenant, err := h.tenantService.Update(c.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			return resp.error(c, http.StatusNotFound, "tenant not found", nil)
		case errors.Is(err, service.ErrInvalidCountry):
			return resp.error(c, http.StatusBadRequest, "invalid country code", nil)
		default:
			return resp.error(c, http.StatusBadRequest, err.Error(), nil)
		}
	}

	return resp.ok(c, tenant)
}

// Delete handles DELETE /api/v1/tenants/:id (soft delete).
func (h *TenantHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid tenant id", nil)
	}

	if err := h.tenantService.Delete(c.Context(), id); err != nil {
		if errors.Is(err, service.ErrTenantNotFound) {
			return resp.error(c, http.StatusNotFound, "tenant not found", nil)
		}
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, fiber.Map{"deleted": true})
}

// --- TeamHandler ---

type TeamHandler struct {
	teamService *service.TeamService
}

// ListByTenant handles GET /api/v1/teams?tenant_id=<uuid>.
func (h *TeamHandler) ListByTenant(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(c.Query("tenant_id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "missing or invalid tenant_id query param", nil)
	}

	teams, err := h.teamService.ListByTenant(c.Context(), tenantID)
	if err != nil {
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, teams)
}

// Create handles POST /api/v1/teams.
func (h *TeamHandler) Create(c *fiber.Ctx) error {
	var req model.CreateTeamRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	tenantIDStr := c.Locals("tenant_id")
	if tenantIDStr == nil {
		return resp.error(c, http.StatusUnauthorized, "tenant context missing", nil)
	}
	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid tenant_id in context", nil)
	}

	leaderID, err := uuid.Parse(strings.TrimSpace(req.LeaderID))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid leader_id", nil)
	}

	team, err := h.teamService.Create(c.Context(), req, tenantID, leaderID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLeaderNotFound):
			return resp.error(c, http.StatusBadRequest, "leader not found", nil)
		case errors.Is(err, service.ErrLeaderNotInTenant):
			return resp.error(c, http.StatusBadRequest, "leader does not belong to this tenant", nil)
		default:
			return resp.error(c, http.StatusBadRequest, err.Error(), nil)
		}
	}

	return resp.created(c, team)
}

// ListMembers handles GET /api/v1/teams/:id/members.
func (h *TeamHandler) ListMembers(c *fiber.Ctx) error {
	teamID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid team id", nil)
	}

	members, err := h.teamService.ListMembers(c.Context(), teamID)
	if err != nil {
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, members)
}

// AddMember handles POST /api/v1/teams/:id/members.
func (h *TeamHandler) AddMember(c *fiber.Ctx) error {
	teamID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid team id", nil)
	}

	var req model.AddTeamMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	member, err := h.teamService.AddMember(c.Context(), teamID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			return resp.error(c, http.StatusConflict, "user is already a member", nil)
		case strings.Contains(err.Error(), "member_role"):
			return resp.error(c, http.StatusBadRequest, "invalid member role", nil)
		default:
			return resp.error(c, http.StatusBadRequest, err.Error(), nil)
		}
	}

	return resp.created(c, member)
}

// RemoveMember handles DELETE /api/v1/teams/:id/members/:user_id.
func (h *TeamHandler) RemoveMember(c *fiber.Ctx) error {
	teamID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid team id", nil)
	}

	userID, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid user id", nil)
	}

	if err := h.teamService.RemoveMember(c.Context(), teamID, userID); err != nil {
		return resp.error(c, http.StatusInternalServerError, "failed to remove member", nil)
	}

	return resp.ok(c, fiber.Map{"removed": true})
}

// --- Response helpers ---

func (h *Handlers) OK(c *fiber.Ctx, data interface{}) error {
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "ok",
		"data":    data,
	})
}

func (h *Handlers) Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"code":    0,
		"message": "created",
		"data":    data,
	})
}

func (h *Handlers) Error(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	return c.JSON(fiber.Map{
		"code":    statusCode,
		"message": message,
		"data":    data,
	})
}
