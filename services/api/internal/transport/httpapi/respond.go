// Package httpapi is the inbound HTTP adapter: routing, request decoding,
// and mapping domain errors to status codes.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Data  any        `json:"data,omitempty"`
	Error *errorBody `json:"error,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(envelope{Data: data})
	if err != nil {
		slog.Error("encode response", "error", err)
	}
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(envelope{Error: &errorBody{Code: code, Message: message}})
	if err != nil {
		slog.Error("encode error response", "error", err)
	}
}

// logRequestError records a server-side failure with the request id so it can
// be matched to the client's failed request. Used by respondInternal and by
// handlers that map a known failure to a non-500 status but still want the
// cause recorded.
func logRequestError(r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "request failed",
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", chimw.GetReqID(r.Context()),
	)
}

// respondInternal logs the underlying cause of a 500 and returns the generic
// error message, keeping internals out of the response body. Every unexpected
// server-side failure should funnel through here instead of a bare respondError
// so no 500 goes unrecorded.
func respondInternal(w http.ResponseWriter, r *http.Request, err error) {
	logRequestError(r, err)
	respondError(w, http.StatusInternalServerError, "internal", "something went wrong")
}
