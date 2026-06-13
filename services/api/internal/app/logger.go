package app

import (
	"log/slog"
	"os"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/config"
)

// NewLogger builds the process logger: JSON in production, text in development.
func NewLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		AddSource:   false,
		ReplaceAttr: nil,
	}
	if cfg.IsProduction() {
		opts.Level = slog.LevelInfo

		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
