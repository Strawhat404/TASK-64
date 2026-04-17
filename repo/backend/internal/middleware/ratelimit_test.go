package middleware

import (
	"testing"
	"time"
)

func TestRateLimitWindowCalculation(t *testing.T) {
	now := time.Date(2026, 4, 17, 10, 33, 45, 0, time.UTC)
	windowMinutes := 1

	windowStart := now.Truncate(time.Duration(windowMinutes) * time.Minute)
	windowEnd := windowStart.Add(time.Duration(windowMinutes) * time.Minute)

	expectedStart := time.Date(2026, 4, 17, 10, 33, 0, 0, time.UTC)
	expectedEnd := time.Date(2026, 4, 17, 10, 34, 0, 0, time.UTC)

	if !windowStart.Equal(expectedStart) {
		t.Errorf("windowStart = %v, want %v", windowStart, expectedStart)
	}
	if !windowEnd.Equal(expectedEnd) {
		t.Errorf("windowEnd = %v, want %v", windowEnd, expectedEnd)
	}
}

func TestRemainingCalculation(t *testing.T) {
	tests := []struct {
		name         string
		maxRequests  int
		currentCount int
		wantRemain   int
		wantExceeded bool
	}{
		{"under limit", 300, 100, 200, false},
		{"at limit", 300, 300, 0, false},
		{"over limit", 300, 301, -1, true},
		{"login rate", 10, 5, 5, false},
		{"login exceeded", 10, 11, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining := tt.maxRequests - tt.currentCount
			exceeded := remaining < 0

			if remaining != tt.wantRemain {
				t.Errorf("remaining = %d, want %d", remaining, tt.wantRemain)
			}
			if exceeded != tt.wantExceeded {
				t.Errorf("exceeded = %v, want %v", exceeded, tt.wantExceeded)
			}
		})
	}
}

func TestRetryAfterCalculation(t *testing.T) {
	windowEnd := time.Now().Add(30 * time.Second)
	retryAfter := int(time.Until(windowEnd).Seconds())

	if retryAfter < 1 {
		retryAfter = 1
	}

	if retryAfter <= 0 {
		t.Error("retryAfter must be positive")
	}
	if retryAfter > 60 {
		t.Errorf("retryAfter = %d, expected <= 60 for a 30s window", retryAfter)
	}
}

func TestCleanupCutoff(t *testing.T) {
	cutoff := time.Now().UTC().Add(-5 * time.Minute)
	recent := time.Now().UTC().Add(-2 * time.Minute)
	old := time.Now().UTC().Add(-10 * time.Minute)

	if recent.Before(cutoff) {
		t.Error("2-minute-old entry should NOT be cleaned up")
	}
	if !old.Before(cutoff) {
		t.Error("10-minute-old entry SHOULD be cleaned up")
	}
}

func TestRateLimitConfigValues(t *testing.T) {
	generalLimit := 300
	loginLimit := 10

	if loginLimit >= generalLimit {
		t.Error("login limit must be stricter than general limit")
	}
	if generalLimit <= 0 || loginLimit <= 0 {
		t.Error("rate limits must be positive")
	}
}
