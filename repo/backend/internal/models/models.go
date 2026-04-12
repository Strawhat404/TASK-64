package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Tenant represents a multi-tenant organization.
type Tenant struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Domain    string    `json:"domain" db:"domain"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Role represents a user role in the system.
type Role struct {
	ID   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// User represents an authenticated user.
type User struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	TenantID            uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	RoleID              int        `json:"role_id" db:"role_id"`
	RoleName            string     `json:"role_name,omitempty" db:"role_name"`
	Username            string     `json:"username" db:"username"`
	Email               string     `json:"email" db:"email"`
	FullName            string     `json:"full_name" db:"full_name"`
	PasswordHash        string     `json:"-" db:"password_hash"`
	FailedLoginAttempts int        `json:"failed_login_attempts" db:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"locked_until,omitempty" db:"locked_until"`
	CaptchaRequired     bool       `json:"captcha_required" db:"captcha_required"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	LocationID          *uuid.UUID `json:"location_id,omitempty" db:"location_id"`
	LastFailedAt        *time.Time `json:"-" db:"last_failed_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	IsActive            bool       `json:"is_active" db:"is_active"`
}

// Service represents a bookable service in the catalog.
type Service struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	TenantID             uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name                 string    `json:"name" db:"name"`
	Description          string    `json:"description" db:"description"`
	BasePriceUSD         float64   `json:"base_price_usd" db:"base_price_usd"`
	Tier                 string    `json:"tier" db:"tier"`
	AfterHoursSurchPct   int       `json:"after_hours_surcharge_pct" db:"after_hours_surcharge_pct"`
	SameDaySurchargeUSD  float64   `json:"same_day_surcharge_usd" db:"same_day_surcharge_usd"`
	DurationMinutes      int       `json:"duration_minutes" db:"duration_minutes"`
	IsActive             bool             `json:"is_active" db:"is_active"`
	Headcount            int              `json:"headcount" db:"headcount"`
	RequiredTools        pq.StringArray   `json:"required_tools" db:"required_tools"`
	AddOns               *json.RawMessage `json:"add_ons,omitempty" db:"add_ons"`
	DailyCap             *int             `json:"daily_cap,omitempty" db:"daily_cap"`
	CreatedAt            time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at" db:"updated_at"`
}

// StaffMember represents a member of the staff roster.
type StaffMember struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	FullName       string    `json:"full_name" db:"full_name"`
	Specialization string    `json:"specialization" db:"specialization"`
	IsAvailable    bool      `json:"is_available" db:"is_available"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Schedule represents a scheduled appointment.
type Schedule struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	TenantID             uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ServiceID            uuid.UUID  `json:"service_id" db:"service_id"`
	StaffID              uuid.UUID  `json:"staff_id" db:"staff_id"`
	ClientName           string     `json:"client_name" db:"client_name"`
	ScheduledStart       time.Time  `json:"scheduled_start" db:"scheduled_start"`
	ScheduledEnd         time.Time  `json:"scheduled_end" db:"scheduled_end"`
	Status               string     `json:"status" db:"status"`
	RequiresConfirmation bool       `json:"requires_confirmation" db:"requires_confirmation"`
	ConfirmedAt          *time.Time `json:"confirmed_at,omitempty" db:"confirmed_at"`
	ReassignmentReason     *string    `json:"reassignment_reason,omitempty" db:"reassignment_reason"`
	ReassignmentReasonCode *string    `json:"reassignment_reason_code,omitempty" db:"reassignment_reason_code"`
	PendingStaffID         *uuid.UUID `json:"pending_staff_id,omitempty" db:"pending_staff_id"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
}

