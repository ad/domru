package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ad/domru/internal/api"
)

var ErrAPINotInitialized = errors.New("api wrapper not initialized")

// CamerasHandler uses API wrapper to return cameras list.
func (h *Handler) CamerasHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.API == nil {
		writeJSONError(w, ErrAPINotInitialized)
		return
	}
	cams, err := h.API.Cameras(r.Context())
	if err != nil {
		if se := api.StatusFromError(err); se != http.StatusBadGateway {
			w.WriteHeader(se)
		}
		writeJSONError(w, err)
		return
	}
	b, _ := json.Marshal(cams)
	if _, err := w.Write(b); err != nil {
		log.Println("camerasHandler write error", err)
	}
}
