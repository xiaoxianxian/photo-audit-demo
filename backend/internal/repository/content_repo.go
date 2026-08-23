package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"audit-platform/internal/model"
)

// ContentRepository persists and retrieves Content and type-specific extension rows.
type ContentRepository struct {
	db *pgxpool.Pool
}

// NewContentRepository creates a new ContentRepository backed by the given database pool.
func NewContentRepository(db *pgxpool.Pool) *ContentRepository {
	return &ContentRepository{db: db}
}

// Create inserts a new content record along with its type-specific extension row.
// Both the main content row and the extension row are wrapped in a transaction
// to prevent orphan records if the extension insert fails.
func (r *ContentRepository) Create(ctx context.Context, c *model.Content) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	const q = `
		INSERT INTO contents (id, tenant_id, content_type, review_policy, ai_risk_score, status, creator_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, tenant_id, content_type, review_policy, ai_risk_score, status, creator_id, created_at, updated_at`

	err = tx.QueryRow(ctx, q,
		c.ID, c.TenantID, c.ContentType, c.ReviewPolicy, c.AIRiskScore, c.Status, c.CreatorID,
	).Scan(
		&c.ID, &c.TenantID, &c.ContentType, &c.ReviewPolicy, &c.AIRiskScore, &c.Status,
		&c.CreatorID, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create content: %w", err)
	}

	var extErr error
	switch c.ContentType {
	case model.ContentTypePhoto:
		extErr = r.createPhotoRowTx(tx, c.ID)
	case model.ContentTypeShortVideo:
		extErr = r.createShortVideoRowTx(tx, c.ID)
	case model.ContentTypeLiveStream:
		extErr = r.createLiveStreamRowTx(tx, c.ID)
	default:
		extErr = fmt.Errorf("unknown content type %q", c.ContentType)
	}
	if extErr != nil {
		return extErr
	}

	return tx.Commit(ctx)
}

// FindByID loads a content record together with its type-specific extension fields.
func (r *ContentRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Content, error) {
	c := &model.Content{}

	const baseQ = `
		SELECT id, tenant_id, content_type, review_policy, ai_risk_score, status, creator_id, created_at, updated_at
		FROM contents WHERE id = $1`

	row := r.db.QueryRow(ctx, baseQ, id)
	if err := row.Scan(
		&c.ID, &c.TenantID, &c.ContentType, &c.ReviewPolicy, &c.AIRiskScore,
		&c.Status, &c.CreatorID, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("find content by id %w", err)
	}

	// Load type-specific extension fields
	switch c.ContentType {
	case model.ContentTypePhoto:
		return r.readPhotoRow(ctx, c)
	case model.ContentTypeShortVideo:
		return r.readShortVideoRow(ctx, c)
	case model.ContentTypeLiveStream:
		return r.readLiveStreamRow(ctx, c)
	}

	return c, nil
}

// List returns paginated contents with filtering and sorting.
func (r *ContentRepository) List(ctx context.Context, query model.ContentListQuery) ([]model.Content, int64, error) {
	// Whitelist allowed sort columns and directions to prevent SQL injection.
	allowedSortCols := map[string]bool{
		"id": true, "created_at": true, "updated_at": true, "ai_risk_score": true,
		"title": true, "status": true, "content_type": true,
	}
	if !allowedSortCols[query.SortBy] {
		query.SortBy = "created_at"
	}
	if query.SortOrder != "asc" {
		query.SortOrder = "desc"
	}

	offset := (query.Page - 1) * query.PageSize
	if offset < 0 {
		offset = 0
	}

	// Build dynamic WHERE clause with parameterized values.
	where := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	if query.TenantID != nil && *query.TenantID != "" {
		tenantUUID, err := uuid.Parse(*query.TenantID)
		if err == nil {
			where = append(where, fmt.Sprintf("tenant_id = $%d", argIndex))
			args = append(args, tenantUUID)
			argIndex++
		}
	}
	if query.ContentType != nil && *query.ContentType != "" {
		where = append(where, fmt.Sprintf("content_type = $%d", argIndex))
		args = append(args, *query.ContentType)
		argIndex++
	}
	if query.Status != nil && *query.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *query.Status)
		argIndex++
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	// Count query
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM contents %s", whereClause)
	var total int64
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count contents: %w", err)
	}

	// Data query — ORDER BY columns are validated against whitelist above.
	listQ := fmt.Sprintf(`
		SELECT id, tenant_id, content_type, review_policy, ai_risk_score, status, creator_id, created_at, updated_at
		FROM contents %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`, whereClause, query.SortBy, query.SortOrder, argIndex, argIndex+1)

	args = append(args, query.PageSize, offset)

	rows, err := r.db.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list contents: %w", err)
	}
	defer rows.Close()

	items := make([]model.Content, 0, query.PageSize)
	for rows.Next() {
		var c model.Content
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.ContentType, &c.ReviewPolicy, &c.AIRiskScore,
			&c.Status, &c.CreatorID, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan content row: %w", err)
		}
		items = append(items, c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate content rows: %w", err)
	}

	return items, total, nil
}

