package middleware

import (
	"strings"

	"audit-platform/internal/config"
	"audit-platform/internal/model"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AuthFactory builds the JWT validation key function from config.
func AuthFactory(secret string) func(token *jwt.Token) (interface{}, error) {
	return func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
		}
		return []byte(secret), nil
	}
}

// Auth returns a Fiber middleware that validates a Bearer JWT in the
// Authorization header. On success it populates c.Locals("user_id") and
// c.Locals("role"); on failure it returns 401 with a consistent error body.
func Auth(cfg *config.Config) fiber.Handler {
	keyFunc := AuthFactory(cfg.JWTSecret)

	return func(c *fiber.Ctx) error {
		authorization := c.Get("Authorization")
		if authorization == "" {
			return c.JSON(fiber.Map{
				"code":    401,
				"message": "missing authorization header",
			})
		}

		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.JSON(fiber.Map{
				"code":    401,
				"message": "invalid authorization format",
			})
		}

		tokenString := parts[1]

		token, err := jwt.ParseWithClaims(tokenString, &model.JWTClaims{}, keyFunc)
		if err != nil || !token.Valid {
			return c.JSON(fiber.Map{
				"code":    401,
				"message": "invalid or expired token",
			})
		}

		claims, ok := token.Claims.(*model.JWTClaims)
		if !ok {
			return c.JSON(fiber.Map{
				"code":    401,
				"message": "invalid token claims",
			})
		}

		c.Locals("user_id", claims.UserID.String())
		c.Locals("role", claims.Role)
		if claims.TenantID != nil {
			c.Locals("tenant_id", claims.TenantID.String())
		}

		return c.Next()
	}
}
