package mongostore

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// Analytics tuning knobs. These shape the dashboard payload: how many trailing
// buckets the revenue chart shows, how long each bucket and comparison window
// is, and how many rows the top-N tables and activity feed carry.
const (
	revenueSeriesBuckets = 12
	seriesBucketDays     = 7
	comparisonWindowDays = 30
	topListLimit         = 5
	recentOrderLimit     = 8
	basisPointScale      = 10_000
	hoursPerDay          = 24
)

// AnalyticsRepository implements domain.AnalyticsRepository with MongoDB
// aggregation queries. Booked revenue is attributed to orders that reached a
// paid lifecycle state; for modest data volumes the simple distributions are
// computed by scanning orders while the heavier rankings use aggregation
// pipelines.
type AnalyticsRepository struct {
	db *mongo.Database
}

// NewAnalyticsRepository returns a repository bound to the store database.
func NewAnalyticsRepository(db *mongo.Database) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// EnsureIndexes is a no-op for the analytics aggregate repository because it
// reads from collections whose indexes are owned by other repositories.
func (r *AnalyticsRepository) EnsureIndexes(ctx context.Context) error {
	// Best-effort ensure the indexes that speed up analytics reads exist.
	_, err := r.db.Collection("orders").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "type", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	})
	if err != nil {
		return fmt.Errorf("create analytics order indexes: %w", err)
	}

	return nil
}

// paidStatuses are the order states that count as booked revenue.
func paidStatuses() map[string]bool {
	return map[string]bool{
		string(domain.OrderStatusBooked):       true,
		string(domain.OrderStatusInProduction): true,
		string(domain.OrderStatusReady):        true,
		string(domain.OrderStatusFulfilled):    true,
	}
}

// GetStoreAnalytics assembles the full dashboard snapshot.
func (r *AnalyticsRepository) GetStoreAnalytics(ctx context.Context) (*domain.StoreAnalytics, error) {
	waitlistCount, err := r.countSubscribers(ctx)
	if err != nil {
		return nil, err
	}

	customerCount, err := r.countCustomers(ctx)
	if err != nil {
		return nil, err
	}

	base, err := r.aggregateOrders(ctx)
	if err != nil {
		return nil, err
	}

	rankings, err := r.aggregateRankings(ctx)
	if err != nil {
		return nil, err
	}

	return assembleAnalytics(waitlistCount, customerCount, base, rankings), nil
}

func assembleAnalytics(
	waitlistCount, customerCount int64,
	base orderAggregate,
	rankings orderRankings,
) *domain.StoreAnalytics {
	return &domain.StoreAnalytics{
		WaitlistCount:            waitlistCount,
		CustomerCount:            customerCount,
		OrderCount:               base.bookedOrderCount,
		BookedRevenuePesewas:     base.revenue,
		AverageOrderValuePesewas: averageOrderValue(base.revenue, base.bookedOrderCount),
		OrdersByStatus:           base.ordersByStatus,
		OrdersByType:             base.ordersByType,
		RevenuePesewas:           base.revenue,
		CollectionViews:          0,
		Comparison:               base.comparison,
		RevenueSeries:            base.series,
		TopDesigns:               rankings.designs,
		TopCollections:           rankings.collections,
		RecentOrders:             base.recent,
	}
}

func averageOrderValue(revenue, count int64) int64 {
	if count == 0 {
		return 0
	}

	return revenue / count
}

func (r *AnalyticsRepository) countSubscribers(ctx context.Context) (int64, error) {
	count, err := r.db.Collection("subscribers").CountDocuments(ctx, bson.D{})
	if err != nil {
		return 0, fmt.Errorf("count subscribers: %w", err)
	}

	return count, nil
}

func (r *AnalyticsRepository) countCustomers(ctx context.Context) (int64, error) {
	count, err := r.db.Collection("users").CountDocuments(ctx, bson.M{"role": string(domain.RoleCustomer)})
	if err != nil {
		return 0, fmt.Errorf("count customers: %w", err)
	}

	return count, nil
}

