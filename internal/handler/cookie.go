package handler

import (
	"net/http"
	"time"

	"fintrack/api/internal/auth"
)

func readSessionCookie(r *http.Request) string {
	c, err := r.Cookie(auth.SessionCookie)
	if err != nil || c.Value == "" {
		return ""
	}
	return c.Value
}

func writeSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   auth.JWT_TTL_SEC,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  time.Unix(0, 0),
	})
}
