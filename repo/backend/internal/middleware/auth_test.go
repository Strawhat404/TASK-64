package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"compliance-console/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestAuthMiddleware_NoCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := AuthMiddleware(nil)(func(c echo.Context) error {
		return c.String(200, "OK")
	})
	_ = handler(c)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no cookie: status = %d, want 401", rec.Code)
	}
}

func TestAuthMiddleware_EmptyCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: ""})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := AuthMiddleware(nil)(func(c echo.Context) error {
		return c.String(200, "OK")
	})
	_ = handler(c)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("empty cookie: status = %d, want 401", rec.Code)
	}
}

func TestSessionConstants(t *testing.T) {
	if SessionCookieName != "session_token" {
		t.Errorf("SessionCookieName = %q, want %q", SessionCookieName, "session_token")
	}
	if SessionIdleTimeout != 30*time.Minute {
		t.Errorf("SessionIdleTimeout = %v, want 30m", SessionIdleTimeout)
	}
}

func TestRoleGuard_NoUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := RoleGuard("Administrator")(func(c echo.Context) error {
		return c.String(200, "OK")
	})
	_ = handler(c)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no user: status = %d, want 401", rec.Code)
	}
}

func TestRoleGuard_WrongRole(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", &models.User{ID: uuid.New(), RoleName: "Scheduler"})

	handler := RoleGuard("Administrator")(func(c echo.Context) error {
		return c.String(200, "OK")
	})
	_ = handler(c)

	if rec.Code != http.StatusForbidden {
		t.Errorf("wrong role: status = %d, want 403", rec.Code)
	}
}

func TestRoleGuard_CorrectRole(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", &models.User{ID: uuid.New(), RoleName: "Administrator"})

	handler := RoleGuard("Administrator", "Scheduler")(func(c echo.Context) error {
		return c.String(200, "OK")
	})
	_ = handler(c)

	if rec.Code != http.StatusOK {
		t.Errorf("correct role: status = %d, want 200", rec.Code)
	}
}

func TestRoleGuard_MultipleRoles(t *testing.T) {
	tests := []struct {
		name     string
		userRole string
		allowed  []string
		wantCode int
	}{
		{"admin in admin+scheduler", "Administrator", []string{"Administrator", "Scheduler"}, 200},
		{"scheduler in admin+scheduler", "Scheduler", []string{"Administrator", "Scheduler"}, 200},
		{"reviewer in admin+scheduler", "Reviewer", []string{"Administrator", "Scheduler"}, 403},
		{"auditor in auditor+admin", "Auditor", []string{"Auditor", "Administrator"}, 200},
		{"scheduler in auditor only", "Scheduler", []string{"Auditor"}, 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user", &models.User{ID: uuid.New(), RoleName: tt.userRole})

			handler := RoleGuard(tt.allowed...)(func(c echo.Context) error {
				return c.String(200, "OK")
			})
			_ = handler(c)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

func TestGetUserFromContext_Present(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	expected := &models.User{ID: uuid.New(), Username: "testuser"}
	c.Set("user", expected)

	user := GetUserFromContext(c)
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Username != "testuser" {
		t.Errorf("username = %q, want testuser", user.Username)
	}
}

func TestGetUserFromContext_Absent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	user := GetUserFromContext(c)
	if user != nil {
		t.Error("expected nil user when not set in context")
	}
}

func TestGetTenantIDFromContext_Present(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	expected := uuid.New()
	c.Set("tenant_id", expected)

	tid := GetTenantIDFromContext(c)
	if tid != expected {
		t.Errorf("tenant_id = %v, want %v", tid, expected)
	}
}

func TestGetTenantIDFromContext_Absent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tid := GetTenantIDFromContext(c)
	if tid != uuid.Nil {
		t.Errorf("expected uuid.Nil, got %v", tid)
	}
}

func TestCaptchaCheck_NonLoginPath(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	handler := CaptchaCheck(nil)(func(c echo.Context) error {
		called = true
		return c.String(200, "OK")
	})
	_ = handler(c)

	if !called {
		t.Error("CaptchaCheck should pass through for non-login paths")
	}
	if rec.Code != 200 {
		t.Errorf("non-login path: status = %d, want 200", rec.Code)
	}
}
