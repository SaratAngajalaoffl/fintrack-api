package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"fintrack/api/internal/auth"
)

func TestRequireSessionMissingCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess, ok := requireSession(rec, req, []byte("sixteencharslong"))
	if ok || sess != nil {
		t.Fatalf("expected unauthorized")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestRequireSessionValid(t *testing.T) {
	secret := []byte("sixteencharslong")
	tok, err := auth.SignSessionToken(secret, "user-1", "a@b.co")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: tok})
	sess, ok := requireSession(rec, req, secret)
	if !ok || sess == nil || sess.Sub != "user-1" {
		t.Fatalf("ok=%v sess=%+v", ok, sess)
	}
}

func TestRequireSessionBadJWT(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: "not-a-jwt"})
	sess, ok := requireSession(rec, req, []byte("sixteencharslong"))
	if ok || sess != nil {
		t.Fatal("expected failure")
	}
}
