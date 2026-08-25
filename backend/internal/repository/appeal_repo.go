package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"audit-platform/internal/model"
)

// txConn is a minimal interface for transaction-aware operations.
type txConn interface {
	pgx.Tx
}

// ErrAppealAlreadyResolved is returned when an UPDATE targets an appeal that
// has already been resolved (optimistic lock guard against concurrent resolves).
var ErrAppealAlreadyResolved = errors.New("appeal already resolved")

// AppealRepository persists and retrieves Appeal records.
type AppealRepository struct {
	db *pgxpool.Pool
}

// NewAppealRepository creates a new AppealRepository.
func NewAppealRepository(db *pgxpool.Pool) *AppealRepository {
	return &AppealRepository{db: db}
}

// Create inserts a new appeal record.
func (r *AppealRepository) Create(ctx context.Context, a *model.Appeal) error {
	const q = `
		INSERT INTO appeals (id, tenant_id, content_id, applicant_id, reason, evidence_urls, status, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, tenant_id, content_id, applicant_id, reason, evidence_urls, status, reviewer_id, resolution, penalty_level_code, comment, submitted_at, resolved_at`

	err := r.db.QueryRow(ctx, q,
		a.ID, a.TenantID, a.ContentID, a.ApplicantID, a.Reason, a.EvidenceURLs, a.Status,
	).Scan(
		&a.ID, &a.TenantID, &a.ContentID, &a.ApplicantID, &a.Reason, &a.EvidenceURLs, &a.Status,
		&a.ReviewerID, &a.Resolution, &a.PenaltyLevel, &a.Comment, &a.SubmittedAt, &a.ResolvedAt,
	)
	if err != nil {
		return fmt.Errorf("create appeal: %w", err)
	}
	return nil
}

// FindByID loads an appeal by its UUID.
func (r *AppealRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Appeal, error) {
	a := &model.Appeal{}
	const q = `
		SELECT id, tenant_id, content_id, applicant_id, reason, evidence_urls, status,
		       reviewer_id, resolution, penalty_level_code, comment,
		       submitted_at, resolved_at
		FROM appeals WHERE id = $1`

	err := r.db.QueryRow(ctx, q, id).Scan(
		&a.ID, &a.TenantID, &a.ContentID, &a.ApplicantID, &a.Reason, &a.EvidenceURLs, &a.Status,
		&a.ReviewerID, &a.Resolution, &a.PenaltyLevel, &a.Comment,
		&a.SubmittedAt, &a.ResolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find appeal by id: %w", err)
	}
	return a, nil
}

// FindByContentID returns the most recent appeal for a given content (for uniqueness check).
func (r *AppealRepository) FindByContentID(ctx context.Context, contentID uuid.UUID) (*model.Appeal, error) {
	a := &model.Appeal{}
	const q = `
		SELECT id, tenant_id, content_id, applicant_id, reason, evidence_urls, status,
		       reviewer_id, resolution, penalty_level_code, comment,
		       submitted_at, resolved_at
		FROM appeals WHERE content_id = $1
		ORDER BY submitted_at DESC LIMIT 1`

	err := r.db.QueryRow(ctx, q, contentID).Scan(
		&a.ID, &a.TenantID, &a.ContentID, &a.ApplicantID, &a.Reason, &a.EvidenceURLs, &a.Status,
		&a.ReviewerID, &a.Resolution, &a.PenaltyLevel, &a.Comment,
		&a.SubmittedAt, &a.ResolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find appeal by content id: %w", err)
	}
	return a, nil
}

