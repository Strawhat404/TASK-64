package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"compliance-console/internal/middleware"
	"compliance-console/internal/models"
	"compliance-console/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// GovernanceHandler contains dependencies for content governance endpoints.
type GovernanceHandler struct {
	DB *sql.DB
	AL *services.AuditLedger
}

// NewGovernanceHandler creates a new GovernanceHandler.
func NewGovernanceHandler(db *sql.DB) *GovernanceHandler {
	return &GovernanceHandler{DB: db, AL: services.NewAuditLedger(db)}
}

// govWriteAuditLog is a local helper to insert an audit log entry.
func govWriteAuditLog(db *sql.DB, tenantID *uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, details map[string]interface{}, ipAddress string) {
	id := uuid.New()
	var detailsStr *string
	if details != nil {
		b, _ := json.Marshal(details)
		s := string(b)
		detailsStr = &s
	}
	_, _ = db.Exec(`
		INSERT INTO audit_logs (id, tenant_id, user_id, action, resource_type, resource_id, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::inet)
	`, id, tenantID, userID, action, resourceType, resourceID, detailsStr, ipAddress)
}

// govWriteCriticalAuditLog writes to both mutable audit_logs and immutable audit_ledger.
// Returns an error if the immutable ledger append fails — callers MUST treat this as a hard failure.
func govWriteCriticalAuditLog(al *services.AuditLedger, db *sql.DB, tenantID *uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, details map[string]interface{}, ipAddress string) error {
	// Write to immutable audit_ledger first — if this fails, the action must not proceed
	resID := ""
	if resourceID != nil {
		resID = resourceID.String()
	}
	if err := al.Append(tenantID, userID, action, resourceType, resID, details, ipAddress); err != nil {
		log.Printf("CRITICAL: immutable audit ledger append failed for action=%s: %v", action, err)
		return fmt.Errorf("immutable audit log write failed: %w", err)
	}

	// Also write to mutable audit_logs for general querying
	govWriteAuditLog(db, tenantID, userID, action, resourceType, resourceID, details, ipAddress)
	return nil
}

// ---------- Content CRUD ----------

// CreateContent handles POST for creating new content with auto-block check.
func (h *GovernanceHandler) CreateContent(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	var req models.CreateContentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "title is required",
		})
	}
	if req.Body == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "body is required",
		})
	}

	// Validate content_type
	validTypes := map[string]bool{"article": true, "resource": true, "announcement": true, "policy": true}
	if req.ContentType == "" {
		req.ContentType = "article"
	}
	if !validTypes[req.ContentType] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "content_type must be one of: article, resource, announcement, policy",
		})
	}

	// Auto-block check
	blocked, reasons, err := services.AutoBlockCheck(h.DB, tenantID, req.Title, req.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to run moderation check",
		})
	}

	newID := uuid.New()
	now := time.Now()
	status := "draft"
	if blocked {
		status = "rejected"
	}

	_, err = h.DB.Exec(`
		INSERT INTO content_items (id, tenant_id, title, body, content_type, subject, grade, tags, status, current_version, metadata, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1, $10, $11, $12, $13)
	`, newID, tenantID, req.Title, req.Body, req.ContentType, req.Subject, req.Grade, pq.StringArray(req.Tags), status, req.Metadata, currentUser.ID, now, now)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create content",
		})
	}

	// Create version 1
	_ = services.CreateVersion(h.DB, newID, currentUser.ID, "Initial version")

	// If auto-blocked, create a review record
	if blocked {
		reasonStr := ""
		for i, r := range reasons {
			if i > 0 {
				reasonStr += "; "
			}
			reasonStr += r
		}
		_ = services.CreateReview(h.DB, newID, uuid.Nil, 1, true, reasonStr)
	}

	govWriteAuditLog(h.DB, &tenantID, &currentUser.ID, "create_content", "content_item", &newID, map[string]interface{}{
		"title":   req.Title,
		"blocked": blocked,
	}, c.RealIP())

	// Return the created content
	if blocked {
		return c.JSON(http.StatusCreated, map[string]interface{}{
			"content_id":    newID,
			"status":        status,
			"auto_blocked":  true,
			"block_reasons": reasons,
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"content_id": newID,
		"status":     status,
	})
}

