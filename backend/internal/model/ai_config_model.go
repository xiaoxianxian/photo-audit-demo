package model

import (
	"time"

	"github.com/google/uuid"
)

// AIConfig represents per-tenant AI model configuration.
type AIConfig struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	AgnesAPIKey     string    `json:"agnes_api_key"`
	AgnesEndpoint   string    `json:"agnes_endpoint"`
	AgnesConcurrency int      `json:"agnes_concurrency"`
	DeepSeekAPIKey  string    `json:"deepseek_api_key"`
	DeepSeekModel   string    `json:"deepseek_model"`
	FallbackEnabled bool      `json:"fallback_enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UpdateAIConfigRequest supports partial updates.
type UpdateAIConfigRequest struct {
	AgnesAPIKey      *string `json:"agnes_api_key,omitempty"`
	AgnesEndpoint    *string `json:"agnes_endpoint,omitempty"`
	AgnesConcurrency *int    `json:"agnes_concurrency,omitempty"`
	DeepSeekAPIKey   *string `json:"deepseek_api_key,omitempty"`
	DeepSeekModel    *string `json:"deepseek_model,omitempty"`
	FallbackEnabled  *bool   `json:"fallback_enabled,omitempty"`
}
