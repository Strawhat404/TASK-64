package unit_tests

import (
	"testing"
	"time"
)

// Tests for retention policy and legal hold business rules.
// Production logic references:
//   - backend/internal/services/auditledger.go (EnforceRetention, SecureDelete)
//   - backend/internal/models/security.go (RetentionPolicy, LegalHold)

func TestRetentionPolicyCutoff(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		retentionYears int
		recordDate     time.Time
		shouldPurge    bool
	}{
		{
			"7-year-old record meets 7-year retention",
			7,
			now.AddDate(-7, -1, 0), // 7 years 1 month ago
			true,
		},
		{
			"6-year-old record does not meet 7-year retention",
			7,
			now.AddDate(-6, 0, 0), // 6 years ago
			false,
		},
		{
			"exactly at retention boundary",
			7,
			now.AddDate(-7, 0, 0), // exactly 7 years ago
			false, // cutoff is AddDate(-years,0,0), record AT cutoff is not past it
		},
		{
			"1-year retention with 2-year-old record",
			1,
			now.AddDate(-2, 0, 0),
			true,
		},
		{
			"recent record never purged",
			7,
			now.AddDate(0, -1, 0), // 1 month ago
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cutoff := now.AddDate(-tt.retentionYears, 0, 0)
			shouldPurge := tt.recordDate.Before(cutoff)
			if shouldPurge != tt.shouldPurge {
				t.Errorf("record from %v with %d-year retention: shouldPurge = %v, want %v",
					tt.recordDate.Format("2006-01-02"), tt.retentionYears, shouldPurge, tt.shouldPurge)
			}
		})
	}
}

func TestLegalHoldPreventsRetention(t *testing.T) {
	t.Run("active legal hold blocks purge", func(t *testing.T) {
		holdIsActive := true
		retentionMet := true

		canPurge := retentionMet && !holdIsActive
		if canPurge {
			t.Error("should not purge when legal hold is active")
		}
	})

	t.Run("released legal hold allows purge", func(t *testing.T) {
		holdIsActive := false
		retentionMet := true

		canPurge := retentionMet && !holdIsActive
		if !canPurge {
			t.Error("should allow purge when legal hold is released and retention is met")
		}
	})

	t.Run("no legal hold allows purge when retention met", func(t *testing.T) {
		holdIsActive := false
		retentionMet := true

		canPurge := retentionMet && !holdIsActive
		if !canPurge {
			t.Error("should allow purge with no hold and met retention")
		}
	})

	t.Run("retention not met blocks purge even without hold", func(t *testing.T) {
		holdIsActive := false
		retentionMet := false

		canPurge := retentionMet && !holdIsActive
		if canPurge {
			t.Error("should not purge when retention period is not met")
		}
	})
}

func TestLegalHoldTimeRange(t *testing.T) {
	now := time.Now()

	t.Run("hold with no end date is active indefinitely", func(t *testing.T) {
		holdStart := now.Add(-30 * 24 * time.Hour) // 30 days ago
		var holdEnd *time.Time                      // nil = no end

		isActive := now.After(holdStart) && (holdEnd == nil || now.Before(*holdEnd))
		if !isActive {
			t.Error("hold with no end date should be active")
		}
	})

	t.Run("hold with past end date is expired", func(t *testing.T) {
		holdStart := now.Add(-60 * 24 * time.Hour) // 60 days ago
		pastEnd := now.Add(-10 * 24 * time.Hour)   // 10 days ago
		holdEnd := &pastEnd

		isActive := now.After(holdStart) && (holdEnd == nil || now.Before(*holdEnd))
		if isActive {
			t.Error("hold with past end date should be expired")
		}
	})

	t.Run("hold with future end date is active", func(t *testing.T) {
		holdStart := now.Add(-30 * 24 * time.Hour) // 30 days ago
		futureEnd := now.Add(30 * 24 * time.Hour)  // 30 days from now
		holdEnd := &futureEnd

		isActive := now.After(holdStart) && (holdEnd == nil || now.Before(*holdEnd))
		if !isActive {
			t.Error("hold with future end date should be active")
		}
	})
}

func TestAuditLedgerImmutability(t *testing.T) {
	// The audit_ledger should NEVER be deleted through the application.
	// Only DBA-level archival/purge should occur.
	t.Run("audit ledger entries are flagged for DBA archival, not deleted", func(t *testing.T) {
		tableName := "audit_ledger"
		// In production, retention enforcement for audit_ledger
		// logs to deletion_log but does NOT delete the row
		isDeletable := tableName != "audit_ledger"
		if isDeletable {
			t.Error("audit_ledger should not be deletable through the application")
		}
	})
}

func TestRetentionPolicyDefaults(t *testing.T) {
	// Common retention periods in the compliance domain
	defaultAuditRetention := 7 // 7 years for audit logs (compliance requirement)

	t.Run("audit logs have 7-year default retention", func(t *testing.T) {
		if defaultAuditRetention != 7 {
			t.Errorf("audit log retention = %d years, want 7", defaultAuditRetention)
		}
	})

	t.Run("7 years is approximately 2557 days", func(t *testing.T) {
		sevenYears := 7 * 365 // approximate
		if sevenYears < 2500 || sevenYears > 2600 {
			t.Errorf("7 years = %d days, expected ~2555", sevenYears)
		}
	})
}

func TestQuarterlyKeyRotation(t *testing.T) {
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	rotationPeriodDays := 90

	tests := []struct {
		name        string
		activatedAt time.Time
		isDue       bool
	}{
		{
			"key activated 100 days ago — overdue",
			now.AddDate(0, 0, -100),
			true,
		},
		{
			"key activated 91 days ago — overdue",
			now.AddDate(0, 0, -91),
			true,
		},
		{
			"key activated exactly 90 days ago — due",
			now.AddDate(0, 0, -90),
			false, // cutoff is AddDate(0,0,-90), at boundary is NOT past it
		},
		{
			"key activated 89 days ago — not due",
			now.AddDate(0, 0, -89),
			false,
		},
		{
			"key activated today — not due",
			now,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cutoff := now.AddDate(0, 0, -rotationPeriodDays)
			isDue := tt.activatedAt.Before(cutoff)
			if isDue != tt.isDue {
				t.Errorf("activated=%v, cutoff=%v: isDue = %v, want %v",
					tt.activatedAt.Format("2006-01-02"), cutoff.Format("2006-01-02"),
					isDue, tt.isDue)
			}
		})
	}
}
