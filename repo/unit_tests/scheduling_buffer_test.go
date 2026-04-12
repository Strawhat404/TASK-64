// Package unit_tests contains standalone specification tests that validate
// scheduling buffer invariants. Production conflict detection is DB-backed
// and tested via integration tests in tests/API_tests/.
//
// The production buffer constant is validated in:
//   backend/internal/services/scheduler_test.go
package unit_tests

import (
	"testing"
	"time"
)

// bufferMinutes mirrors the production BufferDuration (30 min).
// If this changes, update to match services.BufferDuration.
const bufferMinutes = 30

// makeTime builds a time on a fixed date for deterministic testing.
func makeTime(hour, minute int) time.Time {
	return time.Date(2026, 4, 10, hour, minute, 0, 0, time.UTC)
}

// hasBufferViolation checks the scheduling invariant:
// a new schedule [newStart, newEnd] must have at least bufferMinutes gap
// from [existingStart, existingEnd].
// This mirrors the production exclusion constraint + query logic.
func hasBufferViolation(existingStart, existingEnd, newStart, newEnd time.Time) bool {
	buffer := time.Duration(bufferMinutes) * time.Minute

	// Direct overlap
	if newStart.Before(existingEnd) && newEnd.After(existingStart) {
		return true
	}
	// Buffer violation after existing
	if !newStart.Before(existingEnd) && newStart.Before(existingEnd.Add(buffer)) {
		return true
	}
	// Buffer violation before existing
	if !newEnd.After(existingStart) && newEnd.After(existingStart.Add(-buffer)) {
		return true
	}
	return false
}

func TestNoConflict_WellSeparated(t *testing.T) {
	if hasBufferViolation(makeTime(9, 0), makeTime(10, 0), makeTime(12, 0), makeTime(13, 0)) {
		t.Error("expected no conflict for well-separated schedules")
	}
}

func TestDirectOverlap(t *testing.T) {
	if !hasBufferViolation(makeTime(9, 0), makeTime(10, 0), makeTime(9, 30), makeTime(10, 30)) {
		t.Error("expected direct overlap conflict")
	}
}

func TestBufferViolation_TooClose(t *testing.T) {
	if !hasBufferViolation(makeTime(9, 0), makeTime(10, 0), makeTime(10, 15), makeTime(11, 15)) {
		t.Error("expected buffer violation for schedule starting 15 min after existing ends")
	}
}

func TestBufferExact_30Minutes(t *testing.T) {
	if hasBufferViolation(makeTime(9, 0), makeTime(10, 0), makeTime(10, 30), makeTime(11, 30)) {
		t.Error("expected no conflict when schedule starts exactly 30 min after existing ends")
	}
}

func TestBackToBack_NoBuffer(t *testing.T) {
	if !hasBufferViolation(makeTime(9, 0), makeTime(10, 0), makeTime(10, 0), makeTime(11, 0)) {
		t.Error("expected conflict for back-to-back schedule with no buffer")
	}
}

func TestBufferBoundaries_TableDriven(t *testing.T) {
	existingStart := makeTime(10, 0)
	existingEnd := makeTime(11, 0)

	tests := []struct {
		name           string
		newStartH      int
		newStartM      int
		newEndH        int
		newEndM        int
		expectConflict bool
	}{
		{"well before (OK)", 8, 0, 9, 29, false},
		{"ends at buffer boundary before (conflict)", 8, 30, 9, 31, true},
		{"starts 29 min after end (conflict)", 11, 29, 12, 29, true},
		{"starts exactly 30 min after end (OK)", 11, 30, 12, 30, false},
		{"starts 31 min after end (OK)", 11, 31, 12, 31, false},
		{"complete overlap (conflict)", 10, 0, 11, 0, true},
		{"new contains existing (conflict)", 9, 0, 12, 0, true},
		{"new within existing (conflict)", 10, 15, 10, 45, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			newStart := makeTime(tc.newStartH, tc.newStartM)
			newEnd := makeTime(tc.newEndH, tc.newEndM)
			conflict := hasBufferViolation(existingStart, existingEnd, newStart, newEnd)
			if conflict != tc.expectConflict {
				t.Errorf("expected conflict=%v, got conflict=%v", tc.expectConflict, conflict)
			}
		})
	}
}

func TestMultipleExistingSchedules(t *testing.T) {
	type sched struct{ start, end time.Time }
	existing := []sched{
		{makeTime(8, 0), makeTime(9, 0)},
		{makeTime(11, 0), makeTime(12, 0)},
	}

	checkAll := func(newStart, newEnd time.Time) bool {
		for _, s := range existing {
			if hasBufferViolation(s.start, s.end, newStart, newEnd) {
				return true
			}
		}
		return false
	}

	// Fits between with proper buffer (9:30 - 10:30)
	if checkAll(makeTime(9, 30), makeTime(10, 30)) {
		t.Error("expected no conflict for slot fitting between two schedules with buffer")
	}

	// Violates buffer of second schedule (10:35 - 10:50, too close to 11:00)
	if !checkAll(makeTime(10, 35), makeTime(10, 50)) {
		t.Error("expected conflict for slot ending within buffer before second schedule")
	}
}
