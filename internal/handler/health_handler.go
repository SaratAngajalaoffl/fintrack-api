package handler

import (
	"net/http"
)

// Health responds with a minimal JSON payload for load balancers and compose healthchecks.
// @Summary Health check
// @Description Liveness probe; does not require a database.
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string "Example: {\"status\":\"ok\"}"
// @Router /health [get]
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
