package handlers

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"compliance-console/internal/middleware"
	"compliance-console/internal/models"
	"compliance-console/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"
)

// ReconciliationHandler contains dependencies for reconciliation endpoints.
type ReconciliationHandler struct {
	DB *sql.DB
}

// NewReconciliationHandler creates a new ReconciliationHandler.
func NewReconciliationHandler(db *sql.DB) *ReconciliationHandler {
	return &ReconciliationHandler{DB: db}
}

// ImportFeed handles POST /api/reconciliation/import.
// Accepts a multipart file upload (CSV) and imports transactions.
func (h *ReconciliationHandler) ImportFeed(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	feedType := c.FormValue("feed_type")
	if feedType != "internal" && feedType != "external" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "feed_type must be 'internal' or 'external'",
		})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "File upload required",
		})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == ".xls" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Legacy .xls format is not supported. Please save as .xlsx or .csv (File > Save As > CSV) for import.",
		})
	}
	if ext != ".csv" && ext != ".xlsx" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Only CSV and XLSX files are supported",
		})
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to open uploaded file",
		})
	}
	defer src.Close()

	var records []models.Transaction

	if ext == ".xlsx" {
		records, err = parseXLSXFile(src)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Failed to parse XLSX: " + err.Error(),
			})
		}
	} else {
		// Parse CSV: expected columns: date, amount, counterparty, memo, reference
		reader := csv.NewReader(src)

		// Read and validate header
		header, err := reader.Read()
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Failed to read CSV header",
			})
		}

		colIdx := mapCSVColumns(header)
		if colIdx["date"] < 0 || colIdx["amount"] < 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "CSV must contain at least 'date' and 'amount' columns",
			})
		}

		lineNum := 1
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("Failed to parse CSV at line %d: %s", lineNum+1, err.Error()),
				})
			}
			lineNum++

			txDate, err := parseFlexibleDate(getCSVField(row, colIdx["date"]))
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("Invalid date at line %d: %s", lineNum, err.Error()),
				})
			}

			amountStr := strings.TrimSpace(getCSVField(row, colIdx["amount"]))
			amount, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("Invalid amount at line %d: %s", lineNum, err.Error()),
				})
			}

			tx := models.Transaction{
				TransactionDate: txDate,
				Amount:          amount,
			}

			if cp := getCSVField(row, colIdx["counterparty"]); cp != "" {
				tx.Counterparty = &cp
			}
			if acct := getCSVField(row, colIdx["counterparty_account"]); acct != "" {
				tx.CounterpartyAccount = &acct
			}
			if memo := getCSVField(row, colIdx["memo"]); memo != "" {
				tx.Memo = &memo
			}
			if ref := getCSVField(row, colIdx["reference"]); ref != "" {
				tx.ReferenceNumber = &ref
			}

			records = append(records, tx)
		}
	}

	if len(records) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "File contains no data rows",
		})
	}

	feed, err := services.ImportCSVFeed(h.DB, tenantID, currentUser.ID, feedType, file.Filename, records)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to import feed: " + err.Error(),
		})
	}

	result := models.ImportResult{
		FeedID:          feed.ID,
		RecordsImported: feed.RecordCount,
	}

	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"filename":    file.Filename,
		"feed_type":   feedType,
		"record_count": feed.RecordCount,
	})
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "import_feed", "transaction_feed", &feed.ID, &details, c.RealIP())

	return c.JSON(http.StatusCreated, result)
}

