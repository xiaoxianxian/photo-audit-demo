package model

import (
	"time"

	"github.com/google/uuid"
)

// TenantCustomWord represents a tenant-specific sensitive word.
type TenantCustomWord struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Word     string    `json:"word"`
	Category *string   `json:"category,omitempty"`
	Status   int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateTenantCustomWordRequest is the payload for creating a word.
type CreateTenantCustomWordRequest struct {
	Word     string  `json:"word"`
	Category *string `json:"category,omitempty"`
}

// UpdateTenantCustomWordRequest supports partial updates.
type UpdateTenantCustomWordRequest struct {
	Word     *string `json:"word,omitempty"`
	Category *string `json:"category,omitempty"`
	Status   *int    `json:"status,omitempty"`
}
