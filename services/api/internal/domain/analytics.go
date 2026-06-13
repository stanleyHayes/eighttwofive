package domain

import (
	"context"
	"time"
)

// TimeBucket is one point in a booked-revenue time series. StartAt is the
// inclusive start of the bucket window; Label is a short human caption for an
// axis tick (for example "12 Jan").
type TimeBucket struct {
	Label          string
	StartAt        time.Time
	RevenuePesewas int64
	OrderCount     int64
}

// DesignStat ranks one design by its booked order volume. RevenuePesewas is the
// booked revenue attributed to the design over all time.
type DesignStat struct {
	DesignID       string
	Name           string
	OrderCount     int64
	RevenuePesewas int64
}

// CollectionStat ranks one collection by its booked revenue.
type CollectionStat struct {
	CollectionID   string
	Name           string
	OrderCount     int64
	RevenuePesewas int64
}

// RecentOrder is a compact order row for the activity feed.
type RecentOrder struct {
	Ref          string
	Type         string
	Status       string
	TotalPesewas int64
	CreatedAt    time.Time
}

// PeriodComparison contrasts the trailing window against the window before it.
// Change figures are expressed in integer basis points (1% == 100 bps) so the
// transport layer never has to round a float; a missing prior baseline yields a
// zero change rather than a divide-by-zero.
type PeriodComparison struct {
	CurrentRevenuePesewas int64
	PriorRevenuePesewas   int64
	CurrentOrderCount     int64
	PriorOrderCount       int64
	RevenueChangeBps      int64
	OrderCountChangeBps   int64
}

// StoreAnalytics is a snapshot of the store's commercial health: headline
// totals, a trailing-window comparison, a short booked-revenue time series, the
// best-selling designs and collections, and a recent-orders activity feed. All
// money is integer pesewas; the frontend formats to GH₵.
type StoreAnalytics struct {
	WaitlistCount            int64
	CustomerCount            int64
	OrderCount               int64
	BookedRevenuePesewas     int64
	AverageOrderValuePesewas int64
	OrdersByStatus           map[string]int64
	OrdersByType             map[string]int64
	RevenuePesewas           int64
	CollectionViews          int64
	Comparison               PeriodComparison
	RevenueSeries            []TimeBucket
	TopDesigns               []DesignStat
	TopCollections           []CollectionStat
	RecentOrders             []RecentOrder
}

// AnalyticsRepository is the persistence port for aggregate store metrics.
type AnalyticsRepository interface {
	GetStoreAnalytics(ctx context.Context) (*StoreAnalytics, error)
}