// ListFeeds handles GET /api/reconciliation/feeds.
func (h *ReconciliationHandler) ListFeeds(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	offset := (page - 1) * perPage

	var total int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM transaction_feeds WHERE tenant_id = $1", tenantID).Scan(&total)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to count feeds",
		})
	}

	rows, err := h.DB.Query(`
		SELECT id, tenant_id, filename, feed_type, record_count, imported_by, imported_at, status
		FROM transaction_feeds
		WHERE tenant_id = $1
		ORDER BY imported_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, perPage, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch feeds",
		})
	}
	defer rows.Close()

	var feeds []models.TransactionFeed
	for rows.Next() {
		var f models.TransactionFeed
		if err := rows.Scan(&f.ID, &f.TenantID, &f.Filename, &f.FeedType,
			&f.RecordCount, &f.ImportedBy, &f.ImportedAt, &f.Status); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan feed",
			})
		}
		feeds = append(feeds, f)
	}
	if feeds == nil {
		feeds = []models.TransactionFeed{}
	}

	totalPages := (total + perPage - 1) / perPage

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":        feeds,
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
	})
}

// GetFeed handles GET /api/reconciliation/feeds/:id.
func (h *ReconciliationHandler) GetFeed(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid feed ID",
		})
	}

	var f models.TransactionFeed
	err = h.DB.QueryRow(`
		SELECT id, tenant_id, filename, feed_type, record_count, imported_by, imported_at, status
		FROM transaction_feeds
		WHERE id = $1 AND tenant_id = $2
	`, feedID, tenantID).Scan(&f.ID, &f.TenantID, &f.Filename, &f.FeedType,
		&f.RecordCount, &f.ImportedBy, &f.ImportedAt, &f.Status)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Feed not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch feed",
		})
	}

	// Get transaction count
	var txCount int
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE feed_id = $1", feedID).Scan(&txCount)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"feed":              f,
		"transaction_count": txCount,
	})
}

// RunMatching handles POST /api/reconciliation/feeds/:id/match.
func (h *ReconciliationHandler) RunMatching(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	feedID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid feed ID",
		})
	}

	// Verify feed exists and belongs to tenant
	var feedStatus string
	err = h.DB.QueryRow(`
		SELECT status FROM transaction_feeds WHERE id = $1 AND tenant_id = $2
	`, feedID, tenantID).Scan(&feedStatus)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Feed not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch feed",
		})
	}

	// Update feed status to processing
	_, err = h.DB.Exec(`UPDATE transaction_feeds SET status = 'processing' WHERE id = $1`, feedID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update feed status",
		})
	}

	result, err := services.MatchTransactions(h.DB, tenantID, feedID)
	if err != nil {
		// Mark feed as failed on error
		_, _ = h.DB.Exec(`UPDATE transaction_feeds SET status = 'failed' WHERE id = $1`, feedID)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Matching failed: " + err.Error(),
		})
	}

	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"matched_count":   result.MatchedCount,
		"exception_count": result.ExceptionCount,
	})
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "run_matching", "transaction_feed", &feedID, &details, c.RealIP())

	return c.JSON(http.StatusOK, result)
}

// GetMatchResults handles GET /api/reconciliation/matches.
func (h *ReconciliationHandler) GetMatchResults(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	query := `
		SELECT id, tenant_id, internal_tx_id, external_tx_id, match_confidence,
			match_method, amount_variance, time_variance_minutes, matched_by,
			matched_at, status
		FROM transaction_matches
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if feedID := c.QueryParam("feed_id"); feedID != "" {
		query += fmt.Sprintf(`
			AND (internal_tx_id IN (SELECT id FROM transactions WHERE feed_id = $%d)
			     OR external_tx_id IN (SELECT id FROM transactions WHERE feed_id = $%d))
		`, argIdx, argIdx)
		args = append(args, feedID)
		argIdx++
	}

	query += " ORDER BY matched_at DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch matches",
		})
	}
	defer rows.Close()

	var matches []models.TransactionMatch
	for rows.Next() {
		var m models.TransactionMatch
		if err := rows.Scan(&m.ID, &m.TenantID, &m.InternalTxID, &m.ExternalTxID,
			&m.MatchConfidence, &m.MatchMethod, &m.AmountVariance,
			&m.TimeVarianceMinutes, &m.MatchedBy, &m.MatchedAt, &m.Status); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan match",
			})
		}
		matches = append(matches, m)
	}
	if matches == nil {
		matches = []models.TransactionMatch{}
	}

	return c.JSON(http.StatusOK, matches)
}

