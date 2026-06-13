package mongostore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

type designSnapshotDoc struct {
	Name          string `bson:"name"`
	PhotoPublicID string `bson:"photoPublicId"`
	PricePesewas  int64  `bson:"pricePesewas"`
}

type customisationDoc struct {
	SizeMode     string            `bson:"sizeMode"`
	BandLabel    string            `bson:"bandLabel"`
	Measurements map[string]string `bson:"measurements"`
	DesignChange string            `bson:"designChange"`
}

type quoteDoc struct {
	PricePesewas int64  `bson:"pricePesewas"`
	Timeline     string `bson:"timeline"`
	Notes        string `bson:"notes"`
}

type deliveryDoc struct {
	Mode        string `bson:"mode"`
	Area        string `bson:"area"`
	RatePesewas *int64 `bson:"ratePesewas,omitempty"`
}

type paymentDoc struct {
	ProviderRef   string     `bson:"providerRef"`
	AmountPesewas int64      `bson:"amountPesewas"`
	Status        string     `bson:"status"`
	Method        string     `bson:"method"`
	PaidAt        *time.Time `bson:"paidAt,omitempty"`
}

type statusChangeDoc struct {
	Status string    `bson:"status"`
	At     time.Time `bson:"at"`
	By     string    `bson:"by"`
}

type orderDoc struct {
	ID             bson.ObjectID     `bson:"_id,omitempty"`
	Ref            string            `bson:"ref"`
	CustomerID     bson.ObjectID     `bson:"customerId"`
	DesignID       bson.ObjectID     `bson:"designId"`
	DesignSnapshot designSnapshotDoc `bson:"designSnapshot"`
	Type           string            `bson:"type"`
	Customisation  customisationDoc  `bson:"customisation"`
	Quote          quoteDoc          `bson:"quote"`
	Delivery       deliveryDoc       `bson:"delivery"`
	Payments       []paymentDoc      `bson:"payments"`
	Status         string            `bson:"status"`
	StatusHistory  []statusChangeDoc `bson:"statusHistory"`
	CustomerPhone  string            `bson:"customerPhone"`
	Version        int64             `bson:"version"`
	CreatedAt      time.Time         `bson:"createdAt"`
	UpdatedAt      time.Time         `bson:"updatedAt"`
}

