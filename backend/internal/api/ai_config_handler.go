package api

import (
	"errors"
	"net/http"

	"audit-platform/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AIConfigHandler handles AI model configuration CRUD.
type AIConfigHandler struct {
	configSvc *service.AIConfigService
}

func NewAIConfigHandler(configSvc *service.AIConfigService) *AIConfigHandler {
	return &AIConfigHandler{configSvc: configSvc}
}

// GetConfig handles GET /api/v1/ai-config.
func (h *AIConfigHandler) GetConfig(c *fiber.Ctx) error {
	tenantID, _ := uuid.Parse(c.Locals("tenant_id").(string))
	cfg, err := h.configSvc.Get(c.Context(), tenantID)
	if err != nil {
		if errors.Is(err, service.ErrAIConfigNotFound) {
			return resp.ok(c, fiber.Map{})
		}
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}
	return resp.ok(c, cfg)
}

// SaveConfig handles PUT /api/v1/ai-config.
func (h *AIConfigHandler) SaveConfig(c *fiber.Ctx) error {
	var req service.UpdateAIConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	tenantID, _ := uuid.Parse(c.Locals("tenant_id").(string))
	cfg, err := h.configSvc.Save(c.Context(), tenantID, req)
	if err != nil {
		return resp.error(c, http.StatusBadRequest, err.Error(), nil)
	}

	return resp.ok(c, cfg)
}
