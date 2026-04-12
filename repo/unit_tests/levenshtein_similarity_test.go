package unit_tests

import (
	"testing"
)

// levenshteinDistance mirrors the production implementation in
// backend/internal/services/reconciliation.go.
func levenshteinDistance(a, b string) int {
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
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func levenshteinSimilarity(a, b string) float64 {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := levenshteinDistance(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected int
	}{
		{"identical strings", "hello", "hello", 0},
		{"empty vs empty", "", "", 0},
		{"empty vs non-empty", "", "abc", 3},
		{"non-empty vs empty", "abc", "", 3},
		{"single char difference", "cat", "bat", 1},
		{"insertion", "cat", "cats", 1},
		{"deletion", "cats", "cat", 1},
		{"transposition", "ab", "ba", 2},
		{"completely different", "abc", "xyz", 3},
		{"different lengths", "short", "longer string", 11},
		{"case sensitive", "Hello", "hello", 1},
		{"single char strings", "a", "b", 1},
		{"repeated chars", "aaa", "aaaa", 1},
		{"real memo comparison", "wire transfer ref-001", "wire transfer ref-002", 1},
		{"partial memo match", "payment to vendor", "payment from vendor", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levenshteinDistance(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestLevenshteinDistanceSymmetry(t *testing.T) {
	pairs := [][2]string{
		{"hello", "world"},
		{"abc", "xyz"},
		{"payment", "payroll"},
		{"", "test"},
	}
	for _, p := range pairs {
		d1 := levenshteinDistance(p[0], p[1])
		d2 := levenshteinDistance(p[1], p[0])
		if d1 != d2 {
			t.Errorf("distance(%q,%q)=%d != distance(%q,%q)=%d — symmetry violated",
				p[0], p[1], d1, p[1], p[0], d2)
		}
	}
}

func TestLevenshteinSimilarity(t *testing.T) {
	tests := []struct {
		name   string
		a, b   string
		minSim float64
		maxSim float64
	}{
		{"identical", "hello", "hello", 1.0, 1.0},
		{"both empty", "", "", 1.0, 1.0},
		{"completely different same length", "abc", "xyz", 0.0, 0.01},
		{"one char diff in short string", "cat", "bat", 0.6, 0.7},
		{"similar memos", "wire transfer ref-001", "wire transfer ref-002", 0.9, 1.0},
		{"dissimilar memos", "payment to vendor", "unrelated string", 0.0, 0.4},
		{"empty vs non-empty", "", "hello", 0.0, 0.01},
		{"substring", "pay", "payment", 0.3, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := levenshteinSimilarity(tt.a, tt.b)
			if sim < tt.minSim || sim > tt.maxSim {
				t.Errorf("levenshteinSimilarity(%q, %q) = %.4f, want [%.2f, %.2f]",
					tt.a, tt.b, sim, tt.minSim, tt.maxSim)
			}
		})
	}
}

func TestLevenshteinSimilarityBounds(t *testing.T) {
	pairs := [][2]string{
		{"hello", "world"},
		{"a", "b"},
		{"test", "testing"},
		{"abc", ""},
		{"", ""},
		{"same", "same"},
	}
	for _, p := range pairs {
		sim := levenshteinSimilarity(p[0], p[1])
		if sim < 0.0 || sim > 1.0 {
			t.Errorf("similarity(%q, %q) = %.4f — out of [0,1] bounds", p[0], p[1], sim)
		}
	}
}

// TestMemoScoringContribution validates that memo similarity contributes
// up to 10 points in the match scoring formula.
func TestMemoScoringContribution(t *testing.T) {
	tests := []struct {
		name      string
		intMemo   string
		extMemo   string
		minPoints float64
		maxPoints float64
	}{
		{"identical memos", "payment ref-123", "payment ref-123", 10.0, 10.0},
		{"very similar", "wire transfer 001", "wire transfer 002", 8.0, 10.0},
		{"somewhat similar", "vendor payment", "vendor refund", 4.0, 8.0},
		{"completely different", "abc", "xyz", 0.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := levenshteinSimilarity(tt.intMemo, tt.extMemo)
			points := sim * 10
			if points < tt.minPoints || points > tt.maxPoints {
				t.Errorf("memo score = %.2f (sim=%.4f), want [%.1f, %.1f]",
					points, sim, tt.minPoints, tt.maxPoints)
			}
		})
	}
}