// orderAggregate holds every metric derived from a single scan of the orders
// collection: status/type distributions, booked revenue and count, the trailing
// comparison, the revenue time series, and the recent-orders feed.
type orderAggregate struct {
	ordersByStatus   map[string]int64
	ordersByType     map[string]int64
	revenue          int64
	bookedOrderCount int64
	comparison       domain.PeriodComparison
	series           []domain.TimeBucket
	recent           []domain.RecentOrder
}

func (r *AnalyticsRepository) aggregateOrders(ctx context.Context) (orderAggregate, error) {
	docs, err := r.loadOrders(ctx)
	if err != nil {
		return orderAggregate{}, err
	}

	now := time.Now().UTC()
	agg := orderAggregate{
		ordersByStatus:   zeroStatusCounts(),
		ordersByType:     zeroTypeCounts(),
		revenue:          0,
		bookedOrderCount: 0,
		comparison:       domain.PeriodComparison{},
		series:           newRevenueSeries(now),
		recent:           recentOrders(docs),
	}
	paid := paidStatuses()

	for i := range docs {
		doc := docs[i]
		agg.ordersByStatus[doc.Status]++
		agg.ordersByType[doc.Type]++

		if !paid[doc.Status] {
			continue
		}

		total := doc.toDomain().TotalPesewas()
		agg.revenue += total
		agg.bookedOrderCount++

		addToSeries(agg.series, now, doc.CreatedAt, total)
		addToComparison(&agg.comparison, now, doc.CreatedAt, total)
	}

	finaliseComparison(&agg.comparison)

	return agg, nil
}

func (r *AnalyticsRepository) loadOrders(ctx context.Context) ([]orderDoc, error) {
	cur, err := r.db.Collection("orders").Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("find orders: %w", err)
	}

	var docs []orderDoc

	err = cur.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("decode orders: %w", err)
	}

	return docs, nil
}

// seriesBucketSpan is the duration covered by a single time-series bucket.
func seriesBucketSpan() time.Duration {
	return time.Duration(seriesBucketDays) * hoursPerDay * time.Hour
}

// seriesWindowStart anchors the trailing window so its final bucket ends at
// now: bucket i covers [windowStart + i*span, windowStart + (i+1)*span), and an
// order from the last span before now lands in the last bucket.
func seriesWindowStart(now time.Time) time.Time {
	return now.Add(-revenueSeriesBuckets * seriesBucketSpan())
}

// newRevenueSeries lays out the trailing buckets oldest-first, each labelled by
// its window start so the frontend can draw an axis without further maths.
func newRevenueSeries(now time.Time) []domain.TimeBucket {
	span := seriesBucketSpan()
	windowStart := seriesWindowStart(now)
	series := make([]domain.TimeBucket, revenueSeriesBuckets)

	for i := range revenueSeriesBuckets {
		start := windowStart.Add(time.Duration(i) * span)
		series[i] = domain.TimeBucket{
			Label:          start.Format("2 Jan"),
			StartAt:        start,
			RevenuePesewas: 0,
			OrderCount:     0,
		}
	}

	return series
}

func addToSeries(series []domain.TimeBucket, now, at time.Time, total int64) {
	if len(series) == 0 {
		return
	}

	windowStart := seriesWindowStart(now)
	if at.Before(windowStart) {
		return
	}

	idx := int(at.Sub(windowStart) / seriesBucketSpan())
	if idx < 0 || idx >= len(series) {
		return
	}

	series[idx].RevenuePesewas += total
	series[idx].OrderCount++
}

func addToComparison(cmp *domain.PeriodComparison, now, at time.Time, total int64) {
	window := time.Duration(comparisonWindowDays) * hoursPerDay * time.Hour
	currentStart := now.Add(-window)
	priorStart := now.Add(-2 * window)

	switch {
	case !at.Before(currentStart):
		cmp.CurrentRevenuePesewas += total
		cmp.CurrentOrderCount++
	case !at.Before(priorStart):
		cmp.PriorRevenuePesewas += total
		cmp.PriorOrderCount++
	}
}

