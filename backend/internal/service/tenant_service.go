package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrInvalidCountry = errors.New("invalid country code")
)

// TenantService handles business logic for tenant management.
type TenantService struct {
	tenantRepo *repository.TenantRepository
	userRepo   *repository.UserRepository
}

// NewTenantService creates a new TenantService.
func NewTenantService(tenantRepo *repository.TenantRepository, userRepo *repository.UserRepository) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
	}
}

// Create validates and creates a new tenant.
func (s *TenantService) Create(ctx context.Context, req model.CreateTenantRequest, createdBy uuid.UUID) (*model.Tenant, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("tenant name is required")
	}

	countryCode := strings.TrimSpace(req.CountryCode)
	if countryCode == "" {
		return nil, errors.New("country code is required")
	}
	if !isValidISO3166Alpha2(countryCode) {
		return nil, ErrInvalidCountry
	}

	tenant := &model.Tenant{
		ID:          uuid.New(),
		Name:        name,
		CountryCode: strings.ToUpper(countryCode),
		Status:      1,
		CreatedBy:   createdBy,
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	// In production, log creation in audit_logs table here.

	result := *tenant
	result.CreatedBy = uuid.Nil
	return &result, nil
}

// GetByID retrieves a tenant by its UUID.
func (s *TenantService) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	result := *tenant
	result.CreatedBy = uuid.Nil
	return &result, nil
}

// List returns a paginated list of tenants.
func (s *TenantService) List(ctx context.Context, page, pageSize int) ([]model.Tenant, int64, error) {
	tenants, total, err := s.tenantRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}

	for i := range tenants {
		tenants[i].CreatedBy = uuid.Nil
	}

	return tenants, total, nil
}

// Update applies a partial update to an existing tenant.
func (s *TenantService) Update(ctx context.Context, id uuid.UUID, req model.UpdateTenantRequest) (*model.Tenant, error) {
	_, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	// Validate fields that are being changed.
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, errors.New("tenant name cannot be empty")
		}
	}

	if req.CountryCode != nil {
		cc := strings.TrimSpace(*req.CountryCode)
		if !isValidISO3166Alpha2(cc) {
			return nil, ErrInvalidCountry
		}
		upper := strings.ToUpper(cc)
		req.CountryCode = &upper
	}

	if err := s.tenantRepo.Update(ctx, id, &req); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	// Re-fetch the updated tenant.
	tenant, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("update tenant: re-fetch: %w", err)
	}

	result := *tenant
	result.CreatedBy = uuid.Nil
	return &result, nil
}

// Delete performs a soft delete on the tenant.
func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.tenantRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTenantNotFound
		}
		return fmt.Errorf("delete tenant: %w", err)
	}

	if err := s.tenantRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}

	// In production, log the delete event in audit_logs here.
	_ = existing

	return nil
}

// isValidISO3166Alpha2 checks whether s is a valid 2-letter ISO 3166-1 alpha-2 code.
func isValidISO3166Alpha2(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 2 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// ensure repository is imported.
var _ = repository.ErrDuplicateUser
