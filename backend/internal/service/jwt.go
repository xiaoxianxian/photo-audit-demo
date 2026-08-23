package service

import (
	"errors"
	"time"

	"audit-platform/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

// ParseClaims validates and parses a JWT string into Claims.
func ParseClaims(tokenString string, signingKey []byte) (*model.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return signingKey, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*model.JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateToken creates a signed JWT for the given user identity.
// The token expires 24 hours after issuance.
func GenerateToken(userID uuid.UUID, role string, tenantID *uuid.UUID, signingKey string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	claims := &model.JWTClaims{
		UserID:  userID,
		Role:    role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "audit-platform",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(signingKey))
}

// ValidRoles contains roles accepted during user registration.
var ValidRoles = map[string]bool{
	"platform_admin":  true,
	"tenant_admin":    true,
	"reviewer":        true,
	"quality_checker": true,
}

// ValidMemberRoles contains valid roles for team membership.
var ValidMemberRoles = map[string]bool{
	"reviewer":        true,
	"senior_reviewer": true,
	"quality_checker": true,
}
