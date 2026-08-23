package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"audit-platform/internal/logger"
	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
)

var reviewLog = logger.New("review_service")

// HumanReviewInput is the DTO for human review submissions.
type HumanReviewInput struct {
	ElementID  uuid.UUID `json:"element_id"`
	Action     string    `json:"action"` // "approve" or "reject"
	Reason     string    `json:"reason"`
	Comment    string    `json:"comment"`
	ReviewerID uuid.UUID `json:"reviewer_id"`
}

// BatchReviewInput is the DTO for batch review submissions.
type BatchReviewInput struct {
	ElementIDs []string `json:"element_ids"`
	Action     string   `json:"action"`
	Reason     string   `json:"reason"`
	Comment    string   `json:"comment"`
	ReviewerID string   `json:"reviewer_id"`
}

// ResolveAppealInput is the DTO for resolving an appeal.
type ResolveAppealInput struct {
	Decision   string    `json:"decision"` // "approved" or "maintained"
	Comment    string    `json:"comment"`
	ReviewerID uuid.UUID `json:"reviewer_id"`
}

// ReviewService handles human review logic (single, batch, appeal resolution).
type ReviewService struct {
	elementRepo   *repository.ElementRepository
	appealRepo    *repository.AppealRepository
	AuditLogRepo  *repository.LogRepository
	ingestionSvc  *IngestionService
	notifier      Notifier
	WSHub         *Hub
	contentRepo   *repository.ContentRepository
}

// NewReviewService creates a new ReviewService.
func NewReviewService(elementRepo *repository.ElementRepository, appealRepo *repository.AppealRepository, auditLogRepo *repository.LogRepository, ingestionSvc *IngestionService, notifier Notifier, wsHub *Hub, contentRepo *repository.ContentRepository) *ReviewService {
	return &ReviewService{
		elementRepo:  elementRepo,
		appealRepo:   appealRepo,
		AuditLogRepo: auditLogRepo,
		ingestionSvc: ingestionSvc,
		notifier:     notifier,
		WSHub:        wsHub,
		contentRepo:  contentRepo,
	}
}

// HumanReview processes a single human review decision on an element.
// Validates that the element belongs to the requesting tenant.
func (s *ReviewService) HumanReview(ctx context.Context, input HumanReviewInput, tenantID string) (*model.AuditRecord, error) {
	action := model.ReviewAction(strings.ToLower(input.Action))
	switch action {
	case model.ActionApprove, model.ActionReject:
	default:
		return nil, fmt.Errorf("invalid review action: %s", input.Action)
	}

	elem, err := s.elementRepo.FindByID(ctx, input.ElementID)
	if err != nil {
		return nil, fmt.Errorf("review human: %w", err)
	}

	// Validate tenant isolation: element's content must belong to the same tenant.
	content, err := s.contentRepo.FindByID(ctx, elem.ContentID)
	if err != nil {
		return nil, fmt.Errorf("review human: content not found: %w", err)
	}
	if content.TenantID.String() != tenantID {
		return nil, fmt.Errorf("review human: access denied (tenant mismatch)")
	}

	if elem.HumanStatus == model.ElementHumanPassed || elem.HumanStatus == model.ElementHumanRejected {
		return nil, model.ErrAlreadyReviewed
	}

	scoreAfter := elem.AIRiskScore
	if action == model.ActionApprove {
		scoreAfter = 0
	}

	record := &model.AuditRecord{
		ID:            uuid.New(),
		ElementID:     elem.ID,
		ReviewerID:    &input.ReviewerID,
		ReviewType:    model.ReviewTypeHuman,
		Action:        action,
		CreatedAt:     time.Now(),
		AIScoreBefore: &elem.AIRiskScore,
		AIScoreAfter:  &scoreAfter,
		IsConflict:    elem.IsConflict,
	}

	if input.Reason != "" {
		reason := model.RejectReason(strings.ToLower(input.Reason))
		record.Reason = &reason
	}
	if input.Comment != "" {
		record.Comment = &input.Comment
	}

	if err := s.AuditLogRepo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("review human create log: %w", err)
	}

	humanStatus := model.ElementHumanPassed
	if action == model.ActionReject {
		humanStatus = model.ElementHumanRejected
	}
	if err := s.elementRepo.UpdateStatus(ctx, elem.ID, string(elem.AIStatus), string(humanStatus)); err != nil {
		return nil, fmt.Errorf("review human update element: %w", err)
	}

	// Trigger content-level decision after human review.
	if s.ingestionSvc != nil {
		go func(contentID uuid.UUID) {
			if err := s.ingestionSvc.TriggerContentDecision(ctx, contentID); err != nil {
				reviewLog.Warn("content decision: %v", err)
			}
		}(elem.ContentID)
	}

	return record, nil
}

