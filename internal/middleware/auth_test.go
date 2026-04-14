package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthPlaceholderPassesThrough(t *testing.T) {
	called := false
	h := AuthPlaceholder(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("handler not called")
	}
}