func finaliseComparison(cmp *domain.PeriodComparison) {
	cmp.RevenueChangeBps = changeBps(cmp.CurrentRevenuePesewas, cmp.PriorRevenuePesewas)
	cmp.OrderCountChangeBps = changeBps(cmp.CurrentOrderCount, cmp.PriorOrderCount)
}

// changeBps returns the percent change of current over prior in basis points.
// With no prior baseline a positive current reads as a full +10000 bps (+100%)
// and a flat zero reads as no change.
func changeBps(current, prior int64) int64 {
	if prior == 0 {
		if current == 0 {
			return 0
		}

		return basisPointScale
	}

	return (current - prior) * basisPointScale / prior
}

func recentOrders(docs []orderDoc) []domain.RecentOrder {
	sorted := make([]orderDoc, len(docs))
	copy(sorted, docs)
	sortByCreatedAtDesc(sorted)

	limit := min(len(sorted), recentOrderLimit)
	out := make([]domain.RecentOrder, 0, limit)

	for i := range limit {
		doc := sorted[i]
		out = append(out, domain.RecentOrder{
			Ref:          doc.Ref,
			Type:         doc.Type,
			Status:       doc.Status,
			TotalPesewas: doc.toDomain().TotalPesewas(),
			CreatedAt:    doc.CreatedAt,
		})
	}

	return out
}

func sortByCreatedAtDesc(docs []orderDoc) {
	for i := 1; i < len(docs); i++ {
		for j := i; j > 0 && docs[j].CreatedAt.After(docs[j-1].CreatedAt); j-- {
			docs[j], docs[j-1] = docs[j-1], docs[j]
		}
	}
}

func zeroStatusCounts() map[string]int64 {
	return map[string]int64{
		string(domain.OrderStatusPendingPayment):  0,
		string(domain.OrderStatusRequested):       0,
		string(domain.OrderStatusQuoted):          0,
		string(domain.OrderStatusPaymentLinkSent): 0,
		string(domain.OrderStatusBooked):          0,
		string(domain.OrderStatusInProduction):    0,
		string(domain.OrderStatusReady):           0,
		string(domain.OrderStatusFulfilled):       0,
		string(domain.OrderStatusCancelled):       0,
	}
}

func zeroTypeCounts() map[string]int64 {
	return map[string]int64{
		string(domain.OrderTypeStandard):     0,
		string(domain.OrderTypeCustomSize):   0,
		string(domain.OrderTypeDesignChange): 0,
		string(domain.OrderTypeVisit):        0,
	}
}

// orderRankings carries the top-N designs and collections, both already sorted
// by the relevant booked metric and trimmed to topListLimit.
type orderRankings struct {
	designs     []domain.DesignStat
	collections []domain.CollectionStat
}

func (r *AnalyticsRepository) aggregateRankings(ctx context.Context) (orderRankings, error) {
	designs, err := r.topDesigns(ctx)
	if err != nil {
		return orderRankings{}, err
	}

	collections, err := r.topCollections(ctx)
	if err != nil {
		return orderRankings{}, err
	}

	return orderRankings{designs: designs, collections: collections}, nil
}

// designRankDoc is one row of the top-designs aggregation result.
type designRankDoc struct {
	ID      bson.ObjectID `bson:"_id"`
	Name    string        `bson:"name"`
	Count   int64         `bson:"count"`
	Revenue int64         `bson:"revenue"`
}

