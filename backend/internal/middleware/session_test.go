package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSession_SetsCookieWhenMissing(t *testing.T) {
	handler := Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sid := GetSessionID(r.Context())
		if sid == "" {
			t.Error("expected session ID in context, got empty")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == cookieName {
			found = true
			if c.Value == "" {
				t.Error("cookie value is empty")
			}
			if !c.HttpOnly {
				t.Error("cookie should be HttpOnly")
			}
			if c.SameSite != http.SameSiteLaxMode {
				t.Error("cookie should be SameSite=Lax")
			}
		}
	}
	if !found {
		t.Error("session cookie not set in response")
	}
}

func TestSession_ReusesExistingCookie(t *testing.T) {
	expectedID := "existing-session-id-123"

	handler := Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sid := GetSessionID(r.Context())
		if sid != expectedID {
			t.Errorf("expected session ID %q, got %q", expectedID, sid)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: expectedID})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should not set a new cookie when one already exists
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == cookieName {
			t.Error("should not re-set cookie when one already exists")
		}
	}
}

func TestGetSessionID_EmptyContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sid := GetSessionID(req.Context())
	if sid != "" {
		t.Errorf("expected empty session ID from bare context, got %q", sid)
	}
}
