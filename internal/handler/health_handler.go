package handler

import (
	"net/http"
)

// Health responds with a minimal JSON payload for load balancers and compose healthchecks.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
