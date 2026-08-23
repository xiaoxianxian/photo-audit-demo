package api

import (
	"errors"
	"strconv"
	"strings"

	"audit-platform/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AppealHandler handles HTTP requests for appeal submissions and management.
type AppealHandler struct {
	appealSvc *service.AppealService
}

// NewAppealHandler creates a new AppealHandler.
func NewAppealHandler(appealSvc *service.AppealService) *AppealHandler {
	return &AppealHandler{
		appealSvc: appealSvc,
	}
}

// CreateAppealRequest represents the body for submitting a new appeal.
type CreateAppealRequest struct {
	ContentID    string   `json:"content_id" validate:"required"`
	Reason       string   `json:"reason" validate:"required"`
	EvidenceURLs []string `json:"evidence_urls"`
	ApplicantID  string   `json:"applicant_id" validate:"required"`
}

// UpdateAppealRequest represents the body for updating an existing appeal.
type UpdateAppealRequest struct {
	Status     *string `json:"status"`
	Decision   *string `json:"decision"`
	Comment    *string `json:"comment"`
	ReviewerID *string `json:"reviewer_id"`
}

// Submit handles POST /api/v1/appeals — submit a new appeal for a content item.
func (h *AppealHandler) Submit(c *fiber.Ctx) error {
	var req CreateAppealRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	if strings.TrimSpace(req.Reason) == "" {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "reason is required",
		})
	}
	if req.ContentID == "" {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "content_id is required",
		})
	}
	if req.ApplicantID == "" {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "applicant_id is required",
		})
	}

	contentID, err := uuid.Parse(req.ContentID)
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid content_id"})
	}
	applicantID, err := uuid.Parse(req.ApplicantID)
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid applicant_id"})
	}

	// Extract tenant_id from middleware context.
	tenantIDStr := c.Locals("tenant_id")
	var tenantID uuid.UUID
	if tenantIDStr != nil {
		tenantID, _ = uuid.Parse(tenantIDStr.(string))
	}

	appeal, err := h.appealSvc.SubmitAppeal(c.Context(), service.SubmitAppealInput{
		ContentID:    contentID,
		TenantID:     tenantID,
		Reason:       strings.TrimSpace(req.Reason),
		EvidenceURLs: req.EvidenceURLs,
		ApplicantID:  applicantID,
	})
	if err != nil {
		if errors.Is(err, service.ErrAlreadyAppealed) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    409,
				"message": "You have already submitted an appeal for this content",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to submit appeal: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    appeal,
		"message": "Appeal submitted successfully",
	})
}

// GetByID handles GET /api/v1/appeals/:id — retrieve a single appeal by ID.
func (h *AppealHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid appeal ID format",
		})
	}

	appeal, err := h.appealSvc.GetByID(c.Context(), idStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to fetch appeal: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": appeal,
	})
}

// ListByStatus handles GET /api/v1/appeals — list appeals filtered by status with pagination.
func (h *AppealHandler) ListByStatus(c *fiber.Ctx) error {
	status := c.Query("status", "")
	pageStr := c.Query("page", "1")
	pageSizeStr := c.Query("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	if status != "" {
		status = strings.ToLower(strings.TrimSpace(status))
		validStatuses := map[string]bool{
			"submitted":               true,
			"under_review":            true,
			"resolved":                true,
			"resolved_approved":       true,
			"resolved_maintained":     true,
		}
		if !validStatuses[status] {
			return c.JSON(fiber.Map{
				"code":    400,
				"message": "status must be one of: submitted, under_review, resolved, resolved_approved, resolved_maintained",
			})
		}
	}

	// Get tenant ID from context for tenant isolation.
	tenantID, _ := getTenantID(c)

	appeals, total, err := h.appealSvc.ListByTenantAndStatus(c.Context(), tenantID, status, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to list appeals: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":      appeals,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Update handles PUT /api/v1/appeals/:id — update an existing appeal.
func (h *AppealHandler) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid appeal ID format",
		})
	}

	var req UpdateAppealRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	if req.Status == nil && req.Decision == nil && req.Comment == nil && req.ReviewerID == nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "At least one of status, decision, comment, reviewer_id must be provided",
		})
	}

	if req.Decision != nil {
		switch strings.ToLower(*req.Decision) {
		case "approved", "maintained":
			*req.Decision = strings.ToLower(*req.Decision)
		default:
			return c.JSON(fiber.Map{
				"code":    400,
				"message": "decision must be one of: approved, maintained",
			})
		}
	}

	updated, err := h.appealSvc.Update(c.Context(), idStr, service.UpdateAppealInput{
		Status:     req.Status,
		Decision:   req.Decision,
		Comment:    req.Comment,
		ReviewerID: req.ReviewerID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to update appeal: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    updated,
		"message": "Appeal updated successfully",
	})
}
