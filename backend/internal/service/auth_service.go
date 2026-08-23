package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrInactiveUser    = errors.New("user account is disabled")
)

// AuthService handles authentication and authorization logic.
type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// Login authenticates a user by username and password, returning a JWT token and user info.
func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, errors.New("username is required")
	}

	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		// The repo wraps pgx.ErrNoRows — unwrap it.
		if unwrapped := errors.Unwrap(err); unwrapped != nil {
			if errors.Is(unwrapped, ErrInvalidPassword) {
				return nil, ErrInvalidPassword
			}
		}
		return nil, fmt.Errorf("login: %w", err)
	}

	if err := user.CheckPassword(req.Password); err != nil {
		return nil, ErrInvalidPassword
	}

	if user.Status != 1 {
		return nil, ErrInactiveUser
	}

	token, err := GenerateToken(user.ID, user.Role, user.TenantID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Strip password hash from response.
	respUser := *user
	respUser.PasswordHashBcrypt = ""

	return &model.LoginResponse{
		Token: token,
		User:  respUser,
	}, nil
}

// Register creates a new user account.
func (s *AuthService) Register(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, errors.New("username is required")
	}
	if req.Password == "" {
		return nil, errors.New("password is required")
	}
	if !ValidRoles[strings.ToLower(req.Role)] {
		return nil, fmt.Errorf("invalid role: %s", req.Role)
	}

	// Check username uniqueness within the same tenant.
	if req.TenantID != nil && req.TenantID.String() != uuid.Nil.String() {
		existing, err := s.userRepo.FindByUsernameAndTenant(ctx, username, *req.TenantID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("register: check uniqueness: %w", err)
		}
		if existing != nil {
			return nil, repository.ErrDuplicateUser
		}
	} else {
		// Platform admin: check global uniqueness.
		existing, err := s.userRepo.FindByUsername(ctx, username)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("register: check uniqueness: %w", err)
		}
		if existing != nil {
			return nil, repository.ErrDuplicateUser
		}
	}

	user := &model.User{
		ID:          uuid.New(),
		Username:    username,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Role:        strings.ToLower(req.Role),
		Email:       req.Email,
		Phone:       req.Phone,
		Languages:   req.Languages,
		Status:      1,
	}

	if req.TenantID != nil {
		trimmed := strings.TrimSpace(strings.ToLower(req.TenantID.String()))
		if trimmed == "" {
			user.TenantID = nil
		} else {
			parsed, perr := uuid.Parse(trimmed)
			if perr != nil {
				return nil, fmt.Errorf("register: invalid tenant_id: %w", perr)
			}
			user.TenantID = &parsed
		}
	}

	if err := user.SetPasswordHash(req.Password); err != nil {
		return nil, fmt.Errorf("register: hash password: %w", err)
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	// Clear password hash before returning.
	user.PasswordHashBcrypt = ""
	return user, nil
}

// ValidateToken parses and validates a JWT token string.
func (s *AuthService) ValidateToken(tokenString string) (*model.JWTClaims, error) {
	return ParseClaims(tokenString, []byte(s.jwtSecret))
}
