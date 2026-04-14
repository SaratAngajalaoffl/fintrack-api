package httpx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusTeapot, map[string]string{"a": "b"})
	if rec.Code != http.StatusTeapot {
		t.Fatalf("code %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type: %s", ct)
	}
}

func TestReadJSON(t *testing.T) {
	body := `{"name":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	var dst struct {
		Name string `json:"name"`
	}
	if err := ReadJSON(req, &dst); err != nil {
		t.Fatal(err)
	}
	if dst.Name != "x" {
		t.Fatalf("got %q", dst.Name)
	}
}

func TestReadJSONEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(nil))
	var dst struct{}
	err := ReadJSON(req, &dst)
	if err == nil {
		t.Fatal("expected error")
	}
}
