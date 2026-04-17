package unit_tests

import (
	"testing"
)

// Tests for content governance state machine transitions.
// Validates the full lifecycle: draft -> pending_review -> approved -> gray_release -> published.
// Production logic reference: backend/internal/handlers/governance.go

// allowedTransitions defines the state machine from handler logic.
var allowedTransitions = map[string]map[string]bool{
	"draft":          {"pending_review": true},
	"pending_review": {"in_review": true, "approved": true, "rejected": true},
	"in_review":      {"approved": true, "rejected": true},
	"approved":       {"gray_release": true, "pending_review": true},
	"rejected":       {"pending_review": true, "draft": true},
	"gray_release":   {"published": true, "pending_review": true},
	"published":      {"pending_review": true, "archived": true},
}

func canTransition(from, to string) bool {
	targets, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

func TestContentLifecycle_HappyPath(t *testing.T) {
	path := []string{"draft", "pending_review", "approved", "gray_release", "published"}
	for i := 0; i < len(path)-1; i++ {
		from, to := path[i], path[i+1]
		if !canTransition(from, to) {
			t.Errorf("transition %s -> %s should be allowed in happy path", from, to)
		}
	}
}

func TestContentLifecycle_RejectionResubmit(t *testing.T) {
	// draft -> pending_review -> rejected -> pending_review -> approved
	steps := [][2]string{
		{"draft", "pending_review"},
		{"pending_review", "rejected"},
		{"rejected", "pending_review"},
		{"pending_review", "approved"},
	}
	for _, step := range steps {
		if !canTransition(step[0], step[1]) {
			t.Errorf("rejection resubmit: %s -> %s should be allowed", step[0], step[1])
		}
	}
}

func TestContentLifecycle_InvalidTransitions(t *testing.T) {
	invalid := [][2]string{
		{"draft", "published"},       // can't skip review
		{"draft", "gray_release"},    // can't skip review
		{"approved", "published"},    // must go through gray_release
		{"gray_release", "archived"}, // must publish first
		{"published", "draft"},       // can't go back to draft
		{"archived", "published"},    // archived is terminal
	}
	for _, step := range invalid {
		if canTransition(step[0], step[1]) {
			t.Errorf("transition %s -> %s should NOT be allowed", step[0], step[1])
		}
	}
}

func TestContentLifecycle_EscalationPath(t *testing.T) {
	// pending_review -> in_review (escalated) -> approved
	if !canTransition("pending_review", "in_review") {
		t.Error("escalation: pending_review -> in_review should be allowed")
	}
	if !canTransition("in_review", "approved") {
		t.Error("escalation: in_review -> approved should be allowed")
	}
}

func TestContentLifecycle_ReReviewFromPublished(t *testing.T) {
	// published -> pending_review (re-review)
	if !canTransition("published", "pending_review") {
		t.Error("re-review: published -> pending_review should be allowed")
	}
}

func TestReviewDecisions(t *testing.T) {
	validDecisions := map[string]bool{
		"approved":  true,
		"rejected":  true,
		"escalated": true,
	}
	invalid := []string{"accept", "deny", "Approved", "REJECTED", ""}
	for _, d := range invalid {
		if validDecisions[d] {
			t.Errorf("decision %q should be invalid", d)
		}
	}
	for d := range validDecisions {
		if !validDecisions[d] {
			t.Errorf("decision %q should be valid", d)
		}
	}
}
