package middleware

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"compliance-console/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	SessionCookieName = "session_token"
	SessionIdleTimeout = 30 * time.Minute
)

// AuthMiddleware validates session tokens and refreshes idle timeout.
func AuthMiddleware(db *sql.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			token := cookie.Value

			var session models.Session
			var user models.User
			var roleName string
			err = db.QueryRow(`
				SELECT s.id, s.user_id, s.token, s.expires_at, s.created_at,
				       u.id, u.tenant_id, u.role_id, u.username, u.email, u.full_name,
				       u.failed_login_attempts, u.locked_until, u.captcha_required,
				       u.last_login_at, u.created_at, u.updated_at, u.is_active,
				       u.location_id,
				       r.name
				FROM sessions s
				JOIN users u ON s.user_id = u.id
				JOIN roles r ON u.role_id = r.id
				WHERE s.token = $1
			`, token).Scan(
				&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt,
				&user.ID, &user.TenantID, &user.RoleID, &user.Username, &user.Email, &user.FullName,
				&user.FailedLoginAttempts, &user.LockedUntil, &user.CaptchaRequired,
				&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.IsActive,
				&user.LocationID,
				&roleName,
			)
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or expired session",
				})
			}
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to validate session",
				})
			}

			if time.Now().After(session.ExpiresAt) {
				// Clean up expired session
				_, _ = db.Exec("DELETE FROM sessions WHERE id = $1", session.ID)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Session expired",
				})
			}

			if !user.IsActive {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Account is deactivated",
				})
			}

			// Refresh idle timeout
			newExpiry := time.Now().Add(SessionIdleTimeout)
			_, _ = db.Exec("UPDATE sessions SET expires_at = $1 WHERE id = $2", newExpiry, session.ID)

			user.RoleName = roleName

			// Store user info in context
			c.Set("user", &user)
			c.Set("session", &session)
			c.Set("tenant_id", user.TenantID)

			return next(c)
		}
	}
}

// RoleGuard returns middleware that restricts access to specified roles.
func RoleGuard(allowedRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get("user").(*models.User)
			if !ok || user == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			for _, role := range allowedRoles {
				if user.RoleName == role {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Insufficient permissions",
			})
		}
	}
}

// CaptchaCheck is a placeholder middleware that checks the captcha_required flag.
func CaptchaCheck(db *sql.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// This is a placeholder for CAPTCHA verification.
			// In a real implementation, this would validate a CAPTCHA token
			// from the request against a CAPTCHA service.

			var req models.LoginRequest
			// Only applicable to login requests
			if c.Request().URL.Path != "/api/auth/login" {
				return next(c)
			}

			// Peek at the username to check if captcha is required
			body, readErr := io.ReadAll(c.Request().Body)
			if readErr != nil {
				return next(c)
			}
			// Reset body for downstream handler
			c.Request().Body = io.NopCloser(bytes.NewReader(body))
			if err := json.Unmarshal(body, &req); err != nil {
				return next(c)
			}

			var captchaRequired bool
			err := db.QueryRow(
				"SELECT captcha_required FROM users WHERE username = $1",
				req.Username,
			).Scan(&captchaRequired)
			if err != nil {
				// User not found, let the login handler deal with it
				return next(c)
			}

			if captchaRequired {
				captchaToken := c.Request().Header.Get("X-Captcha-Token")
				if captchaToken == "" {
					// Generate a random math challenge
					a := rand.Intn(20) + 1
					b := rand.Intn(20) + 1
					answer := a + b
					// Store the answer in the database
					_, _ = db.Exec(
						"UPDATE users SET captcha_answer = $1, updated_at = NOW() WHERE username = $2",
						answer, req.Username,
					)
					return c.JSON(http.StatusBadRequest, map[string]string{
						"error":             "CAPTCHA verification required",
						"captcha_required":  "true",
						"captcha_challenge": fmt.Sprintf("What is %d + %d?", a, b),
					})
				}
				// Validate the CAPTCHA answer
				userAnswer, parseErr := strconv.Atoi(captchaToken)
				if parseErr != nil {
					// Generate a new challenge on bad input
					a := rand.Intn(20) + 1
					b := rand.Intn(20) + 1
					answer := a + b
					_, _ = db.Exec(
						"UPDATE users SET captcha_answer = $1, updated_at = NOW() WHERE username = $2",
						answer, req.Username,
					)
					return c.JSON(http.StatusBadRequest, map[string]string{
						"error":             "Invalid CAPTCHA answer",
						"captcha_required":  "true",
						"captcha_challenge": fmt.Sprintf("What is %d + %d?", a, b),
					})
				}
				var storedAnswer int
				err = db.QueryRow(
					"SELECT captcha_answer FROM users WHERE username = $1",
					req.Username,
				).Scan(&storedAnswer)
				if err != nil || userAnswer != storedAnswer {
					// Wrong answer — generate a new challenge
					a := rand.Intn(20) + 1
					b := rand.Intn(20) + 1
					newAnswer := a + b
					_, _ = db.Exec(
						"UPDATE users SET captcha_answer = $1, updated_at = NOW() WHERE username = $2",
						newAnswer, req.Username,
					)
					return c.JSON(http.StatusBadRequest, map[string]string{
						"error":             "Incorrect CAPTCHA answer",
						"captcha_required":  "true",
						"captcha_challenge": fmt.Sprintf("What is %d + %d?", a, b),
					})
				}
			}

			return next(c)
		}
	}
}

// GetUserFromContext extracts the authenticated user from the echo context.
func GetUserFromContext(c echo.Context) *models.User {
	user, ok := c.Get("user").(*models.User)
	if !ok {
		return nil
	}
	return user
}

// GetTenantIDFromContext extracts the tenant ID from the echo context.
func GetTenantIDFromContext(c echo.Context) uuid.UUID {
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return tenantID
}
