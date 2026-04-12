package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"compliance-console/internal/middleware"
	"compliance-console/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// ServiceHandler contains dependencies for service catalog endpoints.
type ServiceHandler struct {
	DB *sql.DB
}

// NewServiceHandler creates a new ServiceHandler.
func NewServiceHandler(db *sql.DB) *ServiceHandler {
	return &ServiceHandler{DB: db}
}

// ListServices returns all active services for the current tenant.
func (h *ServiceHandler) ListServices(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	rows, err := h.DB.Query(`
		SELECT id, tenant_id, name, description, base_price_usd, tier,
		       after_hours_surcharge_pct, same_day_surcharge_usd,
		       duration_minutes, is_active, headcount, required_tools, add_ons, daily_cap,
		       created_at, updated_at
		FROM services
		WHERE tenant_id = $1
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch services",
		})
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var s models.Service
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Description, &s.BasePriceUSD, &s.Tier,
			&s.AfterHoursSurchPct, &s.SameDaySurchargeUSD,
			&s.DurationMinutes, &s.IsActive, &s.Headcount, &s.RequiredTools, &s.AddOns, &s.DailyCap,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan service",
			})
		}
		services = append(services, s)
	}

	if services == nil {
		services = []models.Service{}
	}

	return c.JSON(http.StatusOK, services)
}

// GetService returns a single service by ID.
func (h *ServiceHandler) GetService(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)
	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid service ID",
		})
	}

	var s models.Service
	err = h.DB.QueryRow(`
		SELECT id, tenant_id, name, description, base_price_usd, tier,
		       after_hours_surcharge_pct, same_day_surcharge_usd,
		       duration_minutes, is_active, headcount, required_tools, add_ons, daily_cap,
		       created_at, updated_at
		FROM services
		WHERE id = $1 AND tenant_id = $2
	`, serviceID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.BasePriceUSD, &s.Tier,
		&s.AfterHoursSurchPct, &s.SameDaySurchargeUSD,
		&s.DurationMinutes, &s.IsActive, &s.Headcount, &s.RequiredTools, &s.AddOns, &s.DailyCap,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Service not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch service",
		})
	}

	return c.JSON(http.StatusOK, s)
}

// CreateService creates a new service in the catalog.
func (h *ServiceHandler) CreateService(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	var req models.CreateServiceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate tier
	if req.Tier != "standard" && req.Tier != "premium" && req.Tier != "enterprise" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tier must be one of: standard, premium, enterprise",
		})
	}

	// Validate duration: must be 15-min increment, between 15 and 240
	if req.DurationMinutes < 15 || req.DurationMinutes > 240 || req.DurationMinutes%15 != 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Duration must be between 15 and 240 minutes in 15-minute increments",
		})
	}

	// Validate headcount range
	if req.Headcount < 1 || req.Headcount > 10 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Headcount must be between 1 and 10",
		})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Service name is required",
		})
	}

	newID := uuid.New()
	now := time.Now()

	// Default headcount to 1 if not provided
	headcount := req.Headcount
	if headcount <= 0 {
		headcount = 1
	}

	_, err := h.DB.Exec(`
		INSERT INTO services (id, tenant_id, name, description, base_price_usd, tier,
		                      after_hours_surcharge_pct, same_day_surcharge_usd,
		                      duration_minutes, is_active, headcount, required_tools, add_ons, daily_cap,
		                      created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, $10, $11, $12, $13, $14, $15)
	`, newID, tenantID, req.Name, req.Description, req.BasePriceUSD, req.Tier,
		req.AfterHoursSurchPct, req.SameDaySurchargeUSD, req.DurationMinutes,
		headcount, pq.Array(req.RequiredTools), req.AddOns, req.DailyCap, now, now)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create service",
		})
	}

	detailsJSON, _ := json.Marshal(req)
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "create_service", "service", &newID, &details, c.RealIP())

	// Fetch and return created service
	c.SetParamNames("id")
	c.SetParamValues(newID.String())
	return h.getServiceByID(c, newID, tenantID, http.StatusCreated)
}

// UpdateService modifies an existing service.
func (h *ServiceHandler) UpdateService(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid service ID",
		})
	}

	var req models.UpdateServiceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Tier != nil {
		t := *req.Tier
		if t != "standard" && t != "premium" && t != "enterprise" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Tier must be one of: standard, premium, enterprise",
			})
		}
	}

	if req.DurationMinutes != nil {
		d := *req.DurationMinutes
		if d < 15 || d > 240 || d%15 != 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Duration must be between 15 and 240 minutes in 15-minute increments",
			})
		}
	}

	// Validate headcount range
	if req.Headcount != nil {
		h := *req.Headcount
		if h < 1 || h > 10 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Headcount must be between 1 and 10",
			})
		}
	}

	query := "UPDATE services SET updated_at = NOW()"
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, *req.Description)
		argIdx++
	}
	if req.BasePriceUSD != nil {
		query += fmt.Sprintf(", base_price_usd = $%d", argIdx)
		args = append(args, *req.BasePriceUSD)
		argIdx++
	}
	if req.Tier != nil {
		query += fmt.Sprintf(", tier = $%d", argIdx)
		args = append(args, *req.Tier)
		argIdx++
	}
	if req.AfterHoursSurchPct != nil {
		query += fmt.Sprintf(", after_hours_surcharge_pct = $%d", argIdx)
		args = append(args, *req.AfterHoursSurchPct)
		argIdx++
	}
	if req.SameDaySurchargeUSD != nil {
		query += fmt.Sprintf(", same_day_surcharge_usd = $%d", argIdx)
		args = append(args, *req.SameDaySurchargeUSD)
		argIdx++
	}
	if req.DurationMinutes != nil {
		query += fmt.Sprintf(", duration_minutes = $%d", argIdx)
		args = append(args, *req.DurationMinutes)
		argIdx++
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIdx)
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.Headcount != nil {
		query += fmt.Sprintf(", headcount = $%d", argIdx)
		args = append(args, *req.Headcount)
		argIdx++
	}
	if req.RequiredTools != nil {
		query += fmt.Sprintf(", required_tools = $%d", argIdx)
		args = append(args, pq.Array(req.RequiredTools))
		argIdx++
	}
	if req.AddOns != nil {
		query += fmt.Sprintf(", add_ons = $%d", argIdx)
		args = append(args, req.AddOns)
		argIdx++
	}
	if req.DailyCap != nil {
		query += fmt.Sprintf(", daily_cap = $%d", argIdx)
		args = append(args, *req.DailyCap)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND tenant_id = $%d", argIdx, argIdx+1)
	args = append(args, serviceID, tenantID)

	result, err := h.DB.Exec(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update service",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Service not found",
		})
	}

	detailsJSON, _ := json.Marshal(req)
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "update_service", "service", &serviceID, &details, c.RealIP())

	return h.getServiceByID(c, serviceID, tenantID, http.StatusOK)
}

// DeleteService deactivates a service.
func (h *ServiceHandler) DeleteService(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid service ID",
		})
	}

	result, err := h.DB.Exec(`
		UPDATE services SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`, serviceID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to deactivate service",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Service not found",
		})
	}

	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "delete_service", "service", &serviceID, nil, c.RealIP())

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Service deactivated successfully",
	})
}

// GetPricing calculates the total price for a service with surcharges applied.
// Uses pricing_tiers table for granular duration/headcount-based pricing if configured.
func (h *ServiceHandler) GetPricing(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)
	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid service ID",
		})
	}

	afterHours := c.QueryParam("after_hours") == "true"
	sameDay := c.QueryParam("same_day") == "true"
	requestedDuration := 0
	if d := c.QueryParam("duration"); d != "" {
		fmt.Sscanf(d, "%d", &requestedDuration)
	}
	requestedHeadcount := 0
	if h := c.QueryParam("headcount"); h != "" {
		fmt.Sscanf(h, "%d", &requestedHeadcount)
	}

	var s models.Service
	err = h.DB.QueryRow(`
		SELECT id, tenant_id, name, description, base_price_usd, tier,
		       after_hours_surcharge_pct, same_day_surcharge_usd,
		       duration_minutes, is_active, headcount, required_tools, add_ons, daily_cap,
		       created_at, updated_at
		FROM services
		WHERE id = $1 AND tenant_id = $2
	`, serviceID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.BasePriceUSD, &s.Tier,
		&s.AfterHoursSurchPct, &s.SameDaySurchargeUSD,
		&s.DurationMinutes, &s.IsActive, &s.Headcount, &s.RequiredTools, &s.AddOns, &s.DailyCap,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Service not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch service",
		})
	}

	// Start with base price
	basePrice := s.BasePriceUSD

	// Check for granular duration-based pricing tier
	duration := s.DurationMinutes
	if requestedDuration > 0 {
		duration = requestedDuration
	}
	var durationTierPrice *float64
	_ = h.DB.QueryRow(`
		SELECT price_usd FROM pricing_tiers
		WHERE service_id = $1 AND tier_type = 'duration' AND $2 >= min_value AND $2 <= max_value
		ORDER BY min_value ASC LIMIT 1
	`, serviceID, duration).Scan(&durationTierPrice)
	if durationTierPrice != nil {
		basePrice = *durationTierPrice
	}

	// Check for granular headcount-based pricing tier
	headcount := s.Headcount
	if requestedHeadcount > 0 {
		headcount = requestedHeadcount
	}
	var headcountTierPrice *float64
	_ = h.DB.QueryRow(`
		SELECT price_usd FROM pricing_tiers
		WHERE service_id = $1 AND tier_type = 'headcount' AND $2 >= min_value AND $2 <= max_value
		ORDER BY min_value ASC LIMIT 1
	`, serviceID, headcount).Scan(&headcountTierPrice)
	if headcountTierPrice != nil {
		basePrice = *headcountTierPrice
	}

	// Tier multiplier (standard/premium/enterprise)
	tierMultiplier := 1.0
	switch s.Tier {
	case "premium":
		tierMultiplier = 1.5
	case "enterprise":
		tierMultiplier = 2.0
	}

	tierAdjustedPrice := basePrice * tierMultiplier

	// After-hours surcharge
	afterHoursSurcharge := 0.0
	if afterHours {
		afterHoursSurcharge = tierAdjustedPrice * (float64(s.AfterHoursSurchPct) / 100.0)
	}

	// Same-day surcharge
	sameDaySurcharge := 0.0
	if sameDay {
		sameDaySurcharge = s.SameDaySurchargeUSD
	}

	total := tierAdjustedPrice + afterHoursSurcharge + sameDaySurcharge
	total = math.Round(total*100) / 100

	return c.JSON(http.StatusOK, models.PricingResponse{
		ServiceID:           s.ID,
		ServiceName:         s.Name,
		BasePriceUSD:        basePrice,
		TierMultiplier:      tierMultiplier,
		TierAdjustedPrice:   tierAdjustedPrice,
		AfterHoursSurcharge: afterHoursSurcharge,
		SameDaySurcharge:    sameDaySurcharge,
		TotalUSD:            total,
	})
}

// getServiceByID is a helper to fetch and return a service.
func (h *ServiceHandler) getServiceByID(c echo.Context, serviceID uuid.UUID, tenantID uuid.UUID, statusCode int) error {
	var s models.Service
	err := h.DB.QueryRow(`
		SELECT id, tenant_id, name, description, base_price_usd, tier,
		       after_hours_surcharge_pct, same_day_surcharge_usd,
		       duration_minutes, is_active, headcount, required_tools, add_ons, daily_cap,
		       created_at, updated_at
		FROM services
		WHERE id = $1 AND tenant_id = $2
	`, serviceID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.BasePriceUSD, &s.Tier,
		&s.AfterHoursSurchPct, &s.SameDaySurchargeUSD,
		&s.DurationMinutes, &s.IsActive, &s.Headcount, &s.RequiredTools, &s.AddOns, &s.DailyCap,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch service",
		})
	}
	return c.JSON(statusCode, s)
}
