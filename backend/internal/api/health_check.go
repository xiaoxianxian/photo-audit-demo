package api

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthChecker provides health check endpoints for the service
type HealthChecker struct {
	pool   *pgxpool.Pool
	checks map[string]HealthCheck
}

// HealthCheck defines a health check interface
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
}

// DBHealthCheck checks database connectivity via pgxpool
type DBHealthCheck struct {
	Pool *pgxpool.Pool
}

func (h *DBHealthCheck) Name() string {
	return "database"
}

func (h *DBHealthCheck) Check(ctx context.Context) error {
	if h.Pool == nil {
		return fmt.Errorf("database pool not initialized")
	}
	conn, err := h.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	return conn.Conn().Ping(ctx)
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(pool *pgxpool.Pool) *HealthChecker {
	checker := &HealthChecker{
		checks: make(map[string]HealthCheck),
	}

	if pool != nil {
		checker.checks["database"] = &DBHealthCheck{Pool: pool}
	}

	return checker
}

// HealthHandler handles GET /api/v1/health - basic health check
func HealthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "1.0.0",
		})
	}
}

// ReadyHandler handles GET /api/v1/ready - readiness check (checks dependencies)
func (hc *HealthChecker) ReadyHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		checkResults := make(map[string]string)
		allHealthy := true

		for name, check := range hc.checks {
			if err := check.Check(ctx); err != nil {
				checkResults[name] = fmt.Sprintf("unhealthy: %v", err)
				allHealthy = false
			} else {
				checkResults[name] = "healthy"
			}
		}

		statusCode := fiber.StatusOK
		if !allHealthy {
			statusCode = fiber.StatusServiceUnavailable
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status":      map[bool]string{true: "ready", false: "not_ready"}[allHealthy],
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"checks":      checkResults,
			"all_healthy": allHealthy,
		})
	}
}

// LiveHandler handles GET /api/v1/live - liveness check
func LiveHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		memStats := new(runtime.MemStats)
		runtime.ReadMemStats(memStats)

		return c.JSON(fiber.Map{
			"status":            "alive",
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
			"uptime":            time.Since(startTime).String(),
			"goroutines":        runtime.NumGoroutine(),
			"memory_alloc":      memStats.Alloc,
			"memory_total_alloc": memStats.TotalAlloc,
			"sys_memory":        memStats.Sys,
		})
	}
}

// Global start time for uptime calculation
var startTime = time.Now()

// SetupHealthRoutes registers health check routes
func SetupHealthRoutes(app *fiber.App, checker *HealthChecker) {
	health := app.Group("/api/v1")

	// Basic health check
	health.Get("/health", HealthHandler())

	// Readiness check (checks dependencies)
	health.Get("/ready", checker.ReadyHandler())

	// Liveness check
	health.Get("/live", LiveHandler())
}