// Update modifies an existing appeal record with the supplied fields.
func (r *AppealRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateAppealRequest) (*model.Appeal, error) {
	setParts := make([]string, 0, 5)
	args := make([]interface{}, 0, 5)
	idx := 1

	if req.ReviewerID != nil {
		setParts = append(setParts, fmt.Sprintf("reviewer_id = $%d", idx))
		args = append(args, *req.ReviewerID)
		idx++
	}
	if req.Resolution != nil {
		setParts = append(setParts, fmt.Sprintf("resolution = $%d", idx))
		args = append(args, *req.Resolution)
		idx++
	}
	if req.PenaltyLevel != nil {
		setParts = append(setParts, fmt.Sprintf("penalty_level_code = $%d", idx))
		args = append(args, *req.PenaltyLevel)
		idx++
	}
	if req.Comment != nil {
		setParts = append(setParts, fmt.Sprintf("comment = $%d", idx))
		args = append(args, *req.Comment)
		idx++
	}

	// Always allow update even if no fields changed — just return existing record
	args = append(args, id)

	query := fmt.Sprintf(
		"UPDATE appeals SET %s WHERE id = $%d",
		strings.Join(setParts, ", "),
		idx,
	)

	a := &model.Appeal{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&a.ID, &a.TenantID, &a.ContentID, &a.ApplicantID, &a.Reason, &a.EvidenceURLs, &a.Status,
		&a.ReviewerID, &a.Resolution, &a.PenaltyLevel, &a.Comment,
		&a.SubmittedAt, &a.ResolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update appeal: %w", err)
	}
	return a, nil
}

// UpdateWithTx modifies an existing appeal record within a transaction.
func (r *AppealRepository) UpdateWithTx(ctx context.Context, tx txConn, id uuid.UUID, req model.UpdateAppealRequest) error {
	setParts := make([]string, 0, 5)
	args := make([]interface{}, 0, 5)
	idx := 1

	if req.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", idx))
		args = append(args, *req.Status)
		idx++
	}
	if req.ReviewerID != nil {
		setParts = append(setParts, fmt.Sprintf("reviewer_id = $%d", idx))
		args = append(args, *req.ReviewerID)
		idx++
	}
	if req.Resolution != nil {
		setParts = append(setParts, fmt.Sprintf("resolution = $%d", idx))
		args = append(args, *req.Resolution)
		idx++
	}
	if req.PenaltyLevel != nil {
		setParts = append(setParts, fmt.Sprintf("penalty_level_code = $%d", idx))
		args = append(args, *req.PenaltyLevel)
		idx++
	}
	if req.Comment != nil {
		setParts = append(setParts, fmt.Sprintf("comment = $%d", idx))
		args = append(args, *req.Comment)
		idx++
	}

	if len(setParts) == 0 {
		return nil // nothing to update
	}

	args = append(args, id)

	// Optimistic locking: only update appeals that are not yet resolved.
	// This prevents two concurrent reviewers from double-resolving the same
	// appeal (the second UPDATE matches 0 rows and is reported as an error).
	query := fmt.Sprintf(
		"UPDATE appeals SET %s WHERE id = $%d AND status NOT IN ('resolved_approved', 'resolved_maintained')",
		strings.Join(setParts, ", "),
		idx,
	)

	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update appeal (tx): %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAppealAlreadyResolved
	}
	return nil
}

