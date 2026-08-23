package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"audit-platform/internal/model"
)

// ErrDuplicateUser is returned when a user with the same username already exists.
var ErrDuplicateUser = errors.New("username already exists")

// UserRepository provides data access for User entities.
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository backed by the given pool.
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user into the database.
// Returns ErrDuplicateUser if a user with the same username already exists.
func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	query := `
		INSERT INTO users (id, tenant_id, username, display_name, password_hash,
		                   role, email, phone, languages, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		u.ID,
		u.TenantID,
		u.Username,
		u.DisplayName,
		u.PasswordHashBcrypt,
		u.Role,
		u.Email,
		u.Phone,
		u.Languages,
		u.Status,
		time.Now(),
	).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: %s", ErrDuplicateUser, u.Username)
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// FindByUsername retrieves a user by their username.
// Returns pgx.ErrNoRows if not found.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	query := `
		SELECT id, tenant_id, username, display_name, password_hash, role,
		       email, phone, languages, status, created_at
		FROM users
		WHERE username = $1`

	err := r.db.QueryRow(ctx, query, username).Scan(
		&u.ID,
		&u.TenantID,
		&u.Username,
		&u.DisplayName,
		&u.PasswordHashBcrypt,
		&u.Role,
		&u.Email,
		&u.Phone,
		&u.Languages,
		&u.Status,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by username '%s': %w", username, pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("find user by username '%s': %w", username, err)
	}
	return &u, nil
}

// FindByUsernameAndTenant retrieves a user by username within a specific tenant.
func (r *UserRepository) FindByUsernameAndTenant(ctx context.Context, username string, tenantID uuid.UUID) (*model.User, error) {
	var u model.User
	query := `
		SELECT id, tenant_id, username, display_name, password_hash, role,
		       email, phone, languages, status, created_at
		FROM users
		WHERE username = $1 AND tenant_id = $2`

	err := r.db.QueryRow(ctx, query, username, tenantID).Scan(
		&u.ID,
		&u.TenantID,
		&u.Username,
		&u.DisplayName,
		&u.PasswordHashBcrypt,
		&u.Role,
		&u.Email,
		&u.Phone,
		&u.Languages,
		&u.Status,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by username and tenant: %w", pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("find user by username and tenant: %w", err)
	}
	return &u, nil
}

// FindByID retrieves a user by their UUID.
// Returns pgx.ErrNoRows if not found.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	query := `
		SELECT id, tenant_id, username, display_name, password_hash, role,
		       email, phone, languages, status, created_at
		FROM users
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.TenantID,
		&u.Username,
		&u.DisplayName,
		&u.PasswordHashBcrypt,
		&u.Role,
		&u.Email,
		&u.Phone,
		&u.Languages,
		&u.Status,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by id '%s': %w", id.String(), pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("find user by id '%s': %w", id.String(), err)
	}
	return &u, nil
}

// UpdateStatus sets the status of a user (0=disabled, 1=active).
func (r *UserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status int) error {
	query := `UPDATE users SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update user status %d for id '%s': %w", status, id.String(), err)
	}
	return nil
}

// List returns a paginated list of users. Returns (users, total_count, error).
func (r *UserRepository) List(ctx context.Context, page, pageSize int) ([]model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var users []model.User

	// Count total.
	var total int64
	countQuery := `SELECT COUNT(*) FROM users`
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// Fetch page.
	query := `
		SELECT id, tenant_id, username, display_name, password_hash, role,
		       email, phone, languages, status, created_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID,
			&u.TenantID,
			&u.Username,
			&u.DisplayName,
			&u.PasswordHashBcrypt,
			&u.Role,
			&u.Email,
			&u.Phone,
			&u.Languages,
			&u.Status,
			&u.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate user rows: %w", err)
	}

	return users, total, nil
}
