package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims extends jwt.RegisteredClaims with tenant-specific identity information.
// Shared between service and middleware packages to avoid duplication.
type JWTClaims struct {
	UserID   uuid.UUID  `json:"user_id"`
	Role     string     `json:"role"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
	jwt.RegisteredClaims
}

// NewJWTClaims creates a JWTClaims with the given user ID, role, and optional tenant ID.
// The token expires after expiry duration.
func NewJWTClaims(userID uuid.UUID, role string, tenantID *uuid.UUID, expiry time.Duration) *JWTClaims {
	now := time.Now()
	return &JWTClaims{
		UserID: userID,
		Role:   role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "photo-audit-platform",
		},
	}
}
