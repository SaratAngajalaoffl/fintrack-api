package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewMuxHealthOnlyWithoutDB(t *testing.T) {
	m := NewMux(Deps{DB: nil, JWTSecret: nil})
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	rec2 := httptest.NewRecorder()
	m.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered route, got %d", rec2.Code)
	}
}

func TestNewMuxHealthOnlyWithDBButNoJWT(t *testing.T) {
	m := NewMux(Deps{DB: &pgxpool.Pool{}, JWTSecret: nil})
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
}
