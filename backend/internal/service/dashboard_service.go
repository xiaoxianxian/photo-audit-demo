package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"audit-platform/internal/model"
	"audit-platform/internal/repository"

	"github.com/google/uuid"
)

// ReviewerPerfItem represents a single reviewer's aggregated performance data.
type ReviewerPerfItem struct {
	ReviewerID   uuid.UUID  `json:"reviewer_id"`
	ReviewerName string     `json:"reviewer_name"`
	TotalReviews int64      `json:"total_reviews"`
	Approved     int64      `json:"approved"`
	Rejected     int64      `json:"rejected"`
	Appeals      int64      `json:"appeals"`
	Accuracy     float64    `json:"accuracy"`
	AvgTimeSec   float64    `json:"avg_time_sec"`
}

// ReviewerPerfResult wraps a paginated list of reviewer performance data.
type ReviewerPerfResult struct {
	Items []ReviewerPerfItem
	Total int64
}

// DashboardService computes dashboard statistics and reviewer performance metrics.
type DashboardService struct {
	auditLogRepo *repository.LogRepository
	elementRepo  *repository.ElementRepository
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(auditLogRepo *repository.LogRepository, elementRepo *repository.ElementRepository) *DashboardService {
	return &DashboardService{
		auditLogRepo: auditLogRepo,
		elementRepo:  elementRepo,
	}
}

// GetStats aggregates dashboard statistics for a tenant.
// Uses a single consolidated query instead of 8 separate SELECT calls.
func (s *DashboardService) GetStats(ctx context.Context, tenantID string) (*model.DashboardStats, error) {
	// Pending elements count (elements with ai_status = pending_human).
	pendingElements, err := s.elementRepo.CountByStatus(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get stats pending elements: %w", err)
	}

	// Consolidated query replaces: countRecordsAllTime, countRecordsToday,
	// countApprovalsAndRejections, avgRiskScore, CountConflictRecords, CountPendingAppeals.
	row, err := s.auditLogRepo.GetDashboardStatsConsolidated(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get stats consolidated: %w", err)
	}

	var approvalRate, rejectionRate float64
	if row.TotalReviewed > 0 {
		approvalRate = math.Round(float64(row.Approved)/float64(row.TotalReviewed)*10000) / 100
		rejectionRate = math.Round(float64(row.Rejected)/float64(row.TotalReviewed)*10000) / 100
	}

	return &model.DashboardStats{
		TotalReviewed:   row.TotalReviewed,
		TodayReviewed:   row.TodayReviewed,
		ApprovalRate:    approvalRate,
		RejectionRate:   rejectionRate,
		AvgRiskScore:    row.AvgRiskScore,
		PendingElements: pendingElements["pending_human"],
		ConflictCount:   row.ConflictCount,
		AppealCount:     row.PendingAppeals,
	}, nil
}

// GetReviewerPerformance returns performance stats for all reviewers in the current tenant.
func (s *DashboardService) GetReviewerPerformance(ctx context.Context, tenantID string) ([]ReviewerPerfItem, error) {
	stats, err := s.auditLogRepo.CountByReviewer(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get reviewer performance: %w", err)
	}

	items := make([]ReviewerPerfItem, 0, len(stats))
	for _, st := range stats {
		items = append(items, ReviewerPerfItem{
			ReviewerID:   st.ReviewerID,
			ReviewerName: st.ReviewerName,
			TotalReviews: st.TotalReviews,
			Approved:     st.Approved,
			Rejected:     st.Rejected,
			Appeals:      st.Appeals,
			Accuracy:     st.Accuracy,
			AvgTimeSec:   st.AvgTimeSec,
		})
	}
	return items, nil
}

// --- Internal helper methods (kept for other callers) -----------------------

func (s *DashboardService) countRecordsAllTime(ctx context.Context, tenantID string) (int64, error) {
	return s.auditLogRepo.CountByType(ctx, model.ReviewTypeHuman, tenantID)
}

func (s *DashboardService) countRecordsToday(ctx context.Context, startOfDay time.Time, tenantID string) (int64, error) {
	return s.auditLogRepo.CountByDateRange(ctx, startOfDay, startOfDay.Add(24*time.Hour), tenantID)
}

// countPendingElements counts elements with ai_status = pending_human (truly awaiting human review).
func (s *DashboardService) countPendingElements(ctx context.Context, tenantID string) (int64, error) {
	counts, err := s.elementRepo.CountByStatus(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	// Sum up only the pending_human bucket — that is what "pending elements" means
	// for the dashboard (elements awaiting human review).
	return counts["pending_human"], nil
}

func (s *DashboardService) countApprovalsAndRejections(ctx context.Context, tenantID string) (approved, rejected int64, _ error) {
	a, err := s.auditLogRepo.CountByAction(ctx, model.ActionApprove, tenantID)
	if err != nil {
		return 0, 0, fmt.Errorf("count approved: %w", err)
	}
	r, err := s.auditLogRepo.CountByAction(ctx, model.ActionReject, tenantID)
	if err != nil {
		return 0, 0, fmt.Errorf("count rejected: %w", err)
	}
	return a, r, nil
}

func (s *DashboardService) avgRiskScore(ctx context.Context, tenantID string) (float64, error) {
	return s.auditLogRepo.AvgAIScore(ctx, tenantID)
}

func (s *DashboardService) countApprovalsAndRejectionsForDay(ctx context.Context, start, end time.Time, tenantID string) (approved, rejected int64, _ error) {
	a, err := s.auditLogRepo.CountByActionDateRange(ctx, model.ActionApprove, start, end, tenantID)
	if err != nil {
		return 0, 0, fmt.Errorf("count approved for day: %w", err)
	}
	r, err := s.auditLogRepo.CountByActionDateRange(ctx, model.ActionReject, start, end, tenantID)
	if err != nil {
		return 0, 0, fmt.Errorf("count rejected for day: %w", err)
	}
	return a, r, nil
}

// DailyTrendPoint represents a single data point in the daily trend chart.
type DailyTrendPoint struct {
	Date         string  `json:"date"`
	TotalReviewed int64   `json:"total_reviewed"`
	ApprovalRate float64 `json:"approval_rate"`
	RejectionRate float64 `json:"rejection_rate"`
}

// GetDailyTrend returns the last 7 days of review statistics for the given tenant.
func (s *DashboardService) GetDailyTrend(ctx context.Context, tenantID string) ([]DailyTrendPoint, error) {
	now := time.Now()
	points := make([]DailyTrendPoint, 7)
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
		end := start.Add(24 * time.Hour)

		total, err := s.auditLogRepo.CountByDateRange(ctx, start, end, tenantID)
		if err != nil {
			return nil, fmt.Errorf("get daily trend total: %w", err)
		}

		approved, rejected, _ := s.countApprovalsAndRejectionsForDay(ctx, start, end, tenantID)

		var approvalRate, rejectionRate float64
		if total > 0 {
			approvalRate = math.Round(float64(approved)/float64(total)*10000) / 100
			rejectionRate = math.Round(float64(rejected)/float64(total)*10000) / 100
		}

		points[6-i] = DailyTrendPoint{
			Date:          start.Format("01-02"),
			TotalReviewed: total,
			ApprovalRate:  approvalRate,
			RejectionRate: rejectionRate,
		}
	}
	return points, nil
}
