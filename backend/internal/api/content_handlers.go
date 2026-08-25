package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"audit-platform/internal/logger"
	"audit-platform/internal/model"
	"audit-platform/internal/repository"
	"audit-platform/internal/service"
	"audit-platform/internal/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var contentLog = logger.New("content_handler")

// ContentHandler handles HTTP requests for content CRUD operations.
type ContentHandler struct {
	ingestionSvc *service.IngestionService
	contentRepo  *repository.ContentRepository
	elementRepo  *repository.ElementRepository
	minioStorage *storage.MinIOStorage
	videoProc    *service.VideoProcessor
}

// NewContentHandler creates a new ContentHandler.
func NewContentHandler(ingestionSvc *service.IngestionService, contentRepo *repository.ContentRepository) *ContentHandler {
	return &ContentHandler{
		ingestionSvc: ingestionSvc,
		contentRepo:  contentRepo,
	}
}

// NewContentHandlerWithStorage creates a ContentHandler with MinIO support.
func NewContentHandlerWithStorage(ingestionSvc *service.IngestionService, contentRepo *repository.ContentRepository, minioStorage *storage.MinIOStorage) *ContentHandler {
	return &ContentHandler{
		ingestionSvc:   ingestionSvc,
		contentRepo:    contentRepo,
		minioStorage:   minioStorage,
	}
}

// WithVideoProcessor attaches a VideoProcessor for video preprocessing.
func (h *ContentHandler) WithVideoProcessor(vp *service.VideoProcessor) {
	h.videoProc = vp
}

// UploadRequest represents the body of a content upload request.
type UploadRequest struct {
	ContentType   string   `json:"content_type" validate:"required"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	ReviewPolicy  string   `json:"review_policy" validate:"required"`
	FileURLs      []string `json:"file_urls" validate:"required"`
	TenantID      string   `json:"tenant_id" validate:"required"`
	CreatorID     string   `json:"creator_id" validate:"required"`
}

// Upload handles POST /api/v1/contents — create a new content item.
func (h *ContentHandler) Upload(c *fiber.Ctx) error {
	var req UploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	// Validate content_type enum.
	req.ContentType = strings.TrimSpace(req.ContentType)
	switch model.ContentType(req.ContentType) {
	case model.ContentTypePhoto, model.ContentTypeShortVideo, model.ContentTypeLiveStream:
	default:
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "content_type must be one of: photo, short_video, live_stream",
		})
	}

	// Validate review_policy enum.
	req.ReviewPolicy = strings.TrimSpace(req.ReviewPolicy)
	switch model.ReviewPolicy(req.ReviewPolicy) {
	case model.ReviewPostThenReview, model.ReviewBeforePost:
	default:
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "review_policy must be one of: post_then_review, review_before_post",
		})
	}

	// Validate required fields.
	if len(req.FileURLs) == 0 {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "file_urls is required",
		})
	}
	if req.TenantID == "" {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "tenant_id is required",
		})
	}
	if req.CreatorID == "" {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "creator_id is required",
		})
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid tenant_id format",
		})
	}
	creatorID, err := uuid.Parse(req.CreatorID)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "invalid creator_id format",
		})
	}

	// Build the upload input and call the ingestion service.
	content, elements, err := h.ingestionSvc.UploadContent(c.Context(), model.UploadInput{
		ContentType:    model.ContentType(req.ContentType),
		Title:          req.Title,
		Description:    req.Description,
		ReviewPolicy:   model.ReviewPolicy(req.ReviewPolicy),
		FileURLs:       req.FileURLs,
		TenantID:       tenantID,
		CreatorID:      creatorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code":    404,
				"message": "Resource not found",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":    500,
				"message": "Failed to upload content: " + err.Error(),
			})
		}
	}

	// Trigger AI review asynchronously for all pending elements.
	// P0-5: c.Context() is a fasthttp RequestCtx that gets pooled and reused
	// after the handler returns — never hand it to a background goroutine.
	// Use an independent context with a timeout instead.
	reviewCtx, reviewCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	go func() {
		defer reviewCancel()
		h.ingestionSvc.TriggerAIReview(reviewCtx, content.ID, req.TenantID)
	}()

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"content":  content,
			"elements": elements,
		},
		"message": "Content uploaded successfully",
	})
}

// List handles GET /api/v1/contents — list contents with pagination.
func (h *ContentHandler) List(c *fiber.Ctx) error {
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

	// Build the model query struct (mirrors the repository signature).
	query := model.ContentListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.Query("sort_by", "created_at"),
		SortOrder: c.Query("sort_order", "desc"),
	}

	// Only set pointers when the query param was actually provided.
	if ct := c.Query("content_type"); ct != "" {
		query.ContentType = &ct
	}
	if st := c.Query("status"); st != "" {
		query.Status = &st
	}
	if tid := c.Query("tenant_id"); tid != "" {
		query.TenantID = &tid
	}

	contents, total, err := h.contentRepo.List(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to list contents: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":      contents,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetByID handles GET /api/v1/contents/:id — retrieve a single content item.
func (h *ContentHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid content ID format",
		})
	}

	content, err := h.contentRepo.FindByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code":    404,
				"message": "Content not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to fetch content: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": content,
	})
}

// UpdateStatus handles PUT /api/v1/contents/:id/status — update content status.
func (h *ContentHandler) UpdateStatus(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid content ID format",
		})
	}

	var req struct {
		Status string `json:"status" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "Invalid request body: " + err.Error(),
		})
	}

	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "status is required",
		})
	}

	err = h.contentRepo.UpdateStatus(c.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code":    404,
				"message": fmt.Sprintf("Content with ID %s not found", idStr),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to update status: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":    fiber.Map{"status": req.Status},
		"message": "Content status updated successfully",
	})
}

