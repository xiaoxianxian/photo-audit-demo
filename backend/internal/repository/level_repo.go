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

// TenantLevelRepository provides data access for tenant_audit_levels.
type TenantLevelRepository struct {
	db *pgxpool.Pool
}

func NewTenantLevelRepository(db *pgxpool.Pool) *TenantLevelRepository {
	return &TenantLevelRepository{db: db}
}

// Create inserts a new penalty level.
func (r *TenantLevelRepository) Create(ctx context.Context, level *model.TenantAuditLevel) error {
	query := `
		INSERT INTO tenant_audit_levels
			(id, tenant_id, level_code, level_name, description, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`
	var createdAt interface{}
	err := r.db.QueryRow(ctx, query,
		level.ID, level.TenantID, level.LevelCode, level.LevelName,
		level.Description, level.Status,
		time.Now(),
	).Scan(&createdAt)
	if err != nil {
		return fmt.Errorf("create tenant audit level: %w", err)
	}
	return nil
}

// FindByID retrieves a level by UUID.
func (r *TenantLevelRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.TenantAuditLevel, error) {
	var level model.TenantAuditLevel
	query := `
		SELECT id, tenant_id, level_code, level_name, description, status, created_at
		FROM tenant_audit_levels WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&level.ID, &level.TenantID, &level.LevelCode, &level.LevelName,
		&level.Description, &level.Status, &level.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find level by id '%s': %w", id.String(), pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("find level by id '%s': %w", id.String(), err)
	}
	return &level, nil
}

// ListByTenant returns paginated levels for a tenant.
func (r *TenantLevelRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]model.TenantAuditLevel, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var levels []model.TenantAuditLevel

	var total int64
	countQuery := `SELECT COUNT(*) FROM tenant_audit_levels WHERE tenant_id = $1 AND status = 1`
	if err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tenant audit levels: %w", err)
	}

	query := `
		SELECT id, tenant_id, level_code, level_name, description, status, created_at
		FROM tenant_audit_levels
		WHERE tenant_id = $1 AND status = 1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, tenantID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenant audit levels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var level model.TenantAuditLevel
		if err := rows.Scan(
			&level.ID, &level.TenantID, &level.LevelCode, &level.LevelName,
			&level.Description, &level.Status, &level.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan level row: %w", err)
		}
		levels = append(levels, level)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate level rows: %w", err)
	}
	return levels, total, nil
}

// Update performs a partial update.
func (r *TenantLevelRepository) Update(ctx context.Context, id uuid.UUID, req *model.UpdateTenantAuditLevelRequest) error {
	sets := make([]string, 0, 4)
	args := []interface{}{}
	argIdx := 1

	if req.LevelCode != nil {
		sets = append(sets, fmt.Sprintf("level_code = $%d", argIdx))
		args = append(args, *req.LevelCode)
		argIdx++
	}
	if req.LevelName != nil {
		sets = append(sets, fmt.Sprintf("level_name = $%d", argIdx))
		args = append(args, *req.LevelName)
		argIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
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

	query := fmt.Sprintf("UPDATE tenant_audit_levels SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	args = append(args, id)
	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update tenant audit level '%s': %w", id.String(), err)
	}
	return nil
}

// Delete performs a soft delete.
func (r *TenantLevelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenant_audit_levels SET status = 0 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tenant audit level (soft) '%s': %w", id.String(), err)
	}
	return nil
}
