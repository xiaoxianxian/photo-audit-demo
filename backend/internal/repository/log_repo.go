package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"audit-platform/internal/model"
)

// LogRepository is an append-only store for audit records.
type LogRepository struct {
	db *pgxpool.Pool
}

// NewLogRepository creates a new LogRepository.
func NewLogRepository(db *pgxpool.Pool) *LogRepository {
	return &LogRepository{db: db}
}

// Create inserts an audit record. This is an append-only operation.
func (r *LogRepository) Create(ctx context.Context, record *model.AuditRecord) error {
	const q = `
		INSERT INTO audit_records (id, task_id, element_id, reviewer_id, review_type, action,
		                           penalty_level_code, reason, comment, ai_score_before, ai_score_after,
		                           is_conflict, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		RETURNING id, task_id, element_id, reviewer_id, review_type, action,
		          penalty_level_code, reason, comment, ai_score_before, ai_score_after,
		          is_conflict, created_at`

	err := r.db.QueryRow(ctx, q,
		record.ID, record.TaskID, record.ElementID, record.ReviewerID, record.ReviewType,
		record.Action, record.PenaltyLevel, record.Reason, record.Comment,
		record.AIScoreBefore, record.AIScoreAfter, record.IsConflict,
	).Scan(
		&record.ID, &record.TaskID, &record.ElementID, &record.ReviewerID, &record.ReviewType,
		&record.Action, &record.PenaltyLevel, &record.Reason, &record.Comment,
		&record.AIScoreBefore, &record.AIScoreAfter, &record.IsConflict, &record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create audit record: %w", err)
	}
	return nil
}

// CreateWithTx inserts an audit record within a transaction.
func (r *LogRepository) CreateWithTx(ctx context.Context, tx txConn, record *model.AuditRecord) error {
	const q = `
		INSERT INTO audit_records (id, task_id, element_id, reviewer_id, review_type, action,
		                           penalty_level_code, reason, comment, ai_score_before, ai_score_after,
		                           is_conflict, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())`

	_, err := tx.Exec(ctx, q,
		record.ID, record.TaskID, record.ElementID, record.ReviewerID, record.ReviewType,
		record.Action, record.PenaltyLevel, record.Reason, record.Comment,
		record.AIScoreBefore, record.AIScoreAfter, record.IsConflict,
	)
	if err != nil {
		return fmt.Errorf("create audit record (tx): %w", err)
	}
	return nil
}

// FindByElementID returns the full review history for a content element.
func (r *LogRepository) FindByElementID(ctx context.Context, elementID uuid.UUID) ([]model.AuditRecord, error) {
	const q = `
		SELECT id, task_id, element_id, reviewer_id, review_type, action,
		       penalty_level_code, reason, comment, ai_score_before, ai_score_after,
		       is_conflict, created_at
		FROM audit_records WHERE element_id = $1 ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, q, elementID)
	if err != nil {
		return nil, fmt.Errorf("find audit records by element id: %w", err)
	}
	defer rows.Close()

	var items []model.AuditRecord
	for rows.Next() {
		var rec model.AuditRecord
		if err := rows.Scan(
			&rec.ID, &rec.TaskID, &rec.ElementID, &rec.ReviewerID, &rec.ReviewType,
			&rec.Action, &rec.PenaltyLevel, &rec.Reason, &rec.Comment,
			&rec.AIScoreBefore, &rec.AIScoreAfter, &rec.IsConflict, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit record: %w", err)
		}
		items = append(items, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit record rows: %w", err)
	}

	return items, nil
}

// FindByReviewer returns a paginated list of audit records authored by a reviewer.
func (r *LogRepository) FindByReviewer(ctx context.Context, reviewerID uuid.UUID, page, pageSize int) ([]model.AuditRecord, int64, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	const countQ = `SELECT COUNT(*) FROM audit_records WHERE reviewer_id = $1`
	var total int64
	if err := r.db.QueryRow(ctx, countQ, reviewerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit records by reviewer: %w", err)
	}

	const listQ = `
		SELECT id, task_id, element_id, reviewer_id, review_type, action,
		       penalty_level_code, reason, comment, ai_score_before, ai_score_after,
		       is_conflict, created_at
		FROM audit_records WHERE reviewer_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, listQ, reviewerID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("find audit records by reviewer: %w", err)
	}
	defer rows.Close()

	items := make([]model.AuditRecord, 0, pageSize)
	for rows.Next() {
		var rec model.AuditRecord
		if err := rows.Scan(
			&rec.ID, &rec.TaskID, &rec.ElementID, &rec.ReviewerID, &rec.ReviewType,
			&rec.Action, &rec.PenaltyLevel, &rec.Reason, &rec.Comment,
			&rec.AIScoreBefore, &rec.AIScoreAfter, &rec.IsConflict, &rec.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit record: %w", err)
		}
		items = append(items, rec)
	}

	return items, total, nil
}

