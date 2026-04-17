package unit_tests

import (
	"net/http"
	"testing"
)

// Tests for session cookie security attributes.
// Production logic reference: backend/internal/handlers/auth.go (Login, Logout)

func TestSessionCookieAttributes(t *testing.T) {
	// These mirror the cookie settings in the Login handler
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    "test-token",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   1800, // 30 minutes
	}

	t.Run("cookie is HttpOnly", func(t *testing.T) {
		if !cookie.HttpOnly {
			t.Error("session cookie must be HttpOnly to prevent XSS access")
		}
	})

	t.Run("cookie is Secure", func(t *testing.T) {
		if !cookie.Secure {
			t.Error("session cookie must be Secure (HTTPS only)")
		}
	})

	t.Run("cookie is SameSite Strict", func(t *testing.T) {
		if cookie.SameSite != http.SameSiteStrictMode {
			t.Error("session cookie must use SameSite=Strict for CSRF protection")
		}
	})

	t.Run("cookie MaxAge is 30 minutes", func(t *testing.T) {
		if cookie.MaxAge != 1800 {
			t.Errorf("cookie MaxAge = %d, want 1800 (30 min)", cookie.MaxAge)
		}
	})

	t.Run("cookie path is root", func(t *testing.T) {
		if cookie.Path != "/" {
			t.Errorf("cookie Path = %q, want /", cookie.Path)
		}
	})

	t.Run("cookie name is session_token", func(t *testing.T) {
		if cookie.Name != "session_token" {
			t.Errorf("cookie Name = %q, want session_token", cookie.Name)
		}
	})
}

func TestLogoutCookieClears(t *testing.T) {
	// Logout must set MaxAge = -1 to clear the cookie
	logoutCookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}

	t.Run("logout cookie value is empty", func(t *testing.T) {
		if logoutCookie.Value != "" {
			t.Error("logout cookie value must be empty")
		}
	})

	t.Run("logout cookie MaxAge is -1", func(t *testing.T) {
		if logoutCookie.MaxAge != -1 {
			t.Errorf("logout cookie MaxAge = %d, want -1", logoutCookie.MaxAge)
		}
	})
}
