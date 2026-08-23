package middleware

import (
	"audit-platform/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RequirePlatformAdmin returns a Fiber middleware that rejects requests unless
// the authenticated user has the "platform_admin" role.
func RequirePlatformAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || role != "platform_admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    fiber.StatusForbidden,
				"message": "platform admin access required",
			})
		}
		return c.Next()
	}
}

// Tenant returns a Fiber middleware that enforces tenant isolation.
//
// It reads the X-Tenant-ID header, verifies that the requesting user belongs
// to that tenant, and stores the resolved tenant ID in c.Locals("tenant_id").
// Platform admins bypass tenant checks entirely.
func Tenant(db *pgxpool.Pool, cfg *config.Config) fiber.Handler {
	auth := Auth(cfg) // reuse the auth extractor

	return func(c *fiber.Ctx) error {
		// First run auth to populate c.Locals("role") and c.Locals("user_id").
		if err := auth(c); err != nil {
			return err
		}

		// Platform admins can access all tenants.
		role, ok := c.Locals("role").(string)
		if !ok {
			return c.JSON(fiber.Map{
				"code":    500,
				"message": "internal error: role not found in context",
			})
		}
		if role == "platform_admin" {
			// Strip the header so downstream handlers know it was overridden.
			c.Locals("tenant_bypass", true)
			return c.Next()
		}

		tenantHeader := c.Get("X-Tenant-ID")
		if tenantHeader == "" {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		tenantID, err := uuid.Parse(tenantHeader)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": map[string]interface{}{
					"message": "invalid tenant ID format",
					"code":    fiber.StatusBadRequest,
				},
			})
		}

		// Verify the user belongs to the requested tenant.
		var exists bool
		query := `
			SELECT EXISTS(
				SELECT 1 FROM users
				WHERE id = $1 AND tenant_id = $2
			)`
		row := db.QueryRow(c.UserContext(), query, c.Locals("user_id"), tenantID)
		if err := row.Scan(&exists); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": map[string]interface{}{
					"message": "failed to verify tenant membership",
					"code":    fiber.StatusInternalServerError,
				},
			})
		}

		if !exists {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": map[string]interface{}{
					"message": "access denied to this tenant",
					"code":    fiber.StatusForbidden,
				},
			})
		}

		c.Locals("tenant_id", tenantID.String())
		return c.Next()
	}
}
