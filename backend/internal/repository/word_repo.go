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

// TenantWordRepository provides data access for tenant_custom_words.
type TenantWordRepository struct {
	db *pgxpool.Pool
}

func NewTenantWordRepository(db *pgxpool.Pool) *TenantWordRepository {
	return &TenantWordRepository{db: db}
}

// Create inserts a new custom word.
func (r *TenantWordRepository) Create(ctx context.Context, word *model.TenantCustomWord) error {
	query := `
		INSERT INTO tenant_custom_words
			(id, tenant_id, word, category, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`
	var createdAt interface{}
	err := r.db.QueryRow(ctx, query,
		word.ID, word.TenantID, word.Word, word.Category, word.Status,
		time.Now(),
	).Scan(&createdAt)
	if err != nil {
		return fmt.Errorf("create tenant custom word: %w", err)
	}
	return nil
}

// FindByID retrieves a word by UUID.
func (r *TenantWordRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.TenantCustomWord, error) {
	var w model.TenantCustomWord
	query := `
		SELECT id, tenant_id, word, category, status, created_at
		FROM tenant_custom_words WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&w.ID, &w.TenantID, &w.Word, &w.Category, &w.Status, &w.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find word by id '%s': %w", id.String(), pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("find word by id '%s': %w", id.String(), err)
	}
	return &w, nil
}

// ListByTenant returns paginated words for a tenant.
func (r *TenantWordRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]model.TenantCustomWord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var words []model.TenantCustomWord

	var total int64
	countQuery := `SELECT COUNT(*) FROM tenant_custom_words WHERE tenant_id = $1 AND status = 1`
	if err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tenant custom words: %w", err)
	}

	query := `
		SELECT id, tenant_id, word, category, status, created_at
		FROM tenant_custom_words
		WHERE tenant_id = $1 AND status = 1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, tenantID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenant custom words: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var w model.TenantCustomWord
		if err := rows.Scan(
			&w.ID, &w.TenantID, &w.Word, &w.Category, &w.Status, &w.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan word row: %w", err)
		}
		words = append(words, w)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate word rows: %w", err)
	}
	return words, total, nil
}

// Update performs a partial update.
func (r *TenantWordRepository) Update(ctx context.Context, id uuid.UUID, req *model.UpdateTenantCustomWordRequest) error {
	sets := make([]string, 0, 3)
	args := []interface{}{}
	argIdx := 1

	if req.Word != nil {
		sets = append(sets, fmt.Sprintf("word = $%d", argIdx))
		args = append(args, *req.Word)
		argIdx++
	}
	if req.Category != nil {
		sets = append(sets, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *req.Category)
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

	query := fmt.Sprintf("UPDATE tenant_custom_words SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	args = append(args, id)
	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update tenant custom word '%s': %w", id.String(), err)
	}
	return nil
}

// Delete performs a soft delete.
func (r *TenantWordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenant_custom_words SET status = 0 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tenant custom word (soft) '%s': %w", id.String(), err)
	}
	return nil
}
