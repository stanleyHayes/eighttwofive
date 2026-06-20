package httpapi_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/transport/httpapi"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestHealth_DatabaseUnreachable(t *testing.T) {
	t.Parallel()

	// Only the readiness pinger is exercised by /healthz, so the services can
	// stay nil for this check.
	handlers := httpapi.NewHandlers(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "", false,
		stubPinger{err: errors.New("connection refused")},
	)
	srv := httptest.NewServer(httpapi.NewRouter(handlers, slog.New(slog.DiscardHandler), []string{"*"}))
	t.Cleanup(srv.Close)

	assert.Equal(t, http.StatusServiceUnavailable, getStatus(t, srv.URL+"/healthz"))
}

func TestHealth_DatabaseReachable(t *testing.T) {
	t.Parallel()

	handlers := httpapi.NewHandlers(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "", false,
		stubPinger{err: nil},
	)
	srv := httptest.NewServer(httpapi.NewRouter(handlers, slog.New(slog.DiscardHandler), []string{"*"}))
	t.Cleanup(srv.Close)

	assert.Equal(t, http.StatusOK, getStatus(t, srv.URL+"/healthz"))
}
