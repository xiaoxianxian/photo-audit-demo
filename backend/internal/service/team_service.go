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
	ErrLeaderNotFound    = errors.New("leader user not found")
	ErrLeaderNotInTenant = errors.New("leader does not belong to the specified tenant")
	ErrTeamNotFound      = errors.New("team not found")
	ErrUserExists        = errors.New("user is already a member of this team")
)

// TeamService handles business logic for audit team management.
type TeamService struct {
	teamRepo *repository.TeamRepository
	userRepo *repository.UserRepository
}

// NewTeamService creates a new TeamService.
func NewTeamService(teamRepo *repository.TeamRepository, userRepo *repository.UserRepository) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

// Create creates a new audit team within a tenant, with the specified user as leader.
func (s *TeamService) Create(ctx context.Context, req model.CreateTeamRequest, tenantID uuid.UUID, leaderID uuid.UUID) (*model.AuditTeam, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("team name is required")
	}

	// Validate leader exists and belongs to the same tenant.
	leader, err := s.userRepo.FindByID(ctx, leaderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLeaderNotFound
		}
		return nil, fmt.Errorf("create team: %w", err)
	}

	if leader.TenantID == nil || *leader.TenantID != tenantID {
		return nil, ErrLeaderNotInTenant
	}

	team := &model.AuditTeam{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     name,
		LeaderID: leaderID,
		Status:   1,
	}

	if err := s.teamRepo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}

	return team, nil
}

// GetByID retrieves a team by its UUID.
func (s *TeamService) GetByID(ctx context.Context, id uuid.UUID) (*model.AuditTeam, error) {
	team, err := s.teamRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTeamNotFound
		}
		return nil, fmt.Errorf("get team: %w", err)
	}
	return team, nil
}

// ListByTenant returns all teams for a given tenant.
func (s *TeamService) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.AuditTeam, error) {
	teams, err := s.teamRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	return teams, nil
}

// AddMember adds a user to an audit team.
func (s *TeamService) AddMember(ctx context.Context, teamID uuid.UUID, req model.AddTeamMemberRequest) (*model.AuditTeamMember, error) {
	memberRole := strings.TrimSpace(req.MemberRole)
	if !ValidMemberRoles[strings.ToLower(memberRole)] {
		return nil, fmt.Errorf("invalid member_role: %s", req.MemberRole)
	}

	userID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	// Validate user exists.
	_, err = s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLeaderNotFound
		}
		return nil, fmt.Errorf("add member: %w", err)
	}

	// Check for duplicate membership.
	exists, err := s.teamRepo.HasMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}
	if exists {
		return nil, ErrUserExists
	}

	if err := s.teamRepo.AddMember(ctx, teamID, userID, strings.ToLower(memberRole)); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	return &model.AuditTeamMember{
		TeamID:     teamID,
		UserID:     userID,
		MemberRole: strings.ToLower(memberRole),
	}, nil
}

// RemoveMember removes a user from an audit team.
func (s *TeamService) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	if err := s.teamRepo.RemoveMember(ctx, teamID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// ListMembers returns all members of a given team.
func (s *TeamService) ListMembers(ctx context.Context, teamID uuid.UUID) ([]model.AuditTeamMember, error) {
	members, err := s.teamRepo.ListMembers(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	return members, nil
}