// ListAll returns a paginated list of all audit records, ordered by created_at desc.
func (r *LogRepository) ListAll(ctx context.Context, page, pageSize int) ([]model.AuditRecord, int64, error) {
	return r.ListAllFiltered(ctx, page, pageSize, "", "")
}

// ListAllFiltered returns a paginated list of audit records with optional action/review_type filters.
func (r *LogRepository) ListAllFiltered(ctx context.Context, page, pageSize int, action, reviewType string) ([]model.AuditRecord, int64, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Build count query.
	var countQ string
	var countArgs []interface{}
	if action != "" || reviewType != "" {
		countQ = `SELECT COUNT(*) FROM audit_records WHERE 1=1`
		idx := 1
		if action != "" {
			countQ += fmt.Sprintf(" AND action = $%d", idx)
			countArgs = append(countArgs, action)
			idx++
		}
		if reviewType != "" {
			countQ += fmt.Sprintf(" AND review_type = $%d", idx)
			countArgs = append(countArgs, reviewType)
		}
	} else {
		countQ = `SELECT COUNT(*) FROM audit_records`
	}
	var total int64
	if err := r.db.QueryRow(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count all audit records: %w", err)
	}

	// Build list query.
	var listQ string
	var listArgs []interface{}
	listQ = `SELECT id, task_id, element_id, reviewer_id, review_type, action, penalty_level_code, reason, comment, ai_score_before, ai_score_after, is_conflict, created_at FROM audit_records WHERE 1=1`
	idx := 1
	if action != "" {
		listQ += fmt.Sprintf(" AND action = $%d", idx)
		listArgs = append(listArgs, action)
		idx++
	}
	if reviewType != "" {
		listQ += fmt.Sprintf(" AND review_type = $%d", idx)
		listArgs = append(listArgs, reviewType)
		idx++
	}
	listQ += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	listArgs = append(listArgs, pageSize, offset)

	rows, err := r.db.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit records: %w", err)
	}
	defer rows.Close()

	items := make([]model.AuditRecord, 0, pageSize)
	for rows.Next() {
		var rec model.AuditRecord
		if err := rows.Scan(
			&rec.ID, &rec.TaskID, &rec.ElementID, &rec.ReviewerID, &rec.ReviewType,
			&rec.Action, &rec.PenaltyLevel, &rec.Reason, &rec.Comment,
			&rec.AIScoreBefore, &rec.AIScoreAfter, &rec.IsConflict, &rec.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit record: %w", err)
		}
		items = append(items, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate audit record rows: %w", err)
	}

	return items, total, nil
}

// CountByType returns the total number of audit records for a given review type.
// If tenantID is non-empty, only records for that tenant are counted (via element join).
func (r *LogRepository) CountByType(ctx context.Context, reviewType model.ReviewType, tenantID string) (int64, error) {
	var q string
	var args []interface{}
	if tenantID != "" {
		q = `SELECT COUNT(*) FROM audit_records ar JOIN content_elements ce ON ce.id = ar.element_id JOIN contents c ON c.id = ce.content_id WHERE ar.review_type = $1 AND c.tenant_id = $2`
		args = []interface{}{reviewType, tenantID}
	} else {
		q = `SELECT COUNT(*) FROM audit_records WHERE review_type = $1`
		args = []interface{}{reviewType}
	}
	var count int64
	err := r.db.QueryRow(ctx, q, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count audit records by type: %w", err)
	}
	return count, nil
}

