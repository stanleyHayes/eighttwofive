package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// StoreSettings exposes the merchant-editable store settings (scope §05).
type StoreSettings struct {
	repo domain.SettingsRepository
}

// NewStoreSettings wires the settings service.
func NewStoreSettings(repo domain.SettingsRepository) *StoreSettings {
	return &StoreSettings{repo: repo}
}

// Get returns the current settings (defaults if never saved).
func (s *StoreSettings) Get(ctx context.Context) (*domain.Settings, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	return settings, nil
}

// Update validates and persists new settings.
func (s *StoreSettings) Update(ctx context.Context, settings *domain.Settings) error {
	if settings.DepositPesewas < 0 {
		return fmt.Errorf("%w: deposit cannot be negative", domain.ErrInvalidInput)
	}

	settings.WhatsAppNumber = strings.TrimSpace(settings.WhatsAppNumber)
	settings.VisitLocation = strings.TrimSpace(settings.VisitLocation)
	settings.ContactEmail = strings.TrimSpace(settings.ContactEmail)
	settings.InstagramHandle = normalizeInstagramHandle(settings.InstagramHandle)

	seen := make(map[string]struct{}, len(settings.DeliveryRates))
	for _, rate := range settings.DeliveryRates {
		area := strings.TrimSpace(rate.Area)
		if area == "" {
			return fmt.Errorf("%w: delivery area cannot be empty", domain.ErrInvalidInput)
		}

		if rate.RatePesewas < 0 {
			return fmt.Errorf("%w: delivery rate cannot be negative", domain.ErrInvalidInput)
		}

		if _, exists := seen[area]; exists {
			return fmt.Errorf("%w: %s", domain.ErrDuplicateArea, area)
		}

		seen[area] = struct{}{}
	}

	err := s.repo.Update(ctx, settings)
	if err != nil {
		return fmt.Errorf("update settings: %w", err)
	}

	return nil
}

// normalizeInstagramHandle reduces whatever the merchant pastes — a bare handle,
// an @handle, or a full profile URL — to the bare handle, so the storefront can
// build a clean profile link from it.
func normalizeInstagramHandle(raw string) string {
	handle := strings.TrimSpace(raw)
	if handle == "" {
		return ""
	}

	if i := strings.Index(handle, "instagram.com/"); i >= 0 {
		handle = handle[i+len("instagram.com/"):]
	}

	handle = strings.TrimPrefix(strings.TrimSpace(handle), "@")

	// Keep only the first path segment, dropping any trailing slash/query/hash.
	if i := strings.IndexAny(handle, "/?#"); i >= 0 {
		handle = handle[:i]
	}

	return handle
}
