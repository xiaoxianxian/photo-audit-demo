package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"audit-platform/internal/model"
)

// TeamRepository provides data access for AuditTeam entities.
type TeamRepository struct {
	db *pgxpool.Pool
}

// NewTeamRepository creates a new TeamRepository backed by the given pool.
func NewTeamRepository(db *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{db: db}
}

// Create inserts a new audit team into the database.
func (r *TeamRepository) Create(ctx context.Context, t *model.AuditTeam) error {
	query := `
		INSERT INTO audit_teams (id, tenant_id, name, leader_id, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		t.ID,
		t.TenantID,
		t.Name,
		t.LeaderID,
		t.Status,
	).Scan(&t.ID)
	if err != nil {
		return fmt.Errorf("create team: %w", err)
	}
	return nil
}

// FindByID retrieves a team by its UUID.
// Returns pgx.ErrNoRows if not found.
func (r *TeamRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.AuditTeam, error) {
	var t model.AuditTeam
	query := `
		SELECT id, tenant_id, name, leader_id, status
		FROM audit_teams
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID,
		&t.TenantID,
		&t.Name,
		&t.LeaderID,
		&t.Status,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("find team by id '%s': %w", id.String(), pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("find team by id '%s': %w", id.String(), err)
	}
	return &t, nil
}

// ListByTenant retrieves all active teams for a tenant.
func (r *TeamRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.AuditTeam, error) {
	query := `
		SELECT id, tenant_id, name, leader_id, status
		FROM audit_teams
		WHERE tenant_id = $1
		ORDER BY name`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list teams for tenant '%s': %w", tenantID.String(), err)
	}
	defer rows.Close()

	var teams []model.AuditTeam
	for rows.Next() {
		var t model.AuditTeam
		if err := rows.Scan(
			&t.ID,
			&t.TenantID,
			&t.Name,
			&t.LeaderID,
			&t.Status,
		); err != nil {
			return nil, fmt.Errorf("scan team row: %w", err)
		}
		teams = append(teams, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate team rows: %w", err)
	}

	return teams, nil
}

// AddMember adds a user to an audit team with the specified role.
func (r *TeamRepository) AddMember(ctx context.Context, teamID, userID uuid.UUID, role string) error {
	query := `
		INSERT INTO audit_team_members (team_id, user_id, member_role, joined_at)
		VALUES ($1, $2, $3, $4)`

	_, err := r.db.Exec(ctx, query, teamID, userID, role, time.Now())
	if err != nil {
		return fmt.Errorf("add member '%s' to team '%s': %w", userID.String(), teamID.String(), err)
	}
	return nil
}

// RemoveMember removes a user from an audit team.
func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	query := `DELETE FROM audit_team_members WHERE team_id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, teamID, userID)
	if err != nil {
		return fmt.Errorf("remove member '%s' from team '%s': %w", userID.String(), teamID.String(), err)
	}
	return nil
}

// HasMember checks if a user is already a member of the given team.
func (r *TeamRepository) HasMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM audit_team_members WHERE team_id = $1 AND user_id = $2`
	err := r.db.QueryRow(ctx, query, teamID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check team member: %w", err)
	}
	return count > 0, nil
}

// ListMembers retrieves all members of an audit team.
func (r *TeamRepository) ListMembers(ctx context.Context, teamID uuid.UUID) ([]model.AuditTeamMember, error) {
	query := `
		SELECT team_id, user_id, member_role, joined_at
		FROM audit_team_members
		WHERE team_id = $1
		ORDER BY joined_at ASC`

	rows, err := r.db.Query(ctx, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("list members for team '%s': %w", teamID.String(), err)
	}
	defer rows.Close()

	var members []model.AuditTeamMember
	for rows.Next() {
		var m model.AuditTeamMember
		if err := rows.Scan(
			&m.TeamID,
			&m.UserID,
			&m.MemberRole,
			&m.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("scan team member row: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate team member rows: %w", err)
	}

	return members, nil
}
