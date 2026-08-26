// Package api wires up all HTTP routes.
//
// It provides a single Register method that accepts a fully configured
// *fiber.App and attaches every endpoint with the appropriate middleware chain.
package api

import (
	"audit-platform/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// SetupRoutes attaches all API routes to the given Fiber app.
func SetupRoutes(app *fiber.App, handlers *Handlers, authMW fiber.Handler, tenantMW fiber.Handler) {
	api := app.Group("/api/v1")

	// --- Public routes (no auth required) ---
	api.Post("/auth/login", handlers.AuthHandler.Login)
	api.Post("/auth/register", handlers.AuthHandler.Register)
	api.Get("/tenants/public", handlers.TenantHandler.ListPublic)

	// --- Protected routes (JWT required) ---
	protected := api.Group("", authMW)

	// Tenant-aware routes
	tenant := protected.Group("/tenants", tenantMW)
	tenant.Get("/", handlers.TenantHandler.List)
	tenant.Post("/", handlers.TenantHandler.Create)
	tenant.Get("/:id", handlers.TenantHandler.GetByID)
	tenant.Put("/:id", handlers.TenantHandler.Update)
	tenant.Delete("/:id", handlers.TenantHandler.Delete)

	// Team routes (under tenant isolation)
	team := protected.Group("/teams", tenantMW)
	team.Get("/", handlers.TeamHandler.ListByTenant)
	team.Post("/", handlers.TeamHandler.Create)
	team.Get("/:id/members", handlers.TeamHandler.ListMembers)
	team.Post("/:id/members", handlers.TeamHandler.AddMember)
	team.Delete("/:id/members/:user_id", handlers.TeamHandler.RemoveMember)

	// Tenant config routes (audit rules, penalty levels, custom words)
	rules := protected.Group("/audit-rules", tenantMW)
	rules.Get("/", handlers.RuleHandler.List)
	rules.Post("/", handlers.RuleHandler.Create)
	rules.Get("/:id", handlers.RuleHandler.GetByID)
	rules.Put("/:id", handlers.RuleHandler.Update)
	rules.Delete("/:id", handlers.RuleHandler.Delete)

	levels := protected.Group("/audit-levels", tenantMW)
	levels.Get("/", handlers.LevelHandler.List)
	levels.Post("/", handlers.LevelHandler.Create)
	levels.Get("/:id", handlers.LevelHandler.GetByID)
	levels.Put("/:id", handlers.LevelHandler.Update)
	levels.Delete("/:id", handlers.LevelHandler.Delete)

	words := protected.Group("/custom-words", tenantMW)
	words.Get("/", handlers.WordHandler.List)
	words.Post("/", handlers.WordHandler.Create)
	words.Get("/:id", handlers.WordHandler.GetByID)
	words.Put("/:id", handlers.WordHandler.Update)
	words.Delete("/:id", handlers.WordHandler.Delete)

	// AI config routes (under tenant isolation)
	aiConfig := protected.Group("/ai-config", tenantMW)
	aiConfig.Get("/", handlers.AIConfigHandler.GetConfig)
	aiConfig.Put("/", handlers.AIConfigHandler.SaveConfig)

	// Content routes (under tenant isolation)
	content := protected.Group("/contents", tenantMW)
	content.Post("/", handlers.ContentHandler.Upload)
	content.Post("/upload/file", handlers.ContentHandler.UploadFile)
	content.Get("/", handlers.ContentHandler.List)
	content.Get("/:id", handlers.ContentHandler.GetByID)
	content.Put("/:id/status", handlers.ContentHandler.UpdateStatus)

	// Review routes (under tenant isolation)
	review := protected.Group("/review", tenantMW)
	review.Post("/human", handlers.ReviewHandler.HumanReview)
	review.Post("/batch", handlers.ReviewHandler.BatchReview)
	review.Put("/appeal/:id", handlers.ReviewHandler.ResolveAppeal)
	review.Get("/pending", handlers.ReviewHandler.ListPending)
	review.Get("/stats", handlers.ReviewHandler.ElementStats)
	review.Get("/content/:contentId", handlers.ReviewHandler.GetElementsByContent)
	review.Get("/logs", handlers.ReviewHandler.ListAuditLogs)
	review.Get("/logs/search", handlers.ReviewHandler.SearchAuditLogs)
	review.Get("/ws", websocket.New(handlers.ReviewHandler.WebSocket))

	// Appeal routes (under tenant isolation)
	appeal := protected.Group("/appeals", tenantMW)
	appeal.Post("/", handlers.AppealHandler.Submit)
	appeal.Get("/", handlers.AppealHandler.ListByStatus)
	appeal.Get("/:id", handlers.AppealHandler.GetByID)
	// NOTE: no PUT /:id here — appeal resolution must go through
	// PUT /review/appeal/:id (ResolveAppeal), which enforces the
	// already-resolved guard and notifies the applicant. The old bare
	// update endpoint bypassed both (P1 backdoor, removed).

	// Dashboard routes (under tenant isolation)
	dashboard := protected.Group("/dashboard", tenantMW)
	dashboard.Get("/stats", handlers.DashboardHandler.GetStats)
	dashboard.Get("/reviewers", handlers.DashboardHandler.GetReviewerPerformance)
	dashboard.Get("/trend", handlers.DashboardHandler.GetDailyTrend)

	// Quality audit routes (under tenant isolation)
	quality := protected.Group("/quality", tenantMW)
	quality.Post("/batches", handlers.QualityAuditHandler.CreateBatch)
	quality.Get("/batches", handlers.QualityAuditHandler.ListBatches)
	quality.Get("/batches/:id", handlers.QualityAuditHandler.GetBatch)
	quality.Post("/batches/:id/start", handlers.QualityAuditHandler.StartBatch)
	quality.Post("/batches/:id/complete", handlers.QualityAuditHandler.CompleteBatch)
	quality.Post("/batches/:id/records", handlers.QualityAuditHandler.SubmitQARecord)
	quality.Get("/batches/:id/stats", handlers.QualityAuditHandler.GetBatchStats)
	quality.Get("/batches/:id/records", handlers.QualityAuditHandler.GetQARecords)

	// Live wall routes (under tenant isolation)
	live := protected.Group("/live", tenantMW)
	live.Post("/streams", handlers.LiveWallHandler.StartStream)
	live.Delete("/streams/:id", handlers.LiveWallHandler.StopStream)
	live.Get("/wall", handlers.LiveWallHandler.GetLiveWall)
	live.Post("/streams/:id/snapshot", handlers.LiveWallHandler.CreateSnapshot)
	live.Get("/wall/count", handlers.LiveWallHandler.GetStreamCount)
	live.Get("/ws", websocket.New(handlers.LiveWallHandler.WebSocket))

	// --- Public health endpoint (no auth required) ---
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"version": "0.1.0",
		})
	})

	// --- Platform-admin only routes ---
	admin := protected.Group("/admin", middleware.RequirePlatformAdmin())
	admin.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"version": "0.1.0",
		})
	})
}
