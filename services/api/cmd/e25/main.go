// Command e25 is the operations CLI for the Eight Two Five platform:
// seeding demo data and minting admin sign-in links. It reuses the same
// services and adapters as the API server, so everything it writes obeys
// the domain rules (slugs, validation, role promotion).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/config"
)

func main() {
	root := &cobra.Command{
		Use:           "e25",
		Short:         "Eight Two Five operations CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newSeedCommand(),
		newAdminLinkCommand(),
		newSeedImagesCommand(),
		newSeedOrdersCommand(),
		newSeedMoreCommand(),
	)

	err := root.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// loadEnvironment loads .env (when present) and the typed configuration.
func loadEnvironment() (*config.Config, error) {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return cfg, nil
}

// withDatabase connects, runs fn, and disconnects.
func withDatabase(ctx context.Context, cfg *config.Config, run func(*mongo.Database) error) error {
	client, err := mongostore.Connect(ctx, cfg.MongoURI)
	if err != nil {
		return fmt.Errorf("connect storage: %w", err)
	}

	defer func() {
		_ = client.Disconnect(context.WithoutCancel(ctx))
	}()

	return run(client.Database(cfg.MongoDB))
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelWarn,
		AddSource:   false,
		ReplaceAttr: nil,
	}))
}
