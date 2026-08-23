package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"audit-platform/internal/model"
)

// TenantRepository provides data access for Tenant entities.
type TenantRepository struct {
	db *pgxpool.Pool
}

// NewTenantRepository creates a new TenantRepository backed by the given pool.
func NewTenantRepository(db *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{db: db}
}

// Create inserts a new tenant into the database.
func (r *TenantRepository) Create(ctx context.Context, t *model.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, country_code, status, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		t.ID,
		t.Name,
		t.CountryCode,
		t.Status,
		time.Now(),
		t.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

// FindByID retrieves a tenant by its UUID.
// Returns pgx.ErrNoRows if not found.
func (r *TenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	var t model.Tenant
	query := `
		SELECT id, name, country_code, status, created_at, created_by
		FROM tenants
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID,
		&t.Name,
		&t.CountryCode,
		&t.Status,
		&t.CreatedAt,
		&t.CreatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find tenant by id '%s': %w", id.String(), pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("find tenant by id '%s': %w", id.String(), err)
	}
	return &t, nil
}

// List returns a paginated list of tenants. Returns (tenants, total_count, error).
func (r *TenantRepository) List(ctx context.Context, page, pageSize int) ([]model.Tenant, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var tenants []model.Tenant

	// Count total.
	var total int64
	countQuery := `SELECT COUNT(*) FROM tenants WHERE status != 0`
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	// Fetch page.
	query := `
		SELECT id, name, country_code, status, created_at, created_by
		FROM tenants WHERE status != 0
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.CountryCode,
			&t.Status,
			&t.CreatedAt,
			&t.CreatedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("scan tenant row: %w", err)
		}
		tenants = append(tenants, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate tenant rows: %w", err)
	}

	return tenants, total, nil
}

// Update performs a partial update of a tenant using dynamic SQL based on
// which fields in the request are non-nil.
func (r *TenantRepository) Update(ctx context.Context, id uuid.UUID, req *model.UpdateTenantRequest) error {
	sets := make([]string, 0, 3)
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.CountryCode != nil {
		sets = append(sets, fmt.Sprintf("country_code = $%d", argIdx))
		args = append(args, *req.CountryCode)
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}

	if len(sets) == 0 {
		return nil // Nothing to update.
	}

	query := fmt.Sprintf("UPDATE tenants SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update tenant '%s': %w", id.String(), err)
	}
	return nil
}

// Delete performs a soft delete by setting status to 0.
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET status = 0 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tenant (soft) '%s': %w", id.String(), err)
	}
	return nil
}