// ListExceptions handles GET /api/reconciliation/exceptions.
func (h *ReconciliationHandler) ListExceptions(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	offset := (page - 1) * perPage

	query := `
		SELECT id, tenant_id, transaction_id, match_id, exception_type, severity,
			amount, variance_amount, description, assigned_to, status,
			disposition, resolution_notes, resolved_by, resolved_at, created_at, updated_at
		FROM reconciliation_exceptions
		WHERE tenant_id = $1
	`
	countQuery := "SELECT COUNT(*) FROM reconciliation_exceptions WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	countArgs := []interface{}{tenantID}
	argIdx := 2

	if exType := c.QueryParam("type"); exType != "" {
		filter := fmt.Sprintf(" AND exception_type = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, exType)
		countArgs = append(countArgs, exType)
		argIdx++
	}
	if status := c.QueryParam("status"); status != "" {
		filter := fmt.Sprintf(" AND status = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, status)
		countArgs = append(countArgs, status)
		argIdx++
	}
	if severity := c.QueryParam("severity"); severity != "" {
		filter := fmt.Sprintf(" AND severity = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, severity)
		countArgs = append(countArgs, severity)
		argIdx++
	}
	if assignedTo := c.QueryParam("assigned_to"); assignedTo != "" {
		filter := fmt.Sprintf(" AND assigned_to = $%d", argIdx)
		query += filter
		countQuery += filter
		args = append(args, assignedTo)
		countArgs = append(countArgs, assignedTo)
		argIdx++
	}

	var total int
	if err := h.DB.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to count exceptions",
		})
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch exceptions",
		})
	}
	defer rows.Close()

	var exceptions []models.ReconciliationException
	for rows.Next() {
		var e models.ReconciliationException
		if err := rows.Scan(&e.ID, &e.TenantID, &e.TransactionID, &e.MatchID,
			&e.ExceptionType, &e.Severity, &e.Amount, &e.VarianceAmount,
			&e.Description, &e.AssignedTo, &e.Status, &e.Disposition,
			&e.ResolutionNotes, &e.ResolvedBy, &e.ResolvedAt,
			&e.CreatedAt, &e.UpdatedAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to scan exception",
			})
		}
		exceptions = append(exceptions, e)
	}
	if exceptions == nil {
		exceptions = []models.ReconciliationException{}
	}

	totalPages := (total + perPage - 1) / perPage

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":        exceptions,
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
	})
}

// GetException handles GET /api/reconciliation/exceptions/:id.
func (h *ReconciliationHandler) GetException(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	excID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid exception ID",
		})
	}

	var e models.ReconciliationException
	err = h.DB.QueryRow(`
		SELECT id, tenant_id, transaction_id, match_id, exception_type, severity,
			amount, variance_amount, description, assigned_to, status,
			disposition, resolution_notes, resolved_by, resolved_at, created_at, updated_at
		FROM reconciliation_exceptions
		WHERE id = $1 AND tenant_id = $2
	`, excID, tenantID).Scan(&e.ID, &e.TenantID, &e.TransactionID, &e.MatchID,
		&e.ExceptionType, &e.Severity, &e.Amount, &e.VarianceAmount,
		&e.Description, &e.AssignedTo, &e.Status, &e.Disposition,
		&e.ResolutionNotes, &e.ResolvedBy, &e.ResolvedAt,
		&e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Exception not found",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch exception",
		})
	}

	return c.JSON(http.StatusOK, e)
}

