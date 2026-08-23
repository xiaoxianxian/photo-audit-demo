package model

import (
	"time"

	"github.com/google/uuid"
)

// TenantAuditRule represents a per-tenant audit rule.
type TenantAuditRule struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	RuleName        string    `json:"rule_name"`
	RuleExpression  *string   `json:"rule_expression,omitempty"`
	Action          string    `json:"action"`
	Priority        int       `json:"priority"`
	Status          int       `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateTenantAuditRuleRequest is the payload for creating a rule.
type CreateTenantAuditRuleRequest struct {
	RuleName       string  `json:"rule_name"`
	RuleExpression *string `json:"rule_expression,omitempty"`
	Action         string  `json:"action"`
	Priority       int     `json:"priority"`
}

// UpdateTenantAuditRuleRequest supports partial updates.
type UpdateTenantAuditRuleRequest struct {
	RuleName       *string `json:"rule_name,omitempty"`
	RuleExpression *string `json:"rule_expression,omitempty"`
	Action         *string `json:"action,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
	Status         *int    `json:"status,omitempty"`
}
