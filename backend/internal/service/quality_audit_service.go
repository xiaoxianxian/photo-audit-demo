package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"audit-platform/internal/logger"
	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var qualityLog = logger.New("quality_audit_service")

// ErrBatchNotFound is returned when a quality audit batch doesn't exist.
var ErrBatchNotFound = errors.New("quality audit batch not found")

// QualityAuditService manages quality audit batches and individual QA reviews.
type QualityAuditService struct {
	batchRepo    *repository.QualityAuditRepository
	elementRepo  *repository.ElementRepository
	auditLogRepo *repository.LogRepository
}

// NewQualityAuditService creates a new QualityAuditService.
func NewQualityAuditService(batchRepo *repository.QualityAuditRepository, elementRepo *repository.ElementRepository, auditLogRepo *repository.LogRepository) *QualityAuditService {
	return &QualityAuditService{
		batchRepo:    batchRepo,
		elementRepo:  elementRepo,
		auditLogRepo: auditLogRepo,
	}
}

// CreateBatch creates a new quality audit sampling batch.
func (s *QualityAuditService) CreateBatch(ctx context.Context, tenantID uuid.UUID, createdBy uuid.UUID, req model.CreateQualityBatchRequest) (*model.QualityAuditBatch, error) {
	if req.SampleSize < 1 || req.SampleSize > 1000 {
		return nil, fmt.Errorf("sample_size must be between 1 and 1000")
	}
	if req.Mode != model.ModeLocalCorrection && req.Mode != model.ModeFullCorrection {
		return nil, fmt.Errorf("invalid mode: %s", req.Mode)
	}

	batch := &model.QualityAuditBatch{
		ID:           uuid.New(),
		TenantID:     tenantID,
		CreatedBy:    createdBy,
		Name:         req.Name,
		Mode:         req.Mode,
		FilterStatus: req.FilterStatus,
		SampleSize:   req.SampleSize,
		Status:       "draft",
	}

	if err := s.batchRepo.CreateBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("create quality audit batch: %w", err)
	}

	return batch, nil
}

// GetBatch returns a quality audit batch by ID.
func (s *QualityAuditService) GetBatch(ctx context.Context, id uuid.UUID) (*model.QualityAuditBatch, error) {
	batch, err := s.batchRepo.GetBatchByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBatchNotFound
		}
		return nil, err
	}
	return batch, nil
}

// ListBatches returns a paginated list of quality audit batches.
func (s *QualityAuditService) ListBatches(ctx context.Context, q model.QualityAuditQuery) ([]model.QualityAuditBatch, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	tenantID := ""
	if q.TenantID != nil {
		tenantID = *q.TenantID
	}
	status := ""
	if q.Status != nil {
		status = *q.Status
	}

	return s.batchRepo.ListBatches(ctx, tenantID, status, q.DateFrom, q.DateTo, q.Page, q.PageSize)
}

// StartBatch transitions a batch from "draft" to "in_progress".
func (s *QualityAuditService) StartBatch(ctx context.Context, id uuid.UUID) error {
	batch, err := s.batchRepo.GetBatchByID(ctx, id)
	if err != nil {
		return err
	}
	if batch.Status != "draft" {
		return fmt.Errorf("batch is not in draft status: %s", batch.Status)
	}
	return s.batchRepo.UpdateBatchStatus(ctx, id, "in_progress", batch.ReviewedCount, nil)
}

// CompleteBatch marks a batch as completed.
func (s *QualityAuditService) CompleteBatch(ctx context.Context, id uuid.UUID) error {
	batch, err := s.batchRepo.GetBatchByID(ctx, id)
	if err != nil {
		return err
	}
	if batch.Status != "in_progress" {
		return fmt.Errorf("batch is not in progress: %s", batch.Status)
	}
	now := time.Now()
	return s.batchRepo.UpdateBatchStatus(ctx, id, "completed", batch.ReviewedCount, &now)
}

