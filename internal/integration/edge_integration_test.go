package integration

import (
	"net/http"
	"testing"
)

func TestAPI_UnauthorizedWithoutSession(t *testing.T) {
	h := newHarness(t)
	res, err := http.Get(h.BaseURL + "/api/bank-accounts")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got %d", res.StatusCode)
	}
}
