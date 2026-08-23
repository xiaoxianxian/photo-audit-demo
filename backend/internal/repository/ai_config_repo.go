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

// AIConfigRepository provides data access for ai_configs.
type AIConfigRepository struct {
	db *pgxpool.Pool
}

func NewAIConfigRepository(db *pgxpool.Pool) *AIConfigRepository {
	return &AIConfigRepository{db: db}
}

// Upsert inserts or updates the AI config for a tenant.
func (r *AIConfigRepository) Upsert(ctx context.Context, cfg *model.AIConfig) error {
	query := `
		INSERT INTO ai_configs
			(id, tenant_id, agnes_api_key, agnes_endpoint, agnes_concurrency,
			 deepseek_api_key, deepseek_model, fallback_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_id) DO UPDATE SET
			agnes_api_key = EXCLUDED.agnes_api_key,
			agnes_endpoint = EXCLUDED.agnes_endpoint,
			agnes_concurrency = EXCLUDED.agnes_concurrency,
			deepseek_api_key = EXCLUDED.deepseek_api_key,
			deepseek_model = EXCLUDED.deepseek_model,
			fallback_enabled = EXCLUDED.fallback_enabled,
			updated_at = NOW()
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, query,
		cfg.ID, cfg.TenantID, cfg.AgnesAPIKey, cfg.AgnesEndpoint,
		cfg.AgnesConcurrency, cfg.DeepSeekAPIKey, cfg.DeepSeekModel,
		cfg.FallbackEnabled, time.Now(), time.Now(),
	).Scan(&cfg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert ai config: %w", err)
	}
	return nil
}

// GetByTenant retrieves the AI config for a tenant.
func (r *AIConfigRepository) GetByTenant(ctx context.Context, tenantID uuid.UUID) (*model.AIConfig, error) {
	var cfg model.AIConfig
	query := `
		SELECT id, tenant_id, agnes_api_key, agnes_endpoint, agnes_concurrency,
		       deepseek_api_key, deepseek_model, fallback_enabled, created_at, updated_at
		FROM ai_configs WHERE tenant_id = $1`
	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.AgnesAPIKey, &cfg.AgnesEndpoint,
		&cfg.AgnesConcurrency, &cfg.DeepSeekAPIKey, &cfg.DeepSeekModel,
		&cfg.FallbackEnabled, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("get ai config for tenant '%s': %w", tenantID.String(), pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("get ai config for tenant '%s': %w", tenantID.String(), err)
	}
	return &cfg, nil
}

// UpdatePartial applies a partial update.
func (r *AIConfigRepository) UpdatePartial(ctx context.Context, tenantID uuid.UUID, req *model.UpdateAIConfigRequest) error {
	sets := make([]string, 0, 6)
	args := []interface{}{}
	argIdx := 1

	if req.AgnesAPIKey != nil {
		sets = append(sets, fmt.Sprintf("agnes_api_key = $%d", argIdx))
		args = append(args, *req.AgnesAPIKey)
		argIdx++
	}
	if req.AgnesEndpoint != nil {
		sets = append(sets, fmt.Sprintf("agnes_endpoint = $%d", argIdx))
		args = append(args, *req.AgnesEndpoint)
		argIdx++
	}
	if req.AgnesConcurrency != nil {
		sets = append(sets, fmt.Sprintf("agnes_concurrency = $%d", argIdx))
		args = append(args, *req.AgnesConcurrency)
		argIdx++
	}
	if req.DeepSeekAPIKey != nil {
		sets = append(sets, fmt.Sprintf("deepseek_api_key = $%d", argIdx))
		args = append(args, *req.DeepSeekAPIKey)
		argIdx++
	}
	if req.DeepSeekModel != nil {
		sets = append(sets, fmt.Sprintf("deepseek_model = $%d", argIdx))
		args = append(args, *req.DeepSeekModel)
		argIdx++
	}
	if req.FallbackEnabled != nil {
		sets = append(sets, fmt.Sprintf("fallback_enabled = $%d", argIdx))
		args = append(args, *req.FallbackEnabled)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, tenantID)

	query := fmt.Sprintf("UPDATE ai_configs SET %s WHERE tenant_id = $%d", strings.Join(sets, ", "), argIdx)
	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update ai config: %w", err)
	}
	return nil
}