// AuditLog represents an audit trail entry.
type AuditLog struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TenantID     *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	UserID       *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	Action       string     `json:"action" db:"action"`
	ResourceType string     `json:"resource_type" db:"resource_type"`
	ResourceID   *uuid.UUID `json:"resource_id,omitempty" db:"resource_id"`
	Details      *string    `json:"details,omitempty" db:"details"`
	IPAddress    string     `json:"ip_address" db:"ip_address"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// Session represents an active user session.
type Session struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	User User `json:"user"`
}

// CreateScheduleRequest is the payload for creating a new schedule.
type CreateScheduleRequest struct {
	ServiceID            uuid.UUID `json:"service_id" validate:"required"`
	StaffID              uuid.UUID `json:"staff_id" validate:"required"`
	ClientName           string    `json:"client_name" validate:"required"`
	ScheduledStart       time.Time `json:"scheduled_start" validate:"required"`
	ScheduledEnd         time.Time `json:"scheduled_end" validate:"required"`
	RequiresConfirmation bool      `json:"requires_confirmation"`
}

// Validate checks that the schedule request fields are logically correct.
func (r *CreateScheduleRequest) Validate() []string {
	var errs []string
	if r.ClientName == "" {
		errs = append(errs, "client_name is required")
	}
	if r.ServiceID == uuid.Nil {
		errs = append(errs, "service_id is required")
	}
	if r.StaffID == uuid.Nil {
		errs = append(errs, "staff_id is required")
	}
	if r.ScheduledStart.IsZero() {
		errs = append(errs, "scheduled_start is required")
	}
	if r.ScheduledEnd.IsZero() {
		errs = append(errs, "scheduled_end is required")
	}
	if !r.ScheduledStart.IsZero() && !r.ScheduledEnd.IsZero() {
		if !r.ScheduledEnd.After(r.ScheduledStart) {
			errs = append(errs, "scheduled_end must be after scheduled_start")
		}
	}
	return errs
}

// PricingResponse contains the calculated pricing for a service.
type PricingResponse struct {
	ServiceID          uuid.UUID `json:"service_id"`
	ServiceName        string    `json:"service_name"`
	BasePriceUSD       float64   `json:"base_price_usd"`
	TierMultiplier     float64   `json:"tier_multiplier"`
	TierAdjustedPrice  float64   `json:"tier_adjusted_price"`
	AfterHoursSurcharge float64  `json:"after_hours_surcharge"`
	SameDaySurcharge   float64   `json:"same_day_surcharge"`
	TotalUSD           float64   `json:"total_usd"`
}

// CreateUserRequest is the payload for creating a new user.
type CreateUserRequest struct {
	TenantID uuid.UUID `json:"tenant_id"`
	RoleID   int       `json:"role_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Password string    `json:"password"`
}

