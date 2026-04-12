package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ModerationRule represents a moderation rule for auto-blocking content.
type ModerationRule struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TenantID  uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	RuleType  string     `json:"rule_type" db:"rule_type"`
	Pattern   string     `json:"pattern" db:"pattern"`
	Severity  string     `json:"severity" db:"severity"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// ContentItem represents a content item that goes through moderation.
type ContentItem struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	TenantID       uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	Title          string         `json:"title" db:"title"`
	Body           string         `json:"body" db:"body"`
	ContentType    string         `json:"content_type" db:"content_type"`
	Subject        *string        `json:"subject,omitempty" db:"subject"`
	Grade          *string        `json:"grade,omitempty" db:"grade"`
	Tags           pq.StringArray `json:"tags" db:"tags"`
	Status         string         `json:"status" db:"status"`
	GrayReleaseAt  *time.Time     `json:"gray_release_at,omitempty" db:"gray_release_at"`
	PublishedAt    *time.Time     `json:"published_at,omitempty" db:"published_at"`
	CurrentVersion int              `json:"current_version" db:"current_version"`
	Metadata       *json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CreatedBy      uuid.UUID        `json:"created_by" db:"created_by"`
	SubmittedBy    string           `json:"submitted_by,omitempty"`
	SubmittedAt    *time.Time       `json:"submitted_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
}

// ContentVersion represents a versioned snapshot of a content item.
type ContentVersion struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	ContentID     uuid.UUID      `json:"content_id" db:"content_id"`
	VersionNumber int            `json:"version_number" db:"version_number"`
	Title         string         `json:"title" db:"title"`
	Body          string         `json:"body" db:"body"`
	Subject       *string        `json:"subject,omitempty" db:"subject"`
	Grade         *string        `json:"grade,omitempty" db:"grade"`
	Tags          pq.StringArray `json:"tags" db:"tags"`
	ReleaseNotes  *string        `json:"release_notes,omitempty" db:"release_notes"`
	CreatedBy     uuid.UUID      `json:"created_by" db:"created_by"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
}

// ModerationReview represents a review entry in the moderation queue.
type ModerationReview struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	ContentID     uuid.UUID  `json:"content_id" db:"content_id"`
	ReviewerID    *uuid.UUID `json:"reviewer_id,omitempty" db:"reviewer_id"`
	ReviewLevel   int        `json:"review_level" db:"review_level"`
	Status        string     `json:"status" db:"status"`
	DecisionNotes *string    `json:"decision_notes,omitempty" db:"decision_notes"`
	AutoBlocked   bool       `json:"auto_blocked" db:"auto_blocked"`
	BlockedReason *string    `json:"blocked_reason,omitempty" db:"blocked_reason"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	DecidedAt     *time.Time `json:"decided_at,omitempty" db:"decided_at"`
}

// EnrichedPendingReview is a DTO that includes content metadata for the frontend pending reviews table.
type EnrichedPendingReview struct {
	ID            uuid.UUID  `json:"id"`
	ContentID     uuid.UUID  `json:"content_id"`
	Title         string     `json:"title"`
	ContentType   string     `json:"content_type"`
	SubmittedBy   string     `json:"submitted_by"`
	SubmittedAt   time.Time  `json:"submitted_at"`
	ReviewLevel   int        `json:"review_level"`
	Severity      string     `json:"severity"`
	AutoBlocked   bool       `json:"auto_blocked"`
	BlockedReason *string    `json:"blocked_reason,omitempty"`
}

// CreateContentRequest is the payload for creating new content.
type CreateContentRequest struct {
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	ContentType string           `json:"content_type"`
	Subject     *string          `json:"subject,omitempty"`
	Grade       *string          `json:"grade,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Metadata    *json.RawMessage `json:"metadata,omitempty"`
}

// UpdateContentRequest is the payload for updating content.
type UpdateContentRequest struct {
	Title        *string          `json:"title,omitempty"`
	Body         *string          `json:"body,omitempty"`
	ContentType  *string          `json:"content_type,omitempty"`
	Subject      *string          `json:"subject,omitempty"`
	Grade        *string          `json:"grade,omitempty"`
	Tags         []string         `json:"tags,omitempty"`
	ReleaseNotes *string          `json:"release_notes,omitempty"`
	Metadata     *json.RawMessage `json:"metadata,omitempty"`
}

// SubmitForReviewRequest is the payload for submitting content for review.
type SubmitForReviewRequest struct {
	ReviewerID *uuid.UUID `json:"reviewer_id,omitempty"`
}

// ReviewDecisionRequest is the payload for making a review decision.
type ReviewDecisionRequest struct {
	Decision      string `json:"decision"`
	DecisionNotes string `json:"decision_notes,omitempty"`
}

// CreateModerationRuleRequest is the payload for creating a moderation rule.
type CreateModerationRuleRequest struct {
	RuleType string `json:"rule_type"`
	Pattern  string `json:"pattern"`
	Severity string `json:"severity"`
}

// RollbackRequest is the payload for rolling back content to a specific version.
type RollbackRequest struct {
	TargetVersion int `json:"target_version"`
}

// GrayReleaseStatus shows the gray release state of a content item.
type GrayReleaseStatus struct {
	ContentID          uuid.UUID  `json:"content_id"`
	GrayReleaseAt      *time.Time `json:"gray_release_at"`
	HoursRemaining     float64    `json:"hours_remaining"`
	IsEligibleForPublish bool     `json:"is_eligible_for_publish"`
}

// ResourceRelationship represents a dependency, substitute, or bundle relationship between content items.
type ResourceRelationship struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	TenantID         uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	SourceContentID  uuid.UUID  `json:"source_content_id" db:"source_content_id"`
	TargetContentID  uuid.UUID  `json:"target_content_id" db:"target_content_id"`
	RelationshipType string     `json:"relationship_type" db:"relationship_type"`
	Notes            *string    `json:"notes,omitempty" db:"notes"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// CreateResourceRelationshipRequest is the payload for creating a resource relationship.
type CreateResourceRelationshipRequest struct {
	SourceContentID  uuid.UUID `json:"source_content_id"`
	TargetContentID  uuid.UUID `json:"target_content_id"`
	RelationshipType string    `json:"relationship_type"`
	Notes            *string   `json:"notes,omitempty"`
}