// AssignException handles PUT /api/reconciliation/exceptions/:id/assign.
func (h *ReconciliationHandler) AssignException(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	excID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid exception ID",
		})
	}

	var req struct {
		AssignedTo uuid.UUID `json:"assigned_to"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.AssignedTo == uuid.Nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "assigned_to is required",
		})
	}

	// Validate assignee exists in same tenant
	var assigneeExists bool
	_ = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE)", req.AssignedTo, tenantID).Scan(&assigneeExists)
	if !assigneeExists {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Assignee not found in tenant or inactive",
		})
	}

	result, err := h.DB.Exec(`
		UPDATE reconciliation_exceptions
		SET assigned_to = $1, status = 'in_progress', updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, req.AssignedTo, excID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to assign exception",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Exception not found",
		})
	}

	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"assigned_to": req.AssignedTo,
	})
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "assign_exception", "reconciliation_exception", &excID, &details, c.RealIP())

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Exception assigned successfully",
	})
}

// ResolveException handles PUT /api/reconciliation/exceptions/:id/resolve.
func (h *ReconciliationHandler) ResolveException(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	excID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid exception ID",
		})
	}

	var req models.ExceptionResolution
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Disposition == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "disposition is required",
		})
	}

	if req.ResolutionNotes == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "resolution_notes is required",
		})
	}

	// Verify exception is assigned before resolving
	var assignedTo *uuid.UUID
	_ = h.DB.QueryRow("SELECT assigned_to FROM reconciliation_exceptions WHERE id = $1 AND tenant_id = $2", excID, tenantID).Scan(&assignedTo)
	if assignedTo == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Exception must be assigned before it can be resolved",
		})
	}

	now := time.Now()
	result, err := h.DB.Exec(`
		UPDATE reconciliation_exceptions
		SET status = 'resolved', disposition = $1, resolution_notes = $2,
			resolved_by = $3, resolved_at = $4, updated_at = $5
		WHERE id = $6 AND tenant_id = $7
	`, req.Disposition, req.ResolutionNotes, currentUser.ID, now, now, excID, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to resolve exception",
		})
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Exception not found",
		})
	}

	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"disposition":      req.Disposition,
		"resolution_notes": req.ResolutionNotes,
	})
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "resolve_exception", "reconciliation_exception", &excID, &details, c.RealIP())

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Exception resolved successfully",
	})
}

// GetSummary handles GET /api/reconciliation/summary.
func (h *ReconciliationHandler) GetSummary(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromContext(c)

	var summary models.ReconciliationSummary

	err := h.DB.QueryRow("SELECT COUNT(*) FROM transaction_feeds WHERE tenant_id = $1", tenantID).Scan(&summary.TotalFeeds)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch summary",
		})
	}

	_ = h.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE tenant_id = $1", tenantID).Scan(&summary.TotalTransactions)
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE tenant_id = $1 AND matched = TRUE", tenantID).Scan(&summary.TotalMatched)
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM reconciliation_exceptions WHERE tenant_id = $1", tenantID).Scan(&summary.TotalExceptions)
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM reconciliation_exceptions WHERE tenant_id = $1 AND status IN ('open', 'in_progress')", tenantID).Scan(&summary.OpenExceptions)

	if summary.TotalTransactions > 0 {
		summary.MatchRatePct = float64(summary.TotalMatched) / float64(summary.TotalTransactions) * 100
	}

	// Populate frontend-expected fields
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM reconciliation_exceptions WHERE tenant_id = $1 AND exception_type = 'unmatched' AND status IN ('open', 'in_progress')", tenantID).Scan(&summary.UnmatchedItems)
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM reconciliation_exceptions WHERE tenant_id = $1 AND exception_type = 'duplicate_suspect' AND status IN ('open', 'in_progress')", tenantID).Scan(&summary.SuspectedDuplicates)
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM reconciliation_exceptions WHERE tenant_id = $1 AND exception_type = 'variance_over_threshold' AND status IN ('open', 'in_progress')", tenantID).Scan(&summary.VarianceAlerts)
	summary.TotalOpen = summary.OpenExceptions
	summary.MatchRate = summary.MatchRatePct

	return c.JSON(http.StatusOK, summary)
}

