package unit_tests

import (
	"strings"
	"testing"
	"time"
)

// Tests for model validation logic.
// Production logic references:
//   - backend/internal/models/models.go (CreateScheduleRequest.Validate)
//   - backend/internal/handlers/services.go (tier, duration, headcount validation)
//   - backend/internal/handlers/users.go (user creation validation)

// --- CreateScheduleRequest validation ---

type scheduleRequest struct {
	serviceID      string
	staffID        string
	clientName     string
	scheduledStart time.Time
	scheduledEnd   time.Time
}

func validateScheduleRequest(r scheduleRequest) []string {
	var errs []string
	if r.clientName == "" {
		errs = append(errs, "client_name is required")
	}
	if r.serviceID == "" || r.serviceID == "00000000-0000-0000-0000-000000000000" {
		errs = append(errs, "service_id is required")
	}
	if r.staffID == "" || r.staffID == "00000000-0000-0000-0000-000000000000" {
		errs = append(errs, "staff_id is required")
	}
	if r.scheduledStart.IsZero() {
		errs = append(errs, "scheduled_start is required")
	}
	if r.scheduledEnd.IsZero() {
		errs = append(errs, "scheduled_end is required")
	}
	if !r.scheduledStart.IsZero() && !r.scheduledEnd.IsZero() {
		if !r.scheduledEnd.After(r.scheduledStart) {
			errs = append(errs, "scheduled_end must be after scheduled_start")
		}
	}
	return errs
}

func TestScheduleRequestValidation(t *testing.T) {
	now := time.Now()

	t.Run("valid request has no errors", func(t *testing.T) {
		req := scheduleRequest{
			serviceID:      "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			staffID:        "b2c3d4e5-f6a7-8901-bcde-f12345678901",
			clientName:     "John Doe",
			scheduledStart: now.Add(time.Hour),
			scheduledEnd:   now.Add(2 * time.Hour),
		}
		errs := validateScheduleRequest(req)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("missing client name", func(t *testing.T) {
		req := scheduleRequest{
			serviceID:      "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			staffID:        "b2c3d4e5-f6a7-8901-bcde-f12345678901",
			clientName:     "",
			scheduledStart: now.Add(time.Hour),
			scheduledEnd:   now.Add(2 * time.Hour),
		}
		errs := validateScheduleRequest(req)
		assertContainsError(t, errs, "client_name is required")
	})

	t.Run("nil UUID for service", func(t *testing.T) {
		req := scheduleRequest{
			serviceID:      "00000000-0000-0000-0000-000000000000",
			staffID:        "b2c3d4e5-f6a7-8901-bcde-f12345678901",
			clientName:     "John Doe",
			scheduledStart: now.Add(time.Hour),
			scheduledEnd:   now.Add(2 * time.Hour),
		}
		errs := validateScheduleRequest(req)
		assertContainsError(t, errs, "service_id is required")
	})

	t.Run("end before start", func(t *testing.T) {
		req := scheduleRequest{
			serviceID:      "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			staffID:        "b2c3d4e5-f6a7-8901-bcde-f12345678901",
			clientName:     "John Doe",
			scheduledStart: now.Add(2 * time.Hour),
			scheduledEnd:   now.Add(time.Hour),
		}
		errs := validateScheduleRequest(req)
		assertContainsError(t, errs, "scheduled_end must be after scheduled_start")
	})

	t.Run("end equals start", func(t *testing.T) {
		start := now.Add(time.Hour)
		req := scheduleRequest{
			serviceID:      "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			staffID:        "b2c3d4e5-f6a7-8901-bcde-f12345678901",
			clientName:     "John Doe",
			scheduledStart: start,
			scheduledEnd:   start,
		}
		errs := validateScheduleRequest(req)
		assertContainsError(t, errs, "scheduled_end must be after scheduled_start")
	})

	t.Run("all fields missing", func(t *testing.T) {
		req := scheduleRequest{}
		errs := validateScheduleRequest(req)
		if len(errs) < 3 {
			t.Errorf("expected at least 3 errors for empty request, got %d: %v", len(errs), errs)
		}
	})
}

// --- Service validation ---

func validateTier(tier string) bool {
	return tier == "standard" || tier == "premium" || tier == "enterprise"
}

func validateDuration(minutes int) bool {
	return minutes >= 15 && minutes <= 240 && minutes%15 == 0
}

func validateHeadcount(h int) bool {
	return h >= 1 && h <= 10
}

func TestServiceTierValidation(t *testing.T) {
	valid := []string{"standard", "premium", "enterprise"}
	invalid := []string{"", "basic", "pro", "Standard", "ENTERPRISE", "free"}

	for _, tier := range valid {
		if !validateTier(tier) {
			t.Errorf("tier %q should be valid", tier)
		}
	}
	for _, tier := range invalid {
		if validateTier(tier) {
			t.Errorf("tier %q should be invalid", tier)
		}
	}
}

func TestServiceDurationValidation(t *testing.T) {
	validDurations := []int{15, 30, 45, 60, 90, 120, 150, 180, 210, 240}
	invalidDurations := []int{0, 10, 14, 16, 20, 25, 31, 241, 255, 300, -15}

	for _, d := range validDurations {
		if !validateDuration(d) {
			t.Errorf("duration %d should be valid", d)
		}
	}
	for _, d := range invalidDurations {
		if validateDuration(d) {
			t.Errorf("duration %d should be invalid", d)
		}
	}
}

func TestServiceHeadcountValidation(t *testing.T) {
	tests := []struct {
		headcount int
		valid     bool
	}{
		{0, false},
		{1, true},
		{5, true},
		{10, true},
		{11, false},
		{-1, false},
		{100, false},
	}

	for _, tt := range tests {
		result := validateHeadcount(tt.headcount)
		if result != tt.valid {
			t.Errorf("headcount %d: valid = %v, want %v", tt.headcount, result, tt.valid)
		}
	}
}

// --- User creation validation ---

func validatePassword(pw string) bool {
	return len(pw) >= 12
}

func validateUserFields(username, email string) bool {
	return username != "" && email != ""
}

func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		{"", false},
		{"short", false},
		{"11chars!!!!", false},
		{"12chars!!!!!", true},
		{"a very long and secure password", true},
		{"exactly12345", true},
	}

	for _, tt := range tests {
		result := validatePassword(tt.password)
		if result != tt.valid {
			t.Errorf("password %q (len=%d): valid = %v, want %v",
				tt.password, len(tt.password), result, tt.valid)
		}
	}
}

