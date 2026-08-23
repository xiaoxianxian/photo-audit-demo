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
	ErrRuleNotFound    = errors.New("audit rule not found")
	ErrInvalidRuleAction = errors.New("invalid rule action")
)

// TenantRuleService handles business logic for tenant audit rules.
type TenantRuleService struct {
	ruleRepo *repository.TenantRuleRepository
}

func NewTenantRuleService(ruleRepo *repository.TenantRuleRepository) *TenantRuleService {
	return &TenantRuleService{ruleRepo: ruleRepo}
}

// Create validates and creates a new audit rule.
func (s *TenantRuleService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateTenantAuditRuleRequest) (*model.TenantAuditRule, error) {
	name := strings.TrimSpace(req.RuleName)
	if name == "" {
		return nil, errors.New("rule name is required")
	}

	action := strings.TrimSpace(req.Action)
	if action == "" {
		return nil, errors.New("rule action is required")
	}
	if action != "approve" && action != "reject" && action != "flag" {
		return nil, ErrInvalidRuleAction
	}

	rule := &model.TenantAuditRule{
		ID:             uuid.New(),
		TenantID:       tenantID,
		RuleName:       name,
		RuleExpression: req.RuleExpression,
		Action:         action,
		Priority:       req.Priority,
		Status:         1,
	}

	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}

	return rule, nil
}

// GetByID retrieves a rule by UUID.
func (s *TenantRuleService) GetByID(ctx context.Context, id uuid.UUID) (*model.TenantAuditRule, error) {
	rule, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRuleNotFound
		}
		return nil, fmt.Errorf("get rule: %w", err)
	}
	return rule, nil
}

// List returns paginated rules for a tenant.
func (s *TenantRuleService) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]model.TenantAuditRule, int64, error) {
	return s.ruleRepo.ListByTenant(ctx, tenantID, page, pageSize)
}

// Update applies a partial update.
func (s *TenantRuleService) Update(ctx context.Context, id uuid.UUID, req model.UpdateTenantAuditRuleRequest) (*model.TenantAuditRule, error) {
	if req.Action != nil {
		action := strings.TrimSpace(*req.Action)
		if action != "approve" && action != "reject" && action != "flag" {
			return nil, ErrInvalidRuleAction
		}
		*req.Action = action
	}
	if err := s.ruleRepo.Update(ctx, id, &req); err != nil {
		return nil, fmt.Errorf("update rule: %w", err)
	}
	return s.ruleRepo.FindByID(ctx, id)
}

// Delete performs a soft delete.
func (s *TenantRuleService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRuleNotFound
		}
		return fmt.Errorf("delete rule: %w", err)
	}
	return s.ruleRepo.Delete(ctx, id)
}
