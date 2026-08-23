package api

import (
	"audit-platform/internal/service"

	"github.com/gofiber/fiber/v2"
)

// DashboardHandler handles HTTP requests for dashboard analytics and statistics.
type DashboardHandler struct {
	dashSvc *service.DashboardService
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(dashSvc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashSvc: dashSvc,
	}
}

// GetStats handles GET /api/v1/dashboard/stats — return aggregated dashboard statistics.
func (h *DashboardHandler) GetStats(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id")
	tenantStr := ""
	if tenantID != nil {
		tenantStr = tenantID.(string)
	}

	stats, err := h.dashSvc.GetStats(c.Context(), tenantStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to fetch dashboard stats: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": stats,
	})
}

// GetReviewerPerformance handles GET /api/v1/dashboard/reviewers — return reviewer performance data.
func (h *DashboardHandler) GetReviewerPerformance(c *fiber.Ctx) error {
	tenantID, _ := getTenantID(c)
	items, err := h.dashSvc.GetReviewerPerformance(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to fetch reviewer performance: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": items,
	})
}

// GetDailyTrend handles GET /api/v1/dashboard/trend — return the last 7 days of review stats.
func (h *DashboardHandler) GetDailyTrend(c *fiber.Ctx) error {
	tenantID := c.Locals("tenant_id")
	tenantStr := ""
	if tenantID != nil {
		tenantStr = tenantID.(string)
	}

	trend, err := h.dashSvc.GetDailyTrend(c.Context(), tenantStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"message": "Failed to fetch daily trend: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": trend,
	})
}
