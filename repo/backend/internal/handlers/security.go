package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"compliance-console/internal/middleware"
	"compliance-console/internal/models"
	"compliance-console/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// SecurityHandler contains dependencies for security-related endpoints.
type SecurityHandler struct {
	DB  *sql.DB
	Enc *services.EncryptionService
	AL  *services.AuditLedger
}

// NewSecurityHandler creates a new SecurityHandler.
func NewSecurityHandler(db *sql.DB, enc *services.EncryptionService) *SecurityHandler {
	return &SecurityHandler{
		DB:  db,
		Enc: enc,
		AL:  services.NewAuditLedger(db),
	}
}

// writeAuditLog is a helper that appends an entry to the audit ledger.
func (h *SecurityHandler) writeAuditLog(c echo.Context, action, resourceType, resourceID string, details map[string]interface{}) {
	var tenantID *uuid.UUID
	var userID *uuid.UUID

	user := middleware.GetUserFromContext(c)
	if user != nil {
		tenantID = &user.TenantID
		userID = &user.ID
	}

	ip := c.RealIP()
	_ = h.AL.Append(tenantID, userID, action, resourceType, resourceID, details, ip)
}

// StoreSensitiveData handles POST /api/security/sensitive.
func (h *SecurityHandler) StoreSensitiveData(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	var req models.StoreSensitiveDataRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.DataType == "" || req.Value == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "data_type and value are required"})
	}

	// Use or create the default encryption key (scoped to tenant)
	keyAlias := "default-key"
	var keyExists bool
	_ = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM encryption_keys WHERE key_alias = $1 AND tenant_id = $2 AND status = 'active')", keyAlias, user.TenantID).Scan(&keyExists)
	if !keyExists {
		// Bootstrap the default encryption key — encrypt DEK at rest using master key (envelope encryption)
		newKeyID := uuid.New()
		rawKey := h.Enc.GenerateKeyBytes()
		encKey, encNonce, encErr := h.Enc.Encrypt(rawKey)
		if encErr != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to encrypt key material"})
		}
		now := time.Now()
		_, bootstrapErr := h.DB.Exec(`
			INSERT INTO encryption_keys (id, tenant_id, key_alias, encrypted_key, nonce, algorithm, status, rotation_number, activated_at, created_at)
			VALUES ($1, $2, $3, $4, $5, 'AES-256-GCM', 'active', 0, $6, $7)
			ON CONFLICT (tenant_id, key_alias) DO NOTHING
		`, newKeyID, user.TenantID, keyAlias, encKey, encNonce, now, now)
		if bootstrapErr != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to bootstrap encryption key"})
		}
	}

	err := h.Enc.StoreSensitiveField(h.DB, user.TenantID, user.ID, req.DataType, req.Value, req.Label, keyAlias)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to store sensitive data"})
	}

	// Fetch the ID of the record we just inserted so the caller can reference it.
	var newID uuid.UUID
	_ = h.DB.QueryRow(`
		SELECT id FROM sensitive_data
		WHERE tenant_id = $1 AND owner_id = $2 AND data_type = $3
		ORDER BY created_at DESC LIMIT 1
	`, user.TenantID, user.ID, req.DataType).Scan(&newID)

	h.writeAuditLog(c, "store_sensitive_data", "sensitive_data", newID.String(), map[string]interface{}{
		"data_type": req.DataType,
		"label":     req.Label,
	})

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": newID.String(), "message": "Sensitive data stored successfully"})
}

// GetSensitiveData handles GET /api/security/sensitive/:id.
func (h *SecurityHandler) GetSensitiveData(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
	}

	// Fetch metadata for the response
	var sd models.SensitiveData
	err = h.DB.QueryRow(`
		SELECT id, tenant_id, owner_id, data_type, key_alias, label, created_at, updated_at
		FROM sensitive_data WHERE id = $1 AND tenant_id = $2
	`, id, user.TenantID).Scan(
		&sd.ID, &sd.TenantID, &sd.OwnerID, &sd.DataType,
		&sd.KeyAlias, &sd.Label,
		&sd.CreatedAt, &sd.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Sensitive data not found"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve sensitive data"})
	}

	// Decrypt using DEK-aware retrieval (envelope decryption with fallback)
	plaintext, err := h.Enc.RetrieveSensitiveField(h.DB, id, user.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decrypt sensitive data"})
	}

	resp := models.SensitiveDataResponse{
		ID:          sd.ID,
		DataType:    sd.DataType,
		Label:       sd.Label,
		MaskedValue: h.Enc.MaskValue(plaintext),
		CreatedAt:   sd.CreatedAt,
	}

	return c.JSON(http.StatusOK, resp)
}

// RevealSensitiveData handles POST /api/security/sensitive/:id/reveal.
// Admin only — returns decrypted value and logs the access.
func (h *SecurityHandler) RevealSensitiveData(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	if user.RoleName != "Administrator" {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Admin access required"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
	}

	plaintext, err := h.Enc.RetrieveSensitiveField(h.DB, id, user.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve sensitive data"})
	}

	h.writeAuditLog(c, "reveal_sensitive_data", "sensitive_data", id.String(), map[string]interface{}{
		"revealed_by": user.Username,
	})

	return c.JSON(http.StatusOK, map[string]string{"value": plaintext})
}

