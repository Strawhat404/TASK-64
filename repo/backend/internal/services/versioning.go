package services

import (
	"database/sql"
	"fmt"

	"compliance-console/internal/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const maxVersions = 10

// CreateVersion snapshots the current content_items state into content_versions,
// increments the version number, and enforces a maximum of 10 versions by
// deleting the oldest if exceeded.
func CreateVersion(db *sql.DB, contentID uuid.UUID, userID uuid.UUID, releaseNotes string) error {
	// Fetch the current content item state
	var title, body string
	var subject, grade *string
	var tags []string
	var currentVersion int

	err := db.QueryRow(`
		SELECT title, body, subject, grade, tags, current_version
		FROM content_items
		WHERE id = $1
	`, contentID).Scan(&title, &body, &subject, &grade, (*pq.StringArray)(&tags), &currentVersion)
	if err == sql.ErrNoRows {
		return fmt.Errorf("content item not found: %s", contentID)
	}
	if err != nil {
		return fmt.Errorf("failed to fetch content item: %w", err)
	}

	newVersion := currentVersion

	var releaseNotesPtr *string
	if releaseNotes != "" {
		releaseNotesPtr = &releaseNotes
	}

	// Insert the version snapshot
	versionID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO content_versions (id, content_id, version_number, title, body, subject, grade, tags, release_notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, versionID, contentID, newVersion, title, body, subject, grade, pq.StringArray(tags), releaseNotesPtr, userID)
	if err != nil {
		return fmt.Errorf("failed to create content version: %w", err)
	}

	// Increment current_version on the content item
	_, err = db.Exec(`
		UPDATE content_items SET current_version = current_version + 1, updated_at = NOW()
		WHERE id = $1
	`, contentID)
	if err != nil {
		return fmt.Errorf("failed to increment content version: %w", err)
	}

	// Enforce max 10 versions: delete the oldest if exceeding
	_, err = db.Exec(`
		DELETE FROM content_versions
		WHERE id IN (
			SELECT id FROM content_versions
			WHERE content_id = $1
			ORDER BY version_number DESC
			OFFSET $2
		)
	`, contentID, maxVersions)
	if err != nil {
		return fmt.Errorf("failed to prune old versions: %w", err)
	}

	return nil
}

// GetVersionHistory lists all versions for a content item in descending order.
func GetVersionHistory(db *sql.DB, contentID uuid.UUID) ([]models.ContentVersion, error) {
	rows, err := db.Query(`
		SELECT id, content_id, version_number, title, body, subject, grade, tags,
		       release_notes, created_by, created_at
		FROM content_versions
		WHERE content_id = $1
		ORDER BY version_number DESC
	`, contentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version history: %w", err)
	}
	defer rows.Close()

	var versions []models.ContentVersion
	for rows.Next() {
		var v models.ContentVersion
		err := rows.Scan(
			&v.ID, &v.ContentID, &v.VersionNumber, &v.Title, &v.Body,
			&v.Subject, &v.Grade, &v.Tags, &v.ReleaseNotes, &v.CreatedBy, &v.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan content version: %w", err)
		}
		versions = append(versions, v)
	}

	if versions == nil {
		versions = []models.ContentVersion{}
	}

	return versions, nil
}

// RollbackToVersion restores content_items fields from the target version row,
// creates a new version entry recording the rollback, and validates the target
// version exists and is within the last 10 versions.
func RollbackToVersion(db *sql.DB, contentID uuid.UUID, targetVersion int, userID uuid.UUID) error {
	// Validate the target version exists
	var v models.ContentVersion
	err := db.QueryRow(`
		SELECT id, content_id, version_number, title, body, subject, grade, tags
		FROM content_versions
		WHERE content_id = $1 AND version_number = $2
	`, contentID, targetVersion).Scan(
		&v.ID, &v.ContentID, &v.VersionNumber, &v.Title, &v.Body,
		&v.Subject, &v.Grade, &v.Tags,
	)
	if err == sql.ErrNoRows {
		return fmt.Errorf("version %d not found for content %s", targetVersion, contentID)
	}
	if err != nil {
		return fmt.Errorf("failed to fetch target version: %w", err)
	}

	// Validate the target version is within the last 10 kept versions
	var versionCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM content_versions
		WHERE content_id = $1 AND version_number >= $2
	`, contentID, targetVersion).Scan(&versionCount)
	if err != nil {
		return fmt.Errorf("failed to validate version range: %w", err)
	}
	if versionCount > maxVersions {
		return fmt.Errorf("target version %d is outside the last %d versions", targetVersion, maxVersions)
	}

	// Restore content_items fields from the target version
	_, err = db.Exec(`
		UPDATE content_items
		SET title = $2, body = $3, subject = $4, grade = $5, tags = $6, updated_at = NOW()
		WHERE id = $1
	`, contentID, v.Title, v.Body, v.Subject, v.Grade, v.Tags)
	if err != nil {
		return fmt.Errorf("failed to rollback content: %w", err)
	}

	// Create a new version entry recording the rollback
	releaseNotes := fmt.Sprintf("Rollback to version %d", targetVersion)
	err = CreateVersion(db, contentID, userID, releaseNotes)
	if err != nil {
		return fmt.Errorf("failed to create rollback version entry: %w", err)
	}

	return nil
}
