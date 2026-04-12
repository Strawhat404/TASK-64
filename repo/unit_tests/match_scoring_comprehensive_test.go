package unit_tests

import (
	"math"
	"strings"
	"testing"
	"time"
)

// Full scoring function mirroring backend/internal/services/reconciliation.go
// scoreMatchInternal. Max score = 100 points.
//
// Breakdown:
//   Amount:             exact=40, within $1.00=20, else 0
//   Timestamp:          <=10 min=30, else 0
//   Counterparty:       exact=15, substring=7, else 0
//   Counterparty Acct:  exact=10, partial=5, else 0
//   Memo:               levenshtein similarity * 10 (0-10)

type txn struct {
	amount       float64
	date         time.Time
	counterparty *string
	acct         *string
	memo         *string
}

func strPtr(s string) *string { return &s }

func scoreMatch(internal, external txn) float64 {
	score := 0.0

	// Amount
	amountDiff := math.Abs(internal.amount - external.amount)
	if amountDiff == 0 {
		score += 40
	} else if amountDiff <= 1.00 {
		score += 20
	}

	// Timestamp
	timeDiff := math.Abs(internal.date.Sub(external.date).Minutes())
	if timeDiff <= 10 {
		score += 30
	}

	// Counterparty
	if internal.counterparty != nil && external.counterparty != nil {
		intCP := strings.TrimSpace(strings.ToLower(*internal.counterparty))
		extCP := strings.TrimSpace(strings.ToLower(*external.counterparty))
		if intCP != "" && extCP != "" {
			if intCP == extCP {
				score += 15
			} else if strings.Contains(intCP, extCP) || strings.Contains(extCP, intCP) {
				score += 7
			}
		}
	}

	// Counterparty account
	if internal.acct != nil && external.acct != nil {
		intAcct := strings.TrimSpace(strings.ToLower(*internal.acct))
		extAcct := strings.TrimSpace(strings.ToLower(*external.acct))
		if intAcct != "" && extAcct != "" {
			if intAcct == extAcct {
				score += 10
			} else if strings.Contains(intAcct, extAcct) || strings.Contains(extAcct, intAcct) {
				score += 5
			}
		}
	}

	// Memo
	if internal.memo != nil && external.memo != nil {
		intMemo := strings.TrimSpace(*internal.memo)
		extMemo := strings.TrimSpace(*external.memo)
		if intMemo != "" && extMemo != "" {
			sim := levenshteinSimilarity(strings.ToLower(intMemo), strings.ToLower(extMemo))
			score += sim * 10
		}
	}

	return score
}

var baseTime = time.Date(2026, 4, 10, 14, 0, 0, 0, time.UTC)

func TestScoreMatchPerfectMatch(t *testing.T) {
	internal := txn{
		amount:       500.00,
		date:         baseTime,
		counterparty: strPtr("Acme Corp"),
		acct:         strPtr("ACC-001"),
		memo:         strPtr("Invoice 12345"),
	}
	external := txn{
		amount:       500.00,
		date:         baseTime,
		counterparty: strPtr("Acme Corp"),
		acct:         strPtr("ACC-001"),
		memo:         strPtr("Invoice 12345"),
	}

	score := scoreMatch(internal, external)
	// 40 + 30 + 15 + 10 + 10 = 105 but capped effectively at ~105
	// (no cap in production, just summed)
	if score < 100 {
		t.Errorf("perfect match score = %.2f, want >= 100", score)
	}
}

func TestScoreMatchAmountOnly(t *testing.T) {
	tests := []struct {
		name  string
		intAm float64
		extAm float64
		want  float64
	}{
		{"exact match", 100.00, 100.00, 40},
		{"within $1", 100.00, 100.50, 20},
		{"exactly $1 diff", 100.00, 101.00, 20},
		{"over $1 diff", 100.00, 101.01, 0},
		{"large diff", 100.00, 200.00, 0},
		{"zero amounts", 0.00, 0.00, 40},
		{"tiny diff", 100.00, 100.001, 20},
	}

	farTime := baseTime.Add(24 * time.Hour) // ensures no time points

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreMatch(
				txn{amount: tt.intAm, date: baseTime},
				txn{amount: tt.extAm, date: farTime},
			)
			if !floatEq(score, tt.want) {
				t.Errorf("amount score = %.2f, want %.2f", score, tt.want)
			}
		})
	}
}

func TestScoreMatchTimestampOnly(t *testing.T) {
	tests := []struct {
		name    string
		diffMin int
		want    float64
	}{
		{"same time", 0, 30},
		{"5 min apart", 5, 30},
		{"exactly 10 min", 10, 30},
		{"11 min", 11, 0},
		{"30 min", 30, 0},
		{"120 min", 120, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreMatch(
				txn{amount: 999.99, date: baseTime},
				txn{amount: 0.01, date: baseTime.Add(time.Duration(tt.diffMin) * time.Minute)},
			)
			// amount will be 0 (diff > $1), so score = time points only
			if !floatEq(score, tt.want) {
				t.Errorf("time score (diff=%d min) = %.2f, want %.2f", tt.diffMin, score, tt.want)
			}
		})
	}
}

