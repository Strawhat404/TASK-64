package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"compliance-console/internal/middleware"
	"compliance-console/internal/models"

	"github.com/labstack/echo/v4"
)

// AuditHandler contains dependencies for audit log endpoints.
type AuditHandler struct {
	DB *sql.DB
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(db *sql.DB) *AuditHandler {
	return &AuditHandler{DB: db}
}

// ListAuditLogs returns audit logs with pagination and filters.
func (h *AuditHandler) ListAuditLogs(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	// Pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	// Build query with filters
	query := `
		SELECT id, tenant_id, user_id, action, resource_type, resource_id,
		       details, ip_address, created_at
		FROM audit_logs
		WHERE tenant_id = $1
	`
	countQuery := "SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1"

	args := []interface{}{tenantID}
	countArgs := []interface{}{tenantID}
	argIdx := 2

	// Filter by user
	if userID := c.QueryParam("user_id"); userID != "" {
		filter := fmt.Sprintf(" AND user_id = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, userID)
		countArgs = append(countArgs, userID)
		argIdx++
	}

	// Filter by action
	if action := c.QueryParam("action"); action != "" {
		filter := fmt.Sprintf(" AND action = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, action)
		countArgs = append(countArgs, action)
		argIdx++
	}

	// Filter by resource_type
	if resourceType := c.QueryParam("resource_type"); resourceType != "" {
		filter := fmt.Sprintf(" AND resource_type = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, resourceType)
		countArgs = append(countArgs, resourceType)
		argIdx++
	}

	// Filter by date range
	if startDate := c.QueryParam("start_date"); startDate != "" {
		filter := fmt.Sprintf(" AND created_at >= $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, startDate)
		countArgs = append(countArgs, startDate)
		argIdx++
	}
	if endDate := c.QueryParam("end_date"); endDate != "" {
		filter := fmt.Sprintf(" AND created_at <= $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, endDate)
		countArgs = append(countArgs, endDate)
		argIdx++
	}

	// Get total count
	var total int
	err := h.DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to count audit logs",
		})
	}

	// Add ordering and pagination
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch audit logs",
		})
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		err := rows.Scan(
			&l.ID, &l.TenantID, &l.UserID, &l.Action, &l.ResourceType,
			&l.ResourceID, &l.Details, &l.IPAddress, &l.CreatedAt,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan audit log",
			})
		}
		logs = append(logs, l)
	}

	if logs == nil {
		logs = []models.AuditLog{}
	}

	totalPages := (total + perPage - 1) / perPage

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":        logs,
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
	})
}
