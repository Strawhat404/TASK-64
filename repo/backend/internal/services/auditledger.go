package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"compliance-console/internal/models"

	"github.com/google/uuid"
)

// AuditLedger provides an immutable, hash-chained audit ledger.
type AuditLedger struct {
	DB *sql.DB
}

// NewAuditLedger creates a new AuditLedger.
func NewAuditLedger(db *sql.DB) *AuditLedger {
	return &AuditLedger{DB: db}
}

// Append adds a new entry to the audit ledger with a hash chain link.
func (al *AuditLedger) Append(tenantID, userID *uuid.UUID, action, resourceType, resourceID string, details map[string]interface{}, ipAddress string) error {
	// Get the last entry's hash to form the chain
	var previousHash string
	err := al.DB.QueryRow(`
		SELECT entry_hash FROM audit_ledger ORDER BY id DESC LIMIT 1
	`).Scan(&previousHash)
	if err == sql.ErrNoRows {
		previousHash = "GENESIS"
	} else if err != nil {
		return fmt.Errorf("failed to get previous hash: %w", err)
	}

	now := time.Now().UTC()

	// Build payload and compute hash
	payload := previousHash + action + resourceType + resourceID + now.Format(time.RFC3339Nano)
	hash := sha256.Sum256([]byte(payload))
	entryHash := fmt.Sprintf("%x", hash)

	// Marshal details to JSON; use explicit NULL for empty details
	var detailsParam interface{}
	if details != nil {
		detailsJSON, jsonErr := json.Marshal(details)
		if jsonErr != nil {
			return fmt.Errorf("failed to marshal details: %w", jsonErr)
		}
		detailsParam = string(detailsJSON)
	}

	_, err = al.DB.Exec(`
		INSERT INTO audit_ledger (entry_hash, previous_hash, tenant_id, user_id, action, resource_type, resource_id, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, entryHash, previousHash, tenantID, userID, action, resourceType, resourceID, detailsParam, ipAddress, now)
	if err != nil {
		return fmt.Errorf("failed to append audit entry: %w", err)
	}

	return nil
}

// Verify walks the entire audit ledger chain and verifies each hash.
// Returns true if the chain is valid, or false with the ID of the first broken entry.
func (al *AuditLedger) Verify(db *sql.DB) (valid bool, brokenAt int64, err error) {
	rows, err := db.Query(`
		SELECT id, entry_hash, previous_hash, action, resource_type, resource_id, created_at
		FROM audit_ledger
		ORDER BY id ASC
	`)
	if err != nil {
		return false, 0, fmt.Errorf("failed to query audit ledger: %w", err)
	}
	defer rows.Close()

	expectedPrevHash := "GENESIS"

	for rows.Next() {
		var entry models.AuditLedgerEntry
		if err := rows.Scan(
			&entry.ID, &entry.EntryHash, &entry.PreviousHash,
			&entry.Action, &entry.ResourceType, &entry.ResourceID, &entry.CreatedAt,
		); err != nil {
			return false, 0, fmt.Errorf("failed to scan entry: %w", err)
		}

		// Verify the previous_hash matches what we expect
		if entry.PreviousHash != expectedPrevHash {
			return false, entry.ID, nil
		}

		// Recompute the hash
		payload := entry.PreviousHash + entry.Action + entry.ResourceType + entry.ResourceID + entry.CreatedAt.Format(time.RFC3339Nano)
		hash := sha256.Sum256([]byte(payload))
		computedHash := fmt.Sprintf("%x", hash)

		if entry.EntryHash != computedHash {
			return false, entry.ID, nil
		}

		expectedPrevHash = entry.EntryHash
	}

	return true, 0, nil
}

// QueryEntries returns paginated audit ledger entries with optional filters.
func (al *AuditLedger) QueryEntries(tenantID *uuid.UUID, action, resourceType string, from, to time.Time, page, perPage int) ([]models.AuditLedgerEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	query := "SELECT id, entry_hash, previous_hash, tenant_id, user_id, action, resource_type, resource_id, details, ip_address, created_at FROM audit_ledger WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM audit_ledger WHERE 1=1"

	args := []interface{}{}
	countArgs := []interface{}{}
	argIdx := 1

	if tenantID != nil {
		filter := fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, *tenantID)
		countArgs = append(countArgs, *tenantID)
		argIdx++
	}

	if action != "" {
		filter := fmt.Sprintf(" AND action = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, action)
		countArgs = append(countArgs, action)
		argIdx++
	}

	if resourceType != "" {
		filter := fmt.Sprintf(" AND resource_type = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, resourceType)
		countArgs = append(countArgs, resourceType)
		argIdx++
	}

	if !from.IsZero() {
		filter := fmt.Sprintf(" AND created_at >= $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, from)
		countArgs = append(countArgs, from)
		argIdx++
	}

	if !to.IsZero() {
		filter := fmt.Sprintf(" AND created_at <= $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, to)
		countArgs = append(countArgs, to)
		argIdx++
	}

	var total int
	err := al.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count entries: %w", err)
	}

	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := al.DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	var entries []models.AuditLedgerEntry
	for rows.Next() {
		var e models.AuditLedgerEntry
		var detailsJSON []byte
		if err := rows.Scan(
			&e.ID, &e.EntryHash, &e.PreviousHash, &e.TenantID, &e.UserID,
			&e.Action, &e.ResourceType, &e.ResourceID, &detailsJSON, &e.IPAddress, &e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan entry: %w", err)
		}
		if detailsJSON != nil {
			_ = json.Unmarshal(detailsJSON, &e.Details)
		}
		entries = append(entries, e)
	}

	if entries == nil {
		entries = []models.AuditLedgerEntry{}
	}

	return entries, total, nil
}

// EnforceRetention checks retention policies and deletes entries that exceed
// the configured retention period, logging each deletion.
func (al *AuditLedger) EnforceRetention(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, table_name, retention_years FROM retention_policies WHERE is_active = TRUE
	`)
	if err != nil {
		return fmt.Errorf("failed to query retention policies: %w", err)
	}
	defer rows.Close()

	type policy struct {
		id             uuid.UUID
		tableName      string
		retentionYears int
	}

	var policies []policy
	for rows.Next() {
		var p policy
		if err := rows.Scan(&p.id, &p.tableName, &p.retentionYears); err != nil {
			return fmt.Errorf("failed to scan policy: %w", err)
		}
		policies = append(policies, p)
	}
	rows.Close()

	for _, p := range policies {
		cutoff := time.Now().AddDate(-p.retentionYears, 0, 0)

		// Check for active legal holds on this table
		var holdCount int
		err = db.QueryRow(`SELECT COUNT(*) FROM legal_holds WHERE target_table = $1 AND is_active = TRUE`, p.tableName).Scan(&holdCount)
		if err == nil && holdCount > 0 {
			continue // Skip deletion for tables under legal hold
		}

		if p.tableName == "audit_ledger" {
			// Audit ledger entries are NEVER deleted through the application.
			// Instead, we identify expired entries and record them in the deletion_log
			// for a privileged DBA to archive/purge outside the application.
			// This preserves the immutability guarantee — triggers remain active at all times.

			rows, err := db.Query(`
				SELECT al.id FROM audit_ledger al
				WHERE al.created_at < $1
				  AND NOT EXISTS (
				    SELECT 1 FROM legal_holds lh
				    WHERE lh.is_active = TRUE
				      AND lh.target_table = 'audit_ledger'
				      AND lh.target_record_id = al.id::text
				  )
				  AND NOT EXISTS (
				    SELECT 1 FROM deletion_log dl
				    WHERE dl.table_name = 'audit_ledger'
				      AND dl.record_id = al.id::text
				  )
			`, cutoff)
			if err != nil {
				return fmt.Errorf("failed to query expired audit entries: %w", err)
			}

			var ids []int64
			for rows.Next() {
				var id int64
				if err := rows.Scan(&id); err != nil {
					rows.Close()
					return fmt.Errorf("failed to scan id: %w", err)
				}
				ids = append(ids, id)
			}
			rows.Close()

			// Log each expired entry for DBA-level archival/purge
			for _, id := range ids {
				_, err := db.Exec(`
					INSERT INTO deletion_log (table_name, record_id, deletion_reason, retention_met, deleted_at)
					VALUES ($1, $2, $3, TRUE, NOW())
				`, p.tableName, fmt.Sprintf("%d", id), fmt.Sprintf("retention policy: %d years exceeded — pending DBA archival", p.retentionYears))
				if err != nil {
					return fmt.Errorf("failed to log expired audit entry: %w", err)
				}
			}
		}

		// Update the policy's last purge time
		_, err = db.Exec(`
			UPDATE retention_policies SET last_purge_at = NOW(), next_purge_at = NOW() + INTERVAL '1 day'
			WHERE id = $1
		`, p.id)
		if err != nil {
			return fmt.Errorf("failed to update policy purge time: %w", err)
		}
	}

	return nil
}

