package model

import (
	"time"

	"github.com/google/uuid"
)

// LiveStreamStatus tracks the state of a live stream.
type LiveStreamStatus string

const (
	LiveStatusIdle      LiveStreamStatus = "idle"
	LiveStatusStreaming LiveStreamStatus = "streaming"
	LiveStatusOffline   LiveStreamStatus = "offline"
)

// LiveStream represents an active broadcast.
type LiveStream struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	ContentID   uuid.UUID       `json:"content_id"`
	StreamKey   string          `json:"stream_key"`
	StreamURL   string          `json:"stream_url"`
	PlayURL     string          `json:"play_url"`
	Status      LiveStreamStatus `json:"status"`
	ViewerCount int             `json:"viewer_count"`
	StartedAt   *time.Time      `json:"started_at"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// CreateLiveStreamRequest carries the fields for starting a live stream.
type CreateLiveStreamRequest struct {
	ContentID uuid.UUID `json:"content_id"`
	StreamKey string    `json:"stream_key"`
	StreamURL string    `json:"stream_url"`
	PlayURL   string    `json:"play_url"`
}

// CreateSnapshotRequest carries the fields for recording a live frame snapshot.
type CreateSnapshotRequest struct {
	StreamID     uuid.UUID `json:"stream_id"`
	SnapshotURL  string    `json:"snapshot_url"`
	SnapshotTime time.Time `json:"snapshot_time"`
	AIRiskScore  int       `json:"ai_risk_score"`
	AIRiskTypes  []string  `json:"ai_risk_types"`
	AIConfidence float64   `json:"ai_confidence"`
}

// LiveWallQuery holds filter params for listing streams on the TV wall.
type LiveWallQuery struct {
	TenantID string `form:"tenant_id"`
	Status   string `form:"status"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=50"`
}
