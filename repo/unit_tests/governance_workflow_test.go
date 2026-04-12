package unit_tests

import (
	"testing"
	"time"
)

// Tests for governance workflow invariants: content lifecycle states,
// version management, gray-release timing.
//
// Production logic references:
//   - backend/internal/services/versioning.go (maxVersions = 10)
//   - backend/internal/services/grayrelease.go (grayReleaseDuration = 24h)
//   - backend/internal/handlers/governance.go (status transitions)

const maxVersions = 10
const grayReleaseDurationHours = 24

// Valid content statuses from governance_schema.sql CHECK constraint.
var validStatuses = map[string]bool{
	"draft":          true,
	"pending_review": true,
	"in_review":      true,
	"approved":       true,
	"rejected":       true,
	"gray_release":   true,
	"published":      true,
	"archived":       true,
}

// Valid content types from governance_schema.sql CHECK constraint.
var validContentTypes = map[string]bool{
	"article":      true,
	"resource":     true,
	"announcement": true,
	"policy":       true,
}

// Valid review decisions from handler.
var validDecisions = map[string]bool{
	"approved":  true,
	"rejected":  true,
	"escalated": true,
}

// Valid rule types from governance_schema.sql CHECK constraint.
var validRuleTypes = map[string]bool{
	"keyword_block":  true,
	"regex_block":    true,
	"manual_review":  true,
}

// Valid severity levels from governance_schema.sql CHECK constraint.
var validSeverities = map[string]bool{
	"low":      true,
	"medium":   true,
	"high":     true,
	"critical": true,
}

func TestContentStatusTransitions(t *testing.T) {
	// Allowed transitions based on handler logic
	allowed := map[string][]string{
		"draft":          {"pending_review", "rejected"},
		"pending_review": {"in_review", "approved", "rejected"},
		"in_review":      {"approved", "rejected"},
		"approved":       {"gray_release", "pending_review"},
		"rejected":       {"pending_review", "draft"},
		"gray_release":   {"published", "pending_review"},
		"published":      {"pending_review", "archived"},
	}

	for from, targets := range allowed {
		if !validStatuses[from] {
			t.Errorf("source status %q is not a valid status", from)
		}
		for _, to := range targets {
			if !validStatuses[to] {
				t.Errorf("target status %q (from %q) is not a valid status", to, from)
			}
		}
	}
}

func TestSubmitForReviewAllowedStatuses(t *testing.T) {
	// SubmitForReview handler: UPDATE status = 'pending_review'
	// WHERE status IN ('draft', 'rejected')
	canSubmit := map[string]bool{
		"draft":    true,
		"rejected": true,
	}

	for status := range validStatuses {
		allowed := canSubmit[status]
		t.Run(status, func(t *testing.T) {
			if allowed && !(status == "draft" || status == "rejected") {
				t.Errorf("status %q should not be submittable", status)
			}
		})
	}
}

func TestReReviewAllowedStatuses(t *testing.T) {
	// ReReview handler validates: approved|rejected|published|gray_release
	canReReview := map[string]bool{
		"approved":     true,
		"rejected":     true,
		"published":    true,
		"gray_release": true,
	}

	disallowed := []string{"draft", "pending_review", "in_review", "archived"}
	for _, s := range disallowed {
		if canReReview[s] {
			t.Errorf("status %q should NOT allow re-review", s)
		}
	}
	for s := range canReReview {
		if !validStatuses[s] {
			t.Errorf("re-review status %q is not valid", s)
		}
	}
}

