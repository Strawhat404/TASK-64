package services

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"compliance-console/internal/models"

	"github.com/google/uuid"
)

// CheckContent loads active moderation rules for a tenant and checks the content
// string against keyword_block (case-insensitive substring) and regex_block
// (regexp match) rules. Returns whether the content is blocked and all match reasons.
func CheckContent(db *sql.DB, tenantID, content string) (blocked bool, reasons []string, err error) {
	rows, err := db.Query(`
		SELECT id, rule_type, pattern, severity
		FROM moderation_rules
		WHERE tenant_id = $1 AND is_active = TRUE
		  AND rule_type IN ('keyword_block', 'regex_block')
		ORDER BY severity DESC
	`, tenantID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to load moderation rules: %w", err)
	}
	defer rows.Close()

	lowerContent := strings.ToLower(content)

	for rows.Next() {
		var ruleID uuid.UUID
		var ruleType, pattern, severity string
		if err := rows.Scan(&ruleID, &ruleType, &pattern, &severity); err != nil {
			return false, nil, fmt.Errorf("failed to scan moderation rule: %w", err)
		}

		switch ruleType {
		case "keyword_block":
			if strings.Contains(lowerContent, strings.ToLower(pattern)) {
				blocked = true
				reasons = append(reasons, fmt.Sprintf("keyword_block: matched '%s' (severity: %s)", pattern, severity))
			}
		case "regex_block":
			re, compileErr := regexp.Compile(pattern)
			if compileErr != nil {
				// Skip invalid regex patterns
				continue
			}
			if re.MatchString(content) {
				blocked = true
				reasons = append(reasons, fmt.Sprintf("regex_block: matched pattern '%s' (severity: %s)", pattern, severity))
			}
		}
	}

	return blocked, reasons, nil
}

// AutoBlockCheck combines the title and body of a content item and runs
// moderation checks against all active rules for the tenant.
func AutoBlockCheck(db *sql.DB, tenantID uuid.UUID, title, body string) (bool, []string, error) {
	combined := title + " " + body
	return CheckContent(db, tenantID.String(), combined)
}

// CreateReview inserts a new moderation review row.
func CreateReview(db *sql.DB, contentID, reviewerID uuid.UUID, level int, autoBlocked bool, reason string) error {
	id := uuid.New()

	var reviewerPtr *uuid.UUID
	if reviewerID != uuid.Nil {
		reviewerPtr = &reviewerID
	}

	var blockedReason *string
	if reason != "" {
		blockedReason = &reason
	}

	_, err := db.Exec(`
		INSERT INTO moderation_reviews (id, content_id, reviewer_id, review_level, status, auto_blocked, blocked_reason)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6)
	`, id, contentID, reviewerPtr, level, autoBlocked, blockedReason)
	if err != nil {
		return fmt.Errorf("failed to create moderation review: %w", err)
	}

	return nil
}

// GetPendingReviews fetches enriched pending moderation reviews at a given level for a tenant.
// Joins content_items and users to provide title, content_type, submitted_by, and severity.
func GetPendingReviews(db *sql.DB, tenantID uuid.UUID, level int) ([]models.EnrichedPendingReview, error) {
	rows, err := db.Query(`
		SELECT mr.id, mr.content_id, ci.title, ci.content_type,
		       COALESCE(u.username, 'unknown') AS submitted_by,
		       mr.created_at AS submitted_at,
		       mr.review_level,
		       CASE
		           WHEN mr.auto_blocked THEN 'critical'
		           WHEN mr.review_level >= 3 THEN 'high'
		           WHEN mr.review_level = 2 THEN 'medium'
		           ELSE 'low'
		       END AS severity,
		       mr.auto_blocked, mr.blocked_reason
		FROM moderation_reviews mr
		JOIN content_items ci ON mr.content_id = ci.id
		LEFT JOIN users u ON ci.created_by = u.id
		WHERE ci.tenant_id = $1
		  AND mr.review_level = $2
		  AND mr.status = 'pending'
		ORDER BY mr.created_at ASC
	`, tenantID, level)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending reviews: %w", err)
	}
	defer rows.Close()

	var reviews []models.EnrichedPendingReview
	for rows.Next() {
		var r models.EnrichedPendingReview
		err := rows.Scan(
			&r.ID, &r.ContentID, &r.Title, &r.ContentType,
			&r.SubmittedBy, &r.SubmittedAt, &r.ReviewLevel,
			&r.Severity, &r.AutoBlocked, &r.BlockedReason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan moderation review: %w", err)
		}
		reviews = append(reviews, r)
	}

	if reviews == nil {
		reviews = []models.EnrichedPendingReview{}
	}

	return reviews, nil
}

