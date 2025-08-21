package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ad/domru/internal/api"
)

// EventsHandler now uses API wrapper; expects placeID query param.
func (h *Handler) EventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.API == nil {
		writeJSONError(w, ErrAPINotInitialized)
		return
	}
	placeID := r.URL.Query().Get("placeID")
	if placeID == "" {
		writeJSONError(w, fmt.Errorf("provide placeID"))
		return
	}
	events, err := h.API.Events(r.Context(), placeID)
	if err != nil {
		if se := api.StatusFromError(err); se != http.StatusBadGateway {
			w.WriteHeader(se)
		}
		writeJSONError(w, err)
		return
	}
	b, _ := json.Marshal(events)
	if _, err := w.Write(b); err != nil {
		log.Println("eventsHandler write", err)
	}
}

// LastEventHandler returns only the latest event item when available.
func (h *Handler) LastEventHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.API == nil {
		writeJSONError(w, ErrAPINotInitialized)
		return
	}
	placeID := r.URL.Query().Get("placeID")
	if placeID == "" {
		writeJSONError(w, fmt.Errorf("provide placeID"))
		return
	}
	events, err := h.API.Events(r.Context(), placeID)
	if err != nil {
		if se := api.StatusFromError(err); se != http.StatusBadGateway {
			w.WriteHeader(se)
		}
		writeJSONError(w, err)
		return
	}
	if len(events.Data) == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"events not found"}`))
		return
	}
	b, _ := json.Marshal(events.Data[0])
	if _, err := w.Write(b); err != nil {
		log.Println("lastEventHandler write", err)
	}
}
