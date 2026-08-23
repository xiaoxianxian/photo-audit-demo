package model

import (
	"time"

	"github.com/google/uuid"
)

// TenantAuditLevel represents a penalty level config per tenant.
type TenantAuditLevel struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	LevelCode   string    `json:"level_code"`
	LevelName   string    `json:"level_name"`
	Description *string   `json:"description,omitempty"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateTenantAuditLevelRequest is the payload for creating a level.
type CreateTenantAuditLevelRequest struct {
	LevelCode   string  `json:"level_code"`
	LevelName   string  `json:"level_name"`
	Description *string `json:"description,omitempty"`
}

// UpdateTenantAuditLevelRequest supports partial updates.
type UpdateTenantAuditLevelRequest struct {
	LevelCode   *string `json:"level_code,omitempty"`
	LevelName   *string `json:"level_name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *int    `json:"status,omitempty"`
}
