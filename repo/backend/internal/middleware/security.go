package middleware

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	MaxFailedAttempts       = 5
	CaptchaThreshold        = 3
	LockDuration            = 30 * time.Minute
	FailedAttemptWindow     = 15 * time.Minute
)

// CheckAccountLock verifies whether the user account is currently locked.
// Returns an error message if the account is locked, nil otherwise.
func CheckAccountLock(db *sql.DB, username string) error {
	var lockedUntil *time.Time
	err := db.QueryRow(
		"SELECT locked_until FROM users WHERE username = $1",
		username,
	).Scan(&lockedUntil)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // User not found; let the login handler deal with it
		}
		return fmt.Errorf("failed to check account lock status")
	}

	if lockedUntil != nil && time.Now().Before(*lockedUntil) {
		remaining := time.Until(*lockedUntil).Round(time.Minute)
		return fmt.Errorf("account is locked, try again in %v", remaining)
	}

	// If lock has expired, clear it
	if lockedUntil != nil {
		_, _ = db.Exec(`
			UPDATE users
			SET locked_until = NULL, failed_login_attempts = 0, captcha_required = FALSE, updated_at = NOW()
			WHERE username = $1
		`, username)
	}

	return nil
}

// TrackFailedLogin increments the failed login counter and applies
// progressive security measures (CAPTCHA requirement, account lock).
func TrackFailedLogin(db *sql.DB, username string) {
	// Check if the last failure was within the rolling window
	var lastFailedAt *time.Time
	var failedAttempts int
	_ = db.QueryRow(
		"SELECT last_failed_at, failed_login_attempts FROM users WHERE username = $1",
		username,
	).Scan(&lastFailedAt, &failedAttempts)

	// If outside the 15-minute window, reset the counter
	if lastFailedAt == nil || time.Since(*lastFailedAt) > FailedAttemptWindow {
		_, _ = db.Exec(`
			UPDATE users
			SET failed_login_attempts = 1,
			    last_failed_at = NOW(),
			    updated_at = NOW()
			WHERE username = $1
		`, username)
		failedAttempts = 1
	} else {
		// Within window — increment
		_, _ = db.Exec(`
			UPDATE users
			SET failed_login_attempts = failed_login_attempts + 1,
			    last_failed_at = NOW(),
			    updated_at = NOW()
			WHERE username = $1
		`, username)
		failedAttempts++
	}

	// Require CAPTCHA after 3 failed attempts in window
	if failedAttempts >= CaptchaThreshold {
		_, _ = db.Exec(`
			UPDATE users SET captcha_required = TRUE, updated_at = NOW()
			WHERE username = $1
		`, username)
	}

	// Lock account after 5 failed attempts in window
	if failedAttempts >= MaxFailedAttempts {
		lockUntil := time.Now().Add(LockDuration)
		_, _ = db.Exec(`
			UPDATE users SET locked_until = $1, updated_at = NOW()
			WHERE username = $2
		`, lockUntil, username)
	}
}

// ResetLoginAttempts clears the failed login counter and related flags
// after a successful login.
func ResetLoginAttempts(db *sql.DB, username string) {
	_, _ = db.Exec(`
		UPDATE users
		SET failed_login_attempts = 0,
		    locked_until = NULL,
		    captcha_required = FALSE,
		    last_login_at = NOW(),
		    updated_at = NOW()
		WHERE username = $1
	`, username)
}
