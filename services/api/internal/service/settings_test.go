package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

type memSettings struct {
	saved *domain.Settings
}

func (m *memSettings) Get(_ context.Context) (*domain.Settings, error) {
	if m.saved == nil {
		return domain.DefaultSettings(), nil
	}

	clone := *m.saved

	return &clone, nil
}

func (m *memSettings) Update(_ context.Context, s *domain.Settings) error {
	clone := *s
	m.saved = &clone

	return nil
}

func TestStoreSettings_Get_Defaults(t *testing.T) {
	t.Parallel()

	settings := service.NewStoreSettings(&memSettings{saved: nil})
	got, err := settings.Get(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(500_00), got.DepositPesewas)
	assert.Empty(t, got.WhatsAppNumber)
	assert.Empty(t, got.VisitLocation)
	assert.Empty(t, got.DeliveryRates)
}

func TestStoreSettings_Update_ValidatesDeposit(t *testing.T) {
	t.Parallel()

	settings := service.NewStoreSettings(&memSettings{saved: nil})
	err := settings.Update(context.Background(), &domain.Settings{DepositPesewas: -1})

	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestStoreSettings_Update_ValidatesDeliveryRates(t *testing.T) {
	t.Parallel()

	settings := service.NewStoreSettings(&memSettings{saved: nil})

	cases := []struct {
		name   string
		rates  []domain.DeliveryRate
		errIs  error
		errMsg string
	}{
		{
			name:   "empty area",
			rates:  []domain.DeliveryRate{{Area: "", RatePesewas: 1000}},
			errIs:  domain.ErrInvalidInput,
			errMsg: "area cannot be empty",
		},
		{
			name:   "negative rate",
			rates:  []domain.DeliveryRate{{Area: "Accra", RatePesewas: -1}},
			errIs:  domain.ErrInvalidInput,
			errMsg: "cannot be negative",
		},
		{
			name:   "duplicate areas",
			rates:  []domain.DeliveryRate{{Area: "Accra", RatePesewas: 1000}, {Area: "Accra", RatePesewas: 2000}},
			errIs:  domain.ErrDuplicateArea,
			errMsg: "Accra",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := settings.Update(context.Background(), &domain.Settings{
				DepositPesewas: 500_00,
				DeliveryRates:  tc.rates,
			})

			require.ErrorIs(t, err, tc.errIs)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

func TestStoreSettings_Update_PersistsValidRates(t *testing.T) {
	t.Parallel()

	repo := &memSettings{saved: nil}
	settings := service.NewStoreSettings(repo)

	input := &domain.Settings{
		DepositPesewas: 600_00,
		WhatsAppNumber: "+233200000000",
		VisitLocation:  "Osu, Accra",
		DeliveryRates: []domain.DeliveryRate{
			{Area: "Accra", RatePesewas: 1000},
			{Area: "Tema", RatePesewas: 2500},
		},
	}

	require.NoError(t, settings.Update(context.Background(), input))

	got, err := settings.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, input.DeliveryRates, got.DeliveryRates)
}
