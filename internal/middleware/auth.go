package middleware

import "net/http"

// AuthPlaceholder is reserved for session/JWT validation when auth routes move here.
// It currently passes all requests through unchanged.
func AuthPlaceholder(next http.Handler) http.Handler {
	return next
}
