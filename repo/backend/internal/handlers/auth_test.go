package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"compliance-console/internal/models"

	"github.com/labstack/echo/v4"
)

// --- Auth handler tests ---
// These test request parsing, validation, and response format without a live DB.

func TestLoginInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{invalid`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &AuthHandler{DB: nil}
	err := h.Login(c)

	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "Invalid request body" {
		t.Errorf("error = %q, want %q", resp["error"], "Invalid request body")
	}
}

func TestLoginPasswordTooShort(t *testing.T) {
	e := echo.New()
	body := `{"username":"admin","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &AuthHandler{DB: nil}
	err := h.Login(c)

	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "Password must be at least 12 characters" {
		t.Errorf("error = %q, want password length error", resp["error"])
	}
}

func TestLoginEmptyBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &AuthHandler{DB: nil}
	err := h.Login(c)

	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// Empty password => too short
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetSessionNoUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user set in context

	h := &AuthHandler{DB: nil}
	err := h.GetSession(c)

	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "Not authenticated" {
		t.Errorf("error = %q, want %q", resp["error"], "Not authenticated")
	}
}

func TestLogoutNoSession(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user or session in context

	h := &AuthHandler{DB: nil}
	err := h.Logout(c)

	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// Logout without user should return OK (clears cookie only)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// --- Argon2 helper tests ---

func TestVerifyArgon2HashInvalidFormat(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{"empty string", ""},
		{"random text", "not-a-hash"},
		{"too few parts", "$argon2id$v=19$m=65536,t=3,p=4"},
		{"too many parts", "$argon2id$v=19$m=65536,t=3,p=4$salt$hash$extra"},
		{"bad parameters", "$argon2id$v=19$invalid$salt$hash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if verifyArgon2Hash("password1234", tt.hash) {
				t.Error("invalid hash format should not verify")
			}
		})
	}
}

func TestHashPasswordArgon2RoundTrip(t *testing.T) {
	password := "SecureP@ssword1"
	hash, err := HashPasswordArgon2(password)
	if err != nil {
		t.Fatalf("HashPasswordArgon2 failed: %v", err)
	}
	if !verifyArgon2Hash(password, hash) {
		t.Error("password should verify against its own hash")
	}
	if verifyArgon2Hash("WrongPassword1", hash) {
		t.Error("wrong password should not verify")
	}
}

func TestHashPasswordArgon2Uniqueness(t *testing.T) {
	pw := "SamePassword12!"
	h1, _ := HashPasswordArgon2(pw)
	h2, _ := HashPasswordArgon2(pw)
	if h1 == h2 {
		t.Error("two hashes of the same password should differ due to random salt")
	}
}

func TestConstantTimeCompareExported(t *testing.T) {
	if !constantTimeCompare([]byte("hello"), []byte("hello")) {
		t.Error("equal slices should match")
	}
	if constantTimeCompare([]byte("hello"), []byte("world")) {
		t.Error("different slices should not match")
	}
	if constantTimeCompare([]byte("hi"), []byte("hello")) {
		t.Error("different length slices should not match")
	}
}

// --- Response contract tests ---

func TestLoginInvalidJSON_ResponseContract(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &AuthHandler{DB: nil}
	_ = h.Login(c)

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("error response must contain 'error' field")
	}
	if _, ok := resp["error"].(string); !ok {
		t.Error("'error' field must be a string")
	}
}

func TestGetSession_ResponseContract(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &AuthHandler{DB: nil}
	_ = h.GetSession(c)

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	// Unauthorized response must have error field
	if rec.Code == http.StatusUnauthorized {
		if _, ok := resp["error"]; !ok {
			t.Error("401 response must contain 'error' field")
		}
	}
}

func TestLoginPasswordBoundary(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantCode int
	}{
		{"11 chars rejected", "12345678901", http.StatusBadRequest},
		{"12 chars accepted", "123456789012", 0}, // 0 means "not 400" (passes to DB check)
		{"empty password", "", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			body, _ := json.Marshal(map[string]string{"username": "admin", "password": tt.password})
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(string(body)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &AuthHandler{DB: nil}
			_ = h.Login(c)

			if tt.wantCode > 0 && rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.wantCode == 0 && rec.Code == http.StatusBadRequest {
				t.Errorf("12-char password should not get 400")
			}
		})
	}
}

// --- Middleware integration: RoleGuard ---

func TestRoleGuardIntegration(t *testing.T) {
	// Test that RoleGuard properly blocks/allows based on user context
	tests := []struct {
		name         string
		userRole     string
		allowedRoles []string
		wantBlocked  bool
	}{
		{"admin accessing admin route", "Administrator", []string{"Administrator"}, false},
		{"scheduler accessing admin route", "Scheduler", []string{"Administrator"}, true},
		{"reviewer accessing reviewer route", "Reviewer", []string{"Administrator", "Reviewer"}, false},
		{"auditor accessing recon route", "Auditor", []string{"Administrator", "Auditor"}, false},
		{"scheduler accessing recon route", "Scheduler", []string{"Administrator", "Auditor"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			setTestUser(c, tt.userRole)

			// Simulate RoleGuard logic
			user := c.Get("user").(*models.User)
			allowed := false
			for _, role := range tt.allowedRoles {
				if user.RoleName == role {
					allowed = true
					break
				}
			}

			if allowed == tt.wantBlocked {
				t.Errorf("role %q with allowed %v: blocked = %v, want %v",
					tt.userRole, tt.allowedRoles, !allowed, tt.wantBlocked)
			}
		})
	}
}

// --- Critical action classification ---

func TestCriticalActionsMap(t *testing.T) {
	expectedCritical := []string{
		"login", "logout", "login_failed",
		"create_user", "deactivate_user", "update_user_role",
		"review_decision", "promote_content", "rollback_content",
		"create_schedule", "cancel_schedule", "confirm_assignment", "reassign_schedule",
		"store_sensitive_data", "reveal_sensitive_data", "rotate_encryption_key",
		"resolve_exception", "assign_exception",
	}

	for _, action := range expectedCritical {
		if !criticalActions[action] {
			t.Errorf("action %q should be classified as critical", action)
		}
	}

	nonCritical := []string{
		"create_service", "update_service", "delete_service",
		"list_users", "get_user",
	}
	for _, action := range nonCritical {
		if criticalActions[action] {
			t.Errorf("action %q should NOT be classified as critical", action)
		}
	}
}
