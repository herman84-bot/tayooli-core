package handler

import (
	"encoding/json"
	"net/http"
)

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// respondError is a convenience wrapper around RespondError for handlers that
// already have access to the request. Use RespondError directly for new code.
func respondError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	RespondError(w, r, status, msg)
}
