package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"compliance-console/internal/middleware"
	"compliance-console/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// StaffHandler contains dependencies for staff roster endpoints.
type StaffHandler struct {
	DB *sql.DB
}

// NewStaffHandler creates a new StaffHandler.
func NewStaffHandler(db *sql.DB) *StaffHandler {
	return &StaffHandler{DB: db}
}

// ListStaff returns all staff members for the current tenant.
func (h *StaffHandler) ListStaff(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	rows, err := h.DB.Query(`
		SELECT id, tenant_id, user_id, full_name, specialization,
		       is_available, created_at, updated_at
		FROM staff_roster
		WHERE tenant_id = $1
		ORDER BY full_name ASC
	`, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch staff",
		})
	}
	defer rows.Close()

	var staff []models.StaffMember
	for rows.Next() {
		var s models.StaffMember
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.UserID, &s.FullName, &s.Specialization,
			&s.IsAvailable, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan staff member",
			})
		}
		staff = append(staff, s)
	}

	if staff == nil {
		staff = []models.StaffMember{}
	}

	return c.JSON(http.StatusOK, staff)
}

// GetStaff returns a single staff member by ID.
func (h *StaffHandler) GetStaff(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid staff ID",
		})
	}

	var s models.StaffMember
	err = h.DB.QueryRow(`
		SELECT id, tenant_id, user_id, full_name, specialization,
		       is_available, created_at, updated_at
		FROM staff_roster
		WHERE id = $1 AND tenant_id = $2
	`, staffID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.UserID, &s.FullName, &s.Specialization,
		&s.IsAvailable, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Staff member not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch staff member",
		})
	}

	return c.JSON(http.StatusOK, s)
}

// CreateStaff creates a new staff member.
func (h *StaffHandler) CreateStaff(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	var req models.CreateStaffRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.FullName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Full name is required",
		})
	}

	if req.UserID == uuid.Nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID is required",
		})
	}

	// Verify the user exists in the same tenant
	var userExists bool
	err := h.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND tenant_id = $2)",
		req.UserID, tenantID,
	).Scan(&userExists)
	if err != nil || !userExists {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User not found in tenant",
		})
	}

	newID := uuid.New()
	now := time.Now()

	_, err = h.DB.Exec(`
		INSERT INTO staff_roster (id, tenant_id, user_id, full_name, specialization, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, newID, tenantID, req.UserID, req.FullName, req.Specialization, now, now)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create staff member",
		})
	}

	detailsJSON, _ := json.Marshal(req)
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "create_staff", "staff", &newID, &details, c.RealIP())

	return h.getStaffByID(c, newID, tenantID, http.StatusCreated)
}

// UpdateStaff modifies an existing staff member.
func (h *StaffHandler) UpdateStaff(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid staff ID",
		})
	}

	var req models.UpdateStaffRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	query := "UPDATE staff_roster SET updated_at = NOW()"
	args := []interface{}{}
	argIdx := 1

	if req.FullName != nil {
		query += fmt.Sprintf(", full_name = $%d", argIdx)
		args = append(args, *req.FullName)
		argIdx++
	}
	if req.Specialization != nil {
		query += fmt.Sprintf(", specialization = $%d", argIdx)
		args = append(args, *req.Specialization)
		argIdx++
	}
	if req.IsAvailable != nil {
		query += fmt.Sprintf(", is_available = $%d", argIdx)
		args = append(args, *req.IsAvailable)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argIdx, argIdx+1)
	args = append(args, staffID, tenantID)

	result, err := h.DB.Exec(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update staff member",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Staff member not found",
		})
	}

	detailsJSON, _ := json.Marshal(req)
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "update_staff", "staff", &staffID, &details, c.RealIP())

	return h.getStaffByID(c, staffID, tenantID, http.StatusOK)
}

// DeleteStaff removes a staff member (sets unavailable).
func (h *StaffHandler) DeleteStaff(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid staff ID",
		})
	}

	result, err := h.DB.Exec(`
		UPDATE staff_roster SET is_available = FALSE, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`, staffID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to remove staff member",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Staff member not found",
		})
	}

	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "delete_staff", "staff", &staffID, nil, c.RealIP())

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Staff member removed successfully",
	})
}

// getStaffByID fetches and returns a single staff member.
func (h *StaffHandler) getStaffByID(c echo.Context, staffID uuid.UUID, tenantID uuid.UUID, statusCode int) error {
	var s models.StaffMember
	err := h.DB.QueryRow(`
		SELECT id, tenant_id, user_id, full_name, specialization,
		       is_available, created_at, updated_at
		FROM staff_roster
		WHERE id = $1 AND tenant_id = $2
	`, staffID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.UserID, &s.FullName, &s.Specialization,
		&s.IsAvailable, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch staff member",
		})
	}
	return c.JSON(statusCode, s)
}

// ListCredentials handles GET /api/staff/:id/credentials.
func (h *StaffHandler) ListCredentials(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid staff ID"})
	}

	// Verify staff belongs to tenant
	var exists bool
	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM staff_roster WHERE id = $1 AND tenant_id = $2)", staffID, tenantID).Scan(&exists)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Staff member not found"})
	}

	rows, err := h.DB.Query(`
		SELECT id, staff_id, credential_name, issuing_authority, credential_number,
		       issued_date, expiry_date, status, created_at, updated_at
		FROM staff_credentials WHERE staff_id = $1 ORDER BY created_at DESC
	`, staffID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch credentials"})
	}
	defer rows.Close()

	var creds []models.StaffCredential
	for rows.Next() {
		var cr models.StaffCredential
		if err := rows.Scan(&cr.ID, &cr.StaffID, &cr.CredentialName, &cr.IssuingAuthority,
			&cr.CredentialNumber, &cr.IssuedDate, &cr.ExpiryDate, &cr.Status,
			&cr.CreatedAt, &cr.UpdatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to scan credential"})
		}
		creds = append(creds, cr)
	}
	if creds == nil {
		creds = []models.StaffCredential{}
	}
	return c.JSON(http.StatusOK, creds)
}

// AddCredential handles POST /api/staff/:id/credentials.
func (h *StaffHandler) AddCredential(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid staff ID"})
	}

	// Verify staff belongs to tenant
	var staffExists bool
	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM staff_roster WHERE id = $1 AND tenant_id = $2)", staffID, tenantID).Scan(&staffExists)
	if !staffExists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Staff member not found in tenant"})
	}

	var req struct {
		CredentialName   string  `json:"credential_name"`
		IssuingAuthority *string `json:"issuing_authority,omitempty"`
		CredentialNumber *string `json:"credential_number,omitempty"`
		IssuedDate       *string `json:"issued_date,omitempty"`
		ExpiryDate       *string `json:"expiry_date,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.CredentialName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "credential_name is required"})
	}

	newID := uuid.New()
	now := time.Now()
	_, err = h.DB.Exec(`
		INSERT INTO staff_credentials (id, staff_id, credential_name, issuing_authority, credential_number, issued_date, expiry_date, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::date, $7::date, 'active', $8, $9)
	`, newID, staffID, req.CredentialName, req.IssuingAuthority, req.CredentialNumber, req.IssuedDate, req.ExpiryDate, now, now)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add credential"})
	}

	detailsJSON, _ := json.Marshal(req)
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "add_credential", "staff_credential", &newID, &details, c.RealIP())

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": newID})
}

