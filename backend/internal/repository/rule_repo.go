package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"audit-platform/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantRuleRepository provides data access for tenant_audit_rules.
type TenantRuleRepository struct {
	db *pgxpool.Pool
}

func NewTenantRuleRepository(db *pgxpool.Pool) *TenantRuleRepository {
	return &TenantRuleRepository{db: db}
}

// Create inserts a new audit rule.
func (r *TenantRuleRepository) Create(ctx context.Context, rule *model.TenantAuditRule) error {
	query := `
		INSERT INTO tenant_audit_rules
			(id, tenant_id, rule_name, rule_expression, action, priority, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`
	var createdAt interface{}
	err := r.db.QueryRow(ctx, query,
		rule.ID, rule.TenantID, rule.RuleName,
		rule.RuleExpression, rule.Action, rule.Priority, rule.Status,
		time.Now(),
	).Scan(&createdAt)
	if err != nil {
		return fmt.Errorf("create tenant audit rule: %w", err)
	}
	return nil
}

// FindByID retrieves a rule by UUID.
func (r *TenantRuleRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.TenantAuditRule, error) {
	var rule model.TenantAuditRule
	query := `
		SELECT id, tenant_id, rule_name, rule_expression, action, priority, status, created_at
		FROM tenant_audit_rules WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rule.ID, &rule.TenantID, &rule.RuleName,
		&rule.RuleExpression, &rule.Action, &rule.Priority, &rule.Status,
		&rule.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find rule by id '%s': %w", id.String(), pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("find rule by id '%s': %w", id.String(), err)
	}
	return &rule, nil
}

// ListByTenant returns paginated rules for a tenant.
func (r *TenantRuleRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]model.TenantAuditRule, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var rules []model.TenantAuditRule

	var total int64
	countQuery := `SELECT COUNT(*) FROM tenant_audit_rules WHERE tenant_id = $1 AND status = 1`
	if err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tenant audit rules: %w", err)
	}

	query := `
		SELECT id, tenant_id, rule_name, rule_expression, action, priority, status, created_at
		FROM tenant_audit_rules
		WHERE tenant_id = $1 AND status = 1
		ORDER BY priority ASC, created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, tenantID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenant audit rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rule model.TenantAuditRule
		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.RuleName,
			&rule.RuleExpression, &rule.Action, &rule.Priority, &rule.Status,
			&rule.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan rule row: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate rule rows: %w", err)
	}
	return rules, total, nil
}

// Update performs a partial update.
func (r *TenantRuleRepository) Update(ctx context.Context, id uuid.UUID, req *model.UpdateTenantAuditRuleRequest) error {
	sets := make([]string, 0, 5)
	args := []interface{}{}
	argIdx := 1

	if req.RuleName != nil {
		sets = append(sets, fmt.Sprintf("rule_name = $%d", argIdx))
		args = append(args, *req.RuleName)
		argIdx++
	}
	if req.RuleExpression != nil {
		sets = append(sets, fmt.Sprintf("rule_expression = $%d", argIdx))
		args = append(args, *req.RuleExpression)
		argIdx++
	}
	if req.Action != nil {
		sets = append(sets, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, *req.Action)
		argIdx++
	}
	if req.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE tenant_audit_rules SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	args = append(args, id)
	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update tenant audit rule '%s': %w", id.String(), err)
	}
	return nil
}

// Delete performs a soft delete (sets status to 0).
func (r *TenantRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenant_audit_rules SET status = 0 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tenant audit rule (soft) '%s': %w", id.String(), err)
	}
	return nil
}
