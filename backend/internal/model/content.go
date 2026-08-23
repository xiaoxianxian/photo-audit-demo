package model

import (
	"time"

	"github.com/google/uuid"
)

// ContentType defines the kind of content being submitted for review.
type ContentType string

const (
	ContentTypePhoto       ContentType = "photo"
	ContentTypeShortVideo  ContentType = "short_video"
	ContentTypeLiveStream  ContentType = "live_stream"
)

// ReviewPolicy defines when human review happens relative to publishing.
type ReviewPolicy string

const (
	ReviewPostThenReview  ReviewPolicy = "post_then_review"
	ReviewBeforePost      ReviewPolicy = "review_before_post"
)

// ElementStatus tracks the processing stage of a content element.
type ElementStatus string

const (
	ElementPendingAI       ElementStatus = "pending_ai"
	ElementAIProcessing    ElementStatus = "ai_processing"
	ElementAIPassed        ElementStatus = "ai_passed"
	ElementAIRejected      ElementStatus = "ai_rejected"
	ElementPendingHuman    ElementStatus = "pending_human"
	ElementInHumanReview   ElementStatus = "in_human_review"
	ElementHumanPassed     ElementStatus = "human_passed"
	ElementHumanRejected   ElementStatus = "human_rejected"
)

// ElementKind identifies what part of the content an element represents.
type ElementKind string

const (
	ElementCoverImage   ElementKind = "cover_image"
	ElementVideoFrame   ElementKind = "video_frame"
	ElementTitle        ElementKind = "title"
	ElementComment      ElementKind = "comment"
	ElementASRText      ElementKind = "asr_text"
	ElementUserAvatar   ElementKind = "user_avatar"
	ElementNickname     ElementKind = "nickname"
	ElementDescription  ElementKind = "description"
	ElementLiveSnapshot ElementKind = "live_snapshot"
)

// Content is the top-level entity representing uploaded media.
type Content struct {
	ID             uuid.UUID     `json:"id"`
	TenantID       uuid.UUID     `json:"tenant_id"`
	ContentType    ContentType   `json:"content_type"`
	ReviewPolicy   ReviewPolicy  `json:"review_policy"`
	AIRiskScore    int           `json:"ai_risk_score"`
	Status         string        `json:"status"`
	CreatorID      uuid.UUID     `json:"creator_id"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// CreateContentRequest carries the fields needed to create a new content record.
type CreateContentRequest struct {
	ContentType      ContentType  `json:"content_type"`
	ReviewPolicy     ReviewPolicy `json:"review_policy"`
	CreatorID        uuid.UUID    `json:"creator_id"`
	Title            string       `json:"title,omitempty"`
	Description      string       `json:"description,omitempty"`
	OriginalURL      string       `json:"original_url,omitempty"`
	ThumbnailURL     string       `json:"thumbnail_url,omitempty"`
	FileName         string       `json:"file_name,omitempty"`
	FileSize         int64        `json:"file_size,omitempty"`
	MIMEType         string       `json:"mime_type,omitempty"`
	Width            int          `json:"width,omitempty"`
	Height           int          `json:"height,omitempty"`
	Duration         int          `json:"duration,omitempty"`
	VideoFingerprint string       `json:"video_fingerprint,omitempty"`
	ASRText          string       `json:"asr_text,omitempty"`
	StreamURL        string       `json:"stream_url,omitempty"`
	PlayURL          string       `json:"play_url,omitempty"`
	FrameInterval    int          `json:"frame_interval,omitempty"`
}

// ContentElement is a decomposed piece of a Content (image, title, comment, etc.).
type ContentElement struct {
	ID             uuid.UUID     `json:"id"`
	ContentID      uuid.UUID     `json:"content_id"`
	ElementKind    ElementKind   `json:"element_kind"`
	ElementContent string        `json:"element_content"`
	AIRiskScore    int           `json:"ai_risk_score"`
	AIRiskTypes    []string      `json:"ai_risk_types"`
	AIConfidence   float64       `json:"ai_confidence"`
	AIStatus       ElementStatus `json:"ai_status"`
	HumanStatus    ElementStatus `json:"human_status"`
	IsConflict     bool          `json:"is_conflict"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// CreateElementRequest carries the minimal fields to add an element to existing content.
type CreateElementRequest struct {
	ContentID      uuid.UUID   `json:"content_id"`
	ElementKind    ElementKind `json:"element_kind"`
	ElementContent string      `json:"element_content"`
}

// LiveFrameSnapshot captures an AI-reviewed snapshot from a live stream.
type LiveFrameSnapshot struct {
	ID           uuid.UUID    `json:"id"`
	ContentID    uuid.UUID    `json:"content_id"`
	StreamID     uuid.UUID    `json:"stream_id,omitempty"`
	SnapshotURL  string       `json:"snapshot_url"`
	SnapshotTime time.Time    `json:"snapshot_time"`
	AIRiskScore  int          `json:"ai_risk_score"`
	AIRiskTypes  []string     `json:"ai_risk_types"`
	AIConfidence float64      `json:"ai_confidence"`
}

// ContentListQuery holds pagination and filter parameters for listing contents.
type ContentListQuery struct {
	Page       int     `form:"page,default=1"`
	PageSize   int     `form:"page_size,default=20"`
	ContentType *string `form:"content_type"`
	Status     *string `form:"status"`
	TenantID   *string `form:"tenant_id"`
	SortBy     string  `form:"sort_by,default=created_at"`
	SortOrder  string  `form:"sort_order,default=desc"`
}
