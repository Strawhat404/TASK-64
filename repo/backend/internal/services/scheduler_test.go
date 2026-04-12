package services

import (
	"testing"
	"time"
)

func TestBufferDuration(t *testing.T) {
	expected := 30 * time.Minute
	if bufferDuration != expected {
		t.Errorf("bufferDuration = %v, want %v", bufferDuration, expected)
	}
}

func TestBufferDurationIsExported(t *testing.T) {
	if BufferDuration != 30*time.Minute {
		t.Errorf("BufferDuration = %v, want 30m", BufferDuration)
	}
}

func TestBufferDurationConsistency(t *testing.T) {
	if BufferDuration != bufferDuration {
		t.Errorf("exported BufferDuration (%v) != internal bufferDuration (%v)", BufferDuration, bufferDuration)
	}
}

// TestConflictDetectionTimeRangeLogic validates the buffered overlap math
// used by CheckConflicts. The DB query checks:
//   scheduled_start < bufferedEnd AND scheduled_end > bufferedStart
// This test validates the edge-case arithmetic without a DB.
func TestConflictDetectionTimeRangeLogic(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	cases := []struct {
		name           string
		existingStart  time.Time
		existingEnd    time.Time
		requestStart   time.Time
		requestEnd     time.Time
		expectConflict bool
	}{
		{
			name:           "direct overlap",
			existingStart:  now,
			existingEnd:    now.Add(1 * time.Hour),
			requestStart:   now.Add(30 * time.Minute),
			requestEnd:     now.Add(90 * time.Minute),
			expectConflict: true,
		},
		{
			name:           "within buffer window before",
			existingStart:  now,
			existingEnd:    now.Add(1 * time.Hour),
			requestStart:   now.Add(1*time.Hour + 15*time.Minute),
			requestEnd:     now.Add(2*time.Hour + 15*time.Minute),
			expectConflict: true, // 15 min gap < 30 min buffer
		},
		{
			name:           "within buffer window after",
			existingStart:  now.Add(2 * time.Hour),
			existingEnd:    now.Add(3 * time.Hour),
			requestStart:   now,
			requestEnd:     now.Add(1*time.Hour + 45*time.Minute),
			expectConflict: true, // 15 min gap < 30 min buffer
		},
		{
			name:           "exactly at buffer boundary",
			existingStart:  now,
			existingEnd:    now.Add(1 * time.Hour),
			requestStart:   now.Add(1*time.Hour + 30*time.Minute),
			requestEnd:     now.Add(2*time.Hour + 30*time.Minute),
			expectConflict: false, // exactly 30 min gap = no conflict (boundary is exclusive)
		},
		{
			name:           "well separated",
			existingStart:  now,
			existingEnd:    now.Add(1 * time.Hour),
			requestStart:   now.Add(3 * time.Hour),
			requestEnd:     now.Add(4 * time.Hour),
			expectConflict: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bufferedStart := tc.requestStart.Add(-bufferDuration)
			bufferedEnd := tc.requestEnd.Add(bufferDuration)
			// DB query: existing.scheduled_start < bufferedEnd AND existing.scheduled_end > bufferedStart
			overlaps := tc.existingStart.Before(bufferedEnd) && tc.existingEnd.After(bufferedStart)
			if overlaps != tc.expectConflict {
				t.Errorf("overlap=%v, want %v (existing=[%v,%v] request=[%v,%v] buffered=[%v,%v])",
					overlaps, tc.expectConflict,
					tc.existingStart, tc.existingEnd,
					tc.requestStart, tc.requestEnd,
					bufferedStart, bufferedEnd)
			}
		})
	}
}

// TestBufferValidationTimeRangeLogic validates the ValidateBuffer boundary arithmetic.
func TestBufferValidationTimeRangeLogic(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	// "before" check: existing.scheduled_end > (start - 30min) AND existing.scheduled_end <= start
	// "after" check: existing.scheduled_start >= end AND existing.scheduled_start < (end + 30min)

	cases := []struct {
		name        string
		existEnd    time.Time
		existStart  time.Time
		reqStart    time.Time
		reqEnd      time.Time
		expectBlock bool
		reason      string
	}{
		{
			name:        "existing ends 20 min before request starts — within buffer",
			existEnd:    now.Add(-20 * time.Minute),
			existStart:  now.Add(-80 * time.Minute),
			reqStart:    now,
			reqEnd:      now.Add(1 * time.Hour),
			expectBlock: true,
			reason:      "20 min gap < 30 min buffer",
		},
		{
			name:        "existing ends exactly 30 min before request starts — at boundary",
			existEnd:    now.Add(-30 * time.Minute),
			existStart:  now.Add(-90 * time.Minute),
			reqStart:    now,
			reqEnd:      now.Add(1 * time.Hour),
			expectBlock: false,
			reason:      "exactly at 30 min boundary, should pass",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the "before" check
			beforeConflict := tc.existEnd.After(tc.reqStart.Add(-bufferDuration)) && !tc.existEnd.After(tc.reqStart)
			if beforeConflict != tc.expectBlock {
				t.Errorf("beforeConflict=%v, want %v (%s)", beforeConflict, tc.expectBlock, tc.reason)
			}
		})
	}
}
