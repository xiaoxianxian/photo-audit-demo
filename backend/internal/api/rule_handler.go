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

// --- TenantRuleHandler ---

type TenantRuleHandler struct {
	ruleService *service.TenantRuleService
}

func NewTenantRuleHandler(ruleSvc *service.TenantRuleService) *TenantRuleHandler {
	return &TenantRuleHandler{ruleService: ruleSvc}
}

// List handles GET /api/v1/audit-rules?tenant_id=<uuid>&page=&page_size=.
func (h *TenantRuleHandler) List(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(c.Query("tenant_id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "missing or invalid tenant_id", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	rules, total, err := h.ruleService.List(c.Context(), tenantID, page, pageSize)
	if err != nil {
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "ok",
		"data":    rules,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

// Create handles POST /api/v1/audit-rules.
func (h *TenantRuleHandler) Create(c *fiber.Ctx) error {
	var req model.CreateTenantAuditRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	tenantID, _ := uuid.Parse(c.Locals("tenant_id").(string))
	rule, err := h.ruleService.Create(c.Context(), tenantID, req)
	if err != nil {
		return resp.error(c, http.StatusBadRequest, err.Error(), nil)
	}

	return resp.created(c, rule)
}

// GetByID handles GET /api/v1/audit-rules/:id.
func (h *TenantRuleHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid rule id", nil)
	}

	rule, err := h.ruleService.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrRuleNotFound) {
			return resp.error(c, http.StatusNotFound, "rule not found", nil)
		}
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, rule)
}

// Update handles PUT /api/v1/audit-rules/:id.
func (h *TenantRuleHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid rule id", nil)
	}

	var req model.UpdateTenantAuditRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	rule, err := h.ruleService.Update(c.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRuleNotFound):
			return resp.error(c, http.StatusNotFound, "rule not found", nil)
		case errors.Is(err, service.ErrInvalidRuleAction):
			return resp.error(c, http.StatusBadRequest, "invalid rule action", nil)
		default:
			return resp.error(c, http.StatusBadRequest, err.Error(), nil)
		}
	}

	return resp.ok(c, rule)
}

// Delete handles DELETE /api/v1/audit-rules/:id.
func (h *TenantRuleHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid rule id", nil)
	}

	if err := h.ruleService.Delete(c.Context(), id); err != nil {
		if errors.Is(err, service.ErrRuleNotFound) {
			return resp.error(c, http.StatusNotFound, "rule not found", nil)
		}
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, fiber.Map{"deleted": true})
}
