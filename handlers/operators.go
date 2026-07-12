package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ad/domru/internal/api"
)

// Operators ...
func (h *Handler) Operators(r *http.Request) (interface{}, error) {
	if h.API == nil {
		return nil, api.ErrUnknown
	}
	data, err := h.API.Operators(r.Context())
	if err != nil {
		return nil, err
	}
	return data, nil
}

// OperatorsHandler ...
func (h *Handler) OperatorsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, err := h.Operators(r)
	if err != nil {
		if se := api.StatusFromError(err); se != http.StatusBadGateway {
			w.WriteHeader(se)
		}
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		w.Write(b)
		return
	}
	b, _ := json.Marshal(data)
	if _, err := w.Write(b); err != nil {
		log.Println("operatorsHandler write", err)
	}
}