// CountByDateRange returns the number of audit records within a time window.
// If tenantID is non-empty, only records for that tenant are counted.
func (r *LogRepository) CountByDateRange(ctx context.Context, start, end time.Time, tenantID string) (int64, error) {
	var q string
	var args []interface{}
	if tenantID != "" {
		q = `SELECT COUNT(*) FROM audit_records ar JOIN content_elements ce ON ce.id = ar.element_id JOIN contents c ON c.id = ce.content_id WHERE ar.created_at >= $1 AND ar.created_at < $2 AND c.tenant_id = $3`
		args = []interface{}{start, end, tenantID}
	} else {
		q = `SELECT COUNT(*) FROM audit_records WHERE created_at >= $1 AND created_at < $2`
		args = []interface{}{start, end}
	}
	var count int64
	err := r.db.QueryRow(ctx, q, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count audit records by date range: %w", err)
	}
	return count, nil
}

// CountByAction returns the total number of audit records for a given review action.
// If tenantID is non-empty, only records for that tenant are counted.
func (r *LogRepository) CountByAction(ctx context.Context, action model.ReviewAction, tenantID string) (int64, error) {
	var q string
	var args []interface{}
	if tenantID != "" {
		q = `SELECT COUNT(*) FROM audit_records ar JOIN content_elements ce ON ce.id = ar.element_id JOIN contents c ON c.id = ce.content_id WHERE ar.action = $1 AND c.tenant_id = $2`
		args = []interface{}{action, tenantID}
	} else {
		q = `SELECT COUNT(*) FROM audit_records WHERE action = $1`
		args = []interface{}{action}
	}
	var count int64
	err := r.db.QueryRow(ctx, q, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count audit records by action: %w", err)
	}
	return count, nil
}

// CountByActionDateRange returns the number of audit records for a given action within a time window.
// If tenantID is non-empty, only records for that tenant are counted.
func (r *LogRepository) CountByActionDateRange(ctx context.Context, action model.ReviewAction, start, end time.Time, tenantID string) (int64, error) {
	var q string
	var args []interface{}
	if tenantID != "" {
		q = `SELECT COUNT(*) FROM audit_records ar JOIN content_elements ce ON ce.id = ar.element_id JOIN contents c ON c.id = ce.content_id WHERE ar.action = $1 AND ar.created_at >= $2 AND ar.created_at < $3 AND c.tenant_id = $4`
		args = []interface{}{action, start, end, tenantID}
	} else {
		q = `SELECT COUNT(*) FROM audit_records WHERE action = $1 AND created_at >= $2 AND created_at < $3`
		args = []interface{}{action, start, end}
	}
	var count int64
	err := r.db.QueryRow(ctx, q, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count audit records by action and date range: %w", err)
	}
	return count, nil
}

// AvgAIScore returns the average ai_score_after across all audit records.
// Returns 0 if there are no records. If tenantID is non-empty, only records for that tenant are counted.
func (r *LogRepository) AvgAIScore(ctx context.Context, tenantID string) (float64, error) {
	var q string
	var args []interface{}
	if tenantID != "" {
		q = `SELECT COALESCE(AVG(ar.ai_score_after), 0)::numeric FROM audit_records ar JOIN content_elements ce ON ce.id = ar.element_id JOIN contents c ON c.id = ce.content_id WHERE ar.ai_score_after IS NOT NULL AND c.tenant_id = $1`
		args = []interface{}{tenantID}
	} else {
		q = `SELECT COALESCE(AVG(ai_score_after), 0)::numeric FROM audit_records WHERE ai_score_after IS NOT NULL`
	}
	var avg float64
	err := r.db.QueryRow(ctx, q, args...).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("avg ai score: %w", err)
	}
	return avg, nil
}

// CountConflictRecords returns the number of audit records with is_conflict = true.
// If tenantID is non-empty, only records for that tenant are counted.
func (r *LogRepository) CountConflictRecords(ctx context.Context, tenantID string) (int64, error) {
	var q string
	var args []interface{}
	if tenantID != "" {
		q = `SELECT COUNT(*) FROM audit_records ar JOIN content_elements ce ON ce.id = ar.element_id JOIN contents c ON c.id = ce.content_id WHERE ar.is_conflict = true AND c.tenant_id = $1`
		args = []interface{}{tenantID}
	} else {
		q = `SELECT COUNT(*) FROM audit_records WHERE is_conflict = true`
	}
	var count int64
	err := r.db.QueryRow(ctx, q, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count conflict records: %w", err)
	}
	return count, nil
}

