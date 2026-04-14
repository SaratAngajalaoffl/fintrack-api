package httpx

import (
	"encoding/json"
	"io"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// WriteJSON writes a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ReadJSON unmarshals the request body into dst (empty body -> err).
func ReadJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxBodyBytes)
	return json.NewDecoder(limited).Decode(dst)
}