// ListSensitiveData handles GET /api/security/sensitive.
func (h *SecurityHandler) ListSensitiveData(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	rows, err := h.DB.Query(`
		SELECT id, data_type, encrypted_value, nonce, key_alias, label, created_at
		FROM sensitive_data
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, user.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list sensitive data"})
	}
	defer rows.Close()

	var results []models.SensitiveDataResponse
	for rows.Next() {
		var id uuid.UUID
		var dataType string
		var encryptedValue, nonce []byte
		var keyAlias, label string
		var createdAt time.Time

		if err := rows.Scan(&id, &dataType, &encryptedValue, &nonce, &keyAlias, &label, &createdAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to scan sensitive data"})
		}

		// Use DEK-aware decryption (envelope decryption with master-key fallback)
		masked := "****"
		plaintext, err := h.Enc.DecryptValue(h.DB, user.TenantID, keyAlias, encryptedValue, nonce)
		if err == nil {
			masked = h.Enc.MaskValue(string(plaintext))
		}

		results = append(results, models.SensitiveDataResponse{
			ID:          id,
			DataType:    dataType,
			Label:       label,
			MaskedValue: masked,
			CreatedAt:   createdAt,
		})
	}

	if results == nil {
		results = []models.SensitiveDataResponse{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": results})
}

// DeleteSensitiveData handles DELETE /api/security/sensitive/:id.
func (h *SecurityHandler) DeleteSensitiveData(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
	}

	// Verify record exists and belongs to tenant
	var exists bool
	err = h.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM sensitive_data WHERE id = $1 AND tenant_id = $2)`, id, user.TenantID).Scan(&exists)
	if err != nil || !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Sensitive data not found"})
	}

	err = h.AL.SecureDelete(h.DB, "sensitive_data", id.String(), "user requested deletion", user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete sensitive data"})
	}

	h.writeAuditLog(c, "delete_sensitive_data", "sensitive_data", id.String(), nil)

	return c.JSON(http.StatusOK, map[string]string{"message": "Sensitive data deleted securely"})
}

// RotateEncryptionKey handles POST /api/security/keys/rotate.
func (h *SecurityHandler) RotateEncryptionKey(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	var req models.KeyRotationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.KeyAlias == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key_alias is required"})
	}

	newAlias := fmt.Sprintf("%s-rotated-%d", req.KeyAlias, time.Now().Unix())

	err := h.Enc.RotateKey(h.DB, user.TenantID, req.KeyAlias, newAlias)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Key rotation failed"})
	}

	h.writeAuditLog(c, "rotate_encryption_key", "encryption_keys", req.KeyAlias, map[string]interface{}{
		"old_alias": req.KeyAlias,
		"new_alias": newAlias,
	})

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Key rotated successfully",
		"old_alias": req.KeyAlias,
		"new_alias": newAlias,
	})
}

