package api

import (
	"errors"
	"net/http"
	"strconv"

	"audit-platform/internal/model"
	"audit-platform/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// --- TenantLevelHandler ---

type TenantLevelHandler struct {
	levelService *service.TenantLevelService
}

func NewTenantLevelHandler(levelSvc *service.TenantLevelService) *TenantLevelHandler {
	return &TenantLevelHandler{levelService: levelSvc}
}

// List handles GET /api/v1/audit-levels.
func (h *TenantLevelHandler) List(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(c.Query("tenant_id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "missing or invalid tenant_id", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	levels, total, err := h.levelService.List(c.Context(), tenantID, page, pageSize)
	if err != nil {
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "ok",
		"data":    levels,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

// Create handles POST /api/v1/audit-levels.
func (h *TenantLevelHandler) Create(c *fiber.Ctx) error {
	var req model.CreateTenantAuditLevelRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	tenantID, _ := uuid.Parse(c.Locals("tenant_id").(string))
	level, err := h.levelService.Create(c.Context(), tenantID, req)
	if err != nil {
		return resp.error(c, http.StatusBadRequest, err.Error(), nil)
	}

	return resp.created(c, level)
}

// GetByID handles GET /api/v1/audit-levels/:id.
func (h *TenantLevelHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid level id", nil)
	}

	level, err := h.levelService.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrLevelNotFound) {
			return resp.error(c, http.StatusNotFound, "level not found", nil)
		}
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, level)
}

// Update handles PUT /api/v1/audit-levels/:id.
func (h *TenantLevelHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid level id", nil)
	}

	var req model.UpdateTenantAuditLevelRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	level, err := h.levelService.Update(c.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLevelNotFound):
			return resp.error(c, http.StatusNotFound, "level not found", nil)
		default:
			return resp.error(c, http.StatusBadRequest, err.Error(), nil)
		}
	}

	return resp.ok(c, level)
}

// Delete handles DELETE /api/v1/audit-levels/:id.
func (h *TenantLevelHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid level id", nil)
	}

	if err := h.levelService.Delete(c.Context(), id); err != nil {
		if errors.Is(err, service.ErrLevelNotFound) {
			return resp.error(c, http.StatusNotFound, "level not found", nil)
		}
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, fiber.Map{"deleted": true})
}
