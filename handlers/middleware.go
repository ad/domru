package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ad/domru/internal/api"
)

// JSONErrorWriter maps generic error text patterns to HTTP status codes and writes JSON.
func JSONErrorWriter(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	status := api.StatusFromError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// WrapJSONError converts handler returning error into one writing JSON errors.
func WrapJSONError(next func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := next(w, r); err != nil {
			JSONErrorWriter(w, err)
		}
	}
}

// Backwards compatibility helper.
// writeJSONError kept for backward compatibility references in existing handlers.
func writeJSONError(w http.ResponseWriter, err error) { JSONErrorWriter(w, err) }
