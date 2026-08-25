package model

import (
	"time"

	"github.com/google/uuid"
)

// AppealStatus tracks the lifecycle of a user appeal.
type AppealStatus string

const (
	AppealSubmitted           AppealStatus = "submitted"
	AppealUnderReview         AppealStatus = "under_review"
	AppealResolvedApproved    AppealStatus = "resolved_approved"
	AppealResolvedMaintained  AppealStatus = "resolved_maintained"
)

// Appeal is a formal challenge raised by a content creator against a moderation decision.
type Appeal struct {
	ID            uuid.UUID     `json:"id"`
	TenantID      uuid.UUID     `json:"tenant_id"`
	ContentID     uuid.UUID     `json:"content_id"`
	ApplicantID   uuid.UUID     `json:"applicant_id"`
	Reason        string        `json:"reason"`
	EvidenceURLs  []string      `json:"evidence_urls"`
	Status        AppealStatus  `json:"status"`
	ReviewerID    *uuid.UUID    `json:"reviewer_id"`
	Resolution    *string       `json:"resolution"`
	PenaltyLevel  *string       `json:"penalty_level_code"`
	Comment       *string       `json:"comment"`
	SubmittedAt   time.Time     `json:"submitted_at"`
	ResolvedAt    *time.Time    `json:"resolved_at"`
}

// CreateAppealRequest carries the fields a user submits when filing an appeal.
type CreateAppealRequest struct {
	ContentID    uuid.UUID  `json:"content_id"`
	Reason       string     `json:"reason"`
	EvidenceURLs []string   `json:"evidence_urls"`
}

// UpdateAppealRequest carries the fields a moderator fills when resolving an appeal.
type UpdateAppealRequest struct {
	ReviewerID   *uuid.UUID `json:"reviewer_id"`
	Resolution   *string    `json:"resolution"`
	PenaltyLevel *string    `json:"penalty_level_code"`
	Comment      *string    `json:"comment"`
	Status       *string    `json:"status"` // persisted lifecycle transition (e.g. resolved_maintained)
}
