package services

import (
	"database/sql"
	"fmt"
	"time"

	"compliance-console/internal/models"

	"github.com/google/uuid"
)

// BufferDuration is the required gap between consecutive staff assignments.
const BufferDuration = 30 * time.Minute

const bufferDuration = BufferDuration

// SchedulerService provides scheduling conflict detection and auto-assignment.
type SchedulerService struct {
	DB *sql.DB
}

// NewSchedulerService creates a new SchedulerService.
func NewSchedulerService(db *sql.DB) *SchedulerService {
	return &SchedulerService{DB: db}
}

// CheckConflicts checks if any existing schedule for the given staff overlaps
// with the requested time range (including the 30-minute buffer).
// Returns true and the conflicting schedule if a conflict is found.
func (s *SchedulerService) CheckConflicts(tenantID, staffID uuid.UUID, start, end time.Time) (bool, *models.Schedule, error) {
	bufferedStart := start.Add(-bufferDuration)
	bufferedEnd := end.Add(bufferDuration)

	var schedule models.Schedule
	err := s.DB.QueryRow(`
		SELECT id, tenant_id, service_id, staff_id, client_name,
		       scheduled_start, scheduled_end, status,
		       requires_confirmation, confirmed_at, created_at, updated_at
		FROM schedules
		WHERE staff_id = $1
		  AND tenant_id = $4
		  AND status NOT IN ('cancelled')
		  AND scheduled_start < $3
		  AND scheduled_end > $2
		ORDER BY scheduled_start ASC
		LIMIT 1
	`, staffID, bufferedStart, bufferedEnd, tenantID).Scan(
		&schedule.ID, &schedule.TenantID, &schedule.ServiceID, &schedule.StaffID,
		&schedule.ClientName, &schedule.ScheduledStart, &schedule.ScheduledEnd,
		&schedule.Status, &schedule.RequiresConfirmation, &schedule.ConfirmedAt,
		&schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to check conflicts: %w", err)
	}

	return true, &schedule, nil
}

// CheckConflictsExcluding checks for conflicts while excluding a specific schedule
// (useful for updates where the schedule being updated should not conflict with itself).
func (s *SchedulerService) CheckConflictsExcluding(tenantID, staffID uuid.UUID, start, end time.Time, excludeID uuid.UUID) (bool, *models.Schedule, error) {
	bufferedStart := start.Add(-bufferDuration)
	bufferedEnd := end.Add(bufferDuration)

	var schedule models.Schedule
	err := s.DB.QueryRow(`
		SELECT id, tenant_id, service_id, staff_id, client_name,
		       scheduled_start, scheduled_end, status,
		       requires_confirmation, confirmed_at, created_at, updated_at
		FROM schedules
		WHERE staff_id = $1
		  AND tenant_id = $5
		  AND id != $4
		  AND status NOT IN ('cancelled')
		  AND scheduled_start < $3
		  AND scheduled_end > $2
		ORDER BY scheduled_start ASC
		LIMIT 1
	`, staffID, bufferedStart, bufferedEnd, excludeID, tenantID).Scan(
		&schedule.ID, &schedule.TenantID, &schedule.ServiceID, &schedule.StaffID,
		&schedule.ClientName, &schedule.ScheduledStart, &schedule.ScheduledEnd,
		&schedule.Status, &schedule.RequiresConfirmation, &schedule.ConfirmedAt,
		&schedule.CreatedAt, &schedule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to check conflicts: %w", err)
	}

	return true, &schedule, nil
}

// ValidateBuffer ensures there is at least a 30-minute gap between the requested
// time range and any adjacent schedules for the given staff member.
func (s *SchedulerService) ValidateBuffer(tenantID, staffID uuid.UUID, start, end time.Time) (bool, error) {
	// Check if any schedule ends within 30 minutes before the requested start
	var countBefore int
	err := s.DB.QueryRow(`
		SELECT COUNT(*)
		FROM schedules
		WHERE staff_id = $1
		  AND tenant_id = $4
		  AND status NOT IN ('cancelled')
		  AND scheduled_end > $2
		  AND scheduled_end <= $3
	`, staffID, start.Add(-bufferDuration), start, tenantID).Scan(&countBefore)
	if err != nil {
		return false, fmt.Errorf("failed to validate buffer (before): %w", err)
	}

	if countBefore > 0 {
		return false, nil
	}

	// Check if any schedule starts within 30 minutes after the requested end
	var countAfter int
	err = s.DB.QueryRow(`
		SELECT COUNT(*)
		FROM schedules
		WHERE staff_id = $1
		  AND tenant_id = $4
		  AND status NOT IN ('cancelled')
		  AND scheduled_start >= $2
		  AND scheduled_start < $3
	`, staffID, end, end.Add(bufferDuration), tenantID).Scan(&countAfter)
	if err != nil {
		return false, fmt.Errorf("failed to validate buffer (after): %w", err)
	}

	if countAfter > 0 {
		return false, nil
	}

	return true, nil
}

// FindAvailableStaff returns staff members who are available (not booked) during
// the given time range, optionally filtered by specialization.
func (s *SchedulerService) FindAvailableStaff(tenantID uuid.UUID, start, end time.Time, specialization string, locationID *uuid.UUID) ([]models.StaffMember, error) {
	bufferedStart := start.Add(-bufferDuration)
	bufferedEnd := end.Add(bufferDuration)

	query := `
		SELECT sr.id, sr.tenant_id, sr.user_id, sr.full_name, sr.specialization,
		       sr.is_available, sr.created_at, sr.updated_at
		FROM staff_roster sr
		WHERE sr.tenant_id = $1
		  AND sr.is_available = TRUE
		  AND sr.id NOT IN (
		      SELECT DISTINCT sch.staff_id
		      FROM schedules sch
		      WHERE sch.tenant_id = $1
		        AND sch.status NOT IN ('cancelled')
		        AND sch.scheduled_start < $3
		        AND sch.scheduled_end > $2
		  )
	`
	args := []interface{}{tenantID, bufferedStart, bufferedEnd}
	argIdx := 4

	if specialization != "" {
		query += fmt.Sprintf(" AND sr.specialization = $%d", argIdx)
		args = append(args, specialization)
		argIdx++
	}

	// ABAC: filter by location if the requesting user has a location restriction
	if locationID != nil {
		query += fmt.Sprintf(` AND sr.user_id IN (
			SELECT u.id FROM users u WHERE u.location_id = $%d
		)`, argIdx)
		args = append(args, *locationID)
		argIdx++
	}

	query += " ORDER BY sr.full_name ASC"

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find available staff: %w", err)
	}
	defer rows.Close()

	var staff []models.StaffMember
	for rows.Next() {
		var m models.StaffMember
		err := rows.Scan(
			&m.ID, &m.TenantID, &m.UserID, &m.FullName, &m.Specialization,
			&m.IsAvailable, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan staff member: %w", err)
		}
		staff = append(staff, m)
	}

	if staff == nil {
		staff = []models.StaffMember{}
	}

	return staff, nil
}