// ListAvailability handles GET /api/staff/:id/availability.
func (h *StaffHandler) ListAvailability(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid staff ID"})
	}

	var exists bool
	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM staff_roster WHERE id = $1 AND tenant_id = $2)", staffID, tenantID).Scan(&exists)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Staff member not found"})
	}

	rows, err := h.DB.Query(`
		SELECT id, staff_id, day_of_week, start_time::text, end_time::text, is_recurring, specific_date, created_at
		FROM staff_availability WHERE staff_id = $1 ORDER BY day_of_week, start_time
	`, staffID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch availability"})
	}
	defer rows.Close()

	var avails []models.StaffAvailability
	for rows.Next() {
		var a models.StaffAvailability
		if err := rows.Scan(&a.ID, &a.StaffID, &a.DayOfWeek, &a.StartTime, &a.EndTime,
			&a.IsRecurring, &a.SpecificDate, &a.CreatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to scan availability"})
		}
		avails = append(avails, a)
	}
	if avails == nil {
		avails = []models.StaffAvailability{}
	}
	return c.JSON(http.StatusOK, avails)
}

// AddAvailability handles POST /api/staff/:id/availability.
func (h *StaffHandler) AddAvailability(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)
	staffID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid staff ID"})
	}

	// Verify staff belongs to tenant
	var staffExists bool
	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM staff_roster WHERE id = $1 AND tenant_id = $2)", staffID, tenantID).Scan(&staffExists)
	if !staffExists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Staff member not found in tenant"})
	}

	var req struct {
		DayOfWeek    int     `json:"day_of_week"`
		StartTime    string  `json:"start_time"`
		EndTime      string  `json:"end_time"`
		IsRecurring  bool    `json:"is_recurring"`
		SpecificDate *string `json:"specific_date,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.DayOfWeek < 0 || req.DayOfWeek > 6 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "day_of_week must be 0-6 (Sun-Sat)"})
	}
	if req.StartTime == "" || req.EndTime == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "start_time and end_time are required"})
	}

	newID := uuid.New()
	now := time.Now()
	_, err = h.DB.Exec(`
		INSERT INTO staff_availability (id, staff_id, day_of_week, start_time, end_time, is_recurring, specific_date, created_at)
		VALUES ($1, $2, $3, $4::time, $5::time, $6, $7::date, $8)
	`, newID, staffID, req.DayOfWeek, req.StartTime, req.EndTime, req.IsRecurring, req.SpecificDate, now)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add availability"})
	}

	detailsJSON, _ := json.Marshal(req)
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "add_availability", "staff_availability", &newID, &details, c.RealIP())

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": newID})
}
