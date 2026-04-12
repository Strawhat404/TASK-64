package services

import (
	"testing"
	"time"

	"compliance-console/internal/models"
)

func TestScoreMatch_ExactMatch(t *testing.T) {
	now := time.Now()
	memo1 := "Invoice payment"
	memo2 := "Invoice payment"
	cp1 := "Acme Corp"
	cp2 := "Acme Corp"

	internal := &models.Transaction{Amount: 100.00, TransactionDate: now, Counterparty: &cp1, Memo: &memo1}
	external := &models.Transaction{Amount: 100.00, TransactionDate: now, Counterparty: &cp2, Memo: &memo2}

	score := scoreMatchInternal(internal, external)
	if score < 90 {
		t.Errorf("exact match should score >= 90, got %.1f", score)
	}
}

func TestScoreMatch_AmountVarianceWithin1Dollar(t *testing.T) {
	now := time.Now()
	cp := "Vendor"
	internal := &models.Transaction{Amount: 100.00, TransactionDate: now, Counterparty: &cp}
	external := &models.Transaction{Amount: 100.50, TransactionDate: now, Counterparty: &cp}

	score := scoreMatchInternal(internal, external)
	// Should get 20 (amount within $1) + 30 (exact time) + 20 (exact counterparty) = 70
	if score < 70 {
		t.Errorf("within-$1 variance should score >= 70, got %.1f", score)
	}
}

func TestScoreMatch_TimestampBeyond10Minutes(t *testing.T) {
	now := time.Now()
	later := now.Add(15 * time.Minute)

	internal := &models.Transaction{Amount: 100.00, TransactionDate: now}
	external := &models.Transaction{Amount: 100.00, TransactionDate: later}

	score := scoreMatchInternal(internal, external)
	// 40 (exact amount) + 0 (beyond ±10 min = zero timestamp points per strict spec) = 40
	if score > 45 {
		t.Errorf("15-min offset should score ~40 (zero timestamp points beyond ±10 min), got %.1f", score)
	}
}

func TestScoreMatch_TimestampWithin10Minutes(t *testing.T) {
	now := time.Now()
	later := now.Add(9 * time.Minute)

	internal := &models.Transaction{Amount: 100.00, TransactionDate: now}
	external := &models.Transaction{Amount: 100.00, TransactionDate: later}

	score := scoreMatchInternal(internal, external)
	// 40 (exact amount) + 30 (within 10 min) = 70
	if score < 70 {
		t.Errorf("9-min offset should score >= 70, got %.1f", score)
	}
}

func TestScoreMatch_NoMatch(t *testing.T) {
	now := time.Now()
	later := now.Add(120 * time.Minute)

	internal := &models.Transaction{Amount: 100.00, TransactionDate: now}
	external := &models.Transaction{Amount: 999.00, TransactionDate: later}

	score := scoreMatchInternal(internal, external)
	if score > 10 {
		t.Errorf("completely different transactions should score < 10, got %.1f", score)
	}
}

func TestLevenshteinSimilarity_Identical(t *testing.T) {
	sim := levenshteinSimilarity("hello world", "hello world")
	if sim != 1.0 {
		t.Errorf("identical strings should be 1.0, got %.2f", sim)
	}
}

func TestLevenshteinSimilarity_Similar(t *testing.T) {
	sim := levenshteinSimilarity("invoice payment", "invoice paymentt")
	if sim < 0.8 {
		t.Errorf("similar strings should be >= 0.8, got %.2f", sim)
	}
}

func TestLevenshteinSimilarity_Different(t *testing.T) {
	sim := levenshteinSimilarity("abc", "xyz")
	if sim > 0.1 {
		t.Errorf("completely different strings should be < 0.1, got %.2f", sim)
	}
}

func TestLevenshteinDistance(t *testing.T) {
	cases := []struct{ a, b string; want int }{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"same", "same", 0},
	}
	for _, tc := range cases {
		got := levenshteinDistance(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestClassifyVarianceSeverity(t *testing.T) {
	cases := []struct{ variance float64; want string }{
		{0.50, "low"},
		{15.0, "medium"},
		{75.0, "high"},
		{150.0, "critical"},
	}
	for _, tc := range cases {
		got := classifyVarianceSeverity(tc.variance)
		if got != tc.want {
			t.Errorf("classifyVarianceSeverity(%.2f) = %q, want %q", tc.variance, got, tc.want)
		}
	}
}
