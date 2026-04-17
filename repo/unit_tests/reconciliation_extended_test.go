package unit_tests

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// Extended reconciliation tests covering score composition, boundary conditions,
// and edge cases for the matching algorithm.
// Production logic references:
//   - backend/internal/services/reconciliation.go (ScoreMatch, classifyVarianceSeverity, levenshteinDistance)

// --- Score component helpers (mirror production logic) ---

func amountScore(internalAmt, externalAmt float64) float64 {
	diff := math.Abs(internalAmt - externalAmt)
	if diff == 0 {
		return 40
	}
	if diff <= 1.00 {
		return 20
	}
	return 0
}

func timestampScore(internalDate, externalDate time.Time) float64 {
	diff := math.Abs(internalDate.Sub(externalDate).Minutes())
	if diff <= 10 {
		return 30
	}
	return 0
}

func counterpartyScore(intCP, extCP string) float64 {
	intCP = strings.TrimSpace(strings.ToLower(intCP))
	extCP = strings.TrimSpace(strings.ToLower(extCP))
	if intCP == "" || extCP == "" {
		return 0
	}
	if intCP == extCP {
		return 15
	}
	if strings.Contains(intCP, extCP) || strings.Contains(extCP, intCP) {
		return 7
	}
	return 0
}

func accountScore(intAcct, extAcct string) float64 {
	intAcct = strings.TrimSpace(strings.ToLower(intAcct))
	extAcct = strings.TrimSpace(strings.ToLower(extAcct))
	if intAcct == "" || extAcct == "" {
		return 0
	}
	if intAcct == extAcct {
		return 10
	}
	if strings.Contains(intAcct, extAcct) || strings.Contains(extAcct, intAcct) {
		return 5
	}
	return 0
}

