package handlers

import (
	"fmt"
	"net/http"
)

// StreamHandler now builds redirect via API wrapper.
func (h *Handler) StreamHandler(w http.ResponseWriter, r *http.Request) {
	if h.API == nil {
		w.Header().Set("Content-Type", "application/json")
		writeJSONError(w, ErrAPINotInitialized)
		return
	}
	cameraID := r.URL.Query().Get("cameraID")
	if cameraID == "" {
		w.Header().Set("Content-Type", "application/json")
		writeJSONError(w, fmt.Errorf("provide cameraID"))
		return
	}
	target := h.API.StreamURL(cameraID, r.URL.Query())
	http.Redirect(w, r, target, http.StatusFound)
}