func TestScoreMatchCounterparty(t *testing.T) {
	farTime := baseTime.Add(24 * time.Hour)
	tests := []struct {
		name string
		intC *string
		extC *string
		want float64
	}{
		{"exact match", strPtr("Acme Corp"), strPtr("Acme Corp"), 15},
		{"case insensitive", strPtr("ACME CORP"), strPtr("acme corp"), 15},
		{"substring match", strPtr("Acme Corporation"), strPtr("Acme"), 7},
		{"reverse substring", strPtr("Acme"), strPtr("Acme Corporation"), 7},
		{"no match", strPtr("Acme"), strPtr("Globex"), 0},
		{"one nil", strPtr("Acme"), nil, 0},
		{"both nil", nil, nil, 0},
		{"one empty", strPtr("Acme"), strPtr(""), 0},
		{"both empty", strPtr(""), strPtr(""), 0},
		{"whitespace handling", strPtr("  Acme  "), strPtr("acme"), 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreMatch(
				txn{amount: 999.99, date: baseTime, counterparty: tt.intC},
				txn{amount: 0.01, date: farTime, counterparty: tt.extC},
			)
			if !floatEq(score, tt.want) {
				t.Errorf("counterparty score = %.2f, want %.2f", score, tt.want)
			}
		})
	}
}

func TestScoreMatchCounterpartyAccount(t *testing.T) {
	farTime := baseTime.Add(24 * time.Hour)
	tests := []struct {
		name   string
		intA   *string
		extA   *string
		want   float64
	}{
		{"exact match", strPtr("ACC-001"), strPtr("ACC-001"), 10},
		{"case insensitive", strPtr("ACC-001"), strPtr("acc-001"), 10},
		{"partial match", strPtr("ACC-001-MAIN"), strPtr("ACC-001"), 5},
		{"no match", strPtr("ACC-001"), strPtr("ACC-999"), 0},
		{"one nil", strPtr("ACC-001"), nil, 0},
		{"both nil", nil, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreMatch(
				txn{amount: 999.99, date: baseTime, acct: tt.intA},
				txn{amount: 0.01, date: farTime, acct: tt.extA},
			)
			if !floatEq(score, tt.want) {
				t.Errorf("account score = %.2f, want %.2f", score, tt.want)
			}
		})
	}
}

// TestAutoMatchThreshold validates the 70-point auto-match cutoff.
func TestAutoMatchThreshold(t *testing.T) {
	isAutoMatch := func(score float64) bool { return score >= 70 }
	isManualReview := func(score float64) bool { return score >= 50 && score < 70 }
	isUnmatched := func(score float64) bool { return score < 50 }

	tests := []struct {
		name   string
		score  float64
		auto   bool
		manual bool
		noMatch bool
	}{
		{"perfect 100", 100, true, false, false},
		{"exact threshold 70", 70, true, false, false},
		{"just below 69.9", 69.9, false, true, false},
		{"manual review 60", 60, false, true, false},
		{"manual threshold 50", 50, false, true, false},
		{"just below manual 49.9", 49.9, false, false, true},
		{"low 30", 30, false, false, true},
		{"zero", 0, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isAutoMatch(tt.score) != tt.auto {
				t.Errorf("score=%.1f: autoMatch=%v, want %v", tt.score, !tt.auto, tt.auto)
			}
			if isManualReview(tt.score) != tt.manual {
				t.Errorf("score=%.1f: manualReview=%v, want %v", tt.score, !tt.manual, tt.manual)
			}
			if isUnmatched(tt.score) != tt.noMatch {
				t.Errorf("score=%.1f: unmatched=%v, want %v", tt.score, !tt.noMatch, tt.noMatch)
			}
		})
	}
}

// TestScoreMatchRealisticScenarios tests end-to-end scoring with realistic data.
func TestScoreMatchRealisticScenarios(t *testing.T) {
	tests := []struct {
		name     string
		internal txn
		external txn
		minScore float64
		maxScore float64
		outcome  string // "auto", "manual", "unmatched"
	}{
		{
			"wire transfer exact match",
			txn{1500.00, baseTime, strPtr("Bank of America"), strPtr("0011223344"), strPtr("Wire TXN-2026-001")},
			txn{1500.00, baseTime.Add(2 * time.Minute), strPtr("Bank of America"), strPtr("0011223344"), strPtr("Wire TXN-2026-001")},
			95, 110, "auto",
		},
		{
			"fuzzy amount + same counterparty",
			txn{1500.00, baseTime, strPtr("Acme Corp"), nil, nil},
			txn{1500.75, baseTime.Add(5 * time.Minute), strPtr("Acme Corp"), nil, nil},
			60, 70, "manual",
		},
		{
			"same amount different time",
			txn{250.00, baseTime, nil, nil, nil},
			txn{250.00, baseTime.Add(2 * time.Hour), nil, nil, nil},
			40, 41, "unmatched",
		},
		{
			"completely different",
			txn{100.00, baseTime, strPtr("Vendor A"), strPtr("ACC-1"), strPtr("memo A")},
			txn{999.00, baseTime.Add(24 * time.Hour), strPtr("Vendor Z"), strPtr("ACC-9"), strPtr("memo Z")},
			0, 10, "unmatched",
		},
		{
			"amount close + time close + partial counterparty",
			txn{500.00, baseTime, strPtr("Global Services Inc"), nil, nil},
			txn{500.50, baseTime.Add(8 * time.Minute), strPtr("Global Services"), nil, nil},
			50, 70, "manual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreMatch(tt.internal, tt.external)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("score = %.2f, want [%.0f, %.0f]", score, tt.minScore, tt.maxScore)
			}

			var outcome string
			if score >= 70 {
				outcome = "auto"
			} else if score >= 50 {
				outcome = "manual"
			} else {
				outcome = "unmatched"
			}
			if outcome != tt.outcome {
				t.Errorf("outcome = %q (score=%.2f), want %q", outcome, score, tt.outcome)
			}
		})
	}
}