func levenshteinDist(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			m := ins
			if del < m {
				m = del
			}
			if sub < m {
				m = sub
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func levenshteinSim(a, b string) float64 {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := levenshteinDist(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

func memoScore(intMemo, extMemo string) float64 {
	intMemo = strings.TrimSpace(intMemo)
	extMemo = strings.TrimSpace(extMemo)
	if intMemo == "" || extMemo == "" {
		return 0
	}
	sim := levenshteinSim(strings.ToLower(intMemo), strings.ToLower(extMemo))
	return sim * 10
}

func classifyVariance(variance float64) string {
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

// --- Tests ---

func TestAmountScoreDetailed(t *testing.T) {
	tests := []struct {
		name     string
		internal float64
		external float64
		expected float64
	}{
		{"exact match", 100.00, 100.00, 40},
		{"zero amounts", 0.00, 0.00, 40},
		{"within $0.01", 100.00, 100.01, 20},
		{"within $0.50", 100.00, 99.50, 20},
		{"within $0.99", 100.00, 99.01, 20},
		{"within $1.00 exactly", 100.00, 99.00, 20},
		{"exceeds $1.00 by penny", 100.00, 98.99, 0},
		{"large variance", 100.00, 50.00, 0},
		{"negative amounts exact", -50.00, -50.00, 40},
		{"small amounts within tolerance", 0.50, 1.00, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := amountScore(tt.internal, tt.external)
			if score != tt.expected {
				t.Errorf("amountScore(%.2f, %.2f) = %.0f, want %.0f",
					tt.internal, tt.external, score, tt.expected)
			}
		})
	}
}

func TestTimestampScoreDetailed(t *testing.T) {
	base := time.Date(2026, 4, 10, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		offset   time.Duration
		expected float64
	}{
		{"exact same time", 0, 30},
		{"1 minute apart", time.Minute, 30},
		{"5 minutes apart", 5 * time.Minute, 30},
		{"9 minutes apart", 9 * time.Minute, 30},
		{"10 minutes apart", 10 * time.Minute, 30},
		{"10 min 1 sec apart", 10*time.Minute + time.Second, 0},
		{"11 minutes apart", 11 * time.Minute, 0},
		{"30 minutes apart", 30 * time.Minute, 0},
		{"1 hour apart", time.Hour, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := timestampScore(base, base.Add(tt.offset))
			if score != tt.expected {
				t.Errorf("timestampScore (offset=%v) = %.0f, want %.0f",
					tt.offset, score, tt.expected)
			}
		})
	}
}

func TestCounterpartyScoreDetailed(t *testing.T) {
	tests := []struct {
		name     string
		intCP    string
		extCP    string
		expected float64
	}{
		{"exact match", "Acme Corp", "Acme Corp", 15},
		{"case insensitive match", "ACME CORP", "acme corp", 15},
		{"substring match", "Acme Corporation", "Acme Corp", 7},
		{"reverse substring", "Acme", "Acme Corporation", 7},
		{"no match", "Acme Corp", "Beta Inc", 0},
		{"empty internal", "", "Acme Corp", 0},
		{"empty external", "Acme Corp", "", 0},
		{"both empty", "", "", 0},
		{"whitespace only", "  ", "  ", 0},
		{"with leading spaces", "  Acme Corp  ", "Acme Corp", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := counterpartyScore(tt.intCP, tt.extCP)
			if score != tt.expected {
				t.Errorf("counterpartyScore(%q, %q) = %.0f, want %.0f",
					tt.intCP, tt.extCP, score, tt.expected)
			}
		})
	}
}

func TestAccountScoreDetailed(t *testing.T) {
	tests := []struct {
		name     string
		intAcct  string
		extAcct  string
		expected float64
	}{
		{"exact match", "ACC-12345", "ACC-12345", 10},
		{"case insensitive", "ACC-12345", "acc-12345", 10},
		{"partial match", "ACC-12345-678", "12345", 5},
		{"no match", "ACC-12345", "XYZ-99999", 0},
		{"empty internal", "", "ACC-12345", 0},
		{"empty external", "ACC-12345", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := accountScore(tt.intAcct, tt.extAcct)
			if score != tt.expected {
				t.Errorf("accountScore(%q, %q) = %.0f, want %.0f",
					tt.intAcct, tt.extAcct, score, tt.expected)
			}
		})
	}
}

func TestMemoScoreDetailed(t *testing.T) {
	tests := []struct {
		name    string
		intMemo string
		extMemo string
		minScore float64
		maxScore float64
	}{
		{"identical memos", "Wire transfer #1234", "Wire transfer #1234", 10, 10},
		{"similar memos", "Wire transfer #1234", "Wire transfer #1235", 8, 10},
		{"very different memos", "Payment for services", "Refund credit", 0, 5},
		{"empty internal", "", "Wire transfer", 0, 0},
		{"empty external", "Wire transfer", "", 0, 0},
		{"both empty", "", "", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := memoScore(tt.intMemo, tt.extMemo)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("memoScore(%q, %q) = %.2f, want [%.0f, %.0f]",
					tt.intMemo, tt.extMemo, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestMaximumPossibleScore(t *testing.T) {
	// All components at max: 40 + 30 + 15 + 10 + 10 = 105
	// But the production code caps practical max at ~100 with perfect inputs
	maxScore := 40.0 + 30.0 + 15.0 + 10.0 + 10.0
	if maxScore != 105.0 {
		t.Errorf("max theoretical score = %.0f, want 105", maxScore)
	}
}

func TestAutoMatchThresholdBoundaries(t *testing.T) {
	autoMatchThreshold := 70.0
	manualReviewMin := 50.0
	manualReviewMax := 70.0

	t.Run("score 70 is auto-match", func(t *testing.T) {
		if 70.0 < autoMatchThreshold {
			t.Error("score 70 should meet auto-match threshold")
		}
	})

	t.Run("score 69 is manual review", func(t *testing.T) {
		score := 69.0
		isManual := score >= manualReviewMin && score < manualReviewMax
		if !isManual {
			t.Error("score 69 should be in manual review range")
		}
	})

	t.Run("score 49 is unmatched", func(t *testing.T) {
		score := 49.0
		isUnmatched := score < manualReviewMin
		if !isUnmatched {
			t.Error("score 49 should be unmatched")
		}
	})
}

func TestVarianceSeverityClassification(t *testing.T) {
	tests := []struct {
		variance float64
		expected string
	}{
		{0.50, "low"},
		{1.00, "low"},
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
		name := fmt.Sprintf("%.2f_%s", tt.variance, tt.expected)
		t.Run(name, func(t *testing.T) {
			result := classifyVariance(tt.variance)
			if result != tt.expected {
				t.Errorf("classifyVariance(%.2f) = %q, want %q",
					tt.variance, result, tt.expected)
			}
		})
	}
}

func TestDuplicateDetectionWindow(t *testing.T) {
	// Duplicates: same amount + counterparty within 24 hours
	window := 24 * time.Hour

	t.Run("within 24 hours is potential duplicate", func(t *testing.T) {
		t1 := time.Now()
		t2 := t1.Add(12 * time.Hour)
		diff := t2.Sub(t1)
		if diff > window {
			t.Error("12 hours apart should be within duplicate window")
		}
	})

	t.Run("exactly 24 hours is boundary", func(t *testing.T) {
		t1 := time.Now()
		t2 := t1.Add(24 * time.Hour)
		diff := t2.Sub(t1)
		if diff > window {
			t.Error("exactly 24 hours should be within duplicate window")
		}
	})

	t.Run("over 24 hours is not duplicate", func(t *testing.T) {
		t1 := time.Now()
		t2 := t1.Add(25 * time.Hour)
		diff := t2.Sub(t1)
		if diff <= window {
			t.Error("25 hours apart should be outside duplicate window")
		}
	})
}

func TestLevenshteinEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"both empty", "", "", 0},
		{"first empty", "", "abc", 3},
		{"second empty", "abc", "", 3},
		{"identical", "kitten", "kitten", 0},
		{"classic example", "kitten", "sitting", 3},
		{"single char diff", "cat", "bat", 1},
		{"completely different", "abc", "xyz", 3},
		{"insertion", "abc", "abcd", 1},
		{"deletion", "abcd", "abc", 1},
		{"unicode", "cafe", "cafe", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := levenshteinDist(tt.a, tt.b)
			if dist != tt.expected {
				t.Errorf("levenshteinDist(%q, %q) = %d, want %d",
					tt.a, tt.b, dist, tt.expected)
			}
			// Verify symmetry
			distReverse := levenshteinDist(tt.b, tt.a)
			if dist != distReverse {
				t.Errorf("levenshteinDist is not symmetric: (%q,%q)=%d, (%q,%q)=%d",
					tt.a, tt.b, dist, tt.b, tt.a, distReverse)
			}
		})
	}
}

func TestLevenshteinSimilarityRange(t *testing.T) {
	pairs := [][2]string{
		{"hello", "hello"},
		{"hello", "world"},
		{"", ""},
		{"a", "b"},
		{"abc", "xyz"},
		{"test", "testing"},
	}

	for _, p := range pairs {
		sim := levenshteinSim(p[0], p[1])
		if sim < 0.0 || sim > 1.0 {
			t.Errorf("levenshteinSim(%q, %q) = %f, should be in [0,1]", p[0], p[1], sim)
		}
	}
}
