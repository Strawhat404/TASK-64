// Package unit_tests contains standalone specification tests that validate
// reconciliation matching invariants. Production scoring logic is tested
// directly in compliance-console/internal/services/reconciliation_test.go.
//
// These tests document expected matching behavior at a high level.
package unit_tests

import (
	"math"
	"testing"
	"time"
)

// NOTE: These tests validate matching INVARIANTS using a simplified scoring model.
// The production scoring function is tested directly in:
//   backend/internal/services/reconciliation_test.go
// Any changes to scoring logic should update BOTH test suites.

// Simplified scoring model matching production semantics for specification testing
func specScoreAmount(variance float64) float64 {
	if variance == 0 {
		return 40
	} else if variance <= 1.0 {
		return 20
	}
	return 0
}

func specScoreTime(diffMinutes float64) float64 {
	if diffMinutes <= 10 {
		return 30
	} else if diffMinutes <= 60 {
		return 10
	}
	return 0
}

// txTimeSpec creates a time for testing
func txTimeSpec(hour, minute int) time.Time {
	return time.Date(2026, 4, 10, hour, minute, 0, 0, time.UTC)
}

func TestSpec_ExactAmountMatch_HighScore(t *testing.T) {
	score := specScoreAmount(0)
	if score != 40 {
		t.Errorf("exact amount match should yield 40 points, got %.0f", score)
	}
}

func TestSpec_SmallAmountVariance_PartialScore(t *testing.T) {
	score := specScoreAmount(0.50)
	if score != 20 {
		t.Errorf("$0.50 variance should yield 20 points, got %.0f", score)
	}
}

func TestSpec_LargeAmountVariance_ZeroScore(t *testing.T) {
	score := specScoreAmount(5.00)
	if score != 0 {
		t.Errorf("$5.00 variance should yield 0 points, got %.0f", score)
	}
}

func TestSpec_TimestampWithinWindow(t *testing.T) {
	score := specScoreTime(9)
	if score != 30 {
		t.Errorf("9-min difference should yield 30 points, got %.0f", score)
	}
}

func TestSpec_TimestampWithinHour(t *testing.T) {
	score := specScoreTime(30)
	if score != 10 {
		t.Errorf("30-min difference should yield 10 points, got %.0f", score)
	}
}

func TestSpec_TimestampBeyondHour(t *testing.T) {
	score := specScoreTime(120)
	if score != 0 {
		t.Errorf("120-min difference should yield 0 points, got %.0f", score)
	}
}

func TestSpec_VarianceThresholdBoundary(t *testing.T) {
	tests := []struct {
		name        string
		variance    float64
		expectScore float64
	}{
		{"zero", 0, 40},
		{"$0.01", 0.01, 20},
		{"$0.99", 0.99, 20},
		{"$1.00 exactly", 1.00, 20},
		{"$1.01", 1.01, 0},
		{"$5.00", 5.00, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := specScoreAmount(tc.variance)
			if math.Abs(score-tc.expectScore) > 0.01 {
				t.Errorf("variance $%.2f: expected %.0f points, got %.0f", tc.variance, tc.expectScore, score)
			}
		})
	}
}

func TestSpec_DuplicateDetection_Invariants(t *testing.T) {
	// Same amount + counterparty within 24 hours = duplicate suspect
	t.Run("same_within_24h", func(t *testing.T) {
		a := txTimeSpec(10, 0)
		b := txTimeSpec(10, 5)
		diff := math.Abs(a.Sub(b).Hours())
		if diff > 24 {
			t.Error("5 min apart should be within 24h window")
		}
	})

	t.Run("beyond_24h", func(t *testing.T) {
		a := txTimeSpec(10, 0)
		b := a.Add(25 * time.Hour)
		diff := math.Abs(a.Sub(b).Hours())
		if diff <= 24 {
			t.Error("25h apart should be outside 24h window")
		}
	})
}