// CountPendingAppeals returns the number of appeals in submitted or under_review status.
// If tenantID is non-empty, only appeals for that tenant are counted (via contents join).
func (r *LogRepository) CountPendingAppeals(ctx context.Context, tenantID string) (int64, error) {
	var q string
	var args []interface{}
	if tenantID != "" {
		q = `SELECT COUNT(*) FROM appeals ap JOIN contents c ON c.id = ap.content_id WHERE ap.status IN ($1, $2) AND c.tenant_id = $3`
		args = []interface{}{"submitted", "under_review", tenantID}
	} else {
		q = `SELECT COUNT(*) FROM appeals WHERE status IN ($1, $2)`
		args = []interface{}{"submitted", "under_review"}
	}
	var count int64
	err := r.db.QueryRow(ctx, q, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending appeals: %w", err)
	}
	return count, nil
}

// ReviewerStats aggregates performance metrics per reviewer.
type ReviewerStats struct {
	ReviewerID   uuid.UUID `db:"reviewer_id"`
	ReviewerName string    `db:"reviewer_name"`
	TotalReviews int64     `db:"total_reviews"`
	Approved     int64     `db:"approved"`
	Rejected     int64     `db:"rejected"`
	Appeals      int64     `db:"appeals"`
	Accuracy     float64   `db:"accuracy"`
	AvgTimeSec   float64   `db:"avg_time_sec"`
}

// DashboardStatsRow holds the result of a consolidated dashboard query.
type DashboardStatsRow struct {
	TotalReviewed   int64
	TodayReviewed   int64
	Approved        int64
	Rejected        int64
	AvgRiskScore    float64
	ConflictCount   int64
	PendingAppeals  int64
}

// GetDashboardStatsConsolidated runs a single query to fetch all dashboard
// statistics for a tenant. This replaces 8 separate SELECT calls.
func (r *LogRepository) GetDashboardStatsConsolidated(ctx context.Context, tenantID string) (*DashboardStatsRow, error) {
	var row DashboardStatsRow

	if tenantID == "" {
		// Platform-wide query (no tenant filter).
		const q = `
			WITH stats AS (
				SELECT
					COUNT(*) FILTER (WHERE review_type = 'human') AS total_reviewed,
					COUNT(*) FILTER (WHERE action = 'approve') AS approved,
					COUNT(*) FILTER (WHERE action = 'reject') AS rejected,
					COALESCE(AVG(ai_score_after), 0)::numeric AS avg_risk,
					COUNT(*) FILTER (WHERE is_conflict = true) AS conflicts
				FROM audit_records
			),
			today AS (
				SELECT COUNT(*) AS cnt
				FROM audit_records
				WHERE created_at >= $1
			),
			appeals AS (
				SELECT COUNT(*) AS cnt
				FROM appeals
				WHERE status IN ('submitted', 'under_review')
			)
			SELECT s.total_reviewed, t.cnt, s.approved, s.rejected,
			       ROUND(s.avg_risk::numeric, 2), s.conflicts, a.cnt
			FROM stats s, today t, appeals a`

		err := r.db.QueryRow(ctx, q, time.Now().Truncate(24*time.Hour)).Scan(
			&row.TotalReviewed, &row.TodayReviewed,
			&row.Approved, &row.Rejected,
			&row.AvgRiskScore, &row.ConflictCount, &row.PendingAppeals,
		)
		if err != nil {
			return nil, fmt.Errorf("consolidated dashboard stats: %w", err)
		}
		return &row, nil
	}

	// Tenant-scoped query (JOIN through content_elements + contents).
	const q = `
		WITH stats AS (
			SELECT
				COUNT(*) FILTER (WHERE ar.review_type = 'human') AS total_reviewed,
				COUNT(*) FILTER (WHERE ar.action = 'approve') AS approved,
				COUNT(*) FILTER (WHERE ar.action = 'reject') AS rejected,
				COALESCE(AVG(ar.ai_score_after), 0)::numeric AS avg_risk,
				COUNT(*) FILTER (WHERE ar.is_conflict = true) AS conflicts
			FROM audit_records ar
			JOIN content_elements ce ON ce.id = ar.element_id
			JOIN contents c ON c.id = ce.content_id
			WHERE c.tenant_id = $1
		),
		today AS (
			SELECT COUNT(*) AS cnt
			FROM audit_records ar
			JOIN content_elements ce ON ce.id = ar.element_id
			JOIN contents c ON c.id = ce.content_id
			WHERE c.tenant_id = $1 AND ar.created_at >= $2
		),
		appeals AS (
			SELECT COUNT(*) AS cnt
			FROM appeals ap
			JOIN contents c ON c.id = ap.content_id
			WHERE ap.status IN ('submitted', 'under_review') AND c.tenant_id = $3
		)
		SELECT s.total_reviewed, t.cnt, s.approved, s.rejected,
		       ROUND(s.avg_risk::numeric, 2), s.conflicts, a.cnt
		FROM stats s, today t, appeals a`

	err := r.db.QueryRow(ctx, q, tenantID, time.Now().Truncate(24*time.Hour), tenantID).Scan(
		&row.TotalReviewed, &row.TodayReviewed,
		&row.Approved, &row.Rejected,
		&row.AvgRiskScore, &row.ConflictCount, &row.PendingAppeals,
	)
	if err != nil {
		return nil, fmt.Errorf("consolidated dashboard stats: %w", err)
	}
	return &row, nil
}