// topDesigns groups booked orders by design, summing the garment/quote total
// and counting orders, then keeps the busiest designs by order count.
func (r *AnalyticsRepository) topDesigns(ctx context.Context) ([]domain.DesignStat, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bookedMatch()}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$designId"},
			{Key: "name", Value: bson.D{{Key: "$first", Value: "$designSnapshot.name"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: bookedTotalExpr()}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "revenue", Value: -1}}}},
		bson.D{{Key: "$limit", Value: topListLimit}},
	}

	var rows []designRankDoc

	err := r.runPipeline(ctx, "orders", pipeline, &rows)
	if err != nil {
		return nil, fmt.Errorf("aggregate top designs: %w", err)
	}

	out := make([]domain.DesignStat, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.DesignStat{
			DesignID:       row.ID.Hex(),
			Name:           row.Name,
			OrderCount:     row.Count,
			RevenuePesewas: row.Revenue,
		})
	}

	return out, nil
}

// collectionRankDoc is one row of the top-collections aggregation result.
type collectionRankDoc struct {
	ID      bson.ObjectID `bson:"_id"`
	Name    string        `bson:"name"`
	Count   int64         `bson:"count"`
	Revenue int64         `bson:"revenue"`
}

// topCollections joins booked orders to their design, then to that design's
// collection, and ranks collections by booked revenue.
func (r *AnalyticsRepository) topCollections(ctx context.Context) ([]domain.CollectionStat, error) {
	var rows []collectionRankDoc

	err := r.runPipeline(ctx, "orders", topCollectionsPipeline(), &rows)
	if err != nil {
		return nil, fmt.Errorf("aggregate top collections: %w", err)
	}

	out := make([]domain.CollectionStat, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CollectionStat{
			CollectionID:   row.ID.Hex(),
			Name:           row.Name,
			OrderCount:     row.Count,
			RevenuePesewas: row.Revenue,
		})
	}

	return out, nil
}

func topCollectionsPipeline() mongo.Pipeline {
	return mongo.Pipeline{
		bson.D{{Key: "$match", Value: bookedMatch()}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "designs"},
			{Key: "localField", Value: "designId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "design"},
		}}},
		bson.D{{Key: "$unwind", Value: "$design"}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "collections"},
			{Key: "localField", Value: "design.collectionId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "collection"},
		}}},
		bson.D{{Key: "$unwind", Value: "$collection"}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$collection._id"},
			{Key: "name", Value: bson.D{{Key: "$first", Value: "$collection.name"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: bookedTotalExpr()}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "revenue", Value: -1}, {Key: "count", Value: -1}}}},
		bson.D{{Key: "$limit", Value: topListLimit}},
	}
}

// bookedMatch limits an aggregation to orders that count as booked revenue.
func bookedMatch() bson.D {
	statuses := bson.A{
		string(domain.OrderStatusBooked),
		string(domain.OrderStatusInProduction),
		string(domain.OrderStatusReady),
		string(domain.OrderStatusFulfilled),
	}

	return bson.D{{Key: "status", Value: bson.D{{Key: "$in", Value: statuses}}}}
}

// bookedTotalExpr mirrors domain.Order.TotalPesewas inside the aggregation: the
// quote price replaces the garment price when set, plus any delivery rate.
func bookedTotalExpr() bson.D {
	garment := bson.D{{Key: "$cond", Value: bson.A{
		bson.D{{Key: "$gt", Value: bson.A{"$quote.pricePesewas", 0}}},
		"$quote.pricePesewas",
		"$designSnapshot.pricePesewas",
	}}}

	delivery := bson.D{{Key: "$ifNull", Value: bson.A{"$delivery.ratePesewas", 0}}}

	return bson.D{{Key: "$add", Value: bson.A{garment, delivery}}}
}

func (r *AnalyticsRepository) runPipeline(
	ctx context.Context,
	collection string,
	pipeline mongo.Pipeline,
	out any,
) error {
	cur, err := r.db.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return fmt.Errorf("aggregate %s: %w", collection, err)
	}

	err = cur.All(ctx, out)
	if err != nil {
		return fmt.Errorf("decode %s aggregate: %w", collection, err)
	}

	return nil
}
