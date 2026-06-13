package service

import (
	"context"
	"fmt"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// Analytics implements the admin analytics use-case.
type Analytics struct {
	repo domain.AnalyticsRepository
}

// NewAnalytics wires the analytics service.
func NewAnalytics(repo domain.AnalyticsRepository) *Analytics {
	return &Analytics{repo: repo}
}

// GetStoreAnalytics returns the current aggregate snapshot.
func (s *Analytics) GetStoreAnalytics(ctx context.Context) (*domain.StoreAnalytics, error) {
	analytics, err := s.repo.GetStoreAnalytics(ctx)
	if err != nil {
		return nil, fmt.Errorf("load store analytics: %w", err)
	}

	return analytics, nil
}