// ListAuditLogs returns paginated audit records filtered by action and review type.
func (s *ReviewService) ListAuditLogs(ctx context.Context, page, pageSize int, action, reviewType string) ([]model.AuditRecord, int64, error) {
	return s.AuditLogRepo.ListAllFiltered(ctx, page, pageSize, action, reviewType)
}
func (s *ReviewService) BatchReview(ctx context.Context, input BatchReviewInput, tenantID string) ([]model.AuditRecord, error) {
	action := model.ReviewAction(strings.ToLower(input.Action))
	switch action {
	case model.ActionApprove, model.ActionReject:
	default:
		return nil, fmt.Errorf("invalid batch review action: %s", input.Action)
	}

	reviewerID, err := uuid.Parse(input.ReviewerID)
	if err != nil {
		return nil, fmt.Errorf("batch review: invalid reviewer_id: %w", err)
	}

	records := make([]model.AuditRecord, 0, len(input.ElementIDs))
	for _, eid := range input.ElementIDs {
		elemUUID, err := uuid.Parse(eid)
		if err != nil {
			return nil, fmt.Errorf("batch review invalid element id: %s", eid)
		}

		record, err := s.HumanReview(ctx, HumanReviewInput{
			ElementID:  elemUUID,
			Action:     input.Action,
			Reason:     input.Reason,
			Comment:    input.Comment,
			ReviewerID: reviewerID,
		}, tenantID)
		if err != nil {
			return nil, fmt.Errorf("batch review element %s: %w", eid, err)
		}
		records = append(records, *record)
	}

	return records, nil
}

// ResolveAppeal resolves an appeal via the review workflow.
// decision: "approved" = 改判解封, "maintained" = 维持原判.
// All operations (audit_record, element status rollback, appeal update) are wrapped
// in a single DB transaction to prevent partial failures.
// reviewerID is derived from JWT context, not from request body.
func (s *ReviewService) ResolveAppeal(ctx context.Context, appealID string, input ResolveAppealInput) (*model.Appeal, error) {
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

	decision := strings.ToLower(input.Decision)
	var action model.ReviewAction
	var newStatus model.AppealStatus
	switch decision {
	case "approved":
		action = model.ActionReverse
		newStatus = model.AppealResolvedApproved
	case "maintained":
		action = model.ActionMaintain
		newStatus = model.AppealResolvedMaintained
	default:
		return nil, fmt.Errorf("resolve appeal: invalid decision: %s", decision)
	}

	// Use reviewerID from input (derived from JWT), not from request body.
	reviewerID := input.ReviewerID
	if reviewerID == uuid.Nil {
		// Fallback: derive from JWT context (should not happen if handler is correct).
		if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
			reviewerID, _ = uuid.Parse(userID)
		}
	}

	// Begin transaction: audit_record + element rollback + appeal update.
	tx, err := s.elementRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve appeal begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Create audit record within transaction.
	record := &model.AuditRecord{
		ID:            uuid.New(),
		ElementID:     appeal.ContentID,
		ReviewerID:    &reviewerID,
		ReviewType:    model.ReviewTypeAppeal,
		Action:        action,
		CreatedAt:     time.Now(),
	}
	if input.Comment != "" {
		record.Comment = stringPtr(input.Comment)
	}
	if err := s.AuditLogRepo.CreateWithTx(ctx, tx, record); err != nil {
		return nil, fmt.Errorf("resolve appeal create log: %w", err)
	}

	// 2. If approved (改判), revert rejected elements back to pending_human.
	if decision == "approved" {
		elements, err := s.elementRepo.FindByContentID(ctx, appeal.ContentID)
		if err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("resolve appeal find elements: %w", err)
		}
		for _, elem := range elements {
			if elem.HumanStatus == model.ElementHumanRejected {
				if err := s.elementRepo.UpdateStatusWithTx(ctx, tx, elem.ID, string(elem.AIStatus), string(model.ElementPendingHuman)); err != nil {
					tx.Rollback(ctx)
					return nil, fmt.Errorf("resolve appeal revert element %s: %w", elem.ID, err)
				}
			}
		}
	}

	// 3. Update appeal status within transaction.
	updateReq := model.UpdateAppealRequest{
		ReviewerID: &reviewerID,
		Resolution: stringPtr(string(newStatus)),
	}
	if input.Comment != "" {
		updateReq.Comment = stringPtr(input.Comment)
	}
	if err := s.appealRepo.UpdateWithTx(ctx, tx, appealUUID, updateReq); err != nil {
		tx.Rollback(ctx)
		if errors.Is(err, repository.ErrAppealAlreadyResolved) {
			return nil, fmt.Errorf("resolve appeal: already resolved (concurrent update detected)")
		}
		return nil, fmt.Errorf("resolve appeal update: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("resolve appeal commit tx: %w", err)
	}

	// Update the returned appeal object to reflect the new state.
	appeal.Status = newStatus
	now := time.Now()
	appeal.ResolvedAt = &now

	// 4. Notify the applicant (outside transaction).
	if s.notifier != nil {
		notifType := NotificationAppealMaintained
		if decision == "approved" {
			notifType = NotificationAppealApproved
		}
		_ = s.notifier.Notify(ctx, NotificationPayload{
			Type:      notifType,
			UserID:    appeal.ApplicantID.String(),
			Title:     mapTitle(decision),
			Message:   mapMessage(decision),
			AppealID:  appeal.ID.String(),
			ContentID: appeal.ContentID.String(),
		})
	}

	return appeal, nil
}

func mapTitle(decision string) string {
	if decision == "approved" {
		return "申诉已改判"
	}
	return "申诉已处理"
}

func mapMessage(decision string) string {
	if decision == "approved" {
		return "您的申诉已通过，相关内容已恢复"
	}
	return "您的申诉已维持原判"
}

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string {
	return &s
}
