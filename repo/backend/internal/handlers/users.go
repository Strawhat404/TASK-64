package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"compliance-console/internal/middleware"
	"compliance-console/internal/models"
	"compliance-console/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// UserHandler contains dependencies for user management endpoints.
type UserHandler struct {
	DB *sql.DB
	AL *services.AuditLedger
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{DB: db, AL: services.NewAuditLedger(db)}
}

// ListUsers returns all users for the current tenant.
func (h *UserHandler) ListUsers(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	rows, err := h.DB.Query(`
		SELECT u.id, u.tenant_id, u.role_id, r.name, u.username, u.email, u.full_name,
		       u.failed_login_attempts, u.locked_until, u.captcha_required,
		       u.last_login_at, u.created_at, u.updated_at, u.is_active
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.tenant_id = $1
		ORDER BY u.created_at DESC
	`, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(
			&u.ID, &u.TenantID, &u.RoleID, &u.RoleName, &u.Username, &u.Email, &u.FullName,
			&u.FailedLoginAttempts, &u.LockedUntil, &u.CaptchaRequired,
			&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.IsActive,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan user",
			})
		}
		users = append(users, u)
	}

	if users == nil {
		users = []models.User{}
	}

	return c.JSON(http.StatusOK, users)
}

// GetUser returns a single user by ID.
func (h *UserHandler) GetUser(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	var u models.User
	err = h.DB.QueryRow(`
		SELECT u.id, u.tenant_id, u.role_id, r.name, u.username, u.email, u.full_name,
		       u.failed_login_attempts, u.locked_until, u.captcha_required,
		       u.last_login_at, u.created_at, u.updated_at, u.is_active
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1 AND u.tenant_id = $2
	`, userID, tenantID).Scan(
		&u.ID, &u.TenantID, &u.RoleID, &u.RoleName, &u.Username, &u.Email, &u.FullName,
		&u.FailedLoginAttempts, &u.LockedUntil, &u.CaptchaRequired,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.IsActive,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch user",
		})
	}

	return c.JSON(http.StatusOK, u)
}

// CreateUser creates a new user with an Argon2-hashed password.
func (h *UserHandler) CreateUser(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	var req models.CreateUserRequest
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

	if req.Username == "" || req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Username and email are required",
		})
	}

	passwordHash, err := HashPasswordArgon2(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to hash password",
		})
	}

	newID := uuid.New()
	now := time.Now()

	_, err = h.DB.Exec(`
		INSERT INTO users (id, tenant_id, role_id, username, email, full_name, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, newID, tenantID, req.RoleID, req.Username, req.Email, req.FullName, passwordHash, now, now)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "Username or email already exists",
		})
	}

	detailsJSON, _ := json.Marshal(map[string]string{
		"username": req.Username,
		"email":    req.Email,
	})
	details := string(detailsJSON)
	if err := writeCriticalAuditLog(h.AL, h.DB, &tenantID, &currentUser.ID, "create_user", "user", &newID, &details, c.RealIP()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Audit system unavailable, action denied",
		})
	}

	var u models.User
	_ = h.DB.QueryRow(`
		SELECT u.id, u.tenant_id, u.role_id, r.name, u.username, u.email, u.full_name,
		       u.failed_login_attempts, u.locked_until, u.captcha_required,
		       u.last_login_at, u.created_at, u.updated_at, u.is_active
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`, newID).Scan(
		&u.ID, &u.TenantID, &u.RoleID, &u.RoleName, &u.Username, &u.Email, &u.FullName,
		&u.FailedLoginAttempts, &u.LockedUntil, &u.CaptchaRequired,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.IsActive,
	)

	return c.JSON(http.StatusCreated, u)
}

// UpdateUser modifies an existing user.
func (h *UserHandler) UpdateUser(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	var req models.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{}
	argIdx := 1

	if req.RoleID != nil {
		query += fmt.Sprintf(", role_id = $%d", argIdx)
		args = append(args, *req.RoleID)
		argIdx++
	}
	if req.Username != nil {
		query += fmt.Sprintf(", username = $%d", argIdx)
		args = append(args, *req.Username)
		argIdx++
	}
	if req.Email != nil {
		query += fmt.Sprintf(", email = $%d", argIdx)
		args = append(args, *req.Email)
		argIdx++
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIdx)
		args = append(args, *req.IsActive)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argIdx, argIdx+1)
	args = append(args, userID, tenantID)

	result, err := h.DB.Exec(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update user",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	detailsJSON, _ := json.Marshal(req)
	details := string(detailsJSON)

	// Role changes are critical permission mutations — route through immutable ledger (fail-closed)
	if req.RoleID != nil {
		if err := writeCriticalAuditLog(h.AL, h.DB, &tenantID, &currentUser.ID, "update_user_role", "user", &userID, &details, c.RealIP()); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Audit system unavailable, action denied",
			})
		}
	} else {
		writeAuditLog(h.DB, &tenantID, &currentUser.ID, "update_user", "user", &userID, &details, c.RealIP())
	}

	return h.GetUser(c)
}

// DeactivateUser sets a user as inactive.
func (h *UserHandler) DeactivateUser(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	if userID == currentUser.ID {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Cannot deactivate your own account",
		})
	}

	result, err := h.DB.Exec(`
		UPDATE users SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`, userID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to deactivate user",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	_, _ = h.DB.Exec("DELETE FROM sessions WHERE user_id = $1", userID)

	if err := writeCriticalAuditLog(h.AL, h.DB, &tenantID, &currentUser.ID, "deactivate_user", "user", &userID, nil, c.RealIP()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Audit system unavailable, action denied",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User deactivated successfully",
	})
}
