package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"compliance-console/internal/middleware"
	"compliance-console/internal/models"
	"compliance-console/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ScheduleHandler contains dependencies for scheduling endpoints.
type ScheduleHandler struct {
	DB        *sql.DB
	Scheduler *services.SchedulerService
	AL        *services.AuditLedger
}

// NewScheduleHandler creates a new ScheduleHandler.
func NewScheduleHandler(db *sql.DB) *ScheduleHandler {
	return &ScheduleHandler{
		DB:        db,
		Scheduler: services.NewSchedulerService(db),
		AL:        services.NewAuditLedger(db),
	}
}

// CreateSchedule creates a new schedule entry with conflict detection.
func (h *ScheduleHandler) CreateSchedule(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	var req models.CreateScheduleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": errs,
		})
	}

	// Check for scheduling conflicts (including 30-min buffer)
	hasConflict, conflictingSchedule, err := h.Scheduler.CheckConflicts(tenantID, req.StaffID, req.ScheduledStart, req.ScheduledEnd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to check conflicts",
		})
	}
	if hasConflict {
		return c.JSON(http.StatusConflict, map[string]interface{}{
			"error":                "Scheduling conflict detected",
			"conflicting_schedule": conflictingSchedule,
		})
	}

	// Validate 30-min buffer between consecutive assignments
	bufferOK, err := h.Scheduler.ValidateBuffer(tenantID, req.StaffID, req.ScheduledStart, req.ScheduledEnd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to validate buffer",
		})
	}
	if !bufferOK {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "A 30-minute buffer is required between consecutive assignments",
		})
	}

	// Verify the service exists
	var serviceExists bool
	err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM services WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE)", req.ServiceID, tenantID).Scan(&serviceExists)
	if err != nil || !serviceExists {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Service not found or inactive",
		})
	}

	// Enforce daily capacity cap
	scheduleDate := req.ScheduledStart.Format("2006-01-02")
	var dailyCap *int
	_ = h.DB.QueryRow("SELECT daily_cap FROM services WHERE id = $1", req.ServiceID).Scan(&dailyCap)
	if dailyCap != nil && *dailyCap > 0 {
		var todayCount int
		_ = h.DB.QueryRow(`
			SELECT COUNT(*) FROM schedules
			WHERE service_id = $1 AND tenant_id = $2
			  AND DATE(scheduled_start) = $3
			  AND status NOT IN ('cancelled')
		`, req.ServiceID, tenantID, scheduleDate).Scan(&todayCount)
		if todayCount >= *dailyCap {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": fmt.Sprintf("Daily capacity cap reached (%d bookings for this service today)", *dailyCap),
			})
		}
	}

	// Check capacity calendar if an entry exists
	var calendarBlocked bool
	var calendarMaxCap, calendarBookedCount int
	err = h.DB.QueryRow(`
		SELECT max_capacity, booked_count, is_blocked
		FROM capacity_calendar
		WHERE service_id = $1 AND tenant_id = $2 AND calendar_date = $3
	`, req.ServiceID, tenantID, scheduleDate).Scan(&calendarMaxCap, &calendarBookedCount, &calendarBlocked)
	if err == nil {
		if calendarBlocked {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "This date is blocked on the capacity calendar",
			})
		}
		if calendarBookedCount >= calendarMaxCap {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": fmt.Sprintf("Capacity calendar full (%d/%d for this date)", calendarBookedCount, calendarMaxCap),
			})
		}
	}

	// Verify the staff member exists
	var staffExists bool
	err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM staff_roster WHERE id = $1 AND tenant_id = $2 AND is_available = TRUE)", req.StaffID, tenantID).Scan(&staffExists)
	if err != nil || !staffExists {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Staff member not found or unavailable",
		})
	}

	// ABAC: Enforce location restriction — if the requesting user has a location,
	// the staff member must be in the same location.
	if currentUser.LocationID != nil {
		var staffUserLocationID *uuid.UUID
		err = h.DB.QueryRow(`
			SELECT u.location_id FROM staff_roster sr
			JOIN users u ON sr.user_id = u.id
			WHERE sr.id = $1 AND sr.tenant_id = $2
		`, req.StaffID, tenantID).Scan(&staffUserLocationID)
		if err == nil && staffUserLocationID != nil && *staffUserLocationID != *currentUser.LocationID {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Staff member is not in your assigned location",
			})
		}
	}

	newID := uuid.New()
	now := time.Now()
	status := "pending"
	if req.RequiresConfirmation {
		status = "pending"
	}

	_, err = h.DB.Exec(`
		INSERT INTO schedules (id, tenant_id, service_id, staff_id, client_name,
		                       scheduled_start, scheduled_end, status,
		                       requires_confirmation, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, newID, tenantID, req.ServiceID, req.StaffID, req.ClientName,
		req.ScheduledStart, req.ScheduledEnd, status,
		req.RequiresConfirmation, now, now)
	if err != nil {
		log.Printf("Failed to create schedule: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create schedule",
		})
	}

	detailsJSON, _ := json.Marshal(req)
	details := string(detailsJSON)
	if err := writeCriticalAuditLog(h.AL, h.DB, &tenantID, &currentUser.ID, "create_schedule", "schedule", &newID, &details, c.RealIP()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Audit system unavailable, action denied",
		})
	}

	// Increment capacity calendar booked count
	_, _ = h.DB.Exec(`
		UPDATE capacity_calendar SET booked_count = booked_count + 1, updated_at = NOW()
		WHERE service_id = $1 AND tenant_id = $2 AND calendar_date = $3
	`, req.ServiceID, tenantID, scheduleDate)

	return h.getScheduleByID(c, newID, tenantID, http.StatusCreated)
}

// ListSchedules returns schedules with optional filters.
func (h *ScheduleHandler) ListSchedules(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	query := `
		SELECT id, tenant_id, service_id, staff_id, client_name,
		       scheduled_start, scheduled_end, status,
		       requires_confirmation, confirmed_at, reassignment_reason,
		       reassignment_reason_code, pending_staff_id, created_at, updated_at
		FROM schedules
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	// Filter by date range
	if startDate := c.QueryParam("start_date"); startDate != "" {
		query += fmt.Sprintf(" AND scheduled_start >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate := c.QueryParam("end_date"); endDate != "" {
		query += fmt.Sprintf(" AND scheduled_start < ($%d::date + interval '1 day')", argIdx)
		args = append(args, endDate)
		argIdx++
	}

	// Filter by staff
	if staffID := c.QueryParam("staff_id"); staffID != "" {
		query += fmt.Sprintf(" AND staff_id = $%d", argIdx)
		args = append(args, staffID)
		argIdx++
	}

	// Filter by status
	if status := c.QueryParam("status"); status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	query += " ORDER BY scheduled_start ASC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch schedules",
		})
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.ServiceID, &s.StaffID, &s.ClientName,
			&s.ScheduledStart, &s.ScheduledEnd, &s.Status,
			&s.RequiresConfirmation, &s.ConfirmedAt, &s.ReassignmentReason,
			&s.ReassignmentReasonCode, &s.PendingStaffID, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan schedule",
			})
		}
		schedules = append(schedules, s)
	}

	if schedules == nil {
		schedules = []models.Schedule{}
	}

	return c.JSON(http.StatusOK, schedules)
}