// GetKeyStatus handles GET /api/security/keys.
func (h *SecurityHandler) GetKeyStatus(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	rows, err := h.DB.Query(`
		SELECT id, key_alias, algorithm, status, rotation_number, activated_at, rotated_at, expires_at, created_at
		FROM encryption_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, user.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch keys"})
	}
	defer rows.Close()

	var keys []models.EncryptionKey
	for rows.Next() {
		var k models.EncryptionKey
		if err := rows.Scan(
			&k.ID, &k.KeyAlias, &k.Algorithm, &k.Status, &k.RotationNumber,
			&k.ActivatedAt, &k.RotatedAt, &k.ExpiresAt, &k.CreatedAt,
		); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to scan key"})
		}
		keys = append(keys, k)
	}

	if keys == nil {
		keys = []models.EncryptionKey{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": keys})
}

// GetRotationDue handles GET /api/security/keys/rotation-due.
func (h *SecurityHandler) GetRotationDue(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	keys, err := h.Enc.GetQuarterlyRotationDue(h.DB, user.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check rotation status"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": keys})
}

// GetAuditLedger handles GET /api/security/audit-ledger.
func (h *SecurityHandler) GetAuditLedger(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	action := c.QueryParam("action")
	resourceType := c.QueryParam("resource_type")

	var tenantID *uuid.UUID
	user := middleware.GetUserFromContext(c)
	if user != nil {
		tenantID = &user.TenantID
	}

	var from, to time.Time
	if fromStr := c.QueryParam("from"); fromStr != "" {
		from, _ = time.Parse(time.RFC3339, fromStr)
	}
	if toStr := c.QueryParam("to"); toStr != "" {
		to, _ = time.Parse(time.RFC3339, toStr)
	}

	entries, total, err := h.AL.QueryEntries(tenantID, action, resourceType, from, to, page, perPage)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch audit ledger"})
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	totalPages := (total + perPage - 1) / perPage

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":        entries,
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
	})
}

// VerifyAuditChain handles POST /api/security/audit-ledger/verify.
func (h *SecurityHandler) VerifyAuditChain(c echo.Context) error {
	valid, brokenAt, err := h.AL.Verify(h.DB)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Verification failed"})
	}

	h.writeAuditLog(c, "verify_audit_chain", "audit_ledger", "", map[string]interface{}{
		"valid":     valid,
		"broken_at": brokenAt,
	})

	// Count total entries checked
	var entriesChecked int
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM audit_ledger").Scan(&entriesChecked)

	result := map[string]interface{}{
		"valid":           valid,
		"entries_checked": entriesChecked,
	}
	if !valid {
		result["broken_at"] = brokenAt
	}

	return c.JSON(http.StatusOK, result)
}

// GetRetentionPolicies handles GET /api/security/retention.
func (h *SecurityHandler) GetRetentionPolicies(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	rows, err := h.DB.Query(`
		SELECT id, tenant_id, table_name, retention_years, last_purge_at, next_purge_at, is_active, created_at
		FROM retention_policies
		WHERE tenant_id = $1 OR tenant_id IS NULL
		ORDER BY table_name ASC
	`, user.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch retention policies"})
	}
	defer rows.Close()

	var policies []models.RetentionPolicy
	for rows.Next() {
		var p models.RetentionPolicy
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.TableName, &p.RetentionYears, &p.LastPurgeAt,
			&p.NextPurgeAt, &p.IsActive, &p.CreatedAt,
		); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to scan retention policy"})
		}
		policies = append(policies, p)
	}

	if policies == nil {
		policies = []models.RetentionPolicy{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": policies})
}

// RunRetentionCleanup handles POST /api/security/retention/cleanup.
func (h *SecurityHandler) RunRetentionCleanup(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	err := h.AL.EnforceRetentionForTenant(h.DB, user.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Retention cleanup failed"})
	}

	h.writeAuditLog(c, "run_retention_cleanup", "retention_policies", "", nil)

	return c.JSON(http.StatusOK, map[string]string{"message": "Retention cleanup completed"})
}

// CreateLegalHold handles POST /api/security/legal-holds.
func (h *SecurityHandler) CreateLegalHold(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	var req struct {
		HoldReason     string  `json:"hold_reason"`
		TargetTable    string  `json:"target_table"`
		TargetRecordID *string `json:"target_record_id,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	if req.HoldReason == "" || req.TargetTable == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "hold_reason and target_table are required"})
	}

	id := uuid.New()
	_, err := h.DB.Exec(`
		INSERT INTO legal_holds (id, tenant_id, hold_reason, held_by, target_table, target_record_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, user.TenantID, req.HoldReason, user.ID, req.TargetTable, req.TargetRecordID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create legal hold"})
	}

	h.writeAuditLog(c, "create_legal_hold", "legal_hold", id.String(), map[string]interface{}{
		"target_table": req.TargetTable, "reason": req.HoldReason,
	})

	return c.JSON(http.StatusCreated, map[string]string{"id": id.String(), "status": "active"})
}

// ListLegalHolds handles GET /api/security/legal-holds.
func (h *SecurityHandler) ListLegalHolds(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	rows, err := h.DB.Query(`
		SELECT id, tenant_id, hold_reason, held_by, hold_start, hold_end, is_active, target_table, target_record_id, created_at
		FROM legal_holds WHERE tenant_id = $1 ORDER BY created_at DESC
	`, user.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list legal holds"})
	}
	defer rows.Close()

	var holds []models.LegalHold
	for rows.Next() {
		var lh models.LegalHold
		if err := rows.Scan(&lh.ID, &lh.TenantID, &lh.HoldReason, &lh.HeldBy, &lh.HoldStart, &lh.HoldEnd, &lh.IsActive, &lh.TargetTable, &lh.TargetRecordID, &lh.CreatedAt); err != nil {
			continue
		}
		holds = append(holds, lh)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": holds})
}

// ReleaseLegalHold handles PUT /api/security/legal-holds/:id/release.
func (h *SecurityHandler) ReleaseLegalHold(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
	}

	result, err := h.DB.Exec(`
		UPDATE legal_holds SET is_active = FALSE, hold_end = NOW()
		WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE
	`, id, user.TenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to release legal hold"})
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Legal hold not found or already released"})
	}

	h.writeAuditLog(c, "release_legal_hold", "legal_hold", id.String(), nil)

	return c.JSON(http.StatusOK, map[string]string{"status": "released"})
}

// GetRateLimitStatus handles GET /api/security/rate-limits.
func (h *SecurityHandler) GetRateLimitStatus(c echo.Context) error {
	user := middleware.GetUserFromContext(c)

	var statuses []models.RateLimitStatus

	// Get IP-based status
	ipStatus, err := middleware.GetRateLimitStatus(h.DB, c.RealIP(), "ip", 100, 1)
	if err == nil && ipStatus != nil {
		statuses = append(statuses, *ipStatus)
	}

	// Get user-based status if authenticated
	if user != nil {
		userStatus, err := middleware.GetRateLimitStatus(h.DB, user.ID.String(), "user", 100, 1)
		if err == nil && userStatus != nil {
			statuses = append(statuses, *userStatus)
		}
	}

	if statuses == nil {
		statuses = []models.RateLimitStatus{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": statuses})
}
