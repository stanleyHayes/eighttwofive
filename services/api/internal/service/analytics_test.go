package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/service"
)

type fakeAnalyticsRepo struct {
	analytics *domain.StoreAnalytics
}

func (f *fakeAnalyticsRepo) GetStoreAnalytics(context.Context) (*domain.StoreAnalytics, error) {
	return f.analytics, nil
}

func TestAnalytics_GetStoreAnalytics(t *testing.T) {
	t.Parallel()

	expected := &domain.StoreAnalytics{
		WaitlistCount:  12,
		CustomerCount:  8,
		OrdersByStatus: map[string]int64{"booked": 3, "requested": 2},
		OrdersByType:   map[string]int64{"standard": 3, "custom_size": 2},
		RevenuePesewas: 450_000,
	}

	svc := service.NewAnalytics(&fakeAnalyticsRepo{analytics: expected})
	got, err := svc.GetStoreAnalytics(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expected.WaitlistCount, got.WaitlistCount)
	assert.Equal(t, expected.CustomerCount, got.CustomerCount)
	assert.Equal(t, expected.OrdersByStatus, got.OrdersByStatus)
	assert.Equal(t, expected.OrdersByType, got.OrdersByType)
	assert.Equal(t, expected.RevenuePesewas, got.RevenuePesewas)
}

func TestAnalytics_GetStoreAnalytics_PassesEnrichedFields(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)
	expected := &domain.StoreAnalytics{
		WaitlistCount:            12,
		CustomerCount:            8,
		OrderCount:               5,
		BookedRevenuePesewas:     450_000,
		AverageOrderValuePesewas: 90_000,
		OrdersByStatus:           map[string]int64{"booked": 3, "requested": 2},
		OrdersByType:             map[string]int64{"standard": 3, "custom_size": 2},
		RevenuePesewas:           450_000,
		CollectionViews:          0,
		Comparison: domain.PeriodComparison{
			CurrentRevenuePesewas: 300_000,
			PriorRevenuePesewas:   150_000,
			CurrentOrderCount:     3,
			PriorOrderCount:       2,
			RevenueChangeBps:      10_000,
			OrderCountChangeBps:   5_000,
		},
		RevenueSeries: []domain.TimeBucket{
			{Label: "1 Jun", StartAt: createdAt, RevenuePesewas: 100_000, OrderCount: 1},
		},
		TopDesigns: []domain.DesignStat{
			{DesignID: "d1", Name: "Blazer", OrderCount: 4, RevenuePesewas: 400_000},
		},
		TopCollections: []domain.CollectionStat{
			{CollectionID: "c1", Name: "Heritage", OrderCount: 5, RevenuePesewas: 450_000},
		},
		RecentOrders: []domain.RecentOrder{
			{Ref: "E25-1", Type: "standard", Status: "booked", TotalPesewas: 100_000, CreatedAt: createdAt},
		},
	}

	svc := service.NewAnalytics(&fakeAnalyticsRepo{analytics: expected})
	got, err := svc.GetStoreAnalytics(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(5), got.OrderCount)
	assert.Equal(t, int64(450_000), got.BookedRevenuePesewas)
	assert.Equal(t, int64(90_000), got.AverageOrderValuePesewas)
	assert.Equal(t, int64(10_000), got.Comparison.RevenueChangeBps)
	assert.Equal(t, int64(5_000), got.Comparison.OrderCountChangeBps)
	require.Len(t, got.RevenueSeries, 1)
	assert.Equal(t, int64(100_000), got.RevenueSeries[0].RevenuePesewas)
	require.Len(t, got.TopDesigns, 1)
	assert.Equal(t, "Blazer", got.TopDesigns[0].Name)
	require.Len(t, got.TopCollections, 1)
	assert.Equal(t, int64(450_000), got.TopCollections[0].RevenuePesewas)
	require.Len(t, got.RecentOrders, 1)
	assert.Equal(t, "E25-1", got.RecentOrders[0].Ref)
}
