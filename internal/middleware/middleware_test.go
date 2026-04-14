package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChainOrder(t *testing.T) {
	var order []string
	a := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "a-in")
			next.ServeHTTP(w, r)
			order = append(order, "a-out")
		})
	}
	b := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "b-in")
			next.ServeHTTP(w, r)
			order = append(order, "b-out")
		})
	}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "inner")
	})
	h := Chain(inner, a, b)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	want := []string{"a-in", "b-in", "inner", "b-out", "a-out"}
	if len(order) != len(want) {
		t.Fatalf("got %v want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("got %v want %v", order, want)
		}
	}
}
