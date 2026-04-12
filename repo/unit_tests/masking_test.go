package unit_tests

import (
	"strings"
	"testing"
)

// maskValue mirrors production logic in backend/internal/services/encryption.go.
func maskValue(value string) string {
	if len(value) <= 4 {
		return "**** "
	}
	return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"full SSN", "123-45-6789", "*******6789"},
		{"bank account", "9876543210", "******3210"},
		{"credit card", "4111111111111111", "************1111"},
		{"short value 4 chars", "1234", "**** "},
		{"short value 3 chars", "123", "**** "},
		{"short value 1 char", "X", "**** "},
		{"empty string", "", "**** "},
		{"exactly 5 chars", "12345", "*2345"},
		{"tax ID", "12-3456789", "******6789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskValue(tt.input)
			if got != tt.want {
				t.Errorf("maskValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMaskValueNeverLeaksFullValue(t *testing.T) {
	values := []string{
		"123-45-6789",
		"4111111111111111",
		"sensitive-data-here",
	}
	for _, v := range values {
		masked := maskValue(v)
		// The masked value must NOT equal the original
		if masked == v {
			t.Errorf("maskValue(%q) returned original value unchanged", v)
		}
		// Must contain at least one asterisk
		if !strings.Contains(masked, "*") {
			t.Errorf("maskValue(%q) = %q — contains no masking", v, masked)
		}
	}
}

func TestMaskValuePreservesLastFour(t *testing.T) {
	values := []string{
		"1234567890",
		"ABCDEFGHIJ",
		"sensitive-data",
	}
	for _, v := range values {
		masked := maskValue(v)
		if len(v) > 4 {
			lastFour := v[len(v)-4:]
			if !strings.HasSuffix(masked, lastFour) {
				t.Errorf("maskValue(%q) = %q — does not end with %q", v, masked, lastFour)
			}
		}
	}
}
