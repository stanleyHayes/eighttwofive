// Command server is the entry point for the eightfivetwo HTTP API.
// All wiring lives in internal/app; this file only boots the process.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/app"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/config"
)

func main() {
	err := run()
	if err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
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
