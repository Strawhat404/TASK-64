package middleware

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"compliance-console/internal/models"

	"github.com/labstack/echo/v4"
)

// RateLimiter returns Echo middleware that enforces rate limits using a
// sliding-window approach backed by the rate_limits table.
func RateLimiter(db *sql.DB, maxRequests int, windowMinutes int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			now := time.Now().UTC()
			windowStart := now.Truncate(time.Duration(windowMinutes) * time.Minute)
			windowEnd := windowStart.Add(time.Duration(windowMinutes) * time.Minute)

			// Determine identifiers
			var userIdentifier string
			user := GetUserFromContext(c)
			if user != nil {
				userIdentifier = user.ID.String()
			}
			ipIdentifier := c.RealIP()

			// Check and increment user-based rate limit
			if userIdentifier != "" {
				remaining, err := checkAndIncrement(db, userIdentifier, "user", windowStart, maxRequests)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Rate limit check failed",
					})
				}
				if remaining < 0 {
					retryAfter := int(time.Until(windowEnd).Seconds())
					if retryAfter < 1 {
						retryAfter = 1
					}
					c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfter))
					c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
					c.Response().Header().Set("X-RateLimit-Remaining", "0")
					c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(windowEnd.Unix(), 10))
					return c.JSON(http.StatusTooManyRequests, map[string]string{
						"error": "Rate limit exceeded",
					})
				}

				c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
				c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(windowEnd.Unix(), 10))
			}

			// Check and increment IP-based rate limit
			remaining, err := checkAndIncrement(db, ipIdentifier, "ip", windowStart, maxRequests)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Rate limit check failed",
				})
			}
			if remaining < 0 {
				retryAfter := int(time.Until(windowEnd).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfter))
				c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
				c.Response().Header().Set("X-RateLimit-Remaining", "0")
				c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(windowEnd.Unix(), 10))
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Rate limit exceeded",
				})
			}

			// Set headers if not already set by user-based check
			if userIdentifier == "" {
				c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
				c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(windowEnd.Unix(), 10))
			}

			return next(c)
		}
	}
}

// checkAndIncrement checks the current request count for an identifier in the
// given window and increments it. Returns the remaining requests, or a negative
// value if the limit has been exceeded.
func checkAndIncrement(db *sql.DB, identifier, identifierType string, windowStart time.Time, maxRequests int) (int, error) {
	var count int
	err := db.QueryRow(`
		INSERT INTO rate_limits (identifier, identifier_type, window_start, request_count)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (identifier, identifier_type, window_start)
		DO UPDATE SET request_count = rate_limits.request_count + 1
		RETURNING request_count
	`, identifier, identifierType, windowStart).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert rate limit: %w", err)
	}

	remaining := maxRequests - count
	return remaining, nil
}

// CleanupOldWindows deletes rate_limits entries older than 5 minutes.
func CleanupOldWindows(db *sql.DB) error {
	cutoff := time.Now().UTC().Add(-5 * time.Minute)
	_, err := db.Exec(`DELETE FROM rate_limits WHERE window_start < $1`, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup old rate limit windows: %w", err)
	}
	return nil
}

// GetRateLimitStatus returns the current rate limit status for a given identifier.
func GetRateLimitStatus(db *sql.DB, identifier, identifierType string, maxRequests, windowMinutes int) (*models.RateLimitStatus, error) {
	now := time.Now().UTC()
	windowStart := now.Truncate(time.Duration(windowMinutes) * time.Minute)
	windowEnd := windowStart.Add(time.Duration(windowMinutes) * time.Minute)

	var count int
	err := db.QueryRow(`
		SELECT COALESCE(request_count, 0) FROM rate_limits
		WHERE identifier = $1 AND identifier_type = $2 AND window_start = $3
	`, identifier, identifierType, windowStart).Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		return nil, fmt.Errorf("failed to get rate limit status: %w", err)
	}

	remaining := maxRequests - count
	if remaining < 0 {
		remaining = 0
	}

	return &models.RateLimitStatus{
		Identifier:        identifier,
		Type:              identifierType,
		RequestsRemaining: remaining,
		WindowReset:       windowEnd,
	}, nil
}
