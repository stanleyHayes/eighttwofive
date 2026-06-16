// Package mongostore is the MongoDB adapter for the domain's persistence ports.
package mongostore

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	operationTimeout = 10 * time.Second
	pingTimeout      = 5 * time.Second
	// maxListResults caps every non-paginated List read so a pathologically
	// large collection can never exhaust server memory in a single query.
	// Endpoints that need more rows page explicitly; this is only a safety net.
	maxListResults = 1000
)

// Connect establishes and verifies a MongoDB connection.
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetTimeout(operationTimeout))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	err = client.Ping(pingCtx, nil)
	if err != nil {
		_ = client.Disconnect(context.WithoutCancel(ctx))

		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return client, nil
}

// HealthChecker reports whether MongoDB is reachable, backing the /healthz
// readiness probe.
type HealthChecker struct {
	client *mongo.Client
}

// NewHealthChecker returns a readiness checker bound to the Mongo client.
func NewHealthChecker(client *mongo.Client) *HealthChecker {
	return &HealthChecker{client: client}
}

// Ping verifies connectivity to MongoDB within the caller's deadline.
func (h *HealthChecker) Ping(ctx context.Context) error {
	err := h.client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("ping mongo: %w", err)
	}

	return nil
}
