package mongostore_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func TestSettingsRepository_DeliveryRates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in -short mode")
	}

	t.Parallel()

	db := setupDatabase(t)
	repo := mongostore.NewSettingsRepository(db)
	ctx := context.Background()

	// Defaults before first save.
	defaults, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(500_00), defaults.DepositPesewas)
	assert.Empty(t, defaults.DeliveryRates)

	// Save and round-trip delivery rates.
	saved := &domain.Settings{
		DepositPesewas: 600_00,
		WhatsAppNumber: "+233200000000",
		VisitLocation:  "Osu, Accra",
		DeliveryRates: []domain.DeliveryRate{
			{Area: "Accra", RatePesewas: 1000},
			{Area: "Tema", RatePesewas: 2500},
		},
	}
	require.NoError(t, repo.Update(ctx, saved))

	loaded, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, saved.DepositPesewas, loaded.DepositPesewas)
	assert.Equal(t, saved.WhatsAppNumber, loaded.WhatsAppNumber)
	assert.Equal(t, saved.VisitLocation, loaded.VisitLocation)
	assert.Equal(t, saved.DeliveryRates, loaded.DeliveryRates)
}