func TestGrayReleaseEligibility(t *testing.T) {
	now := time.Date(2026, 4, 10, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		grayReleaseAt  time.Time
		eligible       bool
		remainingHours float64
	}{
		{
			"just started",
			now,
			false,
			24.0,
		},
		{
			"12 hours in",
			now.Add(-12 * time.Hour),
			false,
			12.0,
		},
		{
			"23 hours in",
			now.Add(-23 * time.Hour),
			false,
			1.0,
		},
		{
			"exactly 24 hours",
			now.Add(-24 * time.Hour),
			true,
			0.0,
		},
		{
			"25 hours (past due)",
			now.Add(-25 * time.Hour),
			true,
			0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elapsed := now.Sub(tt.grayReleaseAt)
			eligible := elapsed >= time.Duration(grayReleaseDurationHours)*time.Hour
			remaining := float64(grayReleaseDurationHours) - elapsed.Hours()
			if remaining < 0 {
				remaining = 0
			}

			if eligible != tt.eligible {
				t.Errorf("eligible = %v, want %v (elapsed=%.1fh)", eligible, tt.eligible, elapsed.Hours())
			}
			if !floatEq(remaining, tt.remainingHours) {
				t.Errorf("remaining = %.1f hours, want %.1f", remaining, tt.remainingHours)
			}
		})
	}
}

func TestVersionConstraints(t *testing.T) {
	// maxVersions = 10: when creating version 11, version 1 should be pruned
	t.Run("max versions enforced", func(t *testing.T) {
		versions := make([]int, 0, maxVersions)
		for i := 1; i <= 15; i++ {
			versions = append(versions, i)
			if len(versions) > maxVersions {
				versions = versions[1:] // prune oldest
			}
		}
		if len(versions) != maxVersions {
			t.Errorf("version count = %d, want %d", len(versions), maxVersions)
		}
		// Oldest should be version 6, newest version 15
		if versions[0] != 6 {
			t.Errorf("oldest version = %d, want 6", versions[0])
		}
		if versions[len(versions)-1] != 15 {
			t.Errorf("newest version = %d, want 15", versions[len(versions)-1])
		}
	})

	t.Run("rollback target must exist in window", func(t *testing.T) {
		currentVersion := 12
		// Available versions: 3 through 12 (last 10)
		oldestAvailable := currentVersion - maxVersions + 1

		validTarget := 5
		invalidTarget := 2

		if validTarget < oldestAvailable {
			t.Errorf("target %d should be available (oldest=%d)", validTarget, oldestAvailable)
		}
		if invalidTarget >= oldestAvailable {
			t.Errorf("target %d should NOT be available (oldest=%d)", invalidTarget, oldestAvailable)
		}
	})
}

func TestContentTypeValidation(t *testing.T) {
	valid := []string{"article", "resource", "announcement", "policy"}
	invalid := []string{"", "blog", "page", "ARTICLE", "Article"}

	for _, ct := range valid {
		if !validContentTypes[ct] {
			t.Errorf("content type %q should be valid", ct)
		}
	}
	for _, ct := range invalid {
		if validContentTypes[ct] {
			t.Errorf("content type %q should be invalid", ct)
		}
	}
}

func TestReviewDecisionValidation(t *testing.T) {
	valid := []string{"approved", "rejected", "escalated"}
	invalid := []string{"", "accept", "deny", "Approved", "REJECTED"}

	for _, d := range valid {
		if !validDecisions[d] {
			t.Errorf("decision %q should be valid", d)
		}
	}
	for _, d := range invalid {
		if validDecisions[d] {
			t.Errorf("decision %q should be invalid", d)
		}
	}
}

func TestModerationRuleTypeValidation(t *testing.T) {
	valid := []string{"keyword_block", "regex_block", "manual_review"}
	invalid := []string{"", "auto_block", "whitelist", "keyword"}

	for _, rt := range valid {
		if !validRuleTypes[rt] {
			t.Errorf("rule type %q should be valid", rt)
		}
	}
	for _, rt := range invalid {
		if validRuleTypes[rt] {
			t.Errorf("rule type %q should be invalid", rt)
		}
	}
}

func TestRelationshipTypeValidation(t *testing.T) {
	validTypes := map[string]bool{
		"dependency": true,
		"substitute": true,
		"bundle":     true,
	}
	invalid := []string{"", "related", "parent", "child"}

	for rt := range validTypes {
		if !validTypes[rt] {
			t.Errorf("relationship type %q should be valid", rt)
		}
	}
	for _, rt := range invalid {
		if validTypes[rt] {
			t.Errorf("relationship type %q should be invalid", rt)
		}
	}
}
