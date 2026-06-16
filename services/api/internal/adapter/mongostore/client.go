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
