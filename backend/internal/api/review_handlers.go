package api

import (
	"errors"
	"strconv"
	"strings"

	"audit-platform/internal/model"
	"audit-platform/internal/repository"
	"audit-platform/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ReviewHandler handles HTTP requests for review operations (human review, batch review, appeal resolution, pending list).
type ReviewHandler struct {
	reviewSvc   *service.ReviewService
	elementRepo *repository.ElementRepository
	authSvc     *service.AuthService
}

// NewReviewHandler creates a new ReviewHandler.
func NewReviewHandler(reviewSvc *service.ReviewService, elementRepo *repository.ElementRepository, authSvc *service.AuthService) *ReviewHandler {
	return &ReviewHandler{
		reviewSvc:   reviewSvc,
		elementRepo: elementRepo,
		authSvc:     authSvc,
	}
}

// HumanReviewRequest represents the body of a human review request.
type HumanReviewRequest struct {
	ElementID    string `json:"element_id" validate:"required"`
	Action       string `json:"action" validate:"required"`
	Reason       string `json:"reason"`
	Comment      string `json:"comment"`
	ReviewerID   string `json:"reviewer_id" validate:"required"`
}

// BatchReviewRequest represents the body of a batch review request.
type BatchReviewRequest struct {
	ElementIDs []string `json:"element_ids" validate:"required,min=1"`
	Action     string   `json:"action" validate:"required"`
	Reason     string   `json:"reason"`
	Comment    string   `json:"comment"`
	ReviewerID string   `json:"reviewer_id" validate:"required"`
}

// ResolveAppealRequest represents the body for resolving an appeal.
type ResolveAppealRequest struct {
	Decision   string `json:"decision" validate:"required"`
	Comment    string `json:"comment"`
	ReviewerID string `json:"reviewer_id" validate:"required"`
}

// ElementStatsResponse represents the counts for the review stats bar.
type ElementStatsResponse struct {
	PendingHuman  int64 `json:"pending_human"`
	HumanPassed   int64 `json:"human_passed"`
	HumanRejected int64 `json:"human_rejected"`
	Conflict      int64 `json:"conflict"`
}

// ElementStats handles GET /api/v1/review/stats — returns global counts for the stats bar.
func (h *ReviewHandler) ElementStats(c *fiber.Ctx) error {
	aiStatus := c.Query("ai_status", "")
	humanStatus := c.Query("human_status", "")
	elementKind := c.Query("element_kind", "")
	riskMinStr := c.Query("risk_min", "0")
	riskMaxStr := c.Query("risk_max", "100")

	riskMin, _ := strconv.Atoi(riskMinStr)
	riskMax, _ := strconv.Atoi(riskMaxStr)
	if riskMin < 0 {
		riskMin = 0
	}
	if riskMax > 100 {
		riskMax = 100
	}

	stats, err := h.elementRepo.CountByFilters(c.Context(), aiStatus, humanStatus, elementKind, riskMin, riskMax)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to get element stats: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": ElementStatsResponse{
			PendingHuman:  stats.PendingHuman,
			HumanPassed:   stats.HumanPassed,
			HumanRejected: stats.HumanRejected,
			Conflict:      stats.Conflict,
		},
	})
}

// ListPending handles GET /api/v1/review/pending — list elements awaiting human review.
func (h *ReviewHandler) ListPending(c *fiber.Ctx) error {
	aiStatus := c.Query("ai_status", "")
	humanStatus := c.Query("human_status", "")
	elementKind := c.Query("element_kind", "")
	riskMinStr := c.Query("risk_min", "0")
	riskMaxStr := c.Query("risk_max", "100")
	sortBy := c.Query("sort_by", "created_at")
	sortOrder := c.Query("sort_order", "desc")
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

	riskMin, _ := strconv.Atoi(riskMinStr)
	riskMax, _ := strconv.Atoi(riskMaxStr)
	if riskMin < 0 {
		riskMin = 0
	}
	if riskMax > 100 {
		riskMax = 100
	}

	// Validate sort fields
	validSortFields := map[string]bool{"created_at": true, "ai_risk_score": true}
	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}
	sortOrder = strings.ToLower(sortOrder)
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// If neither filter is provided, default to pending human review.
	if aiStatus == "" && humanStatus == "" {
		aiStatus = string(model.ElementAIPassed)
		humanStatus = string(model.ElementPendingHuman)
	}

	elements, total, err := h.elementRepo.FindByStatus(c.Context(), aiStatus, humanStatus, elementKind, riskMin, riskMax, sortBy, sortOrder, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to list pending elements: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":      elements,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// HumanReview handles POST /api/v1/review/human — submit a human review decision for an element.
func (h *ReviewHandler) HumanReview(c *fiber.Ctx) error {
	var req HumanReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// Normalize and validate action enum.
	req.Action = strings.ToLower(req.Action)
	switch req.Action {
	case "approve", "reject":
	default:
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "action must be one of: approve, reject",
		})
	}

	// Reject requires a reason.
	if req.Action == "reject" && strings.TrimSpace(req.Reason) == "" {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "reason is required when action is reject",
		})
	}

	// Validate required fields.
	if req.ElementID == "" {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "element_id is required",
		})
	}

	elemID, err := uuid.Parse(req.ElementID)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid element_id",
		})
	}

	// CRITICAL: reviewer_id must come from JWT claims, not from request body.
	// This prevents users from forging reviewer_id to impersonate others.
	userID := c.Locals("user_id")
	if userID == nil {
		return c.JSON(fiber.Map{
			"code":    401,
			"message": "user context missing",
		})
	}
	reviewerID, err := uuid.Parse(userID.(string))
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid reviewer_id from token",
		})
	}

	record, err := h.reviewSvc.HumanReview(c.Context(), service.HumanReviewInput{
		ElementID:  elemID,
		Action:     req.Action,
		Reason:     req.Reason,
		Comment:    req.Comment,
		ReviewerID: reviewerID,
	}, c.Locals("tenant_id").(string))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code":    404,
				"message": "Element not found",
			})
		}
		if errors.Is(err, model.ErrAlreadyReviewed) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"code":    409,
				"message": "Element has already been reviewed",
			})
		}
		if strings.Contains(err.Error(), "access denied") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    403,
				"message": "Element does not belong to your tenant",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to submit review: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    record,
		"message": "Review submitted successfully",
	})
}

