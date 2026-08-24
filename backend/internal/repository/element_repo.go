package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"audit-platform/internal/model"
)

// ElementRepository persists and retrieves ContentElement records.
type ElementRepository struct {
	db *pgxpool.Pool
}

// NewElementRepository creates a new ElementRepository.
func NewElementRepository(db *pgxpool.Pool) *ElementRepository {
	return &ElementRepository{db: db}
}

// BeginTx begins a new transaction on the underlying pool.
func (r *ElementRepository) BeginTx(ctx context.Context) (txConn, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	return tx, nil
}

// Create inserts a single content element.
func (r *ElementRepository) Create(ctx context.Context, e *model.ContentElement) error {
	const q = `
		INSERT INTO content_elements (id, content_id, element_kind, element_content, ai_risk_score, ai_risk_types, ai_confidence, ai_status, human_status, is_conflict, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, content_id, element_kind, element_content, ai_risk_score, ai_risk_types, ai_confidence, ai_status, human_status, is_conflict, created_at, updated_at`

	var types []string
	if e.AIRiskTypes != nil {
		types = e.AIRiskTypes
	} else {
		types = []string{} // avoid NULL violating NOT NULL constraint (bypasses column DEFAULT)
	}

	err := r.db.QueryRow(ctx, q,
		e.ID, e.ContentID, e.ElementKind, e.ElementContent, e.AIRiskScore,
		types, e.AIConfidence, e.AIStatus, e.HumanStatus, e.IsConflict,
	).Scan(
		&e.ID, &e.ContentID, &e.ElementKind, &e.ElementContent, &e.AIRiskScore,
		&e.AIRiskTypes, &e.AIConfidence, &e.AIStatus, &e.HumanStatus, &e.IsConflict, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create element: %w", err)
	}
	return nil
}

// CreateBulk batch-inserts multiple elements atomically using a transaction.
func (r *ElementRepository) CreateBulk(ctx context.Context, elements []*model.ContentElement) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bulk insert tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := `
		INSERT INTO content_elements (id, content_id, element_kind, element_content, ai_risk_score, ai_risk_types, ai_confidence, ai_status, human_status, is_conflict, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())`

	for _, e := range elements {
		types := e.AIRiskTypes
		if types == nil {
			types = []string{} // avoid NULL violating NOT NULL constraint (bypasses column DEFAULT)
		}
		_, err := tx.Exec(ctx, q,
			e.ID, e.ContentID, e.ElementKind, e.ElementContent, e.AIRiskScore,
			types, e.AIConfidence, e.AIStatus, e.HumanStatus, e.IsConflict,
		)
		if err != nil {
			return fmt.Errorf("bulk create element %s: %w", e.ID, err)
		}
	}

	return tx.Commit(ctx)
}

// FindByContentID returns all elements belonging to a content item.
func (r *ElementRepository) FindByContentID(ctx context.Context, contentID uuid.UUID) ([]model.ContentElement, error) {
	const q = `
		SELECT id, content_id, element_kind, element_content, ai_risk_score, ai_risk_types, ai_confidence, ai_status, human_status, is_conflict, created_at, updated_at
		FROM content_elements WHERE content_id = $1 ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, q, contentID)
	if err != nil {
		return nil, fmt.Errorf("find elements by content id: %w", err)
	}
	defer rows.Close()

	var items []model.ContentElement
	for rows.Next() {
		var e model.ContentElement
		if err := rows.Scan(
			&e.ID, &e.ContentID, &e.ElementKind, &e.ElementContent, &e.AIRiskScore,
			&e.AIRiskTypes, &e.AIConfidence, &e.AIStatus, &e.HumanStatus, &e.IsConflict, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan element: %w", err)
		}
		items = append(items, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate element rows: %w", err)
	}

	return items, nil
}

// FindByStatus returns elements matching the given AI and human status filters.
// Optional filters: elementKind (for content element type), riskMin, riskMax (risk score range).
// sortField and sortOrder control the ORDER BY clause. page/pageSize control pagination.
func (r *ElementRepository) FindByStatus(ctx context.Context, aiStatus, humanStatus string, elementKind string, riskMin, riskMax int, sortField, sortOrder string, page, pageSize int) ([]model.ContentElement, int64, error) {
	whereParts := []string{"ai_status = $1", "human_status = $2"}
	args := []interface{}{aiStatus, humanStatus}
	idx := 3

	if elementKind != "" {
		whereParts = append(whereParts, fmt.Sprintf("element_kind = $%d", idx))
		args = append(args, elementKind)
		idx++
	}
	if riskMin > 0 {
		whereParts = append(whereParts, fmt.Sprintf("ai_risk_score >= $%d", idx))
		args = append(args, riskMin)
		idx++
	}
	if riskMax < 100 {
		whereParts = append(whereParts, fmt.Sprintf("ai_risk_score <= $%d", idx))
		args = append(args, riskMax)
		idx++
	}

	// Build ORDER BY with whitelist validation
	validSortFields := map[string]string{
		"created_at":    "created_at",
		"ai_risk_score": "ai_risk_score",
	}
	field := validSortFields[sortField]
	if field == "" {
		field = "created_at"
	}
	orderClause := field + " " + sortOrder

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM content_elements WHERE %s", strings.Join(whereParts[:2], " AND "))
	// Rebuild count with same dynamic conditions
	if len(whereParts) > 2 {
		countQ = fmt.Sprintf("SELECT COUNT(*) FROM content_elements WHERE %s", strings.Join(whereParts, " AND "))
	}
	var total int64
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count elements by status: %w", err)
	}

	listQ := fmt.Sprintf(
		`SELECT id, content_id, element_kind, element_content, ai_risk_score, ai_risk_types, ai_confidence, ai_status, human_status, is_conflict, created_at, updated_at
		FROM content_elements WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		strings.Join(whereParts, " AND "),
		orderClause,
		idx, idx+1,
	)

	rows, err := r.db.Query(ctx, listQ, append(args, page, pageSize*(page-1))...)
	if err != nil {
		return nil, 0, fmt.Errorf("find elements by status: %w", err)
	}
	defer rows.Close()

	var items []model.ContentElement
	for rows.Next() {
		var e model.ContentElement
		if err := rows.Scan(
			&e.ID, &e.ContentID, &e.ElementKind, &e.ElementContent, &e.AIRiskScore,
			&e.AIRiskTypes, &e.AIConfidence, &e.AIStatus, &e.HumanStatus, &e.IsConflict, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan element: %w", err)
		}
		items = append(items, e)
	}

	return items, total, nil
}

