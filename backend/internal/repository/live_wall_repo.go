package repository

import (
	"context"
	"fmt"
	"time"

	"audit-platform/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LiveWallRepository persists and retrieves live stream and snapshot data.
type LiveWallRepository struct {
	db *pgxpool.Pool
}

// NewLiveWallRepository creates a new LiveWallRepository.
func NewLiveWallRepository(db *pgxpool.Pool) *LiveWallRepository {
	return &LiveWallRepository{db: db}
}

// --- Live Stream CRUD ---

// CreateLiveStream inserts a new live stream record.
func (r *LiveWallRepository) CreateLiveStream(ctx context.Context, s *model.LiveStream) error {
	const q = `
		INSERT INTO live_streams (id, tenant_id, content_id, stream_key, stream_url, play_url,
		                          status, viewer_count, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, tenant_id, content_id, stream_key, stream_url, play_url,
		          status, viewer_count, started_at, created_at, updated_at`

	err := r.db.QueryRow(ctx, q,
		s.ID, s.TenantID, s.ContentID, s.StreamKey, s.StreamURL, s.PlayURL,
		s.Status, s.ViewerCount, s.StartedAt,
	).Scan(
		&s.ID, &s.TenantID, &s.ContentID, &s.StreamKey, &s.StreamURL, &s.PlayURL,
		&s.Status, &s.ViewerCount, &s.StartedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create live stream: %w", err)
	}
	return nil
}

// UpdateLiveStreamStatus updates the status and timestamps of a live stream.
func (r *LiveWallRepository) UpdateLiveStreamStatus(ctx context.Context, id uuid.UUID, status model.LiveStreamStatus, startedAt *time.Time) error {
	const q = `
		UPDATE live_streams
		SET status = $1, started_at = $2, updated_at = NOW()
		WHERE id = $3`

	result, err := r.db.Exec(ctx, q, status, startedAt, id)
	if err != nil {
		return fmt.Errorf("update live stream status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("update live stream status: %w", pgx.ErrNoRows)
	}
	return nil
}

// GetActiveStreams returns all live streams for a tenant that are streaming.
func (r *LiveWallRepository) GetActiveStreams(ctx context.Context, tenantID string) ([]model.LiveStream, error) {
	const q = `
		SELECT id, tenant_id, content_id, stream_key, stream_url, play_url,
		       status, viewer_count, started_at, created_at, updated_at
		FROM live_streams
		WHERE tenant_id = $1 AND status = 'streaming'
		ORDER BY started_at DESC`

	rows, err := r.db.Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get active live streams: %w", err)
	}
	defer rows.Close()

	var items []model.LiveStream
	for rows.Next() {
		var s model.LiveStream
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.ContentID, &s.StreamKey, &s.StreamURL, &s.PlayURL,
			&s.Status, &s.ViewerCount, &s.StartedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan live stream: %w", err)
		}
		items = append(items, s)
	}
	return items, nil
}

// GetLatestSnapshots returns the most recent snapshot per stream.
func (r *LiveWallRepository) GetLatestSnapshots(ctx context.Context, streamIDs []uuid.UUID) ([]model.LiveFrameSnapshot, error) {
	if len(streamIDs) == 0 {
		return nil, nil
	}

	// Use DISTINCT ON to get the latest snapshot per stream.
	const q = `
		SELECT DISTINCT ON (stream_id)
			id, stream_id, snapshot_url, snapshot_time, ai_risk_score, ai_risk_types, ai_confidence
		FROM live_frame_snapshots
		WHERE stream_id = ANY($1)
		ORDER BY stream_id, snapshot_time DESC`

	rows, err := r.db.Query(ctx, q, streamIDs)
	if err != nil {
		return nil, fmt.Errorf("get latest snapshots: %w", err)
	}
	defer rows.Close()

	var items []model.LiveFrameSnapshot
	for rows.Next() {
		var s model.LiveFrameSnapshot
		var types []string
		if err := rows.Scan(
			&s.ID, &s.StreamID, &s.SnapshotURL, &s.SnapshotTime,
			&s.AIRiskScore, &types, &s.AIConfidence,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		s.AIRiskTypes = types
		items = append(items, s)
	}
	return items, nil
}

// --- Snapshot CRUD ---

// CreateSnapshot inserts a new live frame snapshot.
func (r *LiveWallRepository) CreateSnapshot(ctx context.Context, s *model.LiveFrameSnapshot) error {
	const q = `
		INSERT INTO live_frame_snapshots (id, stream_id, snapshot_url, snapshot_time,
		                                  ai_risk_score, ai_risk_types, ai_confidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, stream_id, snapshot_url, snapshot_time, ai_risk_score, ai_risk_types, ai_confidence`

	var types []string
	if s.AIRiskTypes != nil {
		types = s.AIRiskTypes
	}

	err := r.db.QueryRow(ctx, q,
		s.ID, s.StreamID, s.SnapshotURL, s.SnapshotTime,
		s.AIRiskScore, types, s.AIConfidence,
	).Scan(
		&s.ID, &s.StreamID, &s.SnapshotURL, &s.SnapshotTime,
		&s.AIRiskScore, &s.AIRiskTypes, &s.AIConfidence,
	)
	if err != nil {
		return fmt.Errorf("create live frame snapshot: %w", err)
	}
	return nil
}

// CountActiveStreams returns the number of active streams for a tenant.
func (r *LiveWallRepository) CountActiveStreams(ctx context.Context, tenantID string) (int64, error) {
	const q = `SELECT COUNT(*) FROM live_streams WHERE tenant_id = $1 AND status = 'streaming'`
	var count int64
	err := r.db.QueryRow(ctx, q, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active streams: %w", err)
	}
	return count, nil
}