// CountByReviewer returns aggregated performance stats for all reviewers in the given tenant.
func (r *LogRepository) CountByReviewer(ctx context.Context, tenantID string) ([]ReviewerStats, error) {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	const q = `
		WITH element_review_times AS (
			SELECT
				ar.reviewer_id,
				ar.element_id,
				ar.created_at,
				LAG(ar.created_at) OVER (PARTITION BY ar.element_id ORDER BY ar.created_at) AS prev_created_at
			FROM audit_records ar
			JOIN content_elements ce ON ce.id = ar.element_id
			JOIN contents c ON c.id = ce.content_id
			WHERE ar.reviewer_id IS NOT NULL AND c.tenant_id = $1
		),
		reviewer_counts AS (
			SELECT
				reviewer_id,
				COUNT(*) FILTER (WHERE action = 'approve') AS approved,
				COUNT(*) FILTER (WHERE action = 'reject')  AS rejected,
				COUNT(*)                                       AS total_reviews,
				COALESCE(AVG(EXTRACT(EPOCH FROM (created_at - prev_created_at))), 0) AS avg_time_sec
			FROM element_review_times
			GROUP BY reviewer_id
		),
		reviewer_appeals AS (
			SELECT
				reviewer_id,
				COUNT(*) FILTER (WHERE status IN ('resolved_approved', 'resolved_maintained')) AS appeal_count
			FROM appeals
			JOIN contents c ON c.id = appeals.content_id
			WHERE reviewer_id IS NOT NULL AND c.tenant_id = $2
			GROUP BY reviewer_id
		)
		SELECT
			rc.reviewer_id,
			COALESCE(u.display_name, 'Unknown'),
			rc.total_reviews,
			rc.approved,
			rc.rejected,
			COALESCE(ra.appeal_count, 0) AS appeals,
			CASE WHEN rc.total_reviews > 0
				THEN ROUND((rc.approved::numeric / rc.total_reviews::numeric) * 100, 2)
				ELSE 0 END AS accuracy,
			ROUND(rc.avg_time_sec, 2) AS avg_time_sec
		FROM reviewer_counts rc
		LEFT JOIN users u ON u.id = rc.reviewer_id
		LEFT JOIN reviewer_appeals ra ON ra.reviewer_id = rc.reviewer_id
		ORDER BY rc.total_reviews DESC`

	rows, err := r.db.Query(ctx, q, tenantUUID, tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("count by reviewer: %w", err)
	}
	defer rows.Close()

	var stats []ReviewerStats
	for rows.Next() {
		var s ReviewerStats
		if err := rows.Scan(
			&s.ReviewerID, &s.ReviewerName,
			&s.TotalReviews, &s.Approved, &s.Rejected,
			&s.Appeals, &s.Accuracy, &s.AvgTimeSec,
		); err != nil {
			return nil, fmt.Errorf("scan reviewer stats: %w", err)
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reviewer stats: %w", err)
	}
	return stats, nil
}
