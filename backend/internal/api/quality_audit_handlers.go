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

// QualityAuditHandler handles HTTP requests for quality audit operations.
type QualityAuditHandler struct {
	qaservice *service.QualityAuditService
}

// NewQualityAuditHandler creates a new QualityAuditHandler.
func NewQualityAuditHandler(qaservice *service.QualityAuditService) *QualityAuditHandler {
	return &QualityAuditHandler{qaservice: qaservice}
}

// CreateBatchRequest is the DTO for creating a quality audit batch.
type CreateBatchRequest struct {
	Name         string `json:"name" validate:"required"`
	Mode         string `json:"mode" validate:"required"`
	FilterStatus string `json:"filter_status" validate:"required"`
	SampleSize   int    `json:"sample_size" validate:"required,min=1"`
}

// SubmitQARequest is the DTO for submitting a single QA review record.
type SubmitQARequest struct {
	ElementID string                  `json:"element_id" validate:"required"`
	QAScore   int                     `json:"qa_score" validate:"required,min=0,max=100"`
	QALevel   string                  `json:"qa_level" validate:"required"`
	Disagree  bool                    `json:"disagree"`
	Comment   *string                 `json:"comment"`
}

// ListBatchesQuery is the DTO for listing quality audit batches.
type ListBatchesQuery struct {
	TenantID *string `form:"tenant_id"`
	Status   *string `form:"status"`
	DateFrom *string `form:"date_from"`
	DateTo   *string `form:"date_to"`
	Page     int     `form:"page,default=1"`
	PageSize int     `form:"page_size,default=20"`
}

// CreateBatch handles POST /api/v1/quality/batches.
func (h *QualityAuditHandler) CreateBatch(c *fiber.Ctx) error {
	var req CreateBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid request body: " + err.Error(),
		})
	}

	if strings.TrimSpace(req.Name) == "" {
		return c.JSON(fiber.Map{"code": 400, "message": "name is required"})
	}
	if req.SampleSize < 1 {
		return c.JSON(fiber.Map{"code": 400, "message": "sample_size must be at least 1"})
	}

	// Validate mode enum.
	mode := model.QualityAuditMode(strings.ToLower(req.Mode))
	switch mode {
	case model.ModeLocalCorrection, model.ModeFullCorrection:
	default:
		return c.JSON(fiber.Map{"code": 400, "message": "mode must be one of: local_correction, full_correction"})
	}

	// Validate filter_status enum.
	if strings.TrimSpace(req.FilterStatus) == "" {
		return c.JSON(fiber.Map{"code": 400, "message": "filter_status is required"})
	}

	tenantID, terr := getTenantID(c)
	if terr != nil {
		return c.JSON(fiber.Map{"code": 500, "message": "tenant context missing"})
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid tenant_id"})
	}

	userID := c.Locals("user_id")
	var createdBy uuid.UUID
	if userID != nil {
		if uid, ok := userID.(string); ok && uid != "" {
			createdBy, _ = uuid.Parse(uid)
		}
	}

	batch, err := h.qaservice.CreateBatch(c.Context(), tenantUUID, createdBy, model.CreateQualityBatchRequest{
		Name:         req.Name,
		Mode:         mode,
		FilterStatus: req.FilterStatus,
		SampleSize:   req.SampleSize,
	})
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to create batch: " + err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data":    batch,
		"message": "Quality audit batch created",
	})
}

// ListBatches handles GET /api/v1/quality/batches.
func (h *QualityAuditHandler) ListBatches(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := model.QualityAuditQuery{
		Page:     page,
		PageSize: pageSize,
	}
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		query.TenantID = &tenantID
	}
	if status := c.Query("status"); status != "" {
		query.Status = &status
	}

	batches, total, err := h.qaservice.ListBatches(c.Context(), query)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to list batches: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":      batches,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetBatch handles GET /api/v1/quality/batches/:id.
func (h *QualityAuditHandler) GetBatch(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid batch id"})
	}

	batch, err := h.qaservice.GetBatch(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrBatchNotFound) {
			return c.JSON(fiber.Map{"code": 404, "message": "batch not found"})
		}
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to get batch: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{"data": batch})
}

// StartBatch handles POST /api/v1/quality/batches/:id/start.
func (h *QualityAuditHandler) StartBatch(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid batch id"})
	}

	if err := h.qaservice.StartBatch(c.Context(), id); err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to start batch: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    fiber.Map{"status": "in_progress"},
		"message": "Batch started",
	})
}

// CompleteBatch handles POST /api/v1/quality/batches/:id/complete.
func (h *QualityAuditHandler) CompleteBatch(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid batch id"})
	}

	if err := h.qaservice.CompleteBatch(c.Context(), id); err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to complete batch: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    fiber.Map{"status": "completed"},
		"message": "Batch completed",
	})
}

// SubmitQARecord handles POST /api/v1/quality/batches/:id/records.
func (h *QualityAuditHandler) SubmitQARecord(c *fiber.Ctx) error {
	batchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid batch id"})
	}

	var req SubmitQARequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid request body: " + err.Error(),
		})
	}

	if req.ElementID == "" {
		return c.JSON(fiber.Map{"code": 400, "message": "element_id is required"})
	}
	elementUUID, err := uuid.Parse(req.ElementID)
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid element_id"})
	}
	if req.QAScore < 0 || req.QAScore > 100 {
		return c.JSON(fiber.Map{"code": 400, "message": "qa_score must be between 0 and 100"})
	}

	// Validate qa_level enum.
	qaLevel := model.QualityAuditLevel(strings.ToLower(req.QALevel))
	switch qaLevel {
	case model.QualityLevelPass, model.QualityLevelMinor, model.QualityLevelMajor, model.QualityLevelCritical:
	default:
		return c.JSON(fiber.Map{"code": 400, "message": "invalid qa_level"})
	}

	createdBy := uuid.Nil
	userID := c.Locals("user_id")
	if userID != nil {
		createdBy, _ = uuid.Parse(userID.(string))
	}

	record, err := h.qaservice.SubmitQARecord(c.Context(), batchID, elementUUID, createdBy, model.SubmitQualityRecordRequest{
		ElementID: elementUUID.String(),
		QAScore:   req.QAScore,
		QALevel:   qaLevel,
		Disagree:  req.Disagree,
		Comment:   req.Comment,
	})
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to submit QA record: " + err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data":    record,
		"message": "QA record submitted",
	})
}

// GetBatchStats handles GET /api/v1/quality/batches/:id/stats.
func (h *QualityAuditHandler) GetBatchStats(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid batch id"})
	}

	stats, err := h.qaservice.GetBatchStats(c.Context(), id)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to get stats: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{"data": stats})
}

// GetQARecords handles GET /api/v1/quality/batches/:id/records.
func (h *QualityAuditHandler) GetQARecords(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.JSON(fiber.Map{"code": 400, "message": "invalid batch id"})
	}

	records, err := h.qaservice.GetQARecords(c.Context(), id)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "Failed to get records: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{"data": records})
}
