package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/davcs86/happybabytracker/api/internal/constants"
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

// respondInvalid is the 422 for a field that fails validation.
func respondInvalid(w http.ResponseWriter, field string) {
	respondError(w, http.StatusUnprocessableEntity, "invalid", field)
}

func respondInternal(w http.ResponseWriter) {
	respondError(w, http.StatusInternalServerError, "internal")
}

var errTrailingData = errors.New("trailing data after the JSON value")

const unknownFieldPrefix = "json: unknown field "

// decodeJSON reads one capped JSON value into v. On failure it writes the error response and returns false:
// 413 body_too_large, 422 unknown_field (field = the key), 400 bad_request for anything else.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, constants.MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(v)
	if err == nil {
		if err = dec.Decode(&json.RawMessage{}); err == io.EOF {
			return true
		} else if err == nil {
			err = errTrailingData
		}
	}
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig):
		respondError(w, http.StatusRequestEntityTooLarge, "body_too_large")
	case strings.HasPrefix(err.Error(), unknownFieldPrefix):
		name, _ := strconv.Unquote(strings.TrimPrefix(err.Error(), unknownFieldPrefix))
		respondError(w, http.StatusUnprocessableEntity, "unknown_field", name)
	default:
		respondError(w, http.StatusBadRequest, "bad_request")
	}
	return false
}
