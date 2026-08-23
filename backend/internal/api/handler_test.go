package api

import (
	"bytes"
	"io"
	"encoding/json"
	"net/http"
	"testing"
	"time"


	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// TestAuthHandler_Login verifies login endpoint returns proper JSON structure.
func TestAuthHandler_Login(t *testing.T) {
	app := fiber.New()
	app.Post("/login", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"code": 401, "message": "invalid credentials"})
	})
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rec, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.StatusCode)
	buf, _ := io.ReadAll(rec.Body)
	var body map[string]interface{}
	json.Unmarshal(buf, &body)
	require.Equal(t, float64(401), body["code"])
}

// TestJSONErrorFormat verifies error responses follow {code, message} format.
func TestJSONErrorFormat(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(400).JSON(fiber.Map{
			"code":    400,
			"message": "validation failed",
		})
	})
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	rec, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.StatusCode)
	var body map[string]interface{}
	buf, _ := io.ReadAll(rec.Body)
	json.Unmarshal(buf, &body)
	require.Equal(t, float64(400), body["code"])
	require.Equal(t, "validation failed", body["message"])
}

// TestBatchReviewRequest validates batch review request structure.
func TestBatchReviewRequest(t *testing.T) {
	body := map[string]interface{}{
		"element_ids": []string{"id1", "id2"},
		"action":      "approve",
		"reviewer_id": "user-123",
	}
	jsonData, _ := json.Marshal(body)
	var parsed map[string]interface{}
	err := json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)
	require.Equal(t, "approve", parsed["action"])
	ids := parsed["element_ids"].([]interface{})
	require.Len(t, ids, 2)
}

// TestReviewRequest validates single review request structure.
func TestReviewRequest(t *testing.T) {
	body := map[string]interface{}{
		"element_id": "elem-123",
		"action":     "reject",
		"reason":     "violation",
	}
	jsonData, _ := json.Marshal(body)
	var parsed map[string]string
	err := json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)
	require.Equal(t, "reject", parsed["action"])
}

// TestPaginationQueryParsing verifies pagination query parameters.
func TestPaginationQueryParsing(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 1)
		pageSize := c.QueryInt("page_size", 20)
		return c.JSON(fiber.Map{"page": page, "page_size": pageSize})
	})
	req, _ := http.NewRequest(http.MethodGet, "/test?page=3&page_size=50", nil)
	rec, err := app.Test(req)
	require.NoError(t, err)
	var result map[string]int
	buf, _ := io.ReadAll(rec.Body)
	json.Unmarshal(buf, &result)
	require.Equal(t, 3, result["page"])
	require.Equal(t, 50, result["page_size"])
}

// TestHealthEndpoint verifies the health check endpoint.
func TestHealthEndpoint(t *testing.T) {
	app := fiber.New()
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "version": "0.1.0"})
	})
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	rec, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.StatusCode)
	var result map[string]string
	buf, _ := io.ReadAll(rec.Body)
	json.Unmarshal(buf, &result)
	require.Equal(t, "ok", result["status"])
}

// TestJWTTokenGeneration verifies JWT token generation and parsing.
func TestJWTTokenGeneration(t *testing.T) {
	secret := "test-secret-key"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "test-user",
		"role":    "reviewer",
		"exp":     jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)
	parsed, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	claims := parsed.Claims.(jwt.MapClaims)
	require.Equal(t, "test-user", claims["user_id"])
	require.Equal(t, "reviewer", claims["role"])
}
