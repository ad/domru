package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ad/domru/internal/api"
)

// FinancesHandler now uses API wrapper.
func (h *Handler) FinancesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.API == nil {
		writeJSONError(w, ErrAPINotInitialized)
		return
	}
	finances, err := h.API.Finances(r.Context())
	if err != nil {
		if se := api.StatusFromError(err); se != http.StatusBadGateway {
			w.WriteHeader(se)
		}
		writeJSONError(w, err)
		return
	}
	b, _ := json.Marshal(finances)
	if _, err := w.Write(b); err != nil {
		log.Println("financesHandler write", err)
	}
}