// GetContent handles GET /:id for fetching a single content item.
func (h *GovernanceHandler) GetContent(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid content ID",
		})
	}

	var item models.ContentItem
	var submittedBy *string
	err = h.DB.QueryRow(`
		SELECT ci.id, ci.tenant_id, ci.title, ci.body, ci.content_type, ci.subject, ci.grade, ci.tags,
		       ci.status, ci.gray_release_at, ci.published_at, ci.current_version,
		       ci.metadata, ci.created_by, u.username, ci.created_at, ci.updated_at
		FROM content_items ci
		LEFT JOIN users u ON ci.created_by = u.id
		WHERE ci.id = $1 AND ci.tenant_id = $2
	`, contentID, tenantID).Scan(
		&item.ID, &item.TenantID, &item.Title, &item.Body, &item.ContentType,
		&item.Subject, &item.Grade, &item.Tags,
		&item.Status, &item.GrayReleaseAt, &item.PublishedAt, &item.CurrentVersion,
		&item.Metadata, &item.CreatedBy, &submittedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Content not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch content",
		})
	}
	if submittedBy != nil {
		item.SubmittedBy = *submittedBy
	}
	item.SubmittedAt = &item.CreatedAt

	return c.JSON(http.StatusOK, item)
}

// ListContent handles GET with filters (status, content_type, subject).
func (h *GovernanceHandler) ListContent(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	query := `
		SELECT ci.id, ci.tenant_id, ci.title, ci.body, ci.content_type, ci.subject, ci.grade, ci.tags,
		       ci.status, ci.gray_release_at, ci.published_at, ci.current_version,
		       ci.metadata, ci.created_by, u.username, ci.created_at, ci.updated_at
		FROM content_items ci
		LEFT JOIN users u ON ci.created_by = u.id
		WHERE ci.tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if status := c.QueryParam("status"); status != "" {
		query += fmt.Sprintf(" AND ci.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if contentType := c.QueryParam("content_type"); contentType != "" {
		query += fmt.Sprintf(" AND ci.content_type = $%d", argIdx)
		args = append(args, contentType)
		argIdx++
	}
	if subject := c.QueryParam("subject"); subject != "" {
		query += fmt.Sprintf(" AND ci.subject = $%d", argIdx)
		args = append(args, subject)
		argIdx++
	}

	query += " ORDER BY ci.created_at DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch content",
		})
	}
	defer rows.Close()

	var items []models.ContentItem
	for rows.Next() {
		var item models.ContentItem
		var submittedBy *string
		err := rows.Scan(
			&item.ID, &item.TenantID, &item.Title, &item.Body, &item.ContentType,
			&item.Subject, &item.Grade, &item.Tags,
			&item.Status, &item.GrayReleaseAt, &item.PublishedAt, &item.CurrentVersion,
			&item.Metadata, &item.CreatedBy, &submittedBy, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan content",
			})
		}
		if submittedBy != nil {
			item.SubmittedBy = *submittedBy
		}
		item.SubmittedAt = &item.CreatedAt
		items = append(items, item)
	}

	if items == nil {
		items = []models.ContentItem{}
	}

	return c.JSON(http.StatusOK, items)
}

// UpdateContent handles PUT /:id with auto-block re-check and new version creation.
func (h *GovernanceHandler) UpdateContent(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid content ID",
		})
	}

	var req models.UpdateContentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Fetch existing content
	var existing models.ContentItem
	err = h.DB.QueryRow(`
		SELECT id, tenant_id, title, body, content_type, subject, grade, tags, status
		FROM content_items
		WHERE id = $1 AND tenant_id = $2
	`, contentID, tenantID).Scan(
		&existing.ID, &existing.TenantID, &existing.Title, &existing.Body,
		&existing.ContentType, &existing.Subject, &existing.Grade, &existing.Tags, &existing.Status,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Content not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch content",
		})
	}

	// Apply updates
	title := existing.Title
	body := existing.Body
	contentType := existing.ContentType
	subject := existing.Subject
	grade := existing.Grade
	tags := []string(existing.Tags)

	if req.Title != nil {
		title = *req.Title
	}
	if req.Body != nil {
		body = *req.Body
	}
	if req.ContentType != nil {
		validTypes := map[string]bool{"article": true, "resource": true, "announcement": true, "policy": true}
		if !validTypes[*req.ContentType] {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "content_type must be one of: article, resource, announcement, policy",
			})
		}
		contentType = *req.ContentType
	}
	if req.Subject != nil {
		subject = req.Subject
	}
	if req.Grade != nil {
		grade = req.Grade
	}
	if req.Tags != nil {
		tags = req.Tags
	}

	// Handle metadata update
	if req.Metadata != nil {
		_, _ = h.DB.Exec(`UPDATE content_items SET metadata = $1 WHERE id = $2`, req.Metadata, contentID)
	}

	// Auto-block re-check
	blocked, reasons, err := services.AutoBlockCheck(h.DB, tenantID, title, body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to run moderation check",
		})
	}

	newStatus := existing.Status
	if blocked {
		newStatus = "rejected"
	}

	_, err = h.DB.Exec(`
		UPDATE content_items
		SET title = $2, body = $3, content_type = $4, subject = $5, grade = $6,
		    tags = $7, status = $8, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $9
	`, contentID, title, body, contentType, subject, grade, pq.StringArray(tags), newStatus, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update content",
		})
	}

	// Create new version with release notes
	releaseNotes := ""
	if req.ReleaseNotes != nil {
		releaseNotes = *req.ReleaseNotes
	}
	_ = services.CreateVersion(h.DB, contentID, currentUser.ID, releaseNotes)

	if blocked {
		reasonStr := ""
		for i, r := range reasons {
			if i > 0 {
				reasonStr += "; "
			}
			reasonStr += r
		}
		_ = services.CreateReview(h.DB, contentID, uuid.Nil, 1, true, reasonStr)
	}

	govWriteAuditLog(h.DB, &tenantID, &currentUser.ID, "update_content", "content_item", &contentID, map[string]interface{}{
		"blocked": blocked,
	}, c.RealIP())

	if blocked {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"content_id":    contentID,
			"status":        newStatus,
			"auto_blocked":  true,
			"block_reasons": reasons,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"content_id": contentID,
		"status":     newStatus,
	})
}

// ---------- Review Workflow ----------

// SubmitForReview handles POST /:id/submit to create a level-1 review.
func (h *GovernanceHandler) SubmitForReview(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid content ID",
		})
	}

	var req models.SubmitForReviewRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Update content status to pending_review
	result, err := h.DB.Exec(`
		UPDATE content_items SET status = 'pending_review', updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND status IN ('draft', 'rejected')
	`, contentID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to submit for review",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Content not found or not in a submittable state (must be draft or rejected)",
		})
	}

	// Create level-1 review
	reviewerID := uuid.Nil
	if req.ReviewerID != nil {
		reviewerID = *req.ReviewerID
	}
	err = services.CreateReview(h.DB, contentID, reviewerID, 1, false, "")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create review",
		})
	}

	govWriteAuditLog(h.DB, &tenantID, &currentUser.ID, "submit_for_review", "content_item", &contentID, nil, c.RealIP())

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Content submitted for review",
	})
}

// ReviewDecision handles POST /reviews/:id/decide for approve/reject/escalate.
func (h *GovernanceHandler) ReviewDecision(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	reviewID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid review ID",
		})
	}

	var req models.ReviewDecisionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Decision != "approved" && req.Decision != "rejected" && req.Decision != "escalated" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "decision must be one of: approved, rejected, escalated",
		})
	}

	// Fetch the review
	var review models.ModerationReview
	var contentTenantID uuid.UUID
	err = h.DB.QueryRow(`
		SELECT mr.id, mr.content_id, mr.reviewer_id, mr.review_level, mr.status,
		       ci.tenant_id
		FROM moderation_reviews mr
		JOIN content_items ci ON mr.content_id = ci.id
		WHERE mr.id = $1
	`, reviewID).Scan(
		&review.ID, &review.ContentID, &review.ReviewerID, &review.ReviewLevel, &review.Status,
		&contentTenantID,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Review not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch review",
		})
	}

	if contentTenantID != tenantID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	if review.Status != "pending" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Review has already been decided",
		})
	}

	// Update the review
	var decisionNotes *string
	if req.DecisionNotes != "" {
		decisionNotes = &req.DecisionNotes
	}

	_, err = h.DB.Exec(`
		UPDATE moderation_reviews
		SET status = $2, reviewer_id = $3, decision_notes = $4, decided_at = NOW()
		WHERE id = $1
	`, reviewID, req.Decision, currentUser.ID, decisionNotes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update review",
		})
	}

	// Handle the decision
	switch req.Decision {
	case "approved":
		// Check if there are higher-level pending reviews
		var pendingCount int
		_ = h.DB.QueryRow(`
			SELECT COUNT(*) FROM moderation_reviews
			WHERE content_id = $1 AND review_level > $2 AND status = 'pending'
		`, review.ContentID, review.ReviewLevel).Scan(&pendingCount)

		if pendingCount == 0 {
			// Final approval — update content to approved and start gray release
			_, _ = h.DB.Exec(`
				UPDATE content_items SET status = 'approved', updated_at = NOW()
				WHERE id = $1
			`, review.ContentID)

			// Start gray release
			_ = services.StartGrayRelease(h.DB, review.ContentID)
		} else {
			// Move content to in_review for next level
			_, _ = h.DB.Exec(`
				UPDATE content_items SET status = 'in_review', updated_at = NOW()
				WHERE id = $1
			`, review.ContentID)
		}

	case "rejected":
		_, _ = h.DB.Exec(`
			UPDATE content_items SET status = 'rejected', updated_at = NOW()
			WHERE id = $1
		`, review.ContentID)

	case "escalated":
		// Create a higher-level review
		nextLevel := review.ReviewLevel + 1
		_ = services.CreateReview(h.DB, review.ContentID, uuid.Nil, nextLevel, false, "")

		_, _ = h.DB.Exec(`
			UPDATE content_items SET status = 'in_review', updated_at = NOW()
			WHERE id = $1
		`, review.ContentID)
	}

	if err := govWriteCriticalAuditLog(h.AL, h.DB, &tenantID, &currentUser.ID, "review_decision", "moderation_review", &reviewID, map[string]interface{}{
		"decision":   req.Decision,
		"content_id": review.ContentID,
		"level":      review.ReviewLevel,
	}, c.RealIP()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Audit system unavailable, action denied",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Review %s successfully", req.Decision),
	})
}

// ListPendingReviews handles GET /reviews/pending?level= to list pending reviews.
func (h *GovernanceHandler) ListPendingReviews(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	level := 1
	if l := c.QueryParam("level"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err == nil && parsed > 0 {
			level = parsed
		}
	}

	reviews, err := services.GetPendingReviews(h.DB, tenantID, level)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch pending reviews",
		})
	}

	return c.JSON(http.StatusOK, reviews)
}

// ---------- Gray Release ----------

// GetGrayReleaseItems handles GET /gray-release to list items in gray release.
func (h *GovernanceHandler) GetGrayReleaseItems(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	items, err := services.GetGrayReleaseItems(h.DB, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch gray release items",
		})
	}

	return c.JSON(http.StatusOK, items)
}

// PromoteContent handles POST /:id/promote to promote gray-release content.
func (h *GovernanceHandler) PromoteContent(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid content ID",
		})
	}

	// Verify content belongs to tenant
	var itemTenantID uuid.UUID
	err = h.DB.QueryRow(`SELECT tenant_id FROM content_items WHERE id = $1`, contentID).Scan(&itemTenantID)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Content not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch content",
		})
	}
	if itemTenantID != tenantID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	err = services.PromoteToPublished(h.DB, contentID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if err := govWriteCriticalAuditLog(h.AL, h.DB, &tenantID, &currentUser.ID, "promote_content", "content_item", &contentID, nil, c.RealIP()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Audit system unavailable, action denied",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Content promoted to published",
	})
}

// ---------- Versioning ----------

// GetVersionHistory handles GET /:id/versions to list version history.
func (h *GovernanceHandler) GetVersionHistory(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid content ID",
		})
	}

	// Verify content belongs to tenant
	var itemTenantID uuid.UUID
	err = h.DB.QueryRow(`SELECT tenant_id FROM content_items WHERE id = $1`, contentID).Scan(&itemTenantID)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Content not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch content",
		})
	}
	if itemTenantID != tenantID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	versions, err := services.GetVersionHistory(h.DB, contentID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch version history",
		})
	}

	return c.JSON(http.StatusOK, versions)
}

// RollbackContent handles POST /:id/rollback to rollback content to a target version.
func (h *GovernanceHandler) RollbackContent(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid content ID",
		})
	}

	var req models.RollbackRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.TargetVersion < 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "target_version must be a positive integer",
		})
	}

	// Verify content belongs to tenant
	var itemTenantID uuid.UUID
	err = h.DB.QueryRow(`SELECT tenant_id FROM content_items WHERE id = $1`, contentID).Scan(&itemTenantID)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Content not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch content",
		})
	}
	if itemTenantID != tenantID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	err = services.RollbackToVersion(h.DB, contentID, req.TargetVersion, currentUser.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if err := govWriteCriticalAuditLog(h.AL, h.DB, &tenantID, &currentUser.ID, "rollback_content", "content_item", &contentID, map[string]interface{}{
		"target_version": req.TargetVersion,
	}, c.RealIP()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Audit system unavailable, action denied",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Content rolled back to version %d", req.TargetVersion),
	})
}

// ---------- Moderation Rules ----------

// ListRules handles GET /rules to list moderation rules for the tenant.
func (h *GovernanceHandler) ListRules(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	rows, err := h.DB.Query(`
		SELECT id, tenant_id, rule_type, pattern, severity, is_active, created_by, created_at, updated_at
		FROM moderation_rules
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch moderation rules",
		})
	}
	defer rows.Close()

	var rules []models.ModerationRule
	for rows.Next() {
		var r models.ModerationRule
		err := rows.Scan(
			&r.ID, &r.TenantID, &r.RuleType, &r.Pattern, &r.Severity,
			&r.IsActive, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan moderation rule",
			})
		}
		rules = append(rules, r)
	}

	if rules == nil {
		rules = []models.ModerationRule{}
	}

	return c.JSON(http.StatusOK, rules)
}

