package mongostore

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

const settingsDocID = "store"

type deliveryRateDoc struct {
	Area        string `bson:"area"`
	RatePesewas int64  `bson:"ratePesewas"`
}

type settingsDoc struct {
	ID              string            `bson:"_id"`
	DepositPesewas  int64             `bson:"depositPesewas"`
	WhatsAppNumber  string            `bson:"whatsappNumber"`
	VisitLocation   string            `bson:"visitLocation"`
	InstagramHandle string            `bson:"instagramHandle"`
	DeliveryRates   []deliveryRateDoc `bson:"deliveryRates"`
}

// SettingsRepository implements domain.SettingsRepository on MongoDB as a
// single document.
type SettingsRepository struct {
	col *mongo.Collection
}

// NewSettingsRepository returns a repository over the settings collection.
func NewSettingsRepository(db *mongo.Database) *SettingsRepository {
	return &SettingsRepository{col: db.Collection("settings")}
}

// Get returns the saved settings, or the domain defaults when never saved.
func (r *SettingsRepository) Get(ctx context.Context) (*domain.Settings, error) {
	var doc settingsDoc

	err := r.col.FindOne(ctx, bson.M{"_id": settingsDocID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.DefaultSettings(), nil
	}

	if err != nil {
		return nil, fmt.Errorf("find settings: %w", err)
	}

	rates := make([]domain.DeliveryRate, 0, len(doc.DeliveryRates))
	for _, rate := range doc.DeliveryRates {
		rates = append(rates, domain.DeliveryRate{Area: rate.Area, RatePesewas: rate.RatePesewas})
	}

	return &domain.Settings{
		DepositPesewas:  doc.DepositPesewas,
		WhatsAppNumber:  doc.WhatsAppNumber,
		VisitLocation:   doc.VisitLocation,
		InstagramHandle: doc.InstagramHandle,
		DeliveryRates:   rates,
	}, nil
}

// Update saves the settings, creating the document on first save.
func (r *SettingsRepository) Update(ctx context.Context, s *domain.Settings) error {
	rates := make([]deliveryRateDoc, 0, len(s.DeliveryRates))
	for _, rate := range s.DeliveryRates {
		rates = append(rates, deliveryRateDoc{Area: rate.Area, RatePesewas: rate.RatePesewas})
	}

	doc := settingsDoc{
		ID:              settingsDocID,
		DepositPesewas:  s.DepositPesewas,
		WhatsAppNumber:  s.WhatsAppNumber,
		VisitLocation:   s.VisitLocation,
		InstagramHandle: s.InstagramHandle,
		DeliveryRates:   rates,
	}

	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": settingsDocID}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("replace settings: %w", err)
	}

	return nil
}
