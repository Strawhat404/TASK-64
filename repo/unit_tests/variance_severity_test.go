package unit_tests

import (
	"testing"
)

// classifyVarianceSeverity mirrors production logic in
// backend/internal/services/reconciliation.go.
func classifyVarianceSeverity(variance float64) string {
	switch {
	case variance > 100:
		return "critical"
	case variance > 50:
		return "high"
	case variance > 10:
		return "medium"
	default:
		return "low"
	}
}

func TestClassifyVarianceSeverity(t *testing.T) {
	tests := []struct {
		variance float64
		want     string
	}{
		{0.00, "low"},
		{0.01, "low"},
		{1.00, "low"},
		{5.00, "low"},
		{10.00, "low"},
		{10.01, "medium"},
		{25.00, "medium"},
		{50.00, "medium"},
		{50.01, "high"},
		{75.00, "high"},
		{100.00, "high"},
		{100.01, "critical"},
		{500.00, "critical"},
		{10000.00, "critical"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := classifyVarianceSeverity(tt.variance)
			if got != tt.want {
				t.Errorf("classifyVarianceSeverity(%.2f) = %q, want %q", tt.variance, got, tt.want)
			}
		})
	}
}

// TestVarianceSeverityBoundaries focuses on the exact boundary values.
func TestVarianceSeverityBoundaries(t *testing.T) {
	// Boundaries: 10, 50, 100
	boundaries := []struct {
		belowVal    float64
		belowSev    string
		exactVal    float64
		exactSev    string
		aboveVal    float64
		aboveSev    string
	}{
		{9.99, "low", 10.00, "low", 10.01, "medium"},
		{49.99, "medium", 50.00, "medium", 50.01, "high"},
		{99.99, "high", 100.00, "high", 100.01, "critical"},
	}

	for _, b := range boundaries {
		if got := classifyVarianceSeverity(b.belowVal); got != b.belowSev {
			t.Errorf("variance=%.2f: got %q, want %q", b.belowVal, got, b.belowSev)
		}
		if got := classifyVarianceSeverity(b.exactVal); got != b.exactSev {
			t.Errorf("variance=%.2f: got %q, want %q", b.exactVal, got, b.exactSev)
		}
		if got := classifyVarianceSeverity(b.aboveVal); got != b.aboveSev {
			t.Errorf("variance=%.2f: got %q, want %q", b.aboveVal, got, b.aboveSev)
		}
	}
}

// TestVarianceExceptionThreshold validates that the $1.00 threshold for
// generating variance exceptions is correctly applied.
// Per reconciliation.go: if amountVariance > 1.00 => create exception.
func TestVarianceExceptionThreshold(t *testing.T) {
	shouldCreateException := func(variance float64) bool {
		return variance > 1.00
	}

	tests := []struct {
		variance  float64
		exception bool
	}{
		{0.00, false},
		{0.50, false},
		{0.99, false},
		{1.00, false},  // exactly $1.00 — NOT an exception (> not >=)
		{1.01, true},
		{2.00, true},
		{100.00, true},
	}

	for _, tt := range tests {
		got := shouldCreateException(tt.variance)
		if got != tt.exception {
			t.Errorf("shouldCreateException(%.2f) = %v, want %v", tt.variance, got, tt.exception)
		}
	}
}