func TestUserFieldsValidation(t *testing.T) {
	tests := []struct {
		username string
		email    string
		valid    bool
	}{
		{"admin", "admin@test.com", true},
		{"", "admin@test.com", false},
		{"admin", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		result := validateUserFields(tt.username, tt.email)
		if result != tt.valid {
			t.Errorf("username=%q, email=%q: valid = %v, want %v",
				tt.username, tt.email, result, tt.valid)
		}
	}
}

// --- Exception resolution validation ---

func validateDisposition(disposition string) bool {
	valid := map[string]bool{
		"write_off":       true,
		"adjusted":        true,
		"duplicate_confirmed": true,
		"reversed":        true,
		"escalated":       true,
	}
	return valid[disposition]
}

func TestExceptionDispositionValidation(t *testing.T) {
	valid := []string{"write_off", "adjusted", "duplicate_confirmed", "reversed", "escalated"}
	invalid := []string{"", "resolved", "closed", "rejected"}

	for _, d := range valid {
		if !validateDisposition(d) {
			t.Errorf("disposition %q should be valid", d)
		}
	}
	for _, d := range invalid {
		if validateDisposition(d) {
			t.Errorf("disposition %q should be invalid", d)
		}
	}
}

// --- Sensitive data type validation ---

func validateDataType(dt string) bool {
	valid := map[string]bool{
		"bank_account": true,
		"ssn":          true,
		"credit_card":  true,
		"tax_id":       true,
		"passport":     true,
	}
	return valid[dt]
}

func TestSensitiveDataTypeValidation(t *testing.T) {
	valid := []string{"bank_account", "ssn", "credit_card", "tax_id", "passport"}
	invalid := []string{"", "phone", "address", "name"}

	for _, dt := range valid {
		if !validateDataType(dt) {
			t.Errorf("data type %q should be valid", dt)
		}
	}
	for _, dt := range invalid {
		if validateDataType(dt) {
			t.Errorf("data type %q should be invalid", dt)
		}
	}
}

// --- Schedule status validation ---

func validateScheduleStatus(status string) bool {
	valid := map[string]bool{
		"pending":   true,
		"confirmed": true,
		"cancelled": true,
		"completed": true,
	}
	return valid[status]
}

func TestScheduleStatusValidation(t *testing.T) {
	valid := []string{"pending", "confirmed", "cancelled", "completed"}
	invalid := []string{"", "active", "expired", "Pending"}

	for _, s := range valid {
		if !validateScheduleStatus(s) {
			t.Errorf("schedule status %q should be valid", s)
		}
	}
	for _, s := range invalid {
		if validateScheduleStatus(s) {
			t.Errorf("schedule status %q should be invalid", s)
		}
	}
}

// --- Helper ---

func assertContainsError(t *testing.T, errs []string, expected string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e, expected) {
			return
		}
	}
	t.Errorf("expected error containing %q, got %v", expected, errs)
}
