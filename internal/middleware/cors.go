package middleware

import (
	"net/http"
	"strings"
)

// CORS allows credentialed browser requests from listed Origin values (e.g. Next dev on :3000 → API on :8080).
// If allowed is empty, a small dev default (localhost / 127.0.0.1 :3000) is used so local development works.
func CORS(allowed []string) func(http.Handler) http.Handler {
	trimmed := trimNonEmpty(allowed)
	if len(trimmed) == 0 {
		trimmed = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
	}
	set := make(map[string]struct{}, len(trimmed))
	for _, o := range trimmed {
		set[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := set[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Vary", "Origin")
				}
			}

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func trimNonEmpty(parts []string) []string {
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
