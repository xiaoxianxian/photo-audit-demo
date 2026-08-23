package model

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents a tenant organization in the platform.
type Tenant struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	CountryCode string    `json:"country_code"`
	Status      int       `json:"status"` // 0=disabled, 1=active.
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

// CreateTenantRequest is the payload for creating a new tenant.
type CreateTenantRequest struct {
	Name        string `json:"name"`
	CountryCode string `json:"country_code"`
}

// UpdateTenantRequest supports partial updates via pointer fields.
// Only non-nil fields are applied.
type UpdateTenantRequest struct {
	Name        *string `json:"name,omitempty"`
	CountryCode *string `json:"country_code,omitempty"`
	Status      *int    `json:"status,omitempty"`
}
