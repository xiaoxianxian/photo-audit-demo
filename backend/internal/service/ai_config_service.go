package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrAIConfigNotFound = errors.New("AI config not found")
)

// UpdateAIConfigRequest is an alias to model.UpdateAIConfigRequest for convenience.
type UpdateAIConfigRequest = model.UpdateAIConfigRequest

// AIConfigService handles business logic for tenant AI model configuration.
type AIConfigService struct {
	configRepo *repository.AIConfigRepository
}

func NewAIConfigService(configRepo *repository.AIConfigRepository) *AIConfigService {
	return &AIConfigService{configRepo: configRepo}
}

// Get retrieves the AI config for the given tenant.
func (s *AIConfigService) Get(ctx context.Context, tenantID uuid.UUID) (*model.AIConfig, error) {
	return s.configRepo.GetByTenant(ctx, tenantID)
}

// Save creates or updates the AI config for a tenant.
func (s *AIConfigService) Save(ctx context.Context, tenantID uuid.UUID, req model.UpdateAIConfigRequest) (*model.AIConfig, error) {
	// Validate endpoint if provided.
	if req.AgnesEndpoint != nil {
		endpoint := strings.TrimSpace(*req.AgnesEndpoint)
		if endpoint != "" {
			if _, err := url.ParseRequestURI(endpoint); err != nil {
				return nil, fmt.Errorf("invalid agnes endpoint: %w", err)
			}
		}
	}

	// Validate concurrency.
	if req.AgnesConcurrency != nil {
		c := *req.AgnesConcurrency
		if c < 1 || c > 100 {
			return nil, fmt.Errorf("agnes concurrency must be between 1 and 100")
		}
	}

	// Validate deepseek model.
	if req.DeepSeekModel != nil {
		model := strings.TrimSpace(*req.DeepSeekModel)
		if model != "" {
			validModels := map[string]bool{
				"deepseek-chat": true, "deepseek-coder": true,
				"deepseek-v3": true, "deepseek-r1": true,
			}
			if !validModels[model] {
				return nil, fmt.Errorf("unsupported deepseek model: %s", model)
			}
		}
	}

	// Try to get existing config first.
	_, err := s.configRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create new config.
			cfg := &model.AIConfig{
				ID:             uuid.New(),
				TenantID:       tenantID,
				AgnesAPIKey:    "",
				AgnesEndpoint:  "https://api.agnes.ai/v1/review",
				AgnesConcurrency: 10,
				DeepSeekAPIKey: "",
				DeepSeekModel:  "deepseek-chat",
				FallbackEnabled: true,
			}
			if req.AgnesAPIKey != nil {
				cfg.AgnesAPIKey = *req.AgnesAPIKey
			}
			if req.AgnesEndpoint != nil {
				cfg.AgnesEndpoint = *req.AgnesEndpoint
			}
			if req.AgnesConcurrency != nil {
				cfg.AgnesConcurrency = *req.AgnesConcurrency
			}
			if req.DeepSeekAPIKey != nil {
				cfg.DeepSeekAPIKey = *req.DeepSeekAPIKey
			}
			if req.DeepSeekModel != nil {
				cfg.DeepSeekModel = *req.DeepSeekModel
			}
			if req.FallbackEnabled != nil {
				cfg.FallbackEnabled = *req.FallbackEnabled
			}
			if err := s.configRepo.Upsert(ctx, cfg); err != nil {
				return nil, fmt.Errorf("create ai config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("get ai config: %w", err)
	}

	// Update existing.
	if err := s.configRepo.UpdatePartial(ctx, tenantID, &req); err != nil {
		return nil, fmt.Errorf("update ai config: %w", err)
	}
	return s.configRepo.GetByTenant(ctx, tenantID)
}
