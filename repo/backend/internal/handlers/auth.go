package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"compliance-console/internal/middleware"
	"compliance-console/internal/models"
	"compliance-console/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/argon2"
)

// AuthHandler contains dependencies for authentication endpoints.
type AuthHandler struct {
	DB *sql.DB
	AL *services.AuditLedger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db, AL: services.NewAuditLedger(db)}
}

// Login authenticates a user and creates a session.
func (h *AuthHandler) Login(c echo.Context) error {
	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(req.Password) < 12 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Password must be at least 12 characters",
		})
	}

	// Check account lock
	if err := middleware.CheckAccountLock(h.DB, req.Username); err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": err.Error(),
		})
	}

	// Fetch user
	var user models.User
	var roleName string
	err := h.DB.QueryRow(`
		SELECT u.id, u.tenant_id, u.role_id, u.username, u.email, u.full_name,
		       u.password_hash, u.failed_login_attempts, u.locked_until,
		       u.captcha_required, u.last_login_at, u.created_at, u.updated_at,
		       u.is_active, r.name
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.username = $1
	`, req.Username).Scan(
		&user.ID, &user.TenantID, &user.RoleID, &user.Username, &user.Email, &user.FullName,
		&user.PasswordHash, &user.FailedLoginAttempts, &user.LockedUntil,
		&user.CaptchaRequired, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
		&user.IsActive, &roleName,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid credentials",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	if !user.IsActive {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Account is deactivated",
		})
	}

	// Verify password using Argon2
	if !verifyArgon2Hash(req.Password, user.PasswordHash) {
		middleware.TrackFailedLogin(h.DB, req.Username)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid credentials",
		})
	}

	// Reset failed login attempts on success
	middleware.ResetLoginAttempts(h.DB, req.Username)

	// Generate session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate session token",
		})
	}
	token := hex.EncodeToString(tokenBytes)

	sessionID := uuid.New()
	expiresAt := time.Now().Add(middleware.SessionIdleTimeout)

	_, err = h.DB.Exec(`
		INSERT INTO sessions (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, sessionID, user.ID, token, expiresAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create session",
		})
	}

	// Set session cookie
	c.SetCookie(&http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(middleware.SessionIdleTimeout.Seconds()),
	})

	user.RoleName = roleName

	// Audit log — critical action, route through immutable ledger (fail-closed)
	if err := writeCriticalAuditLog(h.AL, h.DB, &user.TenantID, &user.ID, "login", "user", &user.ID, nil, c.RealIP()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Audit system unavailable, login denied",
		})
	}

	return c.JSON(http.StatusOK, models.LoginResponse{
		User: user,
	})
}

// Logout destroys the current session.
func (h *AuthHandler) Logout(c echo.Context) error {
	user := middleware.GetUserFromContext(c)

	session, ok := c.Get("session").(*models.Session)
	if ok && session != nil {
		_, _ = h.DB.Exec("DELETE FROM sessions WHERE id = $1", session.ID)
	}

	// Clear cookie
	c.SetCookie(&http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	if user != nil {
		if err := writeCriticalAuditLog(h.AL, h.DB, &user.TenantID, &user.ID, "logout", "user", &user.ID, nil, c.RealIP()); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Audit system unavailable, logout denied",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// GetSession returns the current authenticated user's info.
func (h *AuthHandler) GetSession(c echo.Context) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Not authenticated",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// verifyArgon2Hash checks a password against an Argon2id hash string.
func verifyArgon2Hash(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	// Expected format: ["", "argon2id", "v=19", "m=65536,t=3,p=4", "<salt>", "<hash>"]
	if len(parts) != 6 {
		return false
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false
	}

	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		// Try using raw bytes as salt (for base64 or plain text salts)
		salt = []byte(parts[4])
	}

	expectedHash, err := hex.DecodeString(parts[5])
	if err != nil {
		// Try using raw bytes
		expectedHash = []byte(parts[5])
	}

	keyLen := uint32(len(expectedHash))
	if keyLen == 0 {
		keyLen = 32
	}

	derivedKey := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLen)

	return constantTimeCompare(derivedKey, expectedHash)
}

// constantTimeCompare performs a constant-time byte comparison.
func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// HashPasswordArgon2 creates an Argon2id hash of the password.
func HashPasswordArgon2(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	hash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, 64*1024, 3, 4,
		hex.EncodeToString(salt),
		hex.EncodeToString(key),
	)

	return hash, nil
}

// writeAuditLog is a helper to insert an audit log entry.
func writeAuditLog(db *sql.DB, tenantID *uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, details *string, ipAddress string) {
	id := uuid.New()
	_, _ = db.Exec(`
		INSERT INTO audit_logs (id, tenant_id, user_id, action, resource_type, resource_id, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::inet)
	`, id, tenantID, userID, action, resourceType, resourceID, details, ipAddress)
}

// criticalActions defines actions that must be routed through the immutable audit ledger.
var criticalActions = map[string]bool{
	"login": true, "logout": true, "login_failed": true,
	"create_user": true, "deactivate_user": true, "update_user_role": true,
	"review_decision": true, "promote_content": true, "rollback_content": true,
	"create_schedule": true, "cancel_schedule": true, "confirm_assignment": true,
	"reassign_schedule": true,
	"store_sensitive_data": true, "reveal_sensitive_data": true, "rotate_encryption_key": true,
	"resolve_exception": true, "assign_exception": true,
}

// writeCriticalAuditLog routes critical actions to the immutable audit_ledger and also writes to audit_logs.
// Returns an error if the immutable ledger append fails — callers MUST treat this as a hard failure
// to preserve compliance traceability guarantees (fail-closed).
func writeCriticalAuditLog(al *services.AuditLedger, db *sql.DB, tenantID *uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, details *string, ipAddress string) error {
	// Write to immutable audit_ledger first — if this fails, the action must not proceed
	resID := ""
	if resourceID != nil {
		resID = resourceID.String()
	}
	var detailsMap map[string]interface{}
	if details != nil && *details != "" {
		_ = json.Unmarshal([]byte(*details), &detailsMap)
	}
	if err := al.Append(tenantID, userID, action, resourceType, resID, detailsMap, ipAddress); err != nil {
		log.Printf("CRITICAL: immutable audit ledger append failed for action=%s: %v", action, err)
		return fmt.Errorf("immutable audit log write failed: %w", err)
	}

	// Also write to mutable audit_logs for general querying
	writeAuditLog(db, tenantID, userID, action, resourceType, resourceID, details, ipAddress)
	return nil
}
