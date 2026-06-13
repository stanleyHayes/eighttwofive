package mongostore_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
)

// setupDatabase starts a disposable MongoDB container and returns a database
// handle scoped to the calling test.
func setupDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	ctx := context.Background()

	ctr, err := mongodb.Run(ctx, "mongo:8.0")
	testcontainers.CleanupContainer(t, ctr)
	require.NoError(t, err)

	uri, err := ctr.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongostore.Connect(ctx, uri)
	require.NoError(t, err)
	t.Cleanup(func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = client.Disconnect(disconnectCtx)
	})

	return client.Database("eightfivetwo_test")
}
