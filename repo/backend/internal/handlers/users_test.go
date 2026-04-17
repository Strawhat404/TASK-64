package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"compliance-console/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- User handler tests ---
// Tests request validation, parameter parsing, and auth context requirements.

func TestCreateUserInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{broken`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &UserHandler{DB: nil}
	err := h.CreateUser(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateUserPasswordTooShort(t *testing.T) {
	e := echo.New()
	body := `{"username":"testuser","email":"test@test.com","full_name":"Test User","password":"short","role_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &UserHandler{DB: nil}
	err := h.CreateUser(c)
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

func TestCreateUserMissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing username", `{"email":"test@test.com","password":"securepassword1","role_id":1}`},
		{"missing email", `{"username":"testuser","password":"securepassword1","role_id":1}`},
		{"both missing", `{"password":"securepassword1","role_id":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setTestUser(c, "Administrator")

			h := &UserHandler{DB: nil}
			_ = h.CreateUser(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestGetUserInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")

	setTestUser(c, "Administrator")

	h := &UserHandler{DB: nil}
	err := h.GetUser(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "Invalid user ID" {
		t.Errorf("error = %q, want %q", resp["error"], "Invalid user ID")
	}
}

func TestUpdateUserInvalidID(t *testing.T) {
	e := echo.New()
	body := `{"email":"new@test.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/users/bad-id", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &UserHandler{DB: nil}
	err := h.UpdateUser(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateUserInvalidBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/users/"+uuid.New().String(), strings.NewReader(`{bad`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	setTestUser(c, "Administrator")

	h := &UserHandler{DB: nil}
	err := h.UpdateUser(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeactivateUserInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/users/bad-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &UserHandler{DB: nil}
	err := h.DeactivateUser(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeactivateSelfPrevented(t *testing.T) {
	e := echo.New()
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/users/"+userID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(userID.String())

	// Set the current user as the same user being deactivated
	c.Set("user", &models.User{
		ID:       userID,
		TenantID: uuid.New(),
		RoleName: "Administrator",
		IsActive: true,
	})
	c.Set("tenant_id", uuid.New())

	h := &UserHandler{DB: nil}
	err := h.DeactivateUser(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "Cannot deactivate your own account" {
		t.Errorf("error = %q, want self-deactivation prevention", resp["error"])
	}
}

// --- Response contract tests ---

func TestCreateUser_ErrorResponseContract(t *testing.T) {
	e := echo.New()
	body := `{"username":"","email":"","password":"short","role_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setTestUser(c, "Administrator")

	h := &UserHandler{DB: nil}
	_ = h.CreateUser(c)

	// Verify error response is valid JSON with "error" string field
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	errMsg, ok := resp["error"].(string)
	if !ok || errMsg == "" {
		t.Error("error response must have non-empty 'error' string field")
	}
}

func TestGetUser_ErrorResponseContract(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")
	setTestUser(c, "Administrator")

	h := &UserHandler{DB: nil}
	_ = h.GetUser(c)

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("400 response is not valid JSON: %v", err)
	}
	if resp["error"] != "Invalid user ID" {
		t.Errorf("error = %q, want %q", resp["error"], "Invalid user ID")
	}
}

func TestDeactivateSelf_ErrorContract(t *testing.T) {
	e := echo.New()
	userID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/users/"+userID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(userID.String())
	c.Set("user", &models.User{ID: userID, TenantID: uuid.New(), RoleName: "Administrator", IsActive: true})
	c.Set("tenant_id", uuid.New())

	h := &UserHandler{DB: nil}
	_ = h.DeactivateUser(c)

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp["error"] != "Cannot deactivate your own account" {
		t.Errorf("error = %q, want self-deactivation message", resp["error"])
	}
}

// --- Test helpers ---

func setTestUser(c echo.Context, role string) {
	tenantID := uuid.New()
	c.Set("user", &models.User{
		ID:        uuid.New(),
		TenantID:  tenantID,
		RoleName:  role,
		Username:  "testadmin",
		Email:     "admin@test.com",
		FullName:  "Test Admin",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	c.Set("tenant_id", tenantID)
}
