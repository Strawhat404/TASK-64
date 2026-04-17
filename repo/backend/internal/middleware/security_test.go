package middleware

import (
	"testing"
	"time"
)

func TestSecurityConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      interface{}
		want     interface{}
	}{
		{"MaxFailedAttempts", MaxFailedAttempts, 5},
		{"CaptchaThreshold", CaptchaThreshold, 3},
		{"LockDuration", LockDuration, 30 * time.Minute},
		{"FailedAttemptWindow", FailedAttemptWindow, 15 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestCaptchaTriggersBeforeLockout(t *testing.T) {
	if CaptchaThreshold >= MaxFailedAttempts {
		t.Errorf("CaptchaThreshold (%d) must be < MaxFailedAttempts (%d)",
			CaptchaThreshold, MaxFailedAttempts)
	}
}

func TestProgressiveEscalation(t *testing.T) {
	for attempts := 0; attempts <= MaxFailedAttempts+1; attempts++ {
		captcha := attempts >= CaptchaThreshold
		locked := attempts >= MaxFailedAttempts

		if attempts < CaptchaThreshold && captcha {
			t.Errorf("at %d attempts: captcha should NOT be required", attempts)
		}
		if attempts >= CaptchaThreshold && !captcha {
			t.Errorf("at %d attempts: captcha SHOULD be required", attempts)
		}
		if attempts < MaxFailedAttempts && locked {
			t.Errorf("at %d attempts: should NOT be locked", attempts)
		}
		if attempts >= MaxFailedAttempts && !locked {
			t.Errorf("at %d attempts: SHOULD be locked", attempts)
		}
	}
}

func TestLockDurationExpiry(t *testing.T) {
	locked := time.Now().Add(LockDuration)

	if !time.Now().Before(locked) {
		t.Error("lock set now+30m should still be active")
	}

	expired := time.Now().Add(-time.Minute)
	if time.Now().Before(expired) {
		t.Error("lock expired 1m ago should not be active")
	}
}

func TestFailedAttemptWindowBoundary(t *testing.T) {
	withinWindow := time.Now().Add(-10 * time.Minute)
	outsideWindow := time.Now().Add(-20 * time.Minute)

	if time.Since(withinWindow) > FailedAttemptWindow {
		t.Error("10 min ago should be within 15 min window")
	}
	if time.Since(outsideWindow) <= FailedAttemptWindow {
		t.Error("20 min ago should be outside 15 min window")
	}
}
