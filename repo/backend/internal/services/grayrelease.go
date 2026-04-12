package services

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	"compliance-console/internal/models"

	"github.com/google/uuid"
)

const grayReleaseDuration = 24 * time.Hour

// StartGrayRelease sets the content item status to 'gray_release' and records
// the gray_release_at timestamp.
func StartGrayRelease(db *sql.DB, contentID uuid.UUID) error {
	result, err := db.Exec(`
		UPDATE content_items
		SET status = 'gray_release', gray_release_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, contentID)
	if err != nil {
		return fmt.Errorf("failed to start gray release: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("content item not found: %s", contentID)
	}

	return nil
}

// CheckGrayReleaseEligibility checks whether 24 hours have elapsed since the
// gray_release_at timestamp. Returns eligibility status and remaining hours.
func CheckGrayReleaseEligibility(db *sql.DB, contentID uuid.UUID) (eligible bool, hoursRemaining float64, err error) {
	var grayReleaseAt *time.Time
	var status string

	err = db.QueryRow(`
		SELECT status, gray_release_at
		FROM content_items
		WHERE id = $1
	`, contentID).Scan(&status, &grayReleaseAt)
	if err == sql.ErrNoRows {
		return false, 0, fmt.Errorf("content item not found: %s", contentID)
	}
	if err != nil {
		return false, 0, fmt.Errorf("failed to check gray release eligibility: %w", err)
	}

	if status != "gray_release" || grayReleaseAt == nil {
		return false, 0, fmt.Errorf("content item is not in gray_release status")
	}

	elapsed := time.Since(*grayReleaseAt)
	if elapsed >= grayReleaseDuration {
		return true, 0, nil
	}

	remaining := grayReleaseDuration - elapsed
	hoursRemaining = math.Ceil(remaining.Hours()*100) / 100

	return false, hoursRemaining, nil
}

// PromoteToPublished promotes a gray-release content item to published status,
// but only if 24 hours have elapsed since the gray release started.
func PromoteToPublished(db *sql.DB, contentID uuid.UUID) error {
	eligible, hoursRemaining, err := CheckGrayReleaseEligibility(db, contentID)
	if err != nil {
		return err
	}

	if !eligible {
		return fmt.Errorf("content not eligible for publish yet, %.2f hours remaining", hoursRemaining)
	}

	result, err := db.Exec(`
		UPDATE content_items
		SET status = 'published', published_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'gray_release'
	`, contentID)
	if err != nil {
		return fmt.Errorf("failed to promote to published: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("content item not found or not in gray_release status")
	}

	return nil
}

// GetGrayReleaseItems returns all content items in gray_release status for a tenant.
func GetGrayReleaseItems(db *sql.DB, tenantID uuid.UUID) ([]models.ContentItem, error) {
	rows, err := db.Query(`
		SELECT id, tenant_id, title, body, content_type, subject, grade, tags,
		       status, gray_release_at, published_at, current_version,
		       created_by, created_at, updated_at
		FROM content_items
		WHERE tenant_id = $1 AND status = 'gray_release'
		ORDER BY gray_release_at ASC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gray release items: %w", err)
	}
	defer rows.Close()

	var items []models.ContentItem
	for rows.Next() {
		var item models.ContentItem
		err := rows.Scan(
			&item.ID, &item.TenantID, &item.Title, &item.Body, &item.ContentType,
			&item.Subject, &item.Grade, &item.Tags,
			&item.Status, &item.GrayReleaseAt, &item.PublishedAt, &item.CurrentVersion,
			&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan content item: %w", err)
		}
		items = append(items, item)
	}

	if items == nil {
		items = []models.ContentItem{}
	}

	return items, nil
}