// FindByStatus returns up to limit contents matching the given status (for batch processing).
func (r *ContentRepository) FindByStatus(ctx context.Context, status string, limit int) ([]model.Content, error) {
	const q = `
		SELECT id, tenant_id, content_type, review_policy, ai_risk_score, status, creator_id, created_at, updated_at
		FROM contents WHERE status = $1 ORDER BY created_at ASC LIMIT $2`

	rows, err := r.db.Query(ctx, q, status, limit)
	if err != nil {
		return nil, fmt.Errorf("find contents by status: %w", err)
	}
	defer rows.Close()

	var items []model.Content
	for rows.Next() {
		var c model.Content
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.ContentType, &c.ReviewPolicy, &c.AIRiskScore,
			&c.Status, &c.CreatorID, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan content: %w", err)
		}
		items = append(items, c)
	}

	return items, nil
}

// UpdateStatus modifies only the status field of a content record.
func (r *ContentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	const q = `UPDATE contents SET status = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.db.Exec(ctx, q, status, id)
	if err != nil {
		return fmt.Errorf("update content status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("update content status: %w", pgx.ErrNoRows)
	}
	return nil
}

// UpdateAIRisk updates the AI risk score and status of a content record.
func (r *ContentRepository) UpdateAIRisk(ctx context.Context, id uuid.UUID, riskScore int, status string) error {
	const q = `UPDATE contents SET ai_risk_score = $1, status = $2, updated_at = NOW() WHERE id = $3`
	result, err := r.db.Exec(ctx, q, riskScore, status, id)
	if err != nil {
		return fmt.Errorf("update content ai risk: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("update content ai risk: %w", pgx.ErrNoRows)
	}
	return nil
}

// FindByCreator returns all contents authored by a specific user, paginated.
func (r *ContentRepository) FindByCreator(ctx context.Context, creatorID uuid.UUID, page, pageSize int) ([]model.Content, int64, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	const countQ = `SELECT COUNT(*) FROM contents WHERE creator_id = $1`
	var total int64
	if err := r.db.QueryRow(ctx, countQ, creatorID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("find by creator count: %w", err)
	}

	const listQ = `
		SELECT id, tenant_id, content_type, review_policy, ai_risk_score, status, creator_id, created_at, updated_at
		FROM contents WHERE creator_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, listQ, creatorID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("find by creator: %w", err)
	}
	defer rows.Close()

	items := make([]model.Content, 0)
	for rows.Next() {
		var c model.Content
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.ContentType, &c.ReviewPolicy, &c.AIRiskScore,
			&c.Status, &c.CreatorID, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan creator content: %w", err)
		}
		items = append(items, c)
	}

	return items, total, nil
}

// --- Extension helpers --------------------------------------------------

