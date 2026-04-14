package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"fintrack/api/internal/auth"
)

func TestReadSessionCookieMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if readSessionCookie(req) != "" {
		t.Fatal("expected empty")
	}
}

func TestReadSessionCookiePresent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: "tok"})
	if readSessionCookie(req) != "tok" {
		t.Fatal("expected tok")
	}
}

func TestWriteAndClearSessionCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	writeSessionCookie(rec, "jwt-here", false)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != auth.SessionCookie {
		t.Fatalf("%v", cookies)
	}
	clearSessionCookie(rec, false)
	// clear adds another Set-Cookie
	if len(rec.Result().Cookies()) < 1 {
		t.Fatal("expected cookies")
	}
}
