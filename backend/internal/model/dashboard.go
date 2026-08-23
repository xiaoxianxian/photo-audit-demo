package model

import (
	"time"

	"github.com/google/uuid"
)

// DashboardStats aggregates the key metrics surfaced to platform and tenant dashboards.
type DashboardStats struct {
	TotalReviewed   int64   `json:"total_reviewed"`
	TodayReviewed   int64   `json:"today_reviewed"`
	ApprovalRate    float64 `json:"approval_rate"`
	RejectionRate   float64 `json:"rejection_rate"`
	AvgRiskScore    float64 `json:"avg_risk_score"`
	AppealCount     int64   `json:"appeal_count"`
	ActiveStreams   int64   `json:"active_streams"`
	PendingElements int64   `json:"pending_elements"`
	ConflictCount   int64   `json:"conflict_count"`
	AccuracyRate    float64 `json:"accuracy_rate"`
}

// ReviewerPerformance summarizes an individual auditor's productivity and quality.
type ReviewerPerformance struct {
	ReviewerID   uuid.UUID `json:"reviewer_id"`
	ReviewerName string    `json:"reviewer_name"`
	TotalReviews int64     `json:"total_reviews"`
	Approved     int64     `json:"approved"`
	Rejected     int64     `json:"rejected"`
	Appeals      int64     `json:"appeals"`
	Accuracy     float64   `json:"accuracy"`
	AvgTimeSec   float64   `json:"avg_time_sec"`
}

// ReviewerPerformanceQuery holds the filter parameters for querying reviewer metrics.
type ReviewerPerformanceQuery struct {
	TenantID    *string   `form:"tenant_id"`
	DateFrom    *time.Time `form:"date_from"`
	DateTo      *time.Time `form:"date_to"`
}