func (r *ContentRepository) createPhotoRow(ctx context.Context, id uuid.UUID) error {
	const q = `
		INSERT INTO contents_photo (content_id, title, description, original_url, thumbnail_url, file_name, file_size, mime_type, width, height)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(ctx, q, id, "", "", "", "", "", 0, "", 0, 0)
	return err
}

func (r *ContentRepository) createPhotoRowTx(tx pgx.Tx, id uuid.UUID) error {
	const q = `
		INSERT INTO contents_photo (content_id, title, description, original_url, thumbnail_url, file_name, file_size, mime_type, width, height)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := tx.Exec(context.Background(), q, id, "", "", "", "", "", 0, "", 0, 0)
	return err
}

func (r *ContentRepository) createShortVideoRow(ctx context.Context, id uuid.UUID) error {
	const q = `
		INSERT INTO contents_short_video (content_id, title, description, original_url, thumbnail_url, file_name, file_size, mime_type, duration, video_fingerprint, asr_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.Exec(ctx, q, id, "", "", "", "", "", 0, "", "", "")
	return err
}

func (r *ContentRepository) createShortVideoRowTx(tx pgx.Tx, id uuid.UUID) error {
	const q = `
		INSERT INTO contents_short_video (content_id, title, description, original_url, thumbnail_url, file_name, file_size, mime_type, duration, video_fingerprint, asr_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := tx.Exec(context.Background(), q, id, "", "", "", "", "", 0, "", "", "")
	return err
}

func (r *ContentRepository) createLiveStreamRow(ctx context.Context, id uuid.UUID) error {
	const q = `
		INSERT INTO contents_live_stream (content_id, title, description, stream_url, play_url, frame_interval)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.Exec(ctx, q, id, "", "", "", "", 0)
	return err
}

func (r *ContentRepository) createLiveStreamRowTx(tx pgx.Tx, id uuid.UUID) error {
	const q = `
		INSERT INTO contents_live_stream (content_id, title, description, stream_url, play_url, frame_interval)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := tx.Exec(context.Background(), q, id, "", "", "", "", 0)
	return err
}

func (r *ContentRepository) readPhotoRow(ctx context.Context, c *model.Content) (*model.Content, error) {
	const q = `SELECT title, description, original_url, thumbnail_url, file_name, file_size, mime_type, width, height FROM contents_photo WHERE content_id = $1`
	var title, desc, origURL, thumbURL, fileName, mimeType string
	var fileSize int64
	var width, height int
	if err := r.db.QueryRow(ctx, q, c.ID).Scan(
		&title, &desc, &origURL, &thumbURL, &fileName, &fileSize, &mimeType, &width, &height,
	); err != nil {
		return nil, fmt.Errorf("read photo extension: %w", err)
	}
	return c, nil
}

func (r *ContentRepository) readShortVideoRow(ctx context.Context, c *model.Content) (*model.Content, error) {
	const q = `SELECT title, description, original_url, thumbnail_url, file_name, file_size, mime_type, duration, video_fingerprint, asr_text FROM contents_short_video WHERE content_id = $1`
	var title, desc, origURL, thumbURL, fileName, mimeType, fingerprint, asrText string
	var fileSize int64
	var duration int
	if err := r.db.QueryRow(ctx, q, c.ID).Scan(
		&title, &desc, &origURL, &thumbURL, &fileName, &fileSize, &mimeType, &duration, &fingerprint, &asrText,
	); err != nil {
		return nil, fmt.Errorf("read short video extension: %w", err)
	}
	return c, nil
}

func (r *ContentRepository) readLiveStreamRow(ctx context.Context, c *model.Content) (*model.Content, error) {
	const q = `SELECT title, description, stream_url, play_url, frame_interval FROM contents_live_stream WHERE content_id = $1`
	var title, desc, streamURL, playURL string
	var frameInterval int
	if err := r.db.QueryRow(ctx, q, c.ID).Scan(
		&title, &desc, &streamURL, &playURL, &frameInterval,
	); err != nil {
		return nil, fmt.Errorf("read live stream extension: %w", err)
	}
	return c, nil
}