// UploadFile handles POST /api/v1/contents/upload/file — multipart file upload.
// If MinIO is configured, the file is stored there and the presigned URL is returned.
// Otherwise, the file is stored in-memory and a mock URL is returned.
//
// For video files (video/* MIME type), the handler also triggers frame extraction
// and ASR transcription via VideoProcessor. The processed elements are inserted
// into the database and AI review is triggered asynchronously.
func (h *ContentHandler) UploadFile(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "file is required",
		})
	}

	// Validate file size (max 100MB).
	maxSize := int64(100 * 1024 * 1024)
	if file.Size > maxSize {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": "file too large (max 100MB)",
		})
	}

	// Open the file.
	src, err := file.Open()
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "failed to open file",
		})
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return c.JSON(fiber.Map{
			"code":    500,
			"message": "failed to read file",
		})
	}

	// Detect MIME type from file content.
	contentType := http.DetectContentType(data)

	// Validate image format.
	allowedImageTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	if strings.HasPrefix(contentType, "image/") && !allowedImageTypes[contentType] {
		return c.JSON(fiber.Map{
			"code":    400,
			"message": fmt.Sprintf("unsupported image format: %s (allowed: jpeg, png, gif, webp)", contentType),
		})
	}

	// Validate video format and check resolution.
	if strings.HasPrefix(contentType, "video/") {
		if !isValidVideoFormat(contentType) {
			return c.JSON(fiber.Map{
				"code":    400,
				"message": fmt.Sprintf("unsupported video format: %s (allowed: mp4, webm, mov)", contentType),
			})
		}
		width, height, err := probeResolution(data)
		if err != nil {
		contentLog.Warn("video resolution probe failed for %s: %v", file.Filename, err)
		} else if width > 0 && height > 0 {
			if width > 3840 || height > 2160 {
				return c.JSON(fiber.Map{
					"code":    400,
					"message": fmt.Sprintf("video resolution too high: %dx%d (max 3840x2160)", width, height),
				})
			}
			if width < 480 || height < 270 {
				return c.JSON(fiber.Map{
					"code":    400,
					"message": fmt.Sprintf("video resolution too low: %dx%d (min 480x270)", width, height),
				})
			}
		}
	}

	// Upload to MinIO if available.
	var fileURL string
	objectName := ""
	if h.minioStorage != nil {
		objectName = storage.GenerateObjectName(file.Filename, "contents")
		fileURL, err = h.minioStorage.UploadBytes(c.Context(), objectName, data, contentType)
		if err != nil {
			return c.JSON(fiber.Map{
				"code":    500,
				"message": "failed to upload to storage: " + err.Error(),
			})
		}
	} else {
		// Fallback: return a mock URL (for development without MinIO).
		fileURL = fmt.Sprintf("file://%s", file.Filename)
	}

	// Check if this is a video file.
	isVideo := strings.HasPrefix(contentType, "video/")

	result := fiber.Map{
		"code":    0,
		"message": "uploaded",
		"data": fiber.Map{
			"url":          fileURL,
			"filename":     file.Filename,
			"size":         file.Size,
			"content_type": contentType,
			"is_video":     isVideo,
		},
	}

	// For video files, trigger async preprocessing (frame extraction + ASR).
	if isVideo {
		tenantIDStr := c.FormValue("tenant_id")
		go h.processVideoAsync(fileURL, data, file.Filename, contentType, tenantIDStr)
		result["data"].(fiber.Map)["preprocessing"] = "started"
	}

	return c.JSON(result)
}