// CreateRule handles POST /rules to create a new moderation rule.
func (h *GovernanceHandler) CreateRule(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	var req models.CreateModerationRuleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate rule_type
	validTypes := map[string]bool{"keyword_block": true, "regex_block": true, "manual_review": true}
	if !validTypes[req.RuleType] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "rule_type must be one of: keyword_block, regex_block, manual_review",
		})
	}

	if req.Pattern == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "pattern is required",
		})
	}

	// Validate severity
	validSeverities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if req.Severity == "" {
		req.Severity = "medium"
	}
	if !validSeverities[req.Severity] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "severity must be one of: low, medium, high, critical",
		})
	}

	newID := uuid.New()
	now := time.Now()

	_, err := h.DB.Exec(`
		INSERT INTO moderation_rules (id, tenant_id, rule_type, pattern, severity, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, TRUE, $6, $7, $8)
	`, newID, tenantID, req.RuleType, req.Pattern, req.Severity, currentUser.ID, now, now)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create moderation rule",
		})
	}

	govWriteAuditLog(h.DB, &tenantID, &currentUser.ID, "create_moderation_rule", "moderation_rule", &newID, map[string]interface{}{
		"rule_type": req.RuleType,
		"pattern":   req.Pattern,
		"severity":  req.Severity,
	}, c.RealIP())

	// Fetch and return
	var rule models.ModerationRule
	_ = h.DB.QueryRow(`
		SELECT id, tenant_id, rule_type, pattern, severity, is_active, created_by, created_at, updated_at
		FROM moderation_rules WHERE id = $1
	`, newID).Scan(
		&rule.ID, &rule.TenantID, &rule.RuleType, &rule.Pattern, &rule.Severity,
		&rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)

	return c.JSON(http.StatusCreated, rule)
}