// ListByStatus returns a paginated list of appeals filtered by status.
func (r *AppealRepository) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]model.Appeal, int64, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	const countQ = `SELECT COUNT(*) FROM appeals WHERE status = $1`
	var total int64
	if err := r.db.QueryRow(ctx, countQ, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count appeals: %w", err)
	}

	const listQ = `
		SELECT id, tenant_id, content_id, applicant_id, reason, evidence_urls, status,
		       reviewer_id, resolution, penalty_level_code, comment,
		       submitted_at, resolved_at
		FROM appeals WHERE status = $1 ORDER BY submitted_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, listQ, status, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list appeals: %w", err)
	}
	defer rows.Close()

	items := make([]model.Appeal, 0, pageSize)
	for rows.Next() {
		var a model.Appeal
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.ContentID, &a.ApplicantID, &a.Reason, &a.EvidenceURLs, &a.Status,
			&a.ReviewerID, &a.Resolution, &a.PenaltyLevel, &a.Comment,
			&a.SubmittedAt, &a.ResolvedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan appeal: %w", err)
		}
		items = append(items, a)
	}

	return items, total, nil
}

// ListByTenantAndStatus returns a paginated list of appeals for a specific tenant filtered by status.
func (r *AppealRepository) ListByTenantAndStatus(ctx context.Context, tenantID, status string, page, pageSize int) ([]model.Appeal, int64, error) {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid tenant ID: %w", err)
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	const countQ = `SELECT COUNT(*) FROM appeals WHERE tenant_id = $1 AND status = $2`
	var total int64
	if err := r.db.QueryRow(ctx, countQ, tenantUUID, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count appeals by tenant: %w", err)
	}

	const listQ = `
		SELECT id, tenant_id, content_id, applicant_id, reason, evidence_urls, status,
		       reviewer_id, resolution, penalty_level_code, comment,
		       submitted_at, resolved_at
		FROM appeals WHERE tenant_id = $1 AND status = $2 ORDER BY submitted_at DESC LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, listQ, tenantUUID, status, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list appeals by tenant: %w", err)
	}
	defer rows.Close()

	items := make([]model.Appeal, 0, pageSize)
	for rows.Next() {
		var a model.Appeal
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.ContentID, &a.ApplicantID, &a.Reason, &a.EvidenceURLs, &a.Status,
			&a.ReviewerID, &a.Resolution, &a.PenaltyLevel, &a.Comment,
			&a.SubmittedAt, &a.ResolvedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan appeal: %w", err)
		}
		items = append(items, a)
	}

	return items, total, nil
}

// CountPending returns the total number of appeals in submitted or under_review status.
func (r *AppealRepository) CountPending(ctx context.Context) (int64, error) {
	const q = `SELECT COUNT(*) FROM appeals WHERE status IN ($1, $2)`
	var count int64
	err := r.db.QueryRow(ctx, q, model.AppealSubmitted, model.AppealUnderReview).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending appeals: %w", err)
	}
	return count, nil
}

// BeginTx begins a new database transaction.
func (r *AppealRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// CreateWithTx inserts a new appeal record within a transaction.
func (r *AppealRepository) CreateWithTx(ctx context.Context, tx txConn, a *model.Appeal) error {
	const q = `
		INSERT INTO appeals (id, tenant_id, content_id, applicant_id, reason, evidence_urls, status, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, tenant_id, content_id, applicant_id, reason, evidence_urls, status, reviewer_id, resolution, penalty_level_code, comment, submitted_at, resolved_at`

	err := tx.QueryRow(ctx, q,
		a.ID, a.TenantID, a.ContentID, a.ApplicantID, a.Reason, a.EvidenceURLs, a.Status,
	).Scan(
		&a.ID, &a.TenantID, &a.ContentID, &a.ApplicantID, &a.Reason, &a.EvidenceURLs, &a.Status,
		&a.ReviewerID, &a.Resolution, &a.PenaltyLevel, &a.Comment, &a.SubmittedAt, &a.ResolvedAt,
	)
	if err != nil {
		return fmt.Errorf("create appeal (tx): %w", err)
	}
	return nil
}

// FindByContentAndApplicant returns the most recent appeal for a given content and applicant.
func (r *AppealRepository) FindByContentAndApplicant(ctx context.Context, contentID, applicantID uuid.UUID) (*model.Appeal, error) {
	a := &model.Appeal{}
	const q = `
		SELECT id, content_id, applicant_id, reason, evidence_urls, status,
		       reviewer_id, resolution, penalty_level_code, comment,
		       submitted_at, resolved_at
		FROM appeals WHERE content_id = $1 AND applicant_id = $2
		ORDER BY submitted_at DESC LIMIT 1`

	err := r.db.QueryRow(ctx, q, contentID, applicantID).Scan(
		&a.ID, &a.ContentID, &a.ApplicantID, &a.Reason, &a.EvidenceURLs, &a.Status,
		&a.ReviewerID, &a.Resolution, &a.PenaltyLevel, &a.Comment,
		&a.SubmittedAt, &a.ResolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find appeal by content and applicant: %w", err)
	}
	return a, nil
}
