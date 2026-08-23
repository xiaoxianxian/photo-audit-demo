package model

import (
	"time"

	"github.com/google/uuid"
)

// ReviewType identifies the source of an audit record.
type ReviewType string

const (
	ReviewTypeAIPrimary ReviewType = "ai_primary"
	ReviewTypeAIPJudge  ReviewType = "ai_judge"
	ReviewTypeHuman     ReviewType = "human"
	ReviewTypeQuality   ReviewType = "quality"
	ReviewTypeAppeal    ReviewType = "appeal"
)

// ReviewAction is the decision recorded in an audit entry.
type ReviewAction string

const (
	ActionApprove ReviewAction = "approve"
	ActionReject  ReviewAction = "reject"
	ActionMaintain ReviewAction = "maintain"
	ActionReverse ReviewAction = "reverse"
)

// RejectReason enumerates the policy violation categories.
type RejectReason string

const (
	RejectCopyright RejectReason = "copyright"
	RejectBlur      RejectReason = "blur"
	RejectPolitics  RejectReason = "politics"
	RejectPorn      RejectReason = "porn"
	RejectViolence  RejectReason = "violence"
	RejectSpam      RejectReason = "spam"
)

// AuditRecord is an immutable log entry for every review action taken on an element.
type AuditRecord struct {
	ID            uuid.UUID      `json:"id"`
	TaskID        uuid.UUID      `json:"task_id"`
	ElementID     uuid.UUID      `json:"element_id"`
	ReviewerID    *uuid.UUID     `json:"reviewer_id"`
	ReviewType    ReviewType     `json:"review_type"`
	Action        ReviewAction   `json:"action"`
	PenaltyLevel  *string        `json:"penalty_level_code"`
	Reason        *RejectReason  `json:"reason"`
	Comment       *string        `json:"comment"`
	AIScoreBefore *int           `json:"ai_score_before"`
	AIScoreAfter  *int           `json:"ai_score_after"`
	IsConflict    bool           `json:"is_conflict"`
	CreatedAt     time.Time      `json:"created_at"`
}

// HumanReviewRequest is the payload from the审核工作台 when a human auditor acts on an element.
type HumanReviewRequest struct {
	ElementID    uuid.UUID     `json:"element_id"`
	Action       ReviewAction  `json:"action"`
	Reason       *RejectReason `json:"reason,omitempty"`
	Comment      *string       `json:"comment,omitempty"`
	ScoreBefore  int           `json:"score_before"`
	ScoreAfter   int           `json:"score_after"`
	IsConflict   bool          `json:"is_conflict"`
}
