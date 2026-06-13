// Package app is the composition root: it wires configuration, adapters,
// services, and transport together (dependency injection happens here and
// only here) and owns the process lifecycle.
//
// File layout: wire.go builds the dependency graph, server.go configures the
// HTTP server, run.go runs the serve/shutdown loop, logger.go builds the
// process logger.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/config"
)

const disconnectTimeout = 5 * time.Second

// App is the fully wired application.
type App struct {
	cfg    *config.Config
	logger *slog.Logger
	mongo  *mongo.Client
	server *http.Server
}

// New connects infrastructure and injects every dependency. It fails fast:
// an App that constructs successfully is ready to serve.
func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*App, error) {
	client, err := mongostore.Connect(ctx, cfg.MongoURI)
	if err != nil {
		return nil, fmt.Errorf("connect storage: %w", err)
	}

	router, err := buildRouter(ctx, cfg, client, logger)
	if err != nil {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), disconnectTimeout)
		defer cancel()

		_ = client.Disconnect(closeCtx)

		return nil, err
	}

	return &App{
		cfg:    cfg,
		logger: logger,
		mongo:  client,
		server: newHTTPServer(cfg, router),
	}, nil
}

// Close releases infrastructure connections.
func (a *App) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
	defer cancel()

	err := a.mongo.Disconnect(ctx)
	if err != nil {
		a.logger.Error("mongo disconnect", "error", err)
	}
}