// UpdateRule handles PUT /rules/:id to update a moderation rule.
func (h *GovernanceHandler) UpdateRule(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid rule ID",
		})
	}

	var req struct {
		RuleType *string `json:"rule_type,omitempty"`
		Pattern  *string `json:"pattern,omitempty"`
		Severity *string `json:"severity,omitempty"`
		IsActive *bool   `json:"is_active,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	query := "UPDATE moderation_rules SET updated_at = NOW()"
	args := []interface{}{}
	argIdx := 1

	if req.RuleType != nil {
		validTypes := map[string]bool{"keyword_block": true, "regex_block": true, "manual_review": true}
		if !validTypes[*req.RuleType] {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "rule_type must be one of: keyword_block, regex_block, manual_review",
			})
		}
		query += fmt.Sprintf(", rule_type = $%d", argIdx)
		args = append(args, *req.RuleType)
		argIdx++
	}
	if req.Pattern != nil {
		query += fmt.Sprintf(", pattern = $%d", argIdx)
		args = append(args, *req.Pattern)
		argIdx++
	}
	if req.Severity != nil {
		validSeverities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
		if !validSeverities[*req.Severity] {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "severity must be one of: low, medium, high, critical",
			})
		}
		query += fmt.Sprintf(", severity = $%d", argIdx)
		args = append(args, *req.Severity)
		argIdx++
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIdx)
		args = append(args, *req.IsActive)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argIdx, argIdx+1)
	args = append(args, ruleID, tenantID)

	result, err := h.DB.Exec(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update moderation rule",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Moderation rule not found",
		})
	}

	govWriteAuditLog(h.DB, &tenantID, &currentUser.ID, "update_moderation_rule", "moderation_rule", &ruleID, nil, c.RealIP())

	// Fetch and return updated rule
	var rule models.ModerationRule
	_ = h.DB.QueryRow(`
		SELECT id, tenant_id, rule_type, pattern, severity, is_active, created_by, created_at, updated_at
		FROM moderation_rules WHERE id = $1
	`, ruleID).Scan(
		&rule.ID, &rule.TenantID, &rule.RuleType, &rule.Pattern, &rule.Severity,
		&rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)

	return c.JSON(http.StatusOK, rule)
}

// DeleteRule handles DELETE /rules/:id to delete a moderation rule.
func (h *GovernanceHandler) DeleteRule(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid rule ID",
		})
	}

	result, err := h.DB.Exec(`
		DELETE FROM moderation_rules WHERE id = $1 AND tenant_id = $2
	`, ruleID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete moderation rule",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Moderation rule not found",
		})
	}

	govWriteAuditLog(h.DB, &tenantID, &currentUser.ID, "delete_moderation_rule", "moderation_rule", &ruleID, nil, c.RealIP())

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Moderation rule deleted successfully",
	})
}

// ---------- Version Diff ----------

// DiffVersions handles GET /content/:id/versions/diff?v1=&v2= to compare two versions.
func (h *GovernanceHandler) DiffVersions(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid content ID",
		})
	}

	v1, err := strconv.Atoi(c.QueryParam("v1"))
	if err != nil || v1 < 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "v1 query parameter must be a positive integer",
		})
	}
	v2, err := strconv.Atoi(c.QueryParam("v2"))
	if err != nil || v2 < 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "v2 query parameter must be a positive integer",
		})
	}

	// Verify content belongs to tenant
	var itemTenantID uuid.UUID
	err = h.DB.QueryRow(`SELECT tenant_id FROM content_items WHERE id = $1`, contentID).Scan(&itemTenantID)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Content not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch content",
		})
	}
	if itemTenantID != tenantID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	// Fetch both versions
	var ver1, ver2 models.ContentVersion
	err = h.DB.QueryRow(`
		SELECT id, content_id, version_number, title, body, subject, grade, tags, release_notes, created_by, created_at
		FROM content_versions WHERE content_id = $1 AND version_number = $2
	`, contentID, v1).Scan(
		&ver1.ID, &ver1.ContentID, &ver1.VersionNumber, &ver1.Title, &ver1.Body,
		&ver1.Subject, &ver1.Grade, &ver1.Tags, &ver1.ReleaseNotes, &ver1.CreatedBy, &ver1.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Version %d not found", v1),
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch version",
		})
	}

	err = h.DB.QueryRow(`
		SELECT id, content_id, version_number, title, body, subject, grade, tags, release_notes, created_by, created_at
		FROM content_versions WHERE content_id = $1 AND version_number = $2
	`, contentID, v2).Scan(
		&ver2.ID, &ver2.ContentID, &ver2.VersionNumber, &ver2.Title, &ver2.Body,
		&ver2.Subject, &ver2.Grade, &ver2.Tags, &ver2.ReleaseNotes, &ver2.CreatedBy, &ver2.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Version %d not found", v2),
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch version",
		})
	}

	// Build diff response
	diff := map[string]interface{}{
		"content_id": contentID,
		"v1":         v1,
		"v2":         v2,
		"changes":    map[string]interface{}{},
	}

	changes := map[string]interface{}{}
	if ver1.Title != ver2.Title {
		changes["title"] = map[string]string{"old": ver1.Title, "new": ver2.Title}
	}
	if ver1.Body != ver2.Body {
		changes["body"] = map[string]string{"old": ver1.Body, "new": ver2.Body}
	}
	if ptrStrVal(ver1.Subject) != ptrStrVal(ver2.Subject) {
		changes["subject"] = map[string]string{"old": ptrStrVal(ver1.Subject), "new": ptrStrVal(ver2.Subject)}
	}
	if ptrStrVal(ver1.Grade) != ptrStrVal(ver2.Grade) {
		changes["grade"] = map[string]string{"old": ptrStrVal(ver1.Grade), "new": ptrStrVal(ver2.Grade)}
	}

	diff["changes"] = changes

	return c.JSON(http.StatusOK, diff)
}

// ptrStrVal returns the value of a *string or empty string if nil.
func ptrStrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ---------- Resource Relationships ----------

// CreateRelationship handles POST /api/governance/relationships.
func (h *GovernanceHandler) CreateRelationship(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	var req models.CreateResourceRelationshipRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.SourceContentID == uuid.Nil || req.TargetContentID == uuid.Nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "source_content_id and target_content_id are required"})
	}

	validTypes := map[string]bool{"dependency": true, "substitute": true, "bundle": true}
	if !validTypes[req.RelationshipType] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "relationship_type must be one of: dependency, substitute, bundle"})
	}

	if req.SourceContentID == req.TargetContentID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "source and target cannot be the same content"})
	}

	// Validate both content items belong to the caller's tenant
	var sourceExists, targetExists bool
	_ = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM content_items WHERE id = $1 AND tenant_id = $2)", req.SourceContentID, tenantID).Scan(&sourceExists)
	_ = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM content_items WHERE id = $1 AND tenant_id = $2)", req.TargetContentID, tenantID).Scan(&targetExists)
	if !sourceExists || !targetExists {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Both source and target content items must belong to your tenant"})
	}

	newID := uuid.New()
	now := time.Now()

	_, err := h.DB.Exec(`
		INSERT INTO resource_relationships (id, tenant_id, source_content_id, target_content_id, relationship_type, notes, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, newID, tenantID, req.SourceContentID, req.TargetContentID, req.RelationshipType, req.Notes, currentUser.ID, now)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create relationship"})
	}

	govWriteAuditLog(h.DB, &tenantID, &currentUser.ID, "create_relationship", "resource_relationship", &newID, map[string]interface{}{
		"source": req.SourceContentID, "target": req.TargetContentID, "type": req.RelationshipType,
	}, c.RealIP())

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": newID, "relationship_type": req.RelationshipType})
}

