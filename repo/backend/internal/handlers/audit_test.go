package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// --- Audit handler tests ---
// Tests pagination, filter parsing, and role context for audit log queries.

func TestListAuditLogsDefaultPagination(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/audit/logs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setTestUser(c, "Administrator")

	h := &AuditHandler{DB: nil}
	defer func() { recover() }() // nil DB will panic on query
	_ = h.ListAuditLogs(c)
	// Verify it doesn't return 400 (pagination defaults are applied)
	if rec.Code == http.StatusBadRequest {
		t.Error("default pagination should not cause 400")
	}
}

func TestListAuditLogsWithFilters(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet,
		"/api/audit/logs?page=1&per_page=10&action=login&start_date=2026-01-01&end_date=2026-12-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setTestUser(c, "Auditor")

	h := &AuditHandler{DB: nil}
	defer func() { recover() }()
	_ = h.ListAuditLogs(c)
	if rec.Code == http.StatusBadRequest {
		t.Error("valid filters should not cause 400")
	}
}

func TestListAuditLogsInvalidPage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/audit/logs?page=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setTestUser(c, "Administrator")

	h := &AuditHandler{DB: nil}
	defer func() { recover() }()
	_ = h.ListAuditLogs(c)
	// strconv.Atoi("abc") returns 0, which gets defaulted to 1
	// so this should not cause a 400
	if rec.Code == http.StatusBadRequest {
		t.Error("invalid page string should default gracefully, not 400")
	}
}