func (d orderDoc) toDomain() *domain.Order {
	payments := make([]domain.Payment, 0, len(d.Payments))
	for _, p := range d.Payments {
		payments = append(payments, domain.Payment{
			ProviderRef:   p.ProviderRef,
			AmountPesewas: p.AmountPesewas,
			Status:        p.Status,
			Method:        p.Method,
			PaidAt:        p.PaidAt,
		})
	}

	history := make([]domain.StatusChange, 0, len(d.StatusHistory))
	for _, h := range d.StatusHistory {
		history = append(history, domain.StatusChange{
			Status: domain.OrderStatus(h.Status),
			At:     h.At,
			By:     h.By,
		})
	}

	return &domain.Order{
		ID:         d.ID.Hex(),
		Ref:        d.Ref,
		CustomerID: d.CustomerID.Hex(),
		DesignID:   d.DesignID.Hex(),
		DesignSnapshot: domain.DesignSnapshot{
			Name:          d.DesignSnapshot.Name,
			PhotoPublicID: d.DesignSnapshot.PhotoPublicID,
			PricePesewas:  d.DesignSnapshot.PricePesewas,
		},
		Type: domain.OrderType(d.Type),
		Customisation: domain.Customisation{
			SizeMode:     d.Customisation.SizeMode,
			BandLabel:    d.Customisation.BandLabel,
			Measurements: d.Customisation.Measurements,
			DesignChange: d.Customisation.DesignChange,
		},
		Quote: domain.Quote{
			PricePesewas: d.Quote.PricePesewas,
			Timeline:     d.Quote.Timeline,
			Notes:        d.Quote.Notes,
		},
		Delivery: domain.Delivery{
			Mode:        d.Delivery.Mode,
			Area:        d.Delivery.Area,
			RatePesewas: d.Delivery.RatePesewas,
		},
		Payments:      payments,
		Status:        domain.OrderStatus(d.Status),
		StatusHistory: history,
		CustomerPhone: d.CustomerPhone,
		Version:       d.Version,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

//nolint:funlen // Mapping an order to its Mongo document is inherently verbose.
func orderToDoc(o *domain.Order) (*orderDoc, error) {
	customerID, err := bson.ObjectIDFromHex(o.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("%w: bad customer id", domain.ErrInvalidInput)
	}

	designID, err := bson.ObjectIDFromHex(o.DesignID)
	if err != nil {
		return nil, fmt.Errorf("%w: bad design id", domain.ErrInvalidInput)
	}

	payments := make([]paymentDoc, 0, len(o.Payments))
	for _, p := range o.Payments {
		payments = append(payments, paymentDoc{
			ProviderRef:   p.ProviderRef,
			AmountPesewas: p.AmountPesewas,
			Status:        p.Status,
			Method:        p.Method,
			PaidAt:        p.PaidAt,
		})
	}

	history := make([]statusChangeDoc, 0, len(o.StatusHistory))
	for _, h := range o.StatusHistory {
		history = append(history, statusChangeDoc{
			Status: string(h.Status),
			At:     h.At,
			By:     h.By,
		})
	}

	var rate *int64

	if o.Delivery.RatePesewas != nil {
		r := *o.Delivery.RatePesewas
		rate = &r
	}

	return &orderDoc{
		ID:         bson.ObjectID{},
		Ref:        o.Ref,
		CustomerID: customerID,
		DesignID:   designID,
		DesignSnapshot: designSnapshotDoc{
			Name:          o.DesignSnapshot.Name,
			PhotoPublicID: o.DesignSnapshot.PhotoPublicID,
			PricePesewas:  o.DesignSnapshot.PricePesewas,
		},
		Type: string(o.Type),
		Customisation: customisationDoc{
			SizeMode:     o.Customisation.SizeMode,
			BandLabel:    o.Customisation.BandLabel,
			Measurements: o.Customisation.Measurements,
			DesignChange: o.Customisation.DesignChange,
		},
		Quote: quoteDoc{
			PricePesewas: o.Quote.PricePesewas,
			Timeline:     o.Quote.Timeline,
			Notes:        o.Quote.Notes,
		},
		Delivery: deliveryDoc{
			Mode:        o.Delivery.Mode,
			Area:        o.Delivery.Area,
			RatePesewas: rate,
		},
		Payments:      payments,
		Status:        string(o.Status),
		StatusHistory: history,
		CustomerPhone: o.CustomerPhone,
		Version:       o.Version,
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
	}, nil
}

// OrderRepository implements domain.OrderRepository on MongoDB.
type OrderRepository struct {
	col *mongo.Collection
}

// NewOrderRepository returns a repository over the orders collection.
func NewOrderRepository(db *mongo.Database) *OrderRepository {
	return &OrderRepository{col: db.Collection("orders")}
}

// EnsureIndexes creates the required order indexes.
func (r *OrderRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "ref", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "customerId", Value: 1}},
			Options: nil,
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: nil,
		},
		{
			Keys:    bson.D{{Key: "type", Value: 1}},
			Options: nil,
		},
		{
			Keys:    bson.D{{Key: "ref", Value: "text"}, {Key: "customerPhone", Value: "text"}},
			Options: nil,
		},
	})
	if err != nil {
		return fmt.Errorf("create order indexes: %w", err)
	}

	return nil
}

// Create inserts a new order; a duplicate ref maps to ErrDuplicateRef.
func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) error {
	doc, err := orderToDoc(o)
	if err != nil {
		return err
	}

	res, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrDuplicateRef
		}

		return fmt.Errorf("insert order: %w", err)
	}

	if id, ok := res.InsertedID.(bson.ObjectID); ok {
		o.ID = id.Hex()
	}

	return nil
}

