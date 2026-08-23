package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditTeam represents an audit team within a tenant.
type AuditTeam struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Name     string    `json:"name"`
	LeaderID uuid.UUID `json:"leader_id"`
	Status   int       `json:"status"` // 0=disabled, 1=active.
}

// CreateTeamRequest is the payload for creating a new audit team.
type CreateTeamRequest struct {
	Name     string `json:"name"`
	LeaderID string `json:"leader_id"`
}

// AuditTeamMember represents a member of an audit team.
type AuditTeamMember struct {
	TeamID     uuid.UUID `json:"team_id"`
	UserID     uuid.UUID `json:"user_id"`
	MemberRole string    `json:"member_role"` // reviewer, senior_reviewer, quality_checker.
	JoinedAt   time.Time `json:"joined_at"`
}

// AddTeamMemberRequest is the payload for adding a member to a team.
type AddTeamMemberRequest struct {
	UserID     string `json:"user_id"`
	MemberRole string `json:"member_role"`
}
