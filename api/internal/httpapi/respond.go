package httpapi

import (
	"encoding/json"
	"net/http"
)

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorBody struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

// respondError writes {"error": code}, plus "field" when one is given.
func respondError(w http.ResponseWriter, status int, code string, field ...string) {
	b := errorBody{Error: code}
	if len(field) > 0 {
		b.Field = field[0]
	}
	respondJSON(w, status, b)
}
