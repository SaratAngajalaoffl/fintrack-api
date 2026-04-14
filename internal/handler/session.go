package handler

import (
	"net/http"

	"fintrack/api/internal/auth"
	"fintrack/api/internal/httpx"
)

// requireSession reads the session cookie, verifies JWT, or writes 401 JSON.
func requireSession(w http.ResponseWriter, r *http.Request, jwtSecret []byte) (*auth.SessionPayload, bool) {
	raw := readSessionCookie(r)
	if raw == "" {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return nil, false
	}
	sess, err := auth.VerifySessionToken(jwtSecret, raw)
	if err != nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return nil, false
	}
	return sess, true
}