// processVideoAsync handles the async video preprocessing pipeline:
// 1. Create a short_video content record.
// 2. Extract frames and run ASR.
// 3. Insert elements and trigger AI review.
func (h *ContentHandler) processVideoAsync(videoURL string, videoData []byte, filename string, mimeType string, tenantID string) {
	// Parse tenant ID from string to UUID.
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		contentLog.Error("processVideoAsync: invalid tenant_id=%s: %v", tenantID, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create a placeholder content record for the video.
	content := &model.Content{
		ID:           uuid.New(),
		ContentType:  model.ContentTypeShortVideo,
		ReviewPolicy: model.ReviewPostThenReview,
		AIRiskScore:  0,
		Status:       "pending",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		TenantID:     tid,
	}

	// Create the content record (type-specific extension row).
	if err := h.contentRepo.Create(ctx, content); err != nil {
		contentLog.Error("create video content: %v", err)
		return
	}

	// Create cover element from the video URL.
	coverElem := model.ContentElement{
		ID:             uuid.New(),
		ContentID:      content.ID,
		ElementKind:    model.ElementCoverImage,
		ElementContent: videoURL,
		AIRiskScore:    0,
		AIStatus:       model.ElementPendingAI,
		HumanStatus:    model.ElementPendingHuman,
		CreatedAt:      time.Now(),
	}

	// Add title element from filename.
	titleElem := model.ContentElement{
		ID:             uuid.New(),
		ContentID:      content.ID,
		ElementKind:    model.ElementTitle,
		ElementContent: strings.TrimSuffix(filename, filepath.Ext(filename)),
		AIRiskScore:    0,
		AIStatus:       model.ElementPendingAI,
		HumanStatus:    model.ElementPendingHuman,
		CreatedAt:      time.Now(),
	}

	elements := []model.ContentElement{coverElem, titleElem}

	// If video processor is available, extract frames and ASR.
	if h.videoProc != nil {
		videoElements, err := h.videoProc.ProcessVideo(ctx, videoData, filename)
		if err != nil {
			contentLog.Warn("video preprocessing failed for %s: %v", filename, err)
		} else {
			elements = append(elements, videoElements...)
		}
	}

	// Insert elements.
	elemPtrs := make([]*model.ContentElement, len(elements))
	for i := range elements {
		elemPtrs[i] = &elements[i]
	}
	if err := h.elementRepo.CreateBulk(ctx, elemPtrs); err != nil {
		contentLog.Error("bulk insert video elements: %v", err)
		return
	}

	// Trigger AI review.
	if h.ingestionSvc != nil {
		go h.ingestionSvc.TriggerAIReview(ctx, content.ID, tid.String())
	}

	contentLog.Info("video processed: content=%s, elements=%d", content.ID, len(elements))
}

// isValidVideoFormat checks if the MIME type corresponds to an allowed video format.
func isValidVideoFormat(mimeType string) bool {
	allowed := map[string]bool{
		"video/mp4":     true,
		"video/webm":    true,
		"video/quicktime": true, // mov
	}
	return allowed[mimeType]
}

// probeResolution uses ffprobe to extract video width and height.
// Returns (0, 0, nil) if ffprobe is not available or fails.
func probeResolution(data []byte) (int, int, error) {
	cmd := exec.Command("ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		"-i", "pipe:0")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe output: %s", string(out))
	}
	w, _ := strconv.Atoi(parts[0])
	h, _ := strconv.Atoi(parts[1])
	return w, h, nil
}
