package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ad/domru/internal/api"
)

// PlacesHandler now uses API wrapper.
func (h *Handler) PlacesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.API == nil {
		writeJSONError(w, ErrAPINotInitialized)
		return
	}
	places, err := h.API.Places(r.Context())
	if err != nil {
		if se := api.StatusFromError(err); se != http.StatusBadGateway {
			w.WriteHeader(se)
		}
		writeJSONError(w, err)
		return
	}
	b, _ := json.Marshal(places)
	if _, err := w.Write(b); err != nil {
		log.Println("placesHandler write", err)
	}
}