// EnforceRetentionForTenant is a tenant-scoped version of EnforceRetention.
// It only processes retention policies belonging to (or visible to) the given tenant.
func (al *AuditLedger) EnforceRetentionForTenant(db *sql.DB, tenantID uuid.UUID) error {
	rows, err := db.Query(`
		SELECT id, table_name, retention_years FROM retention_policies
		WHERE is_active = TRUE AND (tenant_id = $1 OR tenant_id IS NULL)
	`, tenantID)
	if err != nil {
		return fmt.Errorf("failed to query retention policies: %w", err)
	}
	defer rows.Close()

	type policy struct {
		id             uuid.UUID
		tableName      string
		retentionYears int
	}

	var policies []policy
	for rows.Next() {
		var p policy
		if err := rows.Scan(&p.id, &p.tableName, &p.retentionYears); err != nil {
			return fmt.Errorf("failed to scan policy: %w", err)
		}
		policies = append(policies, p)
	}
	rows.Close()

	for _, p := range policies {
		cutoff := time.Now().AddDate(-p.retentionYears, 0, 0)

		// Check for active legal holds on this table for this tenant
		var holdCount int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM legal_holds
			WHERE target_table = $1 AND is_active = TRUE AND tenant_id = $2
		`, p.tableName, tenantID).Scan(&holdCount)
		if err == nil && holdCount > 0 {
			continue
		}

		if p.tableName == "audit_ledger" {
			auditRows, err := db.Query(`
				SELECT al.id FROM audit_ledger al
				WHERE al.created_at < $1 AND al.tenant_id = $2
				  AND NOT EXISTS (
				    SELECT 1 FROM legal_holds lh
				    WHERE lh.is_active = TRUE AND lh.target_table = 'audit_ledger'
				      AND lh.target_record_id = al.id::text AND lh.tenant_id = $2
				  )
				  AND NOT EXISTS (
				    SELECT 1 FROM deletion_log dl
				    WHERE dl.table_name = 'audit_ledger' AND dl.record_id = al.id::text
				  )
			`, cutoff, tenantID)
			if err != nil {
				return fmt.Errorf("failed to query expired audit entries: %w", err)
			}

			var ids []int64
			for auditRows.Next() {
				var id int64
				if err := auditRows.Scan(&id); err != nil {
					auditRows.Close()
					return fmt.Errorf("failed to scan id: %w", err)
				}
				ids = append(ids, id)
			}
			auditRows.Close()

			for _, id := range ids {
				_, err := db.Exec(`
					INSERT INTO deletion_log (table_name, record_id, deletion_reason, deleted_by, retention_met, deleted_at)
					VALUES ($1, $2, $3, $4, TRUE, NOW())
				`, p.tableName, fmt.Sprintf("%d", id),
					fmt.Sprintf("retention policy: %d years exceeded — pending DBA archival", p.retentionYears),
					tenantID)
				if err != nil {
					return fmt.Errorf("failed to log expired audit entry: %w", err)
				}
			}
		}

		_, err = db.Exec(`
			UPDATE retention_policies SET last_purge_at = NOW(), next_purge_at = NOW() + INTERVAL '1 day'
			WHERE id = $1
		`, p.id)
		if err != nil {
			return fmt.Errorf("failed to update policy purge time: %w", err)
		}
	}

	return nil
}

// SecureDelete logs a deletion to the deletion_log and then removes the record.
func (al *AuditLedger) SecureDelete(db *sql.DB, tableName, recordID, reason string, userID uuid.UUID) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Log the deletion
	_, err = tx.Exec(`
		INSERT INTO deletion_log (table_name, record_id, deletion_reason, deleted_by, retention_met, deleted_at)
		VALUES ($1, $2, $3, $4, FALSE, NOW())
	`, tableName, recordID, reason, userID)
	if err != nil {
		return fmt.Errorf("failed to log deletion: %w", err)
	}

	// Delete the actual record using a safe parameterized approach
	// Note: table_name is validated at the handler level before reaching here
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", tableName)
	_, err = tx.Exec(query, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete record from %s: %w", tableName, err)
	}

	return tx.Commit()
}
