package unit_tests

import (
	"testing"
	"time"
)

// parseFlexibleDate mirrors the production implementation in
// backend/internal/handlers/reconciliation.go.
func parseFlexibleDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"01-02-2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"02 Jan 2006",
		"2006/01/02",
	}

	for _, fmt := range formats {
		if t, err := time.Parse(fmt, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, &time.ParseError{Value: s, Message: "no matching format"}
}

func TestParseFlexibleDate(t *testing.T) {
	wantDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{"ISO date", "2026-03-15", wantDate},
		{"US slash format", "03/15/2026", wantDate},
		{"US slash no padding", "3/15/2026", wantDate},
		{"US dash format", "03-15-2026", wantDate},
		{"Short month name", "Mar 15, 2026", wantDate},
		{"Full month name", "March 15, 2026", wantDate},
		{"Day month year", "15 Mar 2026", wantDate},
		{"Forward slash ISO", "2026/03/15", wantDate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFlexibleDate(tt.input)
			if err != nil {
				t.Fatalf("parseFlexibleDate(%q) error: %v", tt.input, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("parseFlexibleDate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFlexibleDateWithTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"RFC3339", "2026-03-15T10:30:00Z"},
		{"ISO datetime no tz", "2026-03-15T10:30:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFlexibleDate(tt.input)
			if err != nil {
				t.Fatalf("parseFlexibleDate(%q) error: %v", tt.input, err)
			}
			if got.Year() != 2026 || got.Month() != 3 || got.Day() != 15 {
				t.Errorf("date portion wrong: got %v", got)
			}
		})
	}
}

func TestParseFlexibleDateInvalid(t *testing.T) {
	invalids := []string{
		"",
		"not-a-date",
		"2026-13-01",  // month 13
		"2026-00-15",  // month 0
		"32/01/2026",  // day 32
		"15-03-2026",  // DD-MM-YYYY (not supported)
		"2026",
	}

	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			_, err := parseFlexibleDate(input)
			if err == nil {
				t.Errorf("parseFlexibleDate(%q) should fail but didn't", input)
			}
		})
	}
}
