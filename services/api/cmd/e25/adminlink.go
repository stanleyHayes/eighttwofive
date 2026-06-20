package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

var errNoLinkCaptured = errors.New("no sign-in link was produced")

// captureSender implements domain.EmailSender but prints instead of sending,
// so a sign-in link can be minted without email delivery.
type captureSender struct {
	link string
}

func (c *captureSender) SendWelcome(context.Context, string, string) error { return nil }

func (c *captureSender) SendOrderConfirmation(context.Context, string, string, string, string) error {
	return nil
}

func (c *captureSender) SendOrderStatusUpdate(context.Context, string, string, string, string, string) error {
	return nil
}

func (c *captureSender) SendLoginLink(_ context.Context, _, link string) error {
	c.link = link

	return nil
}

func newAdminLinkCommand() *cobra.Command {
	var email, name string

	cmd := &cobra.Command{
		Use:   "admin-link",
		Short: "Mint a single-use admin sign-in link (no email needed)",
		Long: "Upserts the user with the admin role and prints a fresh sign-in link " +
			"for the web app. The link works once and expires in 15 minutes.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadEnvironment()
			if err != nil {
				return err
			}

			return withDatabase(cmd.Context(), cfg, func(db *mongo.Database) error {
				return mintAdminLink(cmd.Context(), cmd.OutOrStdout(), db, cfg.WebURL, email, name)
			})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "admin email address (required)")
	cmd.Flags().StringVar(&name, "name", "Admin", "display name for first sign-in")
	_ = cmd.MarkFlagRequired("email")

	return cmd
}

func mintAdminLink(ctx context.Context, out io.Writer, db *mongo.Database, webURL, email, name string) error {
	users := mongostore.NewUserRepository(db)
	tokens := mongostore.NewTokenRepository(db)
	roles := mongostore.NewRoleRepository(db)

	for _, ensure := range []interface {
		EnsureIndexes(ctx context.Context) error
	}{users, tokens, roles} {
		err := ensure.EnsureIndexes(ctx)
		if err != nil {
			return fmt.Errorf("ensure indexes: %w", err)
		}
	}

	sender := &captureSender{link: ""}
	// Passing the email as the allowlist makes RequestLink assign the admin role.
	auth := service.NewAuth(users, tokens, roles, sender, quietLogger(), webURL, []string{email})

	err := auth.RequestLink(ctx, email, name)
	if err != nil {
		return fmt.Errorf("mint link: %w", err)
	}

	if sender.link == "" {
		return errNoLinkCaptured
	}

	_, _ = fmt.Fprintln(out, sender.link)

	return nil
}
