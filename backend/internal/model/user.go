package model

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the photo audit platform.
// Role values: platform_admin, tenant_admin, reviewer, quality_checker.
type User struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         *uuid.UUID `json:"tenant_id,omitempty"` // NULL for platform admins.
	Username         string     `json:"username"`
	DisplayName      string     `json:"display_name"`
	PasswordHashBcrypt string   `json:"-"` // Never serialized.
	Role             string     `json:"role"`
	Email            string     `json:"email"`
	Phone            string     `json:"phone"`
	Languages        []string   `json:"languages"`
	Status           int        `json:"status"` // 0=disabled, 1=active.
	CreatedAt        time.Time  `json:"created_at"`
}

// SetPasswordHash generates a bcrypt hash from the given plaintext password
// using the provided cost (use bcrypt.DefaultCost if unsure).
func (u *User) SetPasswordHashWithCost(password string, cost int) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return err
	}
	u.PasswordHashBcrypt = string(hash)
	return nil
}

// SetPasswordHash generates a bcrypt hash from the given plaintext password
// using bcrypt.DefaultCost.
func (u *User) SetPasswordHash(password string) error {
	return u.SetPasswordHashWithCost(password, bcrypt.DefaultCost)
}

// CheckPassword compares the given plaintext password against the stored hash.
// Returns nil if they match, bcrypt.ErrMismatchedHash otherwise.
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHashBcrypt), []byte(password))
}

// CreateUserRequest is the payload for creating a new user.
type CreateUserRequest struct {
	Username    string     `json:"username"`
	Password    string     `json:"password"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Languages   []string   `json:"languages"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty"`
}

// LoginRequest is the payload for authenticating a user.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse contains the JWT token and the authenticated user details.
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