// Update replaces an existing order, guarded by an optimistic version check:
// the write only matches when the stored version equals the version the caller
// loaded, so concurrent read-modify-write cycles surface as ErrConflict
// instead of silently overwriting each other.
func (r *OrderRepository) Update(ctx context.Context, o *domain.Order) error {
	doc, err := orderToDoc(o)
	if err != nil {
		return err
	}

	objectID, err := bson.ObjectIDFromHex(o.ID)
	if err != nil {
		return domain.ErrNotFound
	}

	doc.ID = objectID
	doc.Version = o.Version + 1

	res, err := r.col.ReplaceOne(ctx, orderVersionFilter(objectID, o.Version), doc)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	if res.MatchedCount == 0 {
		return domain.ErrConflict
	}

	o.Version = doc.Version

	return nil
}

// orderVersionFilter matches the order at the expected version. Documents
// written before versioning existed have no version field and read as zero.
func orderVersionFilter(id bson.ObjectID, version int64) bson.M {
	if version == 0 {
		return bson.M{
			"_id": id,
			"$or": []bson.M{
				{"version": 0},
				{"version": bson.M{"$exists": false}},
			},
		}
	}

	return bson.M{"_id": id, "version": version}
}

// GetByID loads an order by its hex id.
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return r.getOne(ctx, bson.M{"_id": objectID})
}

// GetByRef loads an order by its human-readable reference.
func (r *OrderRepository) GetByRef(ctx context.Context, ref string) (*domain.Order, error) {
	return r.getOne(ctx, bson.M{"ref": ref})
}

// ListByCustomer returns a customer's orders newest first.
func (r *OrderRepository) ListByCustomer(ctx context.Context, customerID string) ([]domain.Order, error) {
	objectID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	return r.list(ctx, bson.M{"customerId": objectID}, bson.D{{Key: "createdAt", Value: -1}}, nil)
}

// List returns orders matching the filter, newest first within each type.
func (r *OrderRepository) List(ctx context.Context, filter domain.OrderFilter) ([]domain.Order, error) {
	return r.list(ctx, orderQuery(filter), adminOrderSort(), nil)
}

// Count returns the number of orders matching the filter.
func (r *OrderRepository) Count(ctx context.Context, filter domain.OrderFilter) (int64, error) {
	total, err := r.col.CountDocuments(ctx, orderQuery(filter))
	if err != nil {
		return 0, fmt.Errorf("count orders: %w", err)
	}

	return total, nil
}

// ListPaged returns one page of orders matching the filter, newest first.
func (r *OrderRepository) ListPaged(
	ctx context.Context, filter domain.OrderFilter, params domain.PageParams,
) ([]domain.Order, error) {
	return r.list(ctx, orderQuery(filter), adminOrderSort(), &params)
}

// orderQuery builds the Mongo filter for an admin order listing.
func orderQuery(filter domain.OrderFilter) bson.M {
	query := bson.M{}

	if filter.Status != "" {
		query["status"] = string(filter.Status)
	}

	if filter.Type != "" {
		query["type"] = string(filter.Type)
	}

	return query
}

// adminOrderSort groups the inbox by type, newest first within each type, then
// by id so paging boundaries are stable even when createdAt values collide.
func adminOrderSort() bson.D {
	return bson.D{
		{Key: "type", Value: 1},
		{Key: "createdAt", Value: -1},
		{Key: "_id", Value: -1},
	}
}

func (r *OrderRepository) list(
	ctx context.Context,
	query bson.M,
	sort bson.D,
	page *domain.PageParams,
) ([]domain.Order, error) {
	opts := options.Find().SetSort(sort)
	if page != nil {
		opts.SetSkip(page.Skip()).SetLimit(page.Limit())
	}

	cur, err := r.col.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("find orders: %w", err)
	}

	var docs []orderDoc

	err = cur.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("decode orders: %w", err)
	}

	orders := make([]domain.Order, 0, len(docs))
	for _, doc := range docs {
		orders = append(orders, *doc.toDomain())
	}

	return orders, nil
}

func (r *OrderRepository) getOne(ctx context.Context, filter bson.M) (*domain.Order, error) {
	var doc orderDoc

	err := r.col.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find order: %w", err)
	}

	return doc.toDomain(), nil
}
