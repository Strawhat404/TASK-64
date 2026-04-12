package unit_tests

import (
	"math"
	"testing"
)

// Mirrors the pricing logic in backend/internal/handlers/services.go GetPricing.
// Tier multipliers: standard=1.0, premium=1.5, enterprise=2.0
// After-hours surcharge: base * (surcharge_pct / 100)
// Same-day surcharge: fixed USD amount added to total

type pricingInput struct {
	basePriceUSD         float64
	tier                 string
	afterHoursSurchPct   int
	sameDaySurchargeUSD  float64
	afterHours           bool
	sameDay              bool
}

type pricingResult struct {
	basePrice        float64
	tierMultiplier   float64
	afterHoursFee    float64
	sameDayFee       float64
	totalPrice       float64
}

func computePricing(in pricingInput) pricingResult {
	var multiplier float64
	switch in.tier {
	case "premium":
		multiplier = 1.5
	case "enterprise":
		multiplier = 2.0
	default:
		multiplier = 1.0
	}

	adjustedBase := in.basePriceUSD * multiplier

	var afterHoursFee float64
	if in.afterHours {
		afterHoursFee = adjustedBase * float64(in.afterHoursSurchPct) / 100
	}

	var sameDayFee float64
	if in.sameDay {
		sameDayFee = in.sameDaySurchargeUSD
	}

	total := adjustedBase + afterHoursFee + sameDayFee

	return pricingResult{
		basePrice:      adjustedBase,
		tierMultiplier: multiplier,
		afterHoursFee:  afterHoursFee,
		sameDayFee:     sameDayFee,
		totalPrice:     total,
	}
}

func floatEq(a, b float64) bool {
	return math.Abs(a-b) < 0.005
}

func TestPricingTierMultipliers(t *testing.T) {
	tests := []struct {
		tier       string
		base       float64
		wantBase   float64
		wantMult   float64
	}{
		{"standard", 100.00, 100.00, 1.0},
		{"premium", 100.00, 150.00, 1.5},
		{"enterprise", 100.00, 200.00, 2.0},
		{"standard", 50.00, 50.00, 1.0},
		{"premium", 200.00, 300.00, 1.5},
		{"enterprise", 75.50, 151.00, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			r := computePricing(pricingInput{
				basePriceUSD: tt.base,
				tier:         tt.tier,
			})
			if !floatEq(r.basePrice, tt.wantBase) {
				t.Errorf("tier=%s base=%.2f: adjusted base = %.2f, want %.2f",
					tt.tier, tt.base, r.basePrice, tt.wantBase)
			}
			if !floatEq(r.tierMultiplier, tt.wantMult) {
				t.Errorf("tier=%s: multiplier = %.1f, want %.1f",
					tt.tier, r.tierMultiplier, tt.wantMult)
			}
		})
	}
}

func TestPricingAfterHoursSurcharge(t *testing.T) {
	tests := []struct {
		name       string
		base       float64
		tier       string
		pct        int
		afterHours bool
		wantFee    float64
	}{
		{"standard 20% active", 100.00, "standard", 20, true, 20.00},
		{"standard 20% inactive", 100.00, "standard", 20, false, 0.00},
		{"premium 20% active", 100.00, "premium", 20, true, 30.00},    // 150 * 0.20
		{"enterprise 25% active", 100.00, "enterprise", 25, true, 50.00}, // 200 * 0.25
		{"zero surcharge pct", 100.00, "standard", 0, true, 0.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := computePricing(pricingInput{
				basePriceUSD:       tt.base,
				tier:               tt.tier,
				afterHoursSurchPct: tt.pct,
				afterHours:         tt.afterHours,
			})
			if !floatEq(r.afterHoursFee, tt.wantFee) {
				t.Errorf("after-hours fee = %.2f, want %.2f", r.afterHoursFee, tt.wantFee)
			}
		})
	}
}

func TestPricingSameDaySurcharge(t *testing.T) {
	tests := []struct {
		name      string
		surcharge float64
		sameDay   bool
		wantFee   float64
	}{
		{"same-day active", 25.00, true, 25.00},
		{"same-day inactive", 25.00, false, 0.00},
		{"zero surcharge same-day", 0.00, true, 0.00},
		{"large surcharge", 100.00, true, 100.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := computePricing(pricingInput{
				basePriceUSD:        100.00,
				tier:                "standard",
				sameDaySurchargeUSD: tt.surcharge,
				sameDay:             tt.sameDay,
			})
			if !floatEq(r.sameDayFee, tt.wantFee) {
				t.Errorf("same-day fee = %.2f, want %.2f", r.sameDayFee, tt.wantFee)
			}
		})
	}
}

func TestPricingTotalCombinations(t *testing.T) {
	tests := []struct {
		name      string
		input     pricingInput
		wantTotal float64
	}{
		{
			"standard no surcharges",
			pricingInput{100.00, "standard", 20, 25.00, false, false},
			100.00,
		},
		{
			"standard both surcharges",
			pricingInput{100.00, "standard", 20, 25.00, true, true},
			145.00, // 100 + 20 + 25
		},
		{
			"premium both surcharges",
			pricingInput{100.00, "premium", 20, 25.00, true, true},
			205.00, // 150 + 30 + 25
		},
		{
			"enterprise after-hours only",
			pricingInput{100.00, "enterprise", 20, 25.00, true, false},
			240.00, // 200 + 40
		},
		{
			"enterprise same-day only",
			pricingInput{100.00, "enterprise", 20, 25.00, false, true},
			225.00, // 200 + 25
		},
		{
			"zero base price",
			pricingInput{0.00, "premium", 20, 25.00, true, true},
			25.00, // 0 + 0 + 25
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := computePricing(tt.input)
			if !floatEq(r.totalPrice, tt.wantTotal) {
				t.Errorf("total = %.2f, want %.2f (base=%.2f + ah=%.2f + sd=%.2f)",
					r.totalPrice, tt.wantTotal, r.basePrice, r.afterHoursFee, r.sameDayFee)
			}
		})
	}
}

// TestDurationValidation validates the service duration constraints:
// 15-240 minutes, must be divisible by 15.
func TestDurationValidation(t *testing.T) {
	isValidDuration := func(d int) bool {
		return d >= 15 && d <= 240 && d%15 == 0
	}

	tests := []struct {
		duration int
		valid    bool
	}{
		{0, false},
		{14, false},
		{15, true},
		{17, false},
		{30, true},
		{45, true},
		{60, true},
		{90, true},
		{120, true},
		{135, true},
		{180, true},
		{240, true},
		{241, false},
		{255, false},
		{300, false},
		{-15, false},
		{1, false},
		{100, false},
	}

	for _, tt := range tests {
		got := isValidDuration(tt.duration)
		if got != tt.valid {
			t.Errorf("isValidDuration(%d) = %v, want %v", tt.duration, got, tt.valid)
		}
	}
}
