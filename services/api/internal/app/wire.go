package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/email"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/media"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/paystack"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/config"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/transport/httpapi"
)

type indexEnsurer interface {
	EnsureIndexes(ctx context.Context) error
}

// holdReleaser settles lapsed, unpaid slot holds. The background sweeper in
// run.go calls it on a timer so slots are freed even on a quiet calendar.
type holdReleaser interface {
	ReleaseExpiredHolds(ctx context.Context)
}

// buildRouter wires repositories, adapters, services, and handlers.
// Dependencies flow inward: adapters satisfy domain ports, services depend
// on ports only, and the HTTP layer depends on services. It also returns the
// background hold sweeper so the lifecycle owner can run it on a timer.
func buildRouter(
	ctx context.Context,
	cfg *config.Config,
	client *mongo.Client,
	logger *slog.Logger,
) (http.Handler, *service.CalendarVisit, error) {
	db := client.Database(cfg.MongoDB)

	// Driven adapters (persistence).
	subscribers := mongostore.NewSubscriberRepository(db)
	users := mongostore.NewUserRepository(db)
	tokens := mongostore.NewTokenRepository(db)
	settings := mongostore.NewSettingsRepository(db)
	collections := mongostore.NewCollectionRepository(db)
	designs := mongostore.NewDesignRepository(db)
	orders := mongostore.NewOrderRepository(db)
	paymentEvents := mongostore.NewPaymentEventRepository(db)
	analyticsRepo := mongostore.NewAnalyticsRepository(db)
	slots := mongostore.NewSlotRepository(db)
	visits := mongostore.NewVisitRepository(db)
	roles := mongostore.NewRoleRepository(db)

	err := ensureIndexes(ctx, []indexEnsurer{
		subscribers, users, tokens, collections, designs, orders, paymentEvents, analyticsRepo, slots, visits, roles,
	})
	if err != nil {
		return nil, nil, err
	}

	// Driven adapters (integrations) — degrade gracefully when unconfigured.
	var sender domain.EmailSender = email.NewNoopSender(logger)
	if cfg.EmailEnabled() {
		sender = email.NewResendSender(cfg.ResendAPIKey, cfg.EmailFrom)
	}

	var signer domain.UploadSigner
	if cfg.UploadsEnabled() {
		signer = media.NewCloudinarySigner(
			cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
	}

	payments := paystack.NewClient(cfg.PaystackSecretKey)

	// Application services (use-cases over ports).
	waitlist := service.NewWaitlist(subscribers, sender, logger)
	auth := service.NewAuth(users, tokens, roles, sender, logger, cfg.WebURL, cfg.AdminEmails)
	store := service.NewStoreSettings(settings)
	catalog := service.NewCatalog(collections, designs)
	orderService := service.NewOrder(orders, designs, users, payments, paymentEvents, sender, settings, cfg.WebURL, logger)
	analyticsService := service.NewAnalytics(analyticsRepo)
	roleService := service.NewRoles(roles)
	slotService := service.NewCalendarSlot(slots)
	visitService := service.NewCalendarVisit(
		slots, visits, orders, designs, users, payments, settings, sender, cfg.WebURL, logger,
	)

	// Driving adapter (HTTP).
	handlers := httpapi.NewHandlers(
		waitlist, auth, store, catalog, orderService, analyticsService, roleService, slotService, visitService,
		signer, cfg.CloudinaryCloudName, cfg.IsProduction(), mongostore.NewHealthChecker(client),
	)

	return httpapi.NewRouter(handlers, logger, cfg.CORSAllowedOrigins), visitService, nil
}

// ensureIndexes runs EnsureIndexes (which also seeds where applicable) on every
// repository, failing fast on the first error.
func ensureIndexes(ctx context.Context, repos []indexEnsurer) error {
	for _, repo := range repos {
		err := repo.EnsureIndexes(ctx)
		if err != nil {
			return fmt.Errorf("ensure indexes: %w", err)
		}
	}

	return nil
}