// SubmitQARecord records a single QA review within a batch.
func (s *QualityAuditService) SubmitQARecord(ctx context.Context, batchID uuid.UUID, elementID uuid.UUID, createdBy uuid.UUID, req model.SubmitQualityRecordRequest) (*model.QualityAuditRecord, error) {
	batch, err := s.batchRepo.GetBatchByID(ctx, batchID)
	if err != nil {
		return nil, err
	}
	if batch.Status != "in_progress" && batch.Status != "draft" {
		return nil, fmt.Errorf("batch is not active (status: %s)", batch.Status)
	}

	// Use direct FindByID instead of loading all elements.
	elem, err := s.elementRepo.FindByID(ctx, elementID)
	if err != nil {
		return nil, fmt.Errorf("element not found: %w", err)
	}

	if req.QAScore < 0 || req.QAScore > 100 {
		return nil, fmt.Errorf("qa_score must be between 0 and 100")
	}

	switch req.QALevel {
	case model.QualityLevelPass, model.QualityLevelMinor, model.QualityLevelMajor, model.QualityLevelCritical:
	default:
		return nil, fmt.Errorf("invalid qa_level: %s", req.QALevel)
	}

	disagree := req.Disagree
	if !disagree && req.QALevel != model.QualityLevelPass && elem.HumanStatus == model.ElementHumanPassed {
		disagree = true
	}
	if !disagree && req.QALevel == model.QualityLevelPass && elem.HumanStatus == model.ElementHumanRejected {
		disagree = true
	}

	record := &model.QualityAuditRecord{
		ID:            uuid.New(),
		BatchID:       batchID,
		ElementID:     elementID,
		OriginalScore: elem.AIRiskScore,
		QAScore:       req.QAScore,
		QALevel:       req.QALevel,
		Disagree:      disagree,
		Comment:       req.Comment,
		CreatedBy:     createdBy,
	}

	if err := s.batchRepo.CreateQARecord(ctx, record); err != nil {
		return nil, fmt.Errorf("create qa record: %w", err)
	}

	newCount := batch.ReviewedCount + 1
	if err := s.batchRepo.UpdateBatchReviewedCount(ctx, batchID, newCount); err != nil {
		return nil, fmt.Errorf("update batch reviewed count: %w", err)
	}

	if disagree && batch.Mode == model.ModeFullCorrection {
		if err := s.applyCorrection(ctx, elementID, req.QALevel); err != nil {
			qualityLog.Warn("failed to apply correction for element %s: %v", elementID, err)
		}
	}

	return record, nil
}

// GetBatchStats returns aggregated statistics for a quality audit batch.
func (s *QualityAuditService) GetBatchStats(ctx context.Context, batchID uuid.UUID) (*model.QualityAuditStats, error) {
	return s.batchRepo.BatchStats(ctx, batchID)
}

// GetQARecords returns all QA records for a batch.
func (s *QualityAuditService) GetQARecords(ctx context.Context, batchID uuid.UUID) ([]model.QualityAuditRecord, error) {
	return s.batchRepo.GetQARecordsByBatch(ctx, batchID)
}

// applyCorrection updates the element status based on QA disagreement.
func (s *QualityAuditService) applyCorrection(ctx context.Context, elementID uuid.UUID, qaLevel model.QualityAuditLevel) error {
	elem, err := s.elementRepo.FindByID(ctx, elementID)
	if err != nil {
		return err
	}

	if qaLevel == model.QualityLevelMajor || qaLevel == model.QualityLevelCritical {
		if elem.HumanStatus == model.ElementHumanPassed {
			return s.elementRepo.UpdateStatus(ctx, elementID, string(elem.AIStatus), string(model.ElementPendingHuman))
		}
	}

	if qaLevel == model.QualityLevelPass && elem.HumanStatus == model.ElementHumanRejected {
		return s.elementRepo.UpdateStatus(ctx, elementID, string(elem.AIStatus), string(model.ElementHumanPassed))
	}

	return nil
}