// UpdateSchedule modifies an existing schedule.
func (h *ScheduleHandler) UpdateSchedule(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid schedule ID",
		})
	}

	var req models.UpdateScheduleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// ABAC: Resolve the schedule's current location and enforce location gate on ALL mutations
	if currentUser.LocationID != nil {
		var scheduleStaffLocationID *uuid.UUID
		_ = h.DB.QueryRow(`
			SELECT u.location_id FROM schedules s
			JOIN staff_roster sr ON s.staff_id = sr.id
			JOIN users u ON sr.user_id = u.id
			WHERE s.id = $1 AND s.tenant_id = $2
		`, scheduleID, tenantID).Scan(&scheduleStaffLocationID)
		if scheduleStaffLocationID != nil && *scheduleStaffLocationID != *currentUser.LocationID {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Cannot modify schedules outside your assigned location",
			})
		}
	}

	// If staff or time is changing, check for conflicts
	if req.StaffID != nil || req.ScheduledStart != nil || req.ScheduledEnd != nil {
		// Fetch existing schedule
		var existing models.Schedule
		err := h.DB.QueryRow(`
			SELECT id, staff_id, scheduled_start, scheduled_end
			FROM schedules WHERE id = $1 AND tenant_id = $2
		`, scheduleID, tenantID).Scan(
			&existing.ID, &existing.StaffID, &existing.ScheduledStart, &existing.ScheduledEnd,
		)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Schedule not found",
			})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to fetch schedule",
			})
		}

		staffID := existing.StaffID
		start := existing.ScheduledStart
		end := existing.ScheduledEnd

		if req.StaffID != nil {
			staffID = *req.StaffID
			// ABAC: Enforce location restriction on staff reassignment
			if currentUser.LocationID != nil {
				var staffUserLocationID *uuid.UUID
				_ = h.DB.QueryRow(`
					SELECT u.location_id FROM staff_roster sr
					JOIN users u ON sr.user_id = u.id
					WHERE sr.id = $1 AND sr.tenant_id = $2
				`, staffID, tenantID).Scan(&staffUserLocationID)
				if staffUserLocationID != nil && *staffUserLocationID != *currentUser.LocationID {
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "Staff member is not in your assigned location",
					})
				}
			}

			// Enforce mandatory reassignment reason code when staff is being changed
			if req.ReassignmentReasonCode == nil || *req.ReassignmentReasonCode == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "reassignment_reason_code is required when changing staff assignment",
				})
			}
			validReasonCodes := map[string]bool{"sick_leave": true, "emergency": true, "no_show": true, "schedule_conflict": true, "requested": true, "skill_mismatch": true, "other": true}
			if !validReasonCodes[*req.ReassignmentReasonCode] {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid reassignment_reason_code",
				})
			}
		}
		if req.ScheduledStart != nil {
			start = *req.ScheduledStart
		}
		if req.ScheduledEnd != nil {
			end = *req.ScheduledEnd
		}

		// Check conflicts excluding the current schedule
		hasConflict, conflicting, err := h.Scheduler.CheckConflictsExcluding(tenantID, staffID, start, end, scheduleID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to check conflicts",
			})
		}
		if hasConflict {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error":                "Scheduling conflict detected",
				"conflicting_schedule": conflicting,
			})
		}
	}

	// Determine if this is a staff reassignment (requires confirmation workflow)
	isReassignment := req.StaffID != nil

	query := "UPDATE schedules SET updated_at = NOW()"
	args := []interface{}{}
	argIdx := 1

	if req.ServiceID != nil {
		// Validate that the new service exists, belongs to the tenant, and is active
		var serviceExists bool
		err := h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM services WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE)", *req.ServiceID, tenantID).Scan(&serviceExists)
		if err != nil || !serviceExists {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Service not found or inactive",
			})
		}
		query += fmt.Sprintf(", service_id = $%d", argIdx)
		args = append(args, *req.ServiceID)
		argIdx++
	}
	if isReassignment {
		// Staff reassignment: store as pending, don't swap staff_id yet
		query += fmt.Sprintf(", pending_staff_id = $%d", argIdx)
		args = append(args, *req.StaffID)
		argIdx++
		query += fmt.Sprintf(", status = $%d", argIdx)
		args = append(args, "pending_reassignment")
		argIdx++
		query += ", requires_confirmation = TRUE"
	}
	if req.ClientName != nil {
		query += fmt.Sprintf(", client_name = $%d", argIdx)
		args = append(args, *req.ClientName)
		argIdx++
	}
	if req.ScheduledStart != nil {
		query += fmt.Sprintf(", scheduled_start = $%d", argIdx)
		args = append(args, *req.ScheduledStart)
		argIdx++
	}
	if req.ScheduledEnd != nil {
		query += fmt.Sprintf(", scheduled_end = $%d", argIdx)
		args = append(args, *req.ScheduledEnd)
		argIdx++
	}
	if req.Status != nil && !isReassignment {
		query += fmt.Sprintf(", status = $%d", argIdx)
		args = append(args, *req.Status)
		argIdx++
	}
	if req.ReassignmentReason != nil {
		query += fmt.Sprintf(", reassignment_reason = $%d", argIdx)
		args = append(args, *req.ReassignmentReason)
		argIdx++
	}
	if req.ReassignmentReasonCode != nil {
		query += fmt.Sprintf(", reassignment_reason_code = $%d", argIdx)
		args = append(args, *req.ReassignmentReasonCode)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argIdx, argIdx+1)
	args = append(args, scheduleID, tenantID)

	result, err := h.DB.Exec(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update schedule",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Schedule not found",
		})
	}

	detailsJSON, _ := json.Marshal(req)
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "update_schedule", "schedule", &scheduleID, &details, c.RealIP())

	return h.getScheduleByID(c, scheduleID, tenantID, http.StatusOK)
}