// UpdateUserRequest is the payload for updating a user.
type UpdateUserRequest struct {
	RoleID   *int    `json:"role_id,omitempty"`
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// CreateServiceRequest is the payload for creating a new service.
type CreateServiceRequest struct {
	Name                string           `json:"name"`
	Description         string           `json:"description"`
	BasePriceUSD        float64          `json:"base_price_usd"`
	Tier                string           `json:"tier"`
	AfterHoursSurchPct  int              `json:"after_hours_surcharge_pct"`
	SameDaySurchargeUSD float64          `json:"same_day_surcharge_usd"`
	DurationMinutes     int              `json:"duration_minutes"`
	Headcount           int              `json:"headcount"`
	RequiredTools       []string         `json:"required_tools,omitempty"`
	AddOns              *json.RawMessage `json:"add_ons,omitempty"`
	DailyCap            *int             `json:"daily_cap,omitempty"`
}

// UpdateServiceRequest is the payload for updating a service.
type UpdateServiceRequest struct {
	Name                *string          `json:"name,omitempty"`
	Description         *string          `json:"description,omitempty"`
	BasePriceUSD        *float64         `json:"base_price_usd,omitempty"`
	Tier                *string          `json:"tier,omitempty"`
	AfterHoursSurchPct  *int             `json:"after_hours_surcharge_pct,omitempty"`
	SameDaySurchargeUSD *float64         `json:"same_day_surcharge_usd,omitempty"`
	DurationMinutes     *int             `json:"duration_minutes,omitempty"`
	IsActive            *bool            `json:"is_active,omitempty"`
	Headcount           *int             `json:"headcount,omitempty"`
	RequiredTools       []string         `json:"required_tools,omitempty"`
	AddOns              *json.RawMessage `json:"add_ons,omitempty"`
	DailyCap            *int             `json:"daily_cap,omitempty"`
}

// CreateStaffRequest is the payload for creating a staff member.
type CreateStaffRequest struct {
	UserID         uuid.UUID `json:"user_id"`
	FullName       string    `json:"full_name"`
	Specialization string    `json:"specialization"`
}

// UpdateStaffRequest is the payload for updating a staff member.
type UpdateStaffRequest struct {
	FullName       *string `json:"full_name,omitempty"`
	Specialization *string `json:"specialization,omitempty"`
	IsAvailable    *bool   `json:"is_available,omitempty"`
}

// UpdateScheduleRequest is the payload for updating a schedule.
type UpdateScheduleRequest struct {
	ServiceID          *uuid.UUID `json:"service_id,omitempty"`
	StaffID            *uuid.UUID `json:"staff_id,omitempty"`
	ClientName         *string    `json:"client_name,omitempty"`
	ScheduledStart     *time.Time `json:"scheduled_start,omitempty"`
	ScheduledEnd       *time.Time `json:"scheduled_end,omitempty"`
	Status             *string    `json:"status,omitempty"`
	ReassignmentReason     *string    `json:"reassignment_reason,omitempty"`
	ReassignmentReasonCode *string    `json:"reassignment_reason_code,omitempty"`
}

// PricingTier represents a granular pricing tier by duration or headcount.
type PricingTier struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ServiceID uuid.UUID `json:"service_id" db:"service_id"`
	TierType  string    `json:"tier_type" db:"tier_type"`
	MinValue  int       `json:"min_value" db:"min_value"`
	MaxValue  int       `json:"max_value" db:"max_value"`
	PriceUSD  float64   `json:"price_usd" db:"price_usd"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// StaffCredential represents a credential/certification held by a staff member.
type StaffCredential struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	StaffID          uuid.UUID  `json:"staff_id" db:"staff_id"`
	CredentialName   string     `json:"credential_name" db:"credential_name"`
	IssuingAuthority *string    `json:"issuing_authority,omitempty" db:"issuing_authority"`
	CredentialNumber *string    `json:"credential_number,omitempty" db:"credential_number"`
	IssuedDate       *time.Time `json:"issued_date,omitempty" db:"issued_date"`
	ExpiryDate       *time.Time `json:"expiry_date,omitempty" db:"expiry_date"`
	Status           string     `json:"status" db:"status"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// StaffAvailability represents a time window when a staff member is available.
type StaffAvailability struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	StaffID      uuid.UUID  `json:"staff_id" db:"staff_id"`
	DayOfWeek    int        `json:"day_of_week" db:"day_of_week"`
	StartTime    string     `json:"start_time" db:"start_time"`
	EndTime      string     `json:"end_time" db:"end_time"`
	IsRecurring  bool       `json:"is_recurring" db:"is_recurring"`
	SpecificDate *time.Time `json:"specific_date,omitempty" db:"specific_date"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// BackupStaffAssignment represents a backup/substitute staff assignment.
type BackupStaffAssignment struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ScheduleID     uuid.UUID  `json:"schedule_id" db:"schedule_id"`
	PrimaryStaffID uuid.UUID  `json:"primary_staff_id" db:"primary_staff_id"`
	BackupStaffID  uuid.UUID  `json:"backup_staff_id" db:"backup_staff_id"`
	ReasonCode     string     `json:"reason_code" db:"reason_code"`
	Notes          *string    `json:"notes,omitempty" db:"notes"`
	Status         string     `json:"status" db:"status"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty" db:"confirmed_at"`
	CreatedBy      uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// CapacityCalendar represents daily capacity for a service.
type CapacityCalendar struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ServiceID    uuid.UUID `json:"service_id" db:"service_id"`
	CalendarDate time.Time `json:"calendar_date" db:"calendar_date"`
	MaxCapacity  int       `json:"max_capacity" db:"max_capacity"`
	BookedCount  int       `json:"booked_count" db:"booked_count"`
	IsBlocked    bool      `json:"is_blocked" db:"is_blocked"`
	Notes        *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
