package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Reconciliation handler tests ---
// Tests request validation for reconciliation endpoints.

func TestRunMatchingInvalidFeedID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/reconciliation/feeds/bad-id/match", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &ReconciliationHandler{DB: nil}
	err := h.RunMatching(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetFeedInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/reconciliation/feeds/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")

	setTestUser(c, "Administrator")

	h := &ReconciliationHandler{DB: nil}
	err := h.GetFeed(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetExceptionInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/reconciliation/exceptions/bad-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &ReconciliationHandler{DB: nil}
	err := h.GetException(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestResolveExceptionInvalidID(t *testing.T) {
	e := echo.New()
	b, _ := json.Marshal(map[string]string{
		"disposition":      "write_off",
		"resolution_notes": "test",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/reconciliation/exceptions/bad-id/resolve",
		strings.NewReader(string(b)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &ReconciliationHandler{DB: nil}
	err := h.ResolveException(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestResolveExceptionInvalidBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/reconciliation/exceptions/"+uuid.New().String()+"/resolve",
		strings.NewReader(`{broken`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	setTestUser(c, "Administrator")

	h := &ReconciliationHandler{DB: nil}
	err := h.ResolveException(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAssignExceptionInvalidID(t *testing.T) {
	e := echo.New()
	b, _ := json.Marshal(map[string]string{"assigned_to": uuid.New().String()})
	req := httptest.NewRequest(http.MethodPut, "/api/reconciliation/exceptions/bad-id/assign",
		strings.NewReader(string(b)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &ReconciliationHandler{DB: nil}
	err := h.AssignException(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestImportFeedNoFile(t *testing.T) {
	e := echo.New()
	// POST with no multipart file
	req := httptest.NewRequest(http.MethodPost, "/api/reconciliation/import", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &ReconciliationHandler{DB: nil}
	_ = h.ImportFeed(c)
	// Without a file, should get a bad request or internal error
	if rec.Code == http.StatusOK || rec.Code == http.StatusCreated {
		t.Errorf("import without file should not succeed, got %d", rec.Code)
	}
}
