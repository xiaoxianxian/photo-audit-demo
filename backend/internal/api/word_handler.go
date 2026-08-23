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

// --- TenantWordHandler ---

type TenantWordHandler struct {
	wordService *service.TenantWordService
}

func NewTenantWordHandler(wordSvc *service.TenantWordService) *TenantWordHandler {
	return &TenantWordHandler{wordService: wordSvc}
}

// List handles GET /api/v1/custom-words.
func (h *TenantWordHandler) List(c *fiber.Ctx) error {
	tenantID, err := uuid.Parse(c.Query("tenant_id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "missing or invalid tenant_id", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	words, total, err := h.wordService.List(c.Context(), tenantID, page, pageSize)
	if err != nil {
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "ok",
		"data":    words,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	})
}

// Create handles POST /api/v1/custom-words.
func (h *TenantWordHandler) Create(c *fiber.Ctx) error {
	var req model.CreateTenantCustomWordRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	tenantID, _ := uuid.Parse(c.Locals("tenant_id").(string))
	word, err := h.wordService.Create(c.Context(), tenantID, req)
	if err != nil {
		return resp.error(c, http.StatusBadRequest, err.Error(), nil)
	}

	return resp.created(c, word)
}

// GetByID handles GET /api/v1/custom-words/:id.
func (h *TenantWordHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid word id", nil)
	}

	word, err := h.wordService.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrWordNotFound) {
			return resp.error(c, http.StatusNotFound, "word not found", nil)
		}
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, word)
}

// Update handles PUT /api/v1/custom-words/:id.
func (h *TenantWordHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid word id", nil)
	}

	var req model.UpdateTenantCustomWordRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid request body", nil)
	}

	word, err := h.wordService.Update(c.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWordNotFound):
			return resp.error(c, http.StatusNotFound, "word not found", nil)
		default:
			return resp.error(c, http.StatusBadRequest, err.Error(), nil)
		}
	}

	return resp.ok(c, word)
}

// Delete handles DELETE /api/v1/custom-words/:id.
func (h *TenantWordHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return resp.error(c, http.StatusBadRequest, "invalid word id", nil)
	}

	if err := h.wordService.Delete(c.Context(), id); err != nil {
		if errors.Is(err, service.ErrWordNotFound) {
			return resp.error(c, http.StatusNotFound, "word not found", nil)
		}
		return resp.error(c, http.StatusInternalServerError, "internal server error", nil)
	}

	return resp.ok(c, fiber.Map{"deleted": true})
}
