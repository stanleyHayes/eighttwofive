// Package httpapi is the inbound HTTP adapter: routing, request decoding,
// and mapping domain errors to status codes.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
