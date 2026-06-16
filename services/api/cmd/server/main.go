// Command server is the entry point for the eightfivetwo HTTP API.
// All wiring lives in internal/app; this file only boots the process.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/app"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/config"
)

const healthcheckTimeout = 3 * time.Second

func main() {
	// `server -healthcheck` runs as a separate process (e.g. the container
	// HEALTHCHECK) and probes the already-running server, so a distroless image
	// with no shell or curl can still report health.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck())
	}

	err := run()
	if err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func healthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/healthz", nil)
	if err != nil {
		return 1
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 1
	}

	return 0
}

func run() error {
	// .env is a local development convenience; absence is fine.
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load .env: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := app.NewLogger(cfg)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("build app: %w", err)
	}
	defer application.Close()

	err = application.Run(ctx)
	if err != nil {
		return fmt.Errorf("run app: %w", err)
	}

	return nil
}
