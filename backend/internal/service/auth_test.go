package service

import (
	"testing"
	"time"

	"audit-platform/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestGenerateToken verifies JWT generation and parsing roundtrip.
func TestGenerateToken(t *testing.T) {
	signingKey := "test-secret-key-for-unit-tests"
	userID := uuid.New()
	tenantID := uuid.New()

	tokenStr, err := GenerateToken(userID, "reviewer", &tenantID, signingKey)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)

	// Parse and validate
	claims, err := ParseClaims(tokenStr, []byte(signingKey))
	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, "reviewer", claims.Role)
	require.NotNil(t, claims.TenantID)
	require.Equal(t, tenantID, *claims.TenantID)
}

// TestGenerateToken_PlatformAdmin verifies token with nil tenant_id.
func TestGenerateToken_PlatformAdmin(t *testing.T) {
	signingKey := "test-secret-key"
	userID := uuid.New()

	tokenStr, err := GenerateToken(userID, "platform_admin", nil, signingKey)
	require.NoError(t, err)

	claims, err := ParseClaims(tokenStr, []byte(signingKey))
	require.NoError(t, err)
	require.Nil(t, claims.TenantID)
}

// TestParseClaims_InvalidToken verifies invalid token returns error.
func TestParseClaims_InvalidToken(t *testing.T) {
	_, err := ParseClaims("invalid-token-string", []byte("secret"))
	require.Error(t, err)
}

// TestParseClaims_WrongSecret verifies wrong signing key fails.
func TestParseClaims_WrongSecret(t *testing.T) {
	signingKey := "correct-secret"
	userID := uuid.New()

	tokenStr, _ := GenerateToken(userID, "reviewer", nil, signingKey)
	_, err := ParseClaims(tokenStr, []byte("wrong-secret"))
	require.Error(t, err)
}

// TestParseClaims_ExpiredToken verifies expired token returns error.
func TestParseClaims_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	claims := &model.JWTClaims{
		UserID: userID,
		Role:   "reviewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte("secret"))
	_, err := ParseClaims(tokenStr, []byte("secret"))
	require.Error(t, err)
}

// TestValidRoles verifies all valid roles are recognized.
func TestValidRoles(t *testing.T) {
	require.True(t, ValidRoles["platform_admin"])
	require.True(t, ValidRoles["tenant_admin"])
	require.True(t, ValidRoles["reviewer"])
	require.True(t, ValidRoles["quality_checker"])
	require.False(t, ValidRoles["invalid_role"])
}

// TestValidMemberRoles verifies team member role whitelist.
func TestValidMemberRoles(t *testing.T) {
	require.True(t, ValidMemberRoles["reviewer"])
	require.True(t, ValidMemberRoles["senior_reviewer"])
	require.True(t, ValidMemberRoles["quality_checker"])
	require.False(t, ValidMemberRoles["platform_admin"])
}

// TestJWTClaims_Structure verifies JWT claims fields.
func TestJWTClaims_Structure(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	claims := &model.JWTClaims{
		UserID:   userID,
		Role:     "reviewer",
		TenantID: &tenantID,
	}

	require.Equal(t, userID, claims.UserID)
	require.Equal(t, "reviewer", claims.Role)
	require.NotNil(t, claims.TenantID)
}

// TestGenerateToken_Expiry verifies token expires in 24 hours.
func TestGenerateToken_Expiry(t *testing.T) {
	signingKey := "test-secret"
	userID := uuid.New()

	tokenStr, _ := GenerateToken(userID, "reviewer", nil, signingKey)
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)
	require.NotZero(t, claims.ExpiresAt)
	require.WithinDuration(t, time.Now().Add(24*time.Hour), claims.ExpiresAt.Time, 5*time.Second)
}
