package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ad/domru/internal/api"
)

// Door now uses API.OpenDoor
func (h *Handler) Door(r *http.Request) (string, error) {
	q := r.URL.Query()
	placeID := q.Get("placeID")
	accessControlID := q.Get("accessControlID")
	if placeID == "" || accessControlID == "" {
		return "", fmt.Errorf("provide placeID and accessControlID")
	}
	if h.API == nil {
		return "", fmt.Errorf("api not initialized")
	}
	if err := h.API.OpenDoor(r.Context(), placeID, accessControlID); err != nil {
		return "", err
	}
	return `{"status":"ok"}`, nil
}

// DoorHandler ...
func (h *Handler) DoorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, err := h.Door(r)
	if err != nil {
		if se := api.StatusFromError(err); se != http.StatusBadGateway {
			w.WriteHeader(se)
		}
		writeJSONError(w, err)
		return
	}
	if _, err := w.Write([]byte(data)); err != nil {
		log.Println("doorHandler write error", err)
	}
}

// APIClient helper ensures we have an http.Client implementing Do.
// Legacy APIClient helpers removed (handled by API layer)
