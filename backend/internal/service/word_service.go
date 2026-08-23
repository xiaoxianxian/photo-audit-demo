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

var ErrWordNotFound = errors.New("custom word not found")

// TenantWordService handles business logic for tenant custom words.
type TenantWordService struct {
	wordRepo *repository.TenantWordRepository
}

func NewTenantWordService(wordRepo *repository.TenantWordRepository) *TenantWordService {
	return &TenantWordService{wordRepo: wordRepo}
}

// Create validates and creates a new custom word.
func (s *TenantWordService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateTenantCustomWordRequest) (*model.TenantCustomWord, error) {
	word := strings.TrimSpace(req.Word)
	if word == "" {
		return nil, errors.New("word is required")
	}

	w := &model.TenantCustomWord{
		ID:       uuid.New(),
		TenantID: tenantID,
		Word:     word,
		Category: req.Category,
		Status:   1,
	}

	if err := s.wordRepo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("create word: %w", err)
	}
	return w, nil
}

// GetByID retrieves a word by UUID.
func (s *TenantWordService) GetByID(ctx context.Context, id uuid.UUID) (*model.TenantCustomWord, error) {
	w, err := s.wordRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWordNotFound
		}
		return nil, fmt.Errorf("get word: %w", err)
	}
	return w, nil
}

// List returns paginated words for a tenant.
func (s *TenantWordService) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]model.TenantCustomWord, int64, error) {
	return s.wordRepo.ListByTenant(ctx, tenantID, page, pageSize)
}

// Update applies a partial update.
func (s *TenantWordService) Update(ctx context.Context, id uuid.UUID, req model.UpdateTenantCustomWordRequest) (*model.TenantCustomWord, error) {
	if req.Word != nil {
		*req.Word = strings.TrimSpace(*req.Word)
	}
	if err := s.wordRepo.Update(ctx, id, &req); err != nil {
		return nil, fmt.Errorf("update word: %w", err)
	}
	return s.wordRepo.FindByID(ctx, id)
}

// Delete performs a soft delete.
func (s *TenantWordService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.wordRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWordNotFound
		}
		return fmt.Errorf("delete word: %w", err)
	}
	return s.wordRepo.Delete(ctx, id)
}
