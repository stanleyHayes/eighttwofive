package httpapi

import (
	"maps"
	"net/http"
	"time"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// timeBucketDTO is one bar of the booked-revenue time series.
type timeBucketDTO struct {
	Label          string    `json:"label"`
	StartAt        time.Time `json:"startAt"`
	RevenuePesewas int64     `json:"revenuePesewas"`
	OrderCount     int64     `json:"orderCount"`
}

// designStatDTO ranks one design by booked order count.
type designStatDTO struct {
	DesignID       string `json:"designId"`
	Name           string `json:"name"`
	OrderCount     int64  `json:"orderCount"`
	RevenuePesewas int64  `json:"revenuePesewas"`
}

// collectionStatDTO ranks one collection by booked revenue.
type collectionStatDTO struct {
	CollectionID   string `json:"collectionId"`
	Name           string `json:"name"`
	OrderCount     int64  `json:"orderCount"`
	RevenuePesewas int64  `json:"revenuePesewas"`
}

// recentOrderDTO is one row of the activity feed.
type recentOrderDTO struct {
	Ref          string    `json:"ref"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	TotalPesewas int64     `json:"totalPesewas"`
	CreatedAt    time.Time `json:"createdAt"`
}

// periodComparisonDTO contrasts the trailing window with the prior one. Change
// figures are integer basis points (100 bps == 1%).
type periodComparisonDTO struct {
	CurrentRevenuePesewas int64 `json:"currentRevenuePesewas"`
	PriorRevenuePesewas   int64 `json:"priorRevenuePesewas"`
	CurrentOrderCount     int64 `json:"currentOrderCount"`
	PriorOrderCount       int64 `json:"priorOrderCount"`
	RevenueChangeBps      int64 `json:"revenueChangeBps"`
	OrderCountChangeBps   int64 `json:"orderCountChangeBps"`
}

// storeAnalyticsDTO is the envelope payload for GET /api/v1/admin/analytics.
type storeAnalyticsDTO struct {
	WaitlistCount            int64               `json:"waitlistCount"`
	CustomerCount            int64               `json:"customerCount"`
	OrderCount               int64               `json:"orderCount"`
	BookedRevenuePesewas     int64               `json:"bookedRevenuePesewas"`
	AverageOrderValuePesewas int64               `json:"averageOrderValuePesewas"`
	OrdersByStatus           map[string]int64    `json:"ordersByStatus"`
	OrdersByType             map[string]int64    `json:"ordersByType"`
	RevenuePesewas           int64               `json:"revenuePesewas"`
	CollectionViews          int64               `json:"collectionViews"`
	Comparison               periodComparisonDTO `json:"comparison"`
	RevenueSeries            []timeBucketDTO     `json:"revenueSeries"`
	TopDesigns               []designStatDTO     `json:"topDesigns"`
	TopCollections           []collectionStatDTO `json:"topCollections"`
	RecentOrders             []recentOrderDTO    `json:"recentOrders"`
}

func toStoreAnalyticsDTO(analytics *domain.StoreAnalytics) storeAnalyticsDTO {
	return storeAnalyticsDTO{
		WaitlistCount:            analytics.WaitlistCount,
		CustomerCount:            analytics.CustomerCount,
		OrderCount:               analytics.OrderCount,
		BookedRevenuePesewas:     analytics.BookedRevenuePesewas,
		AverageOrderValuePesewas: analytics.AverageOrderValuePesewas,
		OrdersByStatus:           copyCountMap(analytics.OrdersByStatus),
		OrdersByType:             copyCountMap(analytics.OrdersByType),
		RevenuePesewas:           analytics.RevenuePesewas,
		CollectionViews:          analytics.CollectionViews,
		Comparison:               toComparisonDTO(analytics.Comparison),
		RevenueSeries:            toSeriesDTO(analytics.RevenueSeries),
		TopDesigns:               toDesignStatsDTO(analytics.TopDesigns),
		TopCollections:           toCollectionStatsDTO(analytics.TopCollections),
		RecentOrders:             toRecentOrdersDTO(analytics.RecentOrders),
	}
}

func toComparisonDTO(cmp domain.PeriodComparison) periodComparisonDTO {
	return periodComparisonDTO{
		CurrentRevenuePesewas: cmp.CurrentRevenuePesewas,
		PriorRevenuePesewas:   cmp.PriorRevenuePesewas,
		CurrentOrderCount:     cmp.CurrentOrderCount,
		PriorOrderCount:       cmp.PriorOrderCount,
		RevenueChangeBps:      cmp.RevenueChangeBps,
		OrderCountChangeBps:   cmp.OrderCountChangeBps,
	}
}

func toSeriesDTO(buckets []domain.TimeBucket) []timeBucketDTO {
	out := make([]timeBucketDTO, 0, len(buckets))
	for _, bucket := range buckets {
		out = append(out, timeBucketDTO{
			Label:          bucket.Label,
			StartAt:        bucket.StartAt,
			RevenuePesewas: bucket.RevenuePesewas,
			OrderCount:     bucket.OrderCount,
		})
	}

	return out
}

func toDesignStatsDTO(stats []domain.DesignStat) []designStatDTO {
	out := make([]designStatDTO, 0, len(stats))
	for _, stat := range stats {
		out = append(out, designStatDTO{
			DesignID:       stat.DesignID,
			Name:           stat.Name,
			OrderCount:     stat.OrderCount,
			RevenuePesewas: stat.RevenuePesewas,
		})
	}

	return out
}

func toCollectionStatsDTO(stats []domain.CollectionStat) []collectionStatDTO {
	out := make([]collectionStatDTO, 0, len(stats))
	for _, stat := range stats {
		out = append(out, collectionStatDTO{
			CollectionID:   stat.CollectionID,
			Name:           stat.Name,
			OrderCount:     stat.OrderCount,
			RevenuePesewas: stat.RevenuePesewas,
		})
	}

	return out
}

func toRecentOrdersDTO(orders []domain.RecentOrder) []recentOrderDTO {
	out := make([]recentOrderDTO, 0, len(orders))
	for _, order := range orders {
		out = append(out, recentOrderDTO{
			Ref:          order.Ref,
			Type:         order.Type,
			Status:       order.Status,
			TotalPesewas: order.TotalPesewas,
			CreatedAt:    order.CreatedAt,
		})
	}

	return out
}

func copyCountMap(src map[string]int64) map[string]int64 {
	if src == nil {
		return nil
	}

	dst := make(map[string]int64, len(src))
	maps.Copy(dst, src)

	return dst
}

// AdminGetAnalytics handles GET /api/v1/admin/analytics.
func (h *Handlers) AdminGetAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := h.analytics.GetStoreAnalytics(r.Context())
	if err != nil {
		respondInternal(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, toStoreAnalyticsDTO(analytics))
}
