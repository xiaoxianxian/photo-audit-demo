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

var ErrLevelNotFound = errors.New("audit level not found")

// TenantLevelService handles business logic for tenant audit levels.
type TenantLevelService struct {
	levelRepo *repository.TenantLevelRepository
}

func NewTenantLevelService(levelRepo *repository.TenantLevelRepository) *TenantLevelService {
	return &TenantLevelService{levelRepo: levelRepo}
}

// Create validates and creates a new penalty level.
func (s *TenantLevelService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateTenantAuditLevelRequest) (*model.TenantAuditLevel, error) {
	code := strings.TrimSpace(req.LevelCode)
	if code == "" {
		return nil, errors.New("level code is required")
	}
	name := strings.TrimSpace(req.LevelName)
	if name == "" {
		return nil, errors.New("level name is required")
	}

	level := &model.TenantAuditLevel{
		ID:          uuid.New(),
		TenantID:    tenantID,
		LevelCode:   strings.ToLower(code),
		LevelName:   name,
		Description: req.Description,
		Status:      1,
	}

	if err := s.levelRepo.Create(ctx, level); err != nil {
		return nil, fmt.Errorf("create level: %w", err)
	}
	return level, nil
}

// GetByID retrieves a level by UUID.
func (s *TenantLevelService) GetByID(ctx context.Context, id uuid.UUID) (*model.TenantAuditLevel, error) {
	level, err := s.levelRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLevelNotFound
		}
		return nil, fmt.Errorf("get level: %w", err)
	}
	return level, nil
}

// List returns paginated levels for a tenant.
func (s *TenantLevelService) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]model.TenantAuditLevel, int64, error) {
	return s.levelRepo.ListByTenant(ctx, tenantID, page, pageSize)
}

// Update applies a partial update.
func (s *TenantLevelService) Update(ctx context.Context, id uuid.UUID, req model.UpdateTenantAuditLevelRequest) (*model.TenantAuditLevel, error) {
	if req.LevelCode != nil {
		*req.LevelCode = strings.TrimSpace(*req.LevelCode)
	}
	if req.LevelName != nil {
		*req.LevelName = strings.TrimSpace(*req.LevelName)
	}
	if err := s.levelRepo.Update(ctx, id, &req); err != nil {
		return nil, fmt.Errorf("update level: %w", err)
	}
	return s.levelRepo.FindByID(ctx, id)
}

// Delete performs a soft delete.
func (s *TenantLevelService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.levelRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLevelNotFound
		}
		return fmt.Errorf("delete level: %w", err)
	}
	return s.levelRepo.Delete(ctx, id)
}