// ExportExceptions handles GET /api/reconciliation/exceptions/export.
func (h *ReconciliationHandler) ExportExceptions(c echo.Context) error {
	currentUser := middleware.GetUserFromContext(c)
	tenantID := middleware.GetTenantIDFromContext(c)

	format := c.QueryParam("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "xlsx" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Supported export formats: csv, xlsx",
		})
	}

	rows, err := h.DB.Query(`
		SELECT id, transaction_id, match_id, exception_type, severity,
			amount, variance_amount, description, assigned_to, status,
			disposition, resolution_notes, resolved_by, resolved_at, created_at
		FROM reconciliation_exceptions
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch exceptions for export",
		})
	}
	defer rows.Close()

	// Collect all rows into a slice first
	type exceptionRow struct {
		id          uuid.UUID
		txID        *uuid.UUID
		matchID     *uuid.UUID
		exType      string
		severity    string
		amount      *float64
		varianceAmt *float64
		description *string
		assignedTo  *uuid.UUID
		status      string
		disposition *string
		resNotes    *string
		resolvedBy  *uuid.UUID
		resolvedAt  *time.Time
		createdAt   time.Time
	}

	var exRows []exceptionRow
	for rows.Next() {
		var r exceptionRow
		if err := rows.Scan(&r.id, &r.txID, &r.matchID, &r.exType, &r.severity,
			&r.amount, &r.varianceAmt, &r.description, &r.assignedTo, &r.status,
			&r.disposition, &r.resNotes, &r.resolvedBy, &r.resolvedAt, &r.createdAt); err != nil {
			continue
		}
		exRows = append(exRows, r)
	}

	if format == "xlsx" {
		f := excelize.NewFile()
		sheet := "Exceptions"
		f.SetSheetName("Sheet1", sheet)

		// Headers
		headers := []string{"ID", "Transaction ID", "Match ID", "Type", "Severity",
			"Amount", "Variance", "Description", "Assigned To", "Status",
			"Disposition", "Resolution Notes", "Resolved By", "Resolved At", "Created At"}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
		}

		// Data rows
		for rowIdx, r := range exRows {
			rowNum := rowIdx + 2
			vals := []string{
				r.id.String(),
				ptrUUIDStr(r.txID),
				ptrUUIDStr(r.matchID),
				r.exType,
				r.severity,
				ptrFloatStr(r.amount),
				ptrFloatStr(r.varianceAmt),
				ptrStr(r.description),
				ptrUUIDStr(r.assignedTo),
				r.status,
				ptrStr(r.disposition),
				ptrStr(r.resNotes),
				ptrUUIDStr(r.resolvedBy),
				ptrTimeStr(r.resolvedAt),
				r.createdAt.Format(time.RFC3339),
			}
			for colIdx, val := range vals {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowNum)
				f.SetCellValue(sheet, cell, val)
			}
		}

		c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Response().Header().Set("Content-Disposition", "attachment; filename=exceptions.xlsx")

		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"format": format,
		})
		details := string(detailsJSON)
		writeAuditLog(h.DB, &tenantID, &currentUser.ID, "export_exceptions", "reconciliation_exception", nil, &details, c.RealIP())

		return f.Write(c.Response().Writer)
	}

	// CSV export
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=exceptions_export.csv")
	c.Response().WriteHeader(http.StatusOK)

	writer := csv.NewWriter(c.Response().Writer)
	defer writer.Flush()

	// Write header
	_ = writer.Write([]string{
		"id", "transaction_id", "match_id", "exception_type", "severity",
		"amount", "variance_amount", "description", "assigned_to", "status",
		"disposition", "resolution_notes", "resolved_by", "resolved_at", "created_at",
	})

	for _, r := range exRows {
		row := []string{
			r.id.String(),
			ptrUUIDStr(r.txID),
			ptrUUIDStr(r.matchID),
			r.exType,
			r.severity,
			ptrFloatStr(r.amount),
			ptrFloatStr(r.varianceAmt),
			ptrStr(r.description),
			ptrUUIDStr(r.assignedTo),
			r.status,
			ptrStr(r.disposition),
			ptrStr(r.resNotes),
			ptrUUIDStr(r.resolvedBy),
			ptrTimeStr(r.resolvedAt),
			r.createdAt.Format(time.RFC3339),
		}
		_ = writer.Write(row)
	}

	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"format": format,
	})
	details := string(detailsJSON)
	writeAuditLog(h.DB, &tenantID, &currentUser.ID, "export_exceptions", "reconciliation_exception", nil, &details, c.RealIP())

	return nil
}

// mapCSVColumns maps column names from the CSV header to their indices.
func mapCSVColumns(header []string) map[string]int {
	m := map[string]int{
		"date":                 -1,
		"amount":               -1,
		"counterparty":         -1,
		"counterparty_account": -1,
		"memo":                 -1,
		"reference":            -1,
	}
	for i, col := range header {
		normalized := strings.TrimSpace(strings.ToLower(col))
		switch normalized {
		case "date", "transaction_date", "txn_date":
			m["date"] = i
		case "amount", "value", "total":
			m["amount"] = i
		case "counterparty", "party", "vendor", "payee":
			m["counterparty"] = i
		case "memo", "description", "note", "notes":
			m["memo"] = i
		case "reference", "reference_number", "ref", "ref_number":
			m["reference"] = i
		case "counterparty_account", "account", "account_number", "acct", "account_no":
			m["counterparty_account"] = i
		}
	}
	return m
}

// getCSVField safely retrieves a field from a CSV row by index.
func getCSVField(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// parseFlexibleDate attempts to parse a date string using common formats.
func parseFlexibleDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"01-02-2006",
		"Jan 2, 2006",
	}
	s = strings.TrimSpace(s)
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}

// ptrStr converts a *string to a string, returning empty string for nil.
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ptrUUIDStr converts a *uuid.UUID to a string, returning empty string for nil.
func ptrUUIDStr(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

// ptrFloatStr converts a *float64 to a string, returning empty string for nil.
func ptrFloatStr(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', 2, 64)
}

// ptrTimeStr converts a *time.Time to an RFC3339 string, returning empty string for nil.
func ptrTimeStr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// parseXLSXFile parses an xlsx file and returns transaction records.
func parseXLSXFile(reader io.Reader) ([]models.Transaction, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to open xlsx: %w", err)
	}
	defer f.Close()

	// Get the first sheet
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("no sheets found in xlsx file")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("xlsx file must have a header row and at least one data row")
	}

	// Map header columns
	colIdx := mapCSVColumns(rows[0])
	if colIdx["date"] < 0 || colIdx["amount"] < 0 {
		return nil, fmt.Errorf("xlsx must contain at least 'date' and 'amount' columns")
	}

	var records []models.Transaction
	for i, row := range rows[1:] {
		lineNum := i + 2

		dateStr := getCSVField(row, colIdx["date"])
		if dateStr == "" {
			continue
		}
		txDate, err := parseFlexibleDate(dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date at row %d: %w", lineNum, err)
		}

		amountStr := strings.TrimSpace(getCSVField(row, colIdx["amount"]))
		if amountStr == "" {
			continue
		}
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount at row %d: %w", lineNum, err)
		}

		tx := models.Transaction{
			TransactionDate: txDate,
			Amount:          amount,
		}

		if cp := getCSVField(row, colIdx["counterparty"]); cp != "" {
			tx.Counterparty = &cp
		}
		if acct := getCSVField(row, colIdx["counterparty_account"]); acct != "" {
			tx.CounterpartyAccount = &acct
		}
		if memo := getCSVField(row, colIdx["memo"]); memo != "" {
			tx.Memo = &memo
		}
		if ref := getCSVField(row, colIdx["reference"]); ref != "" {
			tx.ReferenceNumber = &ref
		}

		records = append(records, tx)
	}

	return records, nil
}