// CancelSchedule sets a schedule status to cancelled.
func (h *ScheduleHandler) CancelSchedule(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid schedule ID",
		})
	}

	// ABAC: verify cancellation is from the same location
	if currentUser.LocationID != nil {
		var staffUserLocationID *uuid.UUID
		h.DB.QueryRow(`
			SELECT u.location_id FROM schedules s
			JOIN staff_roster sr ON s.staff_id = sr.id
			JOIN users u ON sr.user_id = u.id
			WHERE s.id = $1 AND s.tenant_id = $2
		`, scheduleID, tenantID).Scan(&staffUserLocationID)
		if staffUserLocationID != nil && *staffUserLocationID != *currentUser.LocationID {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Cannot cancel schedules outside your assigned location",
			})
		}
	}

	result, err := h.DB.Exec(`
		UPDATE schedules SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND status NOT IN ('completed', 'cancelled')
	`, scheduleID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to cancel schedule",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Schedule not found or cannot be cancelled",
		})
	}

	if err := writeCriticalAuditLog(h.AL, h.DB, &tenantID, &currentUser.ID, "cancel_schedule", "schedule", &scheduleID, nil, c.RealIP()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Audit system unavailable, action denied",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Schedule cancelled successfully",
	})
}

// ConfirmAssignment confirms a staff assignment for a schedule.
func (h *ScheduleHandler) ConfirmAssignment(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid schedule ID",
		})
	}

	// Fetch schedule to determine confirmation type (new assignment vs reassignment)
	var assignedStaffUserID uuid.UUID
	var scheduleStatus string
	var pendingStaffID *uuid.UUID
	err = h.DB.QueryRow(`
		SELECT sr.user_id, s.status, s.pending_staff_id FROM schedules s
		JOIN staff_roster sr ON s.staff_id = sr.id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, scheduleID, tenantID).Scan(&assignedStaffUserID, &scheduleStatus, &pendingStaffID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Schedule not found"})
	}

	isReassignment := scheduleStatus == "pending_reassignment" && pendingStaffID != nil

	if isReassignment {
		// For reassignment: the new (pending) staff member or Admin must confirm
		var pendingStaffUserID uuid.UUID
		_ = h.DB.QueryRow("SELECT user_id FROM staff_roster WHERE id = $1", *pendingStaffID).Scan(&pendingStaffUserID)
		if currentUser.ID != pendingStaffUserID && currentUser.RoleName != "Administrator" {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Only the reassigned staff member or an Administrator can confirm this reassignment",
			})
		}
	} else {
		// For initial assignment: the assigned staff member or Admin must confirm
		if currentUser.ID != assignedStaffUserID && currentUser.RoleName != "Administrator" {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Only the assigned staff member or an Administrator can confirm this assignment",
			})
		}
	}

	now := time.Now()
	var result sql.Result
	if isReassignment {
		// Confirm reassignment: swap staff_id to pending_staff_id, clear pending
		result, err = h.DB.Exec(`
			UPDATE schedules
			SET staff_id = pending_staff_id, pending_staff_id = NULL,
			    status = 'confirmed', confirmed_at = $1, updated_at = $2,
			    requires_confirmation = FALSE
			WHERE id = $3 AND tenant_id = $4 AND status = 'pending_reassignment' AND pending_staff_id IS NOT NULL
		`, now, now, scheduleID, tenantID)
	} else {
		result, err = h.DB.Exec(`
			UPDATE schedules
			SET status = 'confirmed', confirmed_at = $1, updated_at = $2
			WHERE id = $3 AND tenant_id = $4 AND requires_confirmation = TRUE AND status = 'pending'
		`, now, now, scheduleID, tenantID)
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to confirm assignment",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Schedule not found or does not require confirmation",
		})
	}

	action := "confirm_assignment"
	if isReassignment {
		action = "confirm_reassignment"
	}
	if err := writeCriticalAuditLog(h.AL, h.DB, &tenantID, &currentUser.ID, action, "schedule", &scheduleID, nil, c.RealIP()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Audit system unavailable, action denied",
		})
	}

	return h.getScheduleByID(c, scheduleID, tenantID, http.StatusOK)
}

// FindAvailableStaff finds available staff for a given time range.
func (h *ScheduleHandler) FindAvailableStaff(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	startStr := c.QueryParam("start")
	endStr := c.QueryParam("end")
	specialization := c.QueryParam("specialization")

	if startStr == "" || endStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "start and end query parameters are required",
		})
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid start time format (use RFC3339)",
		})
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid end time format (use RFC3339)",
		})
	}

	// ABAC: pass user's location for location-scoped results
	currentUser := middleware.GetUserFromContext(c)
	var locationID *uuid.UUID
	if currentUser != nil {
		locationID = currentUser.LocationID
	}
	staff, err := h.Scheduler.FindAvailableStaff(tenantID, start, end, specialization, locationID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to find available staff",
		})
	}

	return c.JSON(http.StatusOK, staff)
}

// getScheduleByID fetches and returns a single schedule.
func (h *ScheduleHandler) getScheduleByID(c echo.Context, scheduleID uuid.UUID, tenantID uuid.UUID, statusCode int) error {
	var s models.Schedule
	err := h.DB.QueryRow(`
		SELECT id, tenant_id, service_id, staff_id, client_name,
		       scheduled_start, scheduled_end, status,
		       requires_confirmation, confirmed_at, reassignment_reason,
		       reassignment_reason_code, pending_staff_id, created_at, updated_at
		FROM schedules
		WHERE id = $1 AND tenant_id = $2
	`, scheduleID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.ServiceID, &s.StaffID, &s.ClientName,
		&s.ScheduledStart, &s.ScheduledEnd, &s.Status,
		&s.RequiresConfirmation, &s.ConfirmedAt, &s.ReassignmentReason,
		&s.ReassignmentReasonCode, &s.PendingStaffID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch schedule",
		})
	}
	return c.JSON(statusCode, s)
}

// RequestBackup handles POST /api/schedules/:id/backup.
func (h *ScheduleHandler) RequestBackup(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid schedule ID"})
	}

	var req struct {
		BackupStaffID uuid.UUID `json:"backup_staff_id"`
		ReasonCode    string    `json:"reason_code"`
		Notes         *string   `json:"notes,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	validReasons := map[string]bool{"sick_leave": true, "emergency": true, "no_show": true, "schedule_conflict": true, "requested": true, "other": true}
	if !validReasons[req.ReasonCode] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "reason_code must be one of: sick_leave, emergency, no_show, schedule_conflict, requested, other"})
	}

	// Get the primary staff from the schedule
	var primaryStaffID uuid.UUID
	err = h.DB.QueryRow(`SELECT staff_id FROM schedules WHERE id = $1 AND tenant_id = $2`, scheduleID, tenantID).Scan(&primaryStaffID)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Schedule not found"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch schedule"})
	}

	// Verify backup staff exists and is available
	var backupExists bool
	h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM staff_roster WHERE id = $1 AND tenant_id = $2 AND is_available = TRUE)", req.BackupStaffID, tenantID).Scan(&backupExists)
	if !backupExists {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Backup staff not found or unavailable"})
	}

	newID := uuid.New()
	now := time.Now()
	_, err = h.DB.Exec(`
		INSERT INTO backup_staff_assignments (id, schedule_id, primary_staff_id, backup_staff_id, reason_code, notes, status, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7, $8)
	`, newID, scheduleID, primaryStaffID, req.BackupStaffID, req.ReasonCode, req.Notes, currentUser.ID, now)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create backup assignment"})
	}

	detailsJSON, _ := json.Marshal(map[string]interface{}{"backup_staff_id": req.BackupStaffID, "reason_code": req.ReasonCode})
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "request_backup", "backup_assignment", &newID, &details, c.RealIP())

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": newID, "status": "pending"})
}

