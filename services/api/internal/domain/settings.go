package domain

import (
	"context"
	"errors"
)

// defaultDepositPesewas is GHS 500 — the home-visit deposit the scope names
// as the starting value (§4.3, §09); the merchant can change it later (§05).
const defaultDepositPesewas = 500_00

// ErrDuplicateArea is returned when a delivery-rate area is listed more than
// once in the settings table.
var ErrDuplicateArea = errors.New("duplicate delivery area")

// DeliveryRate is a dispatch fee for one geographical area (scope §4.6).
type DeliveryRate struct {
	Area        string
	RatePesewas int64
}

// Settings are the merchant-editable store settings (scope §05).
type Settings struct {
	DepositPesewas int64
	WhatsAppNumber string
	VisitLocation  string
	DeliveryRates  []DeliveryRate
}

// DefaultSettings returns the store settings before the merchant has saved any.
func DefaultSettings() *Settings {
	return &Settings{
		DepositPesewas: defaultDepositPesewas,
		WhatsAppNumber: "",
		VisitLocation:  "",
		DeliveryRates:  []DeliveryRate{},
	}
}

// SettingsRepository is the persistence port for store settings.
type SettingsRepository interface {
	Get(ctx context.Context) (*Settings, error)
	Update(ctx context.Context, s *Settings) error
}
