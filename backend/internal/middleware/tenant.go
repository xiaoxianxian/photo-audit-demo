package middleware

import (
	"database/sql"
	"errors"

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

// RequireTenantAdmin returns a Fiber middleware that allows only
// "tenant_admin" and "platform_admin" roles. Must run AFTER the auth
// middleware (it reads role from c.Locals). Used for tenant write operations
// (create/update/delete) so regular reviewers cannot mutate tenants.
func RequireTenantAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || (role != "tenant_admin" && role != "platform_admin") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    fiber.StatusForbidden,
				"message": "tenant admin access required",
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
//
// NOTE: this middleware must run AFTER the Auth middleware on the same route.
// It reads role/user_id from c.Locals instead of re-running the auth extractor —
// calling auth(c) directly here would advance fiber's internal route index twice
// (auth ends with c.Next()) and break request routing (manifests as spurious 404s).
func Tenant(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Platform admins can access all tenants.
		role, ok := c.Locals("role").(string)
		if !ok {
			return c.JSON(fiber.Map{
				"code":    500,
				"message": "internal error: role not found in context",
			})
		}
		if role == "platform_admin" {
			// Mark that downstream handlers may ignore the tenant header.
			c.Locals("tenant_bypass", true)
			if hdr := c.Get("X-Tenant-ID"); hdr != "" {
				if tid, err := uuid.Parse(hdr); err == nil {
					c.Locals("tenant_id", tid.String())
				}
			}
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
		userIDLocal, ok := c.Locals("user_id").(string)
		if !ok || userIDLocal == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": map[string]interface{}{
					"message": "authentication required",
					"code":    fiber.StatusUnauthorized,
				},
			})
		}
		userID, err := uuid.Parse(userIDLocal)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": map[string]interface{}{
					"message": "invalid user identity",
					"code":    fiber.StatusUnauthorized,
				},
			})
		}

		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND tenant_id = $2)`
		if err := db.QueryRow(c.UserContext(), query, userID, tenantID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				exists = false
			} else {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": map[string]interface{}{
						"message": "failed to verify tenant membership",
						"code":    fiber.StatusInternalServerError,
					},
				})
			}
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