// UpdateStatus changes the AI and human status of a single element.
func (r *ElementRepository) UpdateStatus(ctx context.Context, id uuid.UUID, aiStatus, humanStatus string) error {
	const q = `
		UPDATE content_elements SET ai_status = $1, human_status = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, q, aiStatus, humanStatus, id)
	if err != nil {
		return fmt.Errorf("update element status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("update element status: %w", pgx.ErrNoRows)
	}
	return nil
}

// UpdateStatusWithTx changes the AI and human status within a transaction.
func (r *ElementRepository) UpdateStatusWithTx(ctx context.Context, tx txConn, id uuid.UUID, aiStatus, humanStatus string) error {
	const q = `
		UPDATE content_elements SET ai_status = $1, human_status = $2 WHERE id = $3`

	result, err := tx.Exec(ctx, q, aiStatus, humanStatus, id)
	if err != nil {
		return fmt.Errorf("update element status (tx): %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("update element status (tx): %w", pgx.ErrNoRows)
	}
	return nil
}

// UpdateAIRisk records the results returned by the AI model for an element.
func (r *ElementRepository) UpdateAIRisk(ctx context.Context, id uuid.UUID, score int, types []string, confidence float64) error {
	const q = `
		UPDATE content_elements
		SET ai_risk_score = $1, ai_risk_types = $2, ai_confidence = $3, ai_status = $4, updated_at = NOW()
		WHERE id = $5`

	var setStatus model.ElementStatus
	if score >= 60 {
		setStatus = model.ElementAIRejected
	} else {
		setStatus = model.ElementAIPassed
	}

	result, err := r.db.Exec(ctx, q, score, types, confidence, setStatus, id)
	if err != nil {
		return fmt.Errorf("update element ai risk: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("update element ai risk: %w", pgx.ErrNoRows)
	}
	return nil
}

// MarkConflict marks an element as having a primary/judge disagreement.
func (r *ElementRepository) MarkConflict(ctx context.Context, id uuid.UUID, isConflict bool) error {
	const q = `UPDATE content_elements SET is_conflict = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.Exec(ctx, q, isConflict, id)
	if err != nil {
		return fmt.Errorf("mark element conflict: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("mark element conflict: %w", pgx.ErrNoRows)
	}
	return nil
}

// CountByFilters returns counts for the review stats bar: pending_human, human_passed, human_rejected, conflict.
// Applies the same filters as FindByStatus (ai_status, human_status, elementKind, riskMin, riskMax).
func (r *ElementRepository) CountByFilters(ctx context.Context, aiStatus, humanStatus, elementKind string, riskMin, riskMax int) (struct {
	PendingHuman    int64
	HumanPassed     int64
	HumanRejected   int64
	Conflict        int64
}, error) {
	var result struct {
		PendingHuman    int64
		HumanPassed     int64
		HumanRejected   int64
		Conflict        int64
	}

	whereParts := []string{}
	var args []interface{}
	idx := 1

	if aiStatus != "" {
		whereParts = append(whereParts, fmt.Sprintf("ai_status = $%d", idx))
		args = append(args, aiStatus)
		idx++
	}
	if humanStatus != "" {
		whereParts = append(whereParts, fmt.Sprintf("human_status = $%d", idx))
		args = append(args, humanStatus)
		idx++
	}
	if elementKind != "" {
		whereParts = append(whereParts, fmt.Sprintf("element_kind = $%d", idx))
		args = append(args, elementKind)
		idx++
	}
	if riskMin > 0 {
		whereParts = append(whereParts, fmt.Sprintf("ai_risk_score >= $%d", idx))
		args = append(args, riskMin)
		idx++
	}
	if riskMax < 100 {
		whereParts = append(whereParts, fmt.Sprintf("ai_risk_score <= $%d", idx))
		args = append(args, riskMax)
		idx++
	}

	baseWhere := "1=1"
	if len(whereParts) > 0 {
		baseWhere = strings.Join(whereParts, " AND ")
	}

	// Count pending_human
	var pendingQ string
	if humanStatus == "" {
		pendingQ = fmt.Sprintf("SELECT COUNT(*) FROM content_elements WHERE %s AND human_status = 'pending_human'", baseWhere)
		if err := r.db.QueryRow(ctx, pendingQ).Scan(&result.PendingHuman); err != nil {
			return result, fmt.Errorf("count pending_human: %w", err)
		}
	}

	// Count human_passed
	passedQ := fmt.Sprintf("SELECT COUNT(*) FROM content_elements WHERE %s AND human_status = 'human_passed'", baseWhere)
	if err := r.db.QueryRow(ctx, passedQ).Scan(&result.HumanPassed); err != nil {
		return result, fmt.Errorf("count human_passed: %w", err)
	}

	// Count human_rejected
	rejectedQ := fmt.Sprintf("SELECT COUNT(*) FROM content_elements WHERE %s AND human_status = 'human_rejected'", baseWhere)
	if err := r.db.QueryRow(ctx, rejectedQ).Scan(&result.HumanRejected); err != nil {
		return result, fmt.Errorf("count human_rejected: %w", err)
	}

	// Count conflict
	conflictQ := fmt.Sprintf("SELECT COUNT(*) FROM content_elements WHERE %s AND is_conflict = true", baseWhere)
	if err := r.db.QueryRow(ctx, conflictQ).Scan(&result.Conflict); err != nil {
		return result, fmt.Errorf("count conflict: %w", err)
	}

	return result, nil
}

// CountByStatus returns the number of elements in each status bucket for dashboard aggregation.
// If tenantID is non-empty, only elements for that tenant are counted.
func (r *ElementRepository) CountByStatus(ctx context.Context, tenantID string) (map[string]int64, error) {
	var countQ string
	var countArgs []interface{}
	if tenantID != "" {
		countQ = `SELECT ce.ai_status, COUNT(*)::int FROM content_elements ce JOIN contents c ON c.id = ce.content_id WHERE c.tenant_id = $1 GROUP BY ce.ai_status`
		countArgs = []interface{}{tenantID}
	} else {
		countQ = "SELECT ai_status, COUNT(*)::int FROM content_elements GROUP BY ai_status"
	}

	rows, err := r.db.Query(ctx, countQ, countArgs...)
	if err != nil {
		return nil, fmt.Errorf("count elements by status: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var status string
		var cnt int64
		if err := rows.Scan(&status, &cnt); err != nil {
			return nil, fmt.Errorf("scan count by status: %w", err)
		}
		result[status] = cnt
	}
	return result, nil
}

// FindByContentAndKind returns the first element of a given kind for a content item.
func (r *ElementRepository) FindByContentAndKind(ctx context.Context, contentID uuid.UUID, kind model.ElementKind) (*model.ContentElement, error) {
	const q = `
		SELECT id, content_id, element_kind, element_content, ai_risk_score, ai_risk_types, ai_confidence, ai_status, human_status, is_conflict, created_at, updated_at
		FROM content_elements WHERE content_id = $1 AND element_kind = $2 ORDER BY created_at ASC LIMIT 1`

	e := &model.ContentElement{}
	err := r.db.QueryRow(ctx, q, contentID, kind).Scan(
		&e.ID, &e.ContentID, &e.ElementKind, &e.ElementContent, &e.AIRiskScore,
		&e.AIRiskTypes, &e.AIConfidence, &e.AIStatus, &e.HumanStatus, &e.IsConflict, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find element by content and kind: %w", err)
	}
	return e, nil
}

// FindByID loads a single content element by its UUID.
func (r *ElementRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.ContentElement, error) {
	const q = `
		SELECT id, content_id, element_kind, element_content, ai_risk_score, ai_risk_types, ai_confidence, ai_status, human_status, is_conflict, created_at, updated_at
		FROM content_elements WHERE id = $1`

	e := &model.ContentElement{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&e.ID, &e.ContentID, &e.ElementKind, &e.ElementContent, &e.AIRiskScore,
		&e.AIRiskTypes, &e.AIConfidence, &e.AIStatus, &e.HumanStatus, &e.IsConflict, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find element by id: %w", err)
	}
	return e, nil
}
