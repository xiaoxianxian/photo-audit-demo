package model

import (
	"time"

	"github.com/google/uuid"
)

// QualityAuditLevel defines the severity of a quality audit finding.
type QualityAuditLevel string

const (
	QualityLevelPass      QualityAuditLevel = "pass"
	QualityLevelMinor     QualityAuditLevel = "minor_issue"
	QualityLevelMajor     QualityAuditLevel = "major_issue"
	QualityLevelCritical  QualityAuditLevel = "critical"
)

// QualityAuditMode defines how a quality audit batch operates.
type QualityAuditMode string

const (
	ModeLocalCorrection  QualityAuditMode = "local_correction"  // only fix the batch, don't change original penalty
	ModeFullCorrection   QualityAuditMode = "full_correction"   // fix batch + cascade to user penalty
)

// QualityAuditBatch represents a QA sampling batch created by a quality checker.
type QualityAuditBatch struct {
	ID              uuid.UUID           `json:"id"`
	TenantID        uuid.UUID           `json:"tenant_id"`
	CreatedBy       uuid.UUID           `json:"created_by"` // quality_checker ID
	Name            string              `json:"name"`
	Mode            QualityAuditMode    `json:"mode"`
	FilterStatus    string              `json:"filter_status"` // e.g. "approved", "rejected" — which elements to sample from
	SampleSize      int                 `json:"sample_size"`
	Status          string              `json:"status"` // "draft", "in_progress", "completed"
	ReviewedCount   int                 `json:"reviewed_count"`
	CreatedAt       time.Time           `json:"created_at"`
	CompletedAt     *time.Time          `json:"completed_at"`
}

// CreateQualityBatchRequest carries the fields for creating a quality audit batch.
type CreateQualityBatchRequest struct {
	Name         string             `json:"name" validate:"required"`
	Mode         QualityAuditMode   `json:"mode" validate:"required"`
	FilterStatus string             `json:"filter_status" validate:"required"`
	SampleSize   int               `json:"sample_size" validate:"required,min=1"`
}

// QualityAuditRecord is a single element review within a quality audit batch.
type QualityAuditRecord struct {
	ID            uuid.UUID          `json:"id"`
	BatchID       uuid.UUID          `json:"batch_id"`
	ElementID     uuid.UUID          `json:"element_id"`
	OriginalScore int                `json:"original_score"`
	QAScore       int                `json:"qa_score"`
	QALevel       QualityAuditLevel  `json:"qa_level"`
	Disagree      bool               `json:"disagree"` // QA disagrees with original human decision
	Comment       *string            `json:"comment"`
	CreatedBy     uuid.UUID          `json:"created_by"`
	CreatedAt     time.Time          `json:"created_at"`
}

// SubmitQualityRecordRequest carries the fields for submitting a single QA review.
type SubmitQualityRecordRequest struct {
	ElementID string            `json:"element_id" validate:"required"`
	QAScore   int              `json:"qa_score" validate:"required,min=0,max=100"`
	QALevel   QualityAuditLevel `json:"qa_level" validate:"required"`
	Disagree  bool             `json:"disagree"`
	Comment   *string          `json:"comment"`
}

// QualityAuditStats aggregates quality audit metrics for a batch.
type QualityAuditStats struct {
	BatchID       uuid.UUID         `json:"batch_id"`
	BatchName     string            `json:"batch_name"`
	TotalSamples  int               `json:"total_samples"`
	ReviewedCount int               `json:"reviewed_count"`
	DisagreeCount int               `json:"disagree_count"`
	DisagreeRate  float64           `json:"disagree_rate"`
	LevelCounts   map[string]int    `json:"level_counts"`
	AvgQAScore    float64           `json:"avg_qa_score"`
}

// QualityAuditQuery holds filter params for listing batches.
type QualityAuditQuery struct {
	TenantID *string     `form:"tenant_id"`
	Status   *string     `form:"status"`
	DateFrom *time.Time  `form:"date_from"`
	DateTo   *time.Time  `form:"date_to"`
	Page     int         `form:"page,default=1"`
	PageSize int         `form:"page_size,default=20"`
}