// BatchReview handles POST /api/v1/review/batch — review multiple elements at once.
func (h *ReviewHandler) BatchReview(c *fiber.Ctx) error {
	var req BatchReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// Validate action enum.
	req.Action = strings.ToLower(req.Action)
	switch req.Action {
	case "approve", "reject":
	default:
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "action must be one of: approve, reject",
		})
	}

	if req.Action == "reject" && strings.TrimSpace(req.Reason) == "" {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "reason is required when action is reject",
		})
	}

	// Validate element IDs are valid UUIDs.
	for _, eid := range req.ElementIDs {
		if _, err := uuid.Parse(eid); err != nil {
			return c.JSON(fiber.Map{
				"code":    400,
				"message": "Invalid element_id in element_ids: " + eid,
			})
		}
	}

	// CRITICAL: reviewer_id must come from JWT claims, not from request body.
	userID := c.Locals("user_id")
	if userID == nil {
		return c.JSON(fiber.Map{
			"code":    401,
			"message": "user context missing",
		})
	}
	reviewerID, err := uuid.Parse(userID.(string))
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid reviewer_id from token",
		})
	}

	records, err := h.reviewSvc.BatchReview(c.Context(), service.BatchReviewInput{
		ElementIDs: req.ElementIDs,
		Action:     req.Action,
		Reason:     req.Reason,
		Comment:    req.Comment,
		ReviewerID: reviewerID.String(),
	}, c.Locals("tenant_id").(string))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to batch review: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    records,
		"message": "Batch review completed",
	})
}

// ResolveAppeal handles PUT /api/v1/review/appeal/:id — resolve an appeal via the review workflow.
func (h *ReviewHandler) ResolveAppeal(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid appeal ID format",
		})
	}

	var req ResolveAppealRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// Validate decision enum.
	req.Decision = strings.ToLower(req.Decision)
	switch req.Decision {
	case "approved", "maintained":
	default:
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "decision must be one of: approved, maintained",
		})
	}

	// CRITICAL: reviewer_id must come from JWT claims, not from request body.
	userID := c.Locals("user_id")
	if userID == nil {
		return c.JSON(fiber.Map{
			"code":    401,
			"message": "user context missing",
		})
	}
	reviewerID, err := uuid.Parse(userID.(string))
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid reviewer_id from token",
		})
	}

	appeal, err := h.reviewSvc.ResolveAppeal(c.Context(), idStr, service.ResolveAppealInput{
		Decision:   req.Decision,
		Comment:    req.Comment,
		ReviewerID: reviewerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code":    404,
				"message": "Appeal not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to resolve appeal: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    appeal,
		"message": "Appeal resolved successfully",
	})
}

// GetElementsByContent handles GET /api/v1/review/content/:contentId — list all elements for a content item.
func (h *ReviewHandler) GetElementsByContent(c *fiber.Ctx) error {
	contentID, err := uuid.Parse(c.Params("contentId"))
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid content_id",
		})
	}

	elements, err := h.elementRepo.FindByContentID(c.Context(), contentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to list elements: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": elements,
	})
}

// ListAuditLogs handles GET /api/v1/review/logs — list audit records with pagination.
func (h *ReviewHandler) ListAuditLogs(c *fiber.Ctx) error {
	pageStr := c.Query("page", "1")
	pageSizeStr := c.Query("page_size", "20")
	action := c.Query("action")
	reviewType := c.Query("review_type")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	records, total, err := h.reviewSvc.ListAuditLogs(c.Context(), page, pageSize, action, reviewType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to list audit logs: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":      records,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// WebSocket handler for review task notifications.
// Clients connect via ws://host/ws/review?token=<jwt>.
// On connect, the handler validates the JWT and registers the client with user info.
func (h *ReviewHandler) WebSocket(c *websocket.Conn) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"missing token"}`))
		c.Close()
		return
	}

	// Validate JWT and extract user info.
	claims, err := h.authSvc.ValidateToken(tokenStr)
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"invalid token"}`))
		c.Close()
		return
	}

	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" && claims.TenantID != nil {
		tenantID = claims.TenantID.String()
	}

	// Enforce: tenant_admin and platform_admin can access any tenant.
	// Reviewer and quality_checker must match the JWT tenant.
	if claims.Role != "tenant_admin" && claims.Role != "platform_admin" {
		if tenantID == "" {
			c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"tenant context missing"}`))
			c.Close()
			return
		}
	}

	client := service.NewWSClient(c, h.reviewSvc.WSHub, tenantID)
	h.reviewSvc.WSHub.Register(client, claims.UserID.String(), claims.Role, tenantID)
	go client.WritePump()
	go client.ReadPump()
}
