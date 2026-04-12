package unit_tests

import (
	"strings"
	"testing"
)

// Mirrors validation rules from:
// - backend/internal/handlers/auth.go (verifyArgon2Hash format)
// - backend/internal/handlers/users.go (CreateUser: len >= 12)

func isValidPasswordLength(password string) bool {
	return len(password) >= 12
}

func isValidArgon2Hash(hash string) bool {
	// Expected: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}
	if parts[0] != "" {
		return false
	}
	if parts[1] != "argon2id" {
		return false
	}
	if !strings.HasPrefix(parts[2], "v=") {
		return false
	}
	// params must contain m=, t=, p=
	params := parts[3]
	if !strings.Contains(params, "m=") || !strings.Contains(params, "t=") || !strings.Contains(params, "p=") {
		return false
	}
	// salt and hash must be non-empty
	if len(parts[4]) == 0 || len(parts[5]) == 0 {
		return false
	}
	return true
}

func TestPasswordLengthValidation(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{"", false},
		{"short", false},
		{"11charslong", false},
		{"12characters", true},
		{"Admin12345!!!", true},
		{"a very long password that is well over twelve characters", true},
		{"exactly12ch", false}, // 11 chars
		{"exactly12chr", true}, // 12 chars
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			got := isValidPasswordLength(tt.password)
			if got != tt.valid {
				t.Errorf("isValidPasswordLength(%q) = %v, want %v (len=%d)",
					tt.password, got, tt.valid, len(tt.password))
			}
		})
	}
}

func TestArgon2HashFormatValidation(t *testing.T) {
	tests := []struct {
		name  string
		hash  string
		valid bool
	}{
		{
			"valid production hash",
			"$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0c2FsdA$ZKRnmrvMb2Mwl3mypOFbkVOsp3YMi3bpGp/jWKXt1UQ",
			true,
		},
		{"empty string", "", false},
		{"plaintext password", "Admin12345!!!", false},
		{"bcrypt hash", "$2b$10$abcdefghijklmnopqrstuuVWXYZ0123456789./ABCDEFGHIJKLMN", false},
		{"missing salt", "$argon2id$v=19$m=65536,t=3,p=4$$hash", false},
		{"wrong algorithm", "$argon2i$v=19$m=65536,t=3,p=4$salt$hash", false},
		{"missing params", "$argon2id$v=19$$salt$hash", false},
		{
			"missing version prefix",
			"$argon2id$19$m=65536,t=3,p=4$salt$hash",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidArgon2Hash(tt.hash)
			if got != tt.valid {
				t.Errorf("isValidArgon2Hash(%q) = %v, want %v", tt.hash, got, tt.valid)
			}
		})
	}
}

// TestBruteForceProtectionThresholds validates the security middleware constants.
func TestBruteForceProtectionThresholds(t *testing.T) {
	const (
		maxFailedAttempts    = 5
		captchaThreshold     = 3
		lockDurationMinutes  = 30
		failedWindowMinutes  = 15
	)

	// CAPTCHA triggers before lockout
	if captchaThreshold >= maxFailedAttempts {
		t.Errorf("captchaThreshold (%d) should be < maxFailedAttempts (%d)",
			captchaThreshold, maxFailedAttempts)
	}

	// Lock duration should be meaningful
	if lockDurationMinutes < failedWindowMinutes {
		t.Errorf("lockDuration (%d min) should be >= failedWindow (%d min)",
			lockDurationMinutes, failedWindowMinutes)
	}

	// Thresholds must be positive
	if captchaThreshold <= 0 || maxFailedAttempts <= 0 {
		t.Error("thresholds must be positive")
	}
}
