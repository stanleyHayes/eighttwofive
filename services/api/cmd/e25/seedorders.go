package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/mongostore"
	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

var errNoDesignsToSeed = errors.New("no designs with size bands to seed orders against (run `e25 seed` first)")

const (
	seedOrderCount   = 26
	seedDaySpacing   = 79 * time.Hour // ~3.3 days between orders => ~12 weeks of history
	seedDeliveryRate = 30_00          // GHS 30 dispatch
	seedCustomerMail = "seed-shopper@example.com"
)

// orderStatusCycle is cycled across seeded orders so the dashboard shows a
// spread of production stages.
func orderStatusCycle() []domain.OrderStatus {
	return []domain.OrderStatus{
		domain.OrderStatusFulfilled, domain.OrderStatusFulfilled, domain.OrderStatusBooked,
		domain.OrderStatusInProduction, domain.OrderStatusReady, domain.OrderStatusFulfilled,
		domain.OrderStatusBooked,
	}
}

func newSeedOrdersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "seed-orders",
		Short: "Insert demo booked orders so the analytics dashboard has data",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadEnvironment()
			if err != nil {
				return err
			}

			return withDatabase(cmd.Context(), cfg, func(db *mongo.Database) error {
				return runSeedOrders(cmd.Context(), cmd.OutOrStdout(), db)
			})
		},
	}
}

func runSeedOrders(ctx context.Context, out io.Writer, db *mongo.Database) error {
	orders := mongostore.NewOrderRepository(db)

	err := orders.EnsureIndexes(ctx)
	if err != nil {
		return fmt.Errorf("ensure order indexes: %w", err)
	}

	customerID, err := seedCustomer(ctx, db)
	if err != nil {
		return err
	}

	designs, err := mongostore.NewDesignRepository(db).List(ctx, domain.DesignFilter{IncludeRetired: true})
	if err != nil {
		return fmt.Errorf("list designs: %w", err)
	}

	priced := make([]domain.Design, 0, len(designs))

	for _, design := range designs {
		if len(design.SizeBands) > 0 {
			priced = append(priced, design)
		}
	}

	if len(priced) == 0 {
		return errNoDesignsToSeed
	}

	now := time.Now().UTC()
	created := 0

	for i := range seedOrderCount {
		order := buildSeedOrder(i, customerID, priced[i%len(priced)], now)

		err = orders.Create(ctx, order)
		if err != nil {
			return fmt.Errorf("create seed order %s: %w", order.Ref, err)
		}

		created++
	}

	_, _ = fmt.Fprintf(out, "seeded %d booked orders across %d designs\n", created, len(priced))

	return nil
}

func seedCustomer(ctx context.Context, db *mongo.Database) (string, error) {
	users := mongostore.NewUserRepository(db)

	err := users.EnsureIndexes(ctx)
	if err != nil {
		return "", fmt.Errorf("ensure user indexes: %w", err)
	}

	customer := &domain.User{
		Email:     seedCustomerMail,
		Name:      "Seed Shopper",
		Role:      domain.RoleCustomer,
		CreatedAt: time.Now().UTC(),
	}

	err = users.Upsert(ctx, customer)
	if err != nil {
		return "", fmt.Errorf("upsert seed customer: %w", err)
	}

	return customer.ID, nil
}

func buildSeedOrder(index int, customerID string, design domain.Design, now time.Time) *domain.Order {
	price := design.SizeBands[0].PricePesewas
	rate := int64(seedDeliveryRate)
	createdAt := now.Add(-time.Duration(index) * seedDaySpacing)
	cycle := orderStatusCycle()
	status := cycle[index%len(cycle)]
	orderType := seedOrderType(index)

	photo := ""
	if len(design.Photos) > 0 {
		photo = design.Photos[0].PublicID
	}

	return &domain.Order{
		Ref:        fmt.Sprintf("E25-S%04d", index+1),
		CustomerID: customerID,
		DesignID:   design.ID,
		DesignSnapshot: domain.DesignSnapshot{
			Name:          design.Name,
			PhotoPublicID: photo,
			PricePesewas:  price,
		},
		Type: orderType,
		Customisation: domain.Customisation{
			SizeMode:  "band",
			BandLabel: design.SizeBands[0].Label,
		},
		Delivery: domain.Delivery{Mode: "dispatch", Area: "Accra", RatePesewas: &rate},
		Payments: []domain.Payment{{
			ProviderRef:   fmt.Sprintf("seed-%04d", index+1),
			AmountPesewas: price + rate,
			Status:        domain.PaymentStatusSuccess,
			Method:        "mobile_money",
			PaidAt:        &createdAt,
		}},
		Status:        status,
		StatusHistory: []domain.StatusChange{{Status: status, At: createdAt, By: "seed"}},
		CustomerPhone: "+233200000000",
		Version:       1,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
}

func seedOrderType(index int) domain.OrderType {
	switch {
	case index%8 == 7:
		return domain.OrderTypeVisit
	case index%6 == 5:
		return domain.OrderTypeCustomSize
	default:
		return domain.OrderTypeStandard
	}
}