// ConfirmBackup handles POST /api/schedules/backup/:id/confirm.
func (h *ScheduleHandler) ConfirmBackup(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid backup ID"})
	}

	now := time.Now()
	var scheduleID, backupStaffID uuid.UUID
	err = h.DB.QueryRow(`
		SELECT ba.schedule_id, ba.backup_staff_id
		FROM backup_staff_assignments ba
		JOIN schedules s ON ba.schedule_id = s.id
		WHERE ba.id = $1 AND s.tenant_id = $2 AND ba.status = 'pending'
	`, backupID, tenantID).Scan(&scheduleID, &backupStaffID)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Backup assignment not found or already processed"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch backup assignment"})
	}

	// Verify the confirming user is the backup staff member or an Administrator
	var backupStaffUserID uuid.UUID
	_ = h.DB.QueryRow("SELECT user_id FROM staff_roster WHERE id = $1", backupStaffID).Scan(&backupStaffUserID)
	if currentUser.ID != backupStaffUserID && currentUser.RoleName != "Administrator" {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only the backup staff member or an Administrator can confirm this backup assignment",
		})
	}

	// Update backup status
	_, err = h.DB.Exec(`UPDATE backup_staff_assignments SET status = 'confirmed', confirmed_at = $1 WHERE id = $2`, now, backupID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to confirm backup"})
	}

	// Reassign the schedule to the backup staff
	_, err = h.DB.Exec(`UPDATE schedules SET staff_id = $1, updated_at = $2 WHERE id = $3`, backupStaffID, now, scheduleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to reassign schedule"})
	}

	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "confirm_backup", "backup_assignment", &backupID, nil, c.RealIP())

	return c.JSON(http.StatusOK, map[string]string{"message": "Backup confirmed and schedule reassigned"})
}
