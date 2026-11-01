package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSetRefreshCookieAppliesSecurityAttributes(t *testing.T) {
	t.Parallel()

	service := newCookieTestService()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/api/auth/login", nil)
	rec := httptest.NewRecorder()

	service.SetRefreshCookie(rec, req, "refresh-value")

	cookie := findCookie(t, rec.Result().Cookies(), "refresh_token")
	if cookie.Value != "refresh-value" {
		t.Fatalf("expected cookie value %q, got %q", "refresh-value", cookie.Value)
	}
	if cookie.Path != RefreshCookiePath {
		t.Fatalf("expected cookie path %q, got %q", RefreshCookiePath, cookie.Path)
	}
	if cookie.MaxAge != int(service.refreshAbsoluteTTL.Seconds()) {
		t.Fatalf("expected max age %d, got %d", int(service.refreshAbsoluteTTL.Seconds()), cookie.MaxAge)
	}
	assertRefreshCookieSecurity(t, cookie)
}

func TestClearRefreshCookieAppliesExpiredSecurityAttributes(t *testing.T) {
	t.Parallel()

	service := newCookieTestService()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/api/auth/logout", nil)
	rec := httptest.NewRecorder()

	service.ClearRefreshCookie(rec, req)

	cookie := findCookie(t, rec.Result().Cookies(), "refresh_token")
	if cookie.Value != "" {
		t.Fatalf("expected empty cookie value, got %q", cookie.Value)
	}
	if cookie.MaxAge != -1 {
		t.Fatalf("expected max age %d, got %d", -1, cookie.MaxAge)
	}
	assertRefreshCookieSecurity(t, cookie)
}

func newCookieTestService() *Service {
	return &Service{
		refreshCookieName:  "refresh_token",
		refreshAbsoluteTTL: 30 * time.Second,
	}
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found", name)
	return nil
}

func assertRefreshCookieSecurity(t *testing.T, cookie *http.Cookie) {
	t.Helper()

	if !cookie.HttpOnly {
		t.Fatal("expected refresh cookie to be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
	if !cookie.Secure {
		t.Fatal("expected refresh cookie to be Secure")
	}
}