// ListRelationships handles GET /api/governance/relationships.
func (h *GovernanceHandler) ListRelationships(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	query := `
		SELECT r.id, r.tenant_id, r.source_content_id, r.target_content_id,
		       r.relationship_type, r.notes, r.created_by, r.created_at
		FROM resource_relationships r
		WHERE r.tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if contentID := c.QueryParam("content_id"); contentID != "" {
		query += fmt.Sprintf(" AND (r.source_content_id = $%d OR r.target_content_id = $%d)", argIdx, argIdx)
		args = append(args, contentID)
		argIdx++
	}
	if relType := c.QueryParam("type"); relType != "" {
		query += fmt.Sprintf(" AND r.relationship_type = $%d", argIdx)
		args = append(args, relType)
		argIdx++
	}

	query += " ORDER BY r.created_at DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch relationships"})
	}
	defer rows.Close()

	var rels []models.ResourceRelationship
	for rows.Next() {
		var r models.ResourceRelationship
		if err := rows.Scan(&r.ID, &r.TenantID, &r.SourceContentID, &r.TargetContentID,
			&r.RelationshipType, &r.Notes, &r.CreatedBy, &r.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to scan relationship"})
		}
		rels = append(rels, r)
	}
	if rels == nil {
		rels = []models.ResourceRelationship{}
	}

	return c.JSON(http.StatusOK, rels)
}

// DeleteRelationship handles DELETE /api/governance/relationships/:id.
func (h *GovernanceHandler) DeleteRelationship(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	relID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid relationship ID"})
	}

	result, err := h.DB.Exec(`DELETE FROM resource_relationships WHERE id = $1 AND tenant_id = $2`, relID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete relationship"})
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Relationship not found"})
	}

	govWriteAuditLog(h.DB, &tenantID, &currentUser.ID, "delete_relationship", "resource_relationship", &relID, nil, c.RealIP())

	return c.JSON(http.StatusOK, map[string]string{"message": "Relationship deleted"})
}

// ReReview handles POST /api/governance/content/:id/re-review.
func (h *GovernanceHandler) ReReview(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid content ID"})
	}

	// Get current content status
	var status string
	err = h.DB.QueryRow("SELECT status FROM content_items WHERE id = $1 AND tenant_id = $2", contentID, tenantID).Scan(&status)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Content not found"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch content"})
	}

	// Can only re-review content that was previously approved, rejected, or published
	allowedStatuses := map[string]bool{"approved": true, "rejected": true, "published": true, "gray_release": true}
	if !allowedStatuses[status] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Content must be approved, rejected, or published to request re-review"})
	}

	// Set status back to pending_review
	_, err = h.DB.Exec(`UPDATE content_items SET status = 'pending_review', updated_at = NOW() WHERE id = $1`, contentID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update content status"})
	}

	// Create a new review at level 1
	_ = services.CreateReview(h.DB, contentID, uuid.Nil, 1, false, "Re-review requested")

	govWriteAuditLog(h.DB, &tenantID, &currentUser.ID, "re_review", "content_item", &contentID, map[string]interface{}{
		"previous_status": status,
	}, c.RealIP())

	return c.JSON(http.StatusOK, map[string]string{"message": "Content submitted for re-review"})
}
