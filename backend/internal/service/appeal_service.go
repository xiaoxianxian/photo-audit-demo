package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrAlreadyAppealed is returned when a user tries to appeal the same content twice.
var ErrAlreadyAppealed = fmt.Errorf("you have already submitted an appeal for this content")

// SubmitAppealInput is the DTO for appeal submission requests.
type SubmitAppealInput struct {
	ContentID    uuid.UUID  `json:"content_id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Reason       string     `json:"reason"`
	EvidenceURLs []string   `json:"evidence_urls"`
	ApplicantID  uuid.UUID  `json:"applicant_id"`
}

// UpdateAppealInput is the DTO for updating an existing appeal.
type UpdateAppealInput struct {
	Status     *string    `json:"status"`
	Decision   *string    `json:"decision"`
	Comment    *string    `json:"comment"`
	ReviewerID *string    `json:"reviewer_id"`
}

// AppealService manages appeal submissions and lifecycle.
type AppealService struct {
	appealRepo *repository.AppealRepository
	contentRepo *repository.ContentRepository
	notifier   Notifier
}

// NewAppealService creates a new AppealService.
func NewAppealService(appealRepo *repository.AppealRepository, contentRepo *repository.ContentRepository, notifier Notifier) *AppealService {
	return &AppealService{
		appealRepo:  appealRepo,
		contentRepo: contentRepo,
		notifier:    notifier,
	}
}

// SubmitAppeal creates a new appeal for a content item.
// Enforces the "one appeal per content per user" constraint atomically
// using a database transaction to prevent race conditions.
func (s *AppealService) SubmitAppeal(ctx context.Context, input SubmitAppealInput) (*model.Appeal, error) {
	// Check if the content exists.
	if _, err := s.contentRepo.FindByID(ctx, input.ContentID); err != nil {
		return nil, fmt.Errorf("submit appeal content check: %w", err)
	}

	tx, err := s.appealRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("submit appeal begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Atomic check for existing appeal within the same transaction.
	var existingID uuid.UUID
	checkQ := `SELECT id FROM appeals WHERE content_id = $1 AND applicant_id = $2 LIMIT 1`
	err = tx.QueryRow(ctx, checkQ, input.ContentID, input.ApplicantID).Scan(&existingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("submit appeal check existing: %w", err)
	}
	if err == nil && existingID != uuid.Nil {
		return nil, ErrAlreadyAppealed
	}

	appeal := &model.Appeal{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		ContentID:    input.ContentID,
		ApplicantID:  input.ApplicantID,
		Reason:       strings.TrimSpace(input.Reason),
		EvidenceURLs: input.EvidenceURLs,
		Status:       model.AppealSubmitted,
		SubmittedAt:  time.Now(),
	}

	if err := s.appealRepo.CreateWithTx(ctx, tx, appeal); err != nil {
		return nil, fmt.Errorf("submit appeal insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("submit appeal commit tx: %w", err)
	}

	// Notify: alert the submitter that their appeal was submitted.
	if s.notifier != nil {
		_ = s.notifier.Notify(ctx, NotificationPayload{
			Type:      NotificationAppealSubmitted,
			UserID:    appeal.ApplicantID.String(),
			Title:     "申诉已提交",
			Message:   "您的申诉已成功提交，等待审核处理",
			AppealID:  appeal.ID.String(),
			ContentID: appeal.ContentID.String(),
		})
	}

	return appeal, nil
}

// GetByID returns a single appeal by ID.
func (s *AppealService) GetByID(ctx context.Context, id string) (*model.Appeal, error) {
	uuidVal, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("get appeal by id: invalid UUID: %w", err)
	}

	appeal, err := s.appealRepo.FindByID(ctx, uuidVal)
	if err != nil {
		return nil, fmt.Errorf("get appeal by id: %w", err)
	}

	return appeal, nil
}

// ListByStatus returns paginated appeals filtered by status.
func (s *AppealService) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]model.Appeal, int64, error) {
	if status == "" {
		status = string(model.AppealSubmitted)
	}
	return s.appealRepo.ListByStatus(ctx, status, page, pageSize)
}

// ListByTenantAndStatus returns appeals for a specific tenant filtered by status, paginated.
func (s *AppealService) ListByTenantAndStatus(ctx context.Context, tenantID, status string, page, pageSize int) ([]model.Appeal, int64, error) {
	return s.appealRepo.ListByTenantAndStatus(ctx, tenantID, status, page, pageSize)
}

// Update modifies an existing appeal with the given fields.
func (s *AppealService) Update(ctx context.Context, id string, input UpdateAppealInput) (*model.Appeal, error) {
	uuidVal, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("update appeal: invalid ID")
	}

	updateReq := model.UpdateAppealRequest{}
	if input.ReviewerID != nil {
		rid, perr := uuid.Parse(*input.ReviewerID)
		if perr == nil {
			updateReq.ReviewerID = &rid
		}
	}
	if input.Decision != nil {
		updateReq.Resolution = input.Decision
	}
	if input.Comment != nil {
		updateReq.Comment = input.Comment
	}

	return s.appealRepo.Update(ctx, uuidVal, updateReq)
}

// ResolveAppeal resolves an appeal and notifies the applicant.
func (s *AppealService) ResolveAppeal(ctx context.Context, appealID string, decision string, comment string, reviewerID uuid.UUID) (*model.Appeal, error) {
	appealUUID, err := uuid.Parse(appealID)
	if err != nil {
		return nil, fmt.Errorf("resolve appeal: invalid ID: %w", err)
	}

	appeal, err := s.appealRepo.FindByID(ctx, appealUUID)
	if err != nil {
		return nil, fmt.Errorf("resolve appeal: %w", err)
	}

	if appeal.Status == model.AppealResolvedApproved || appeal.Status == model.AppealResolvedMaintained {
		return nil, fmt.Errorf("resolve appeal: already resolved")
	}

	var notifType NotificationType
	var title, message string
	switch decision {
	case "approved":
		notifType = NotificationAppealApproved
		title = "申诉已改判"
		message = "您的申诉已通过，相关内容已恢复"
	case "maintained":
		notifType = NotificationAppealMaintained
		title = "申诉已处理"
		message = "您的申诉已维持原判"
	default:
		return nil, fmt.Errorf("resolve appeal: invalid decision: %s", decision)
	}

	updateReq := model.UpdateAppealRequest{
		ReviewerID: &reviewerID,
		Resolution: stringPtr(decision),
		Comment:    stringPtr(comment),
	}

	resolved, err := s.appealRepo.Update(ctx, appealUUID, updateReq)
	if err != nil {
		return nil, fmt.Errorf("resolve appeal update: %w", err)
	}

	resolved.Status = model.AppealStatus(decision)
	now := time.Now()
	resolved.ResolvedAt = &now

	// Notify the applicant.
	if s.notifier != nil {
		_ = s.notifier.Notify(ctx, NotificationPayload{
			Type:      notifType,
			UserID:    appeal.ApplicantID.String(),
			Title:     title,
			Message:   message,
			AppealID:  appeal.ID.String(),
			ContentID: appeal.ContentID.String(),
		})
	}

	return resolved, nil
}
