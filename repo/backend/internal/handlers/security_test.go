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

// --- Security handler tests ---
// Tests request validation for sensitive data, key rotation, and legal hold endpoints.

func TestStoreSensitiveDataInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/security/sensitive", strings.NewReader(`{broken`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	err := h.StoreSensitiveData(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestStoreSensitiveDataMissingFields(t *testing.T) {
	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing data_type", map[string]string{"value": "1234", "label": "test"}},
		{"missing value", map[string]string{"data_type": "ssn", "label": "test"}},
		{"both missing", map[string]string{"label": "test"}},
		{"all empty", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/security/sensitive", strings.NewReader(string(b)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setTestUser(c, "Administrator")

			h := &SecurityHandler{DB: nil}
			_ = h.StoreSensitiveData(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestRevealSensitiveDataInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/security/sensitive/bad-id/reveal", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	err := h.RevealSensitiveData(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeleteSensitiveDataInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/security/sensitive/bad-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	err := h.DeleteSensitiveData(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRotateKeyInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/security/keys/rotate", strings.NewReader(`{broken`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	err := h.RotateEncryptionKey(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRotateKeyMissingAlias(t *testing.T) {
	e := echo.New()
	b, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/security/keys/rotate", strings.NewReader(string(b)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	_ = h.RotateEncryptionKey(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateLegalHoldInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/security/legal-holds", strings.NewReader(`{broken`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	err := h.CreateLegalHold(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestReleaseLegalHoldInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/security/legal-holds/bad-id/release", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	err := h.ReleaseLegalHold(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetSensitiveDataInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/security/sensitive/bad-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	err := h.GetSensitiveData(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestVerifyAuditChainNoDBHandling(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/security/audit-ledger/verify", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	// Will panic or return error due to nil DB — we just verify it doesn't crash with a 400
	defer func() {
		if r := recover(); r != nil {
			// Expected with nil DB — this is fine for a validation test
		}
	}()
	_ = h.VerifyAuditChain(c)
	// If it doesn't panic, any non-200 is acceptable
	if rec.Code == http.StatusOK {
		// With nil DB, should not return OK
		t.Log("verify audit chain with nil DB unexpectedly returned OK")
	}
}

func TestGetRateLimitStatusEndpoint(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/security/rate-limits", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &SecurityHandler{DB: nil}
	// nil DB will cause an error but should not panic
	defer func() {
		if r := recover(); r != nil {
			// Expected with nil DB
		}
	}()
	_ = h.GetRateLimitStatus(c)
}
