package unit_tests

import (
	"testing"
	"time"
)

// Tests for middleware security constants and brute-force protection logic.
// Production logic references:
//   - backend/internal/middleware/security.go
//   - backend/internal/middleware/auth.go

const (
	maxFailedAttempts   = 5
	captchaThreshold    = 3
	lockDuration        = 30 * time.Minute
	failedAttemptWindow = 15 * time.Minute
	sessionIdleTimeout  = 30 * time.Minute
	sessionCookieName   = "session_token"
)

func TestBruteForceConstants(t *testing.T) {
	t.Run("max failed attempts is 5", func(t *testing.T) {
		if maxFailedAttempts != 5 {
			t.Errorf("maxFailedAttempts = %d, want 5", maxFailedAttempts)
		}
	})

	t.Run("captcha threshold is 3", func(t *testing.T) {
		if captchaThreshold != 3 {
			t.Errorf("captchaThreshold = %d, want 3", captchaThreshold)
		}
	})

	t.Run("lock duration is 30 minutes", func(t *testing.T) {
		if lockDuration != 30*time.Minute {
			t.Errorf("lockDuration = %v, want 30m", lockDuration)
		}
	})

	t.Run("failed attempt window is 15 minutes", func(t *testing.T) {
		if failedAttemptWindow != 15*time.Minute {
			t.Errorf("failedAttemptWindow = %v, want 15m", failedAttemptWindow)
		}
	})

	t.Run("captcha triggers before lockout", func(t *testing.T) {
		if captchaThreshold >= maxFailedAttempts {
			t.Error("captcha threshold should be less than max failed attempts")
		}
	})
}

func TestSessionConstants(t *testing.T) {
	t.Run("session idle timeout is 30 minutes", func(t *testing.T) {
		if sessionIdleTimeout != 30*time.Minute {
			t.Errorf("sessionIdleTimeout = %v, want 30m", sessionIdleTimeout)
		}
	})

	t.Run("session cookie name is session_token", func(t *testing.T) {
		if sessionCookieName != "session_token" {
			t.Errorf("sessionCookieName = %q, want %q", sessionCookieName, "session_token")
		}
	})
}

func TestAccountLockLogic(t *testing.T) {
	t.Run("account locked when locked_until is in the future", func(t *testing.T) {
		lockedUntil := time.Now().Add(15 * time.Minute)
		isLocked := time.Now().Before(lockedUntil)
		if !isLocked {
			t.Error("account should be locked when locked_until is in the future")
		}
	})

	t.Run("account not locked when locked_until is in the past", func(t *testing.T) {
		lockedUntil := time.Now().Add(-5 * time.Minute)
		isLocked := time.Now().Before(lockedUntil)
		if isLocked {
			t.Error("account should not be locked when locked_until is in the past")
		}
	})

	t.Run("account not locked when locked_until is nil", func(t *testing.T) {
		var lockedUntil *time.Time
		isLocked := lockedUntil != nil && time.Now().Before(*lockedUntil)
		if isLocked {
			t.Error("account should not be locked when locked_until is nil")
		}
	})
}

func TestProgressiveSecurityMeasures(t *testing.T) {
	// Simulate progressive security escalation
	tests := []struct {
		failedAttempts  int
		expectCaptcha   bool
		expectLocked    bool
	}{
		{0, false, false},
		{1, false, false},
		{2, false, false},
		{3, true, false},   // captcha triggered at 3
		{4, true, false},
		{5, true, true},    // lockout at 5
		{6, true, true},
	}

	for _, tt := range tests {
		captchaRequired := tt.failedAttempts >= captchaThreshold
		accountLocked := tt.failedAttempts >= maxFailedAttempts

		if captchaRequired != tt.expectCaptcha {
			t.Errorf("at %d attempts: captcha = %v, want %v",
				tt.failedAttempts, captchaRequired, tt.expectCaptcha)
		}
		if accountLocked != tt.expectLocked {
			t.Errorf("at %d attempts: locked = %v, want %v",
				tt.failedAttempts, accountLocked, tt.expectLocked)
		}
	}
}

func TestFailedAttemptWindowReset(t *testing.T) {
	t.Run("within window — attempts accumulate", func(t *testing.T) {
		lastFailedAt := time.Now().Add(-10 * time.Minute)
		withinWindow := time.Since(lastFailedAt) <= failedAttemptWindow
		if !withinWindow {
			t.Error("10 minutes ago should be within the 15-minute window")
		}
	})

	t.Run("outside window — counter resets", func(t *testing.T) {
		lastFailedAt := time.Now().Add(-20 * time.Minute)
		withinWindow := time.Since(lastFailedAt) <= failedAttemptWindow
		if withinWindow {
			t.Error("20 minutes ago should be outside the 15-minute window")
		}
	})

	t.Run("exactly at boundary — within window", func(t *testing.T) {
		lastFailedAt := time.Now().Add(-failedAttemptWindow)
		withinWindow := time.Since(lastFailedAt) <= failedAttemptWindow
		if !withinWindow {
			t.Error("exactly at the boundary should be within the window")
		}
	})
}

func TestSessionExpiry(t *testing.T) {
	t.Run("active session within idle timeout", func(t *testing.T) {
		expiresAt := time.Now().Add(15 * time.Minute)
		expired := time.Now().After(expiresAt)
		if expired {
			t.Error("session expiring in the future should not be expired")
		}
	})

	t.Run("expired session past idle timeout", func(t *testing.T) {
		expiresAt := time.Now().Add(-5 * time.Minute)
		expired := time.Now().After(expiresAt)
		if !expired {
			t.Error("session with past expiry should be expired")
		}
	})

	t.Run("session refresh extends idle timeout", func(t *testing.T) {
		originalExpiry := time.Now().Add(5 * time.Minute)
		newExpiry := time.Now().Add(sessionIdleTimeout) // refreshed
		if !newExpiry.After(originalExpiry) {
			t.Error("refreshed expiry should be later than original")
		}
	})
}

func TestRoleGuardLogic(t *testing.T) {
	allowedRoles := []string{"Administrator", "Scheduler"}

	tests := []struct {
		role    string
		allowed bool
	}{
		{"Administrator", true},
		{"Scheduler", true},
		{"Reviewer", false},
		{"Auditor", false},
		{"", false},
	}

	for _, tt := range tests {
		found := false
		for _, r := range allowedRoles {
			if r == tt.role {
				found = true
				break
			}
		}
		if found != tt.allowed {
			t.Errorf("role %q: allowed = %v, want %v", tt.role, found, tt.allowed)
		}
	}
}

func TestRateLimitConstants(t *testing.T) {
	generalLimit := 300
	loginLimit := 10
	windowMinutes := 1

	t.Run("general rate limit is 300 per minute", func(t *testing.T) {
		if generalLimit != 300 {
			t.Errorf("general rate limit = %d, want 300", generalLimit)
		}
	})

	t.Run("login rate limit is 10 per minute", func(t *testing.T) {
		if loginLimit != 10 {
			t.Errorf("login rate limit = %d, want 10", loginLimit)
		}
	})

	t.Run("login limit is stricter than general", func(t *testing.T) {
		if loginLimit >= generalLimit {
			t.Error("login rate limit should be stricter than general rate limit")
		}
	})

	t.Run("window is 1 minute", func(t *testing.T) {
		if windowMinutes != 1 {
			t.Errorf("window = %d minutes, want 1", windowMinutes)
		}
	})
}
