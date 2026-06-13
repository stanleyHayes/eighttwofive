package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

func paidOrder(status domain.OrderStatus) *domain.Order {
	now := time.Now().UTC()

	return &domain.Order{
		Ref:            "E25-TEST",
		DesignSnapshot: domain.DesignSnapshot{Name: "Blazer", PhotoPublicID: "", PricePesewas: 50000},
		Payments: []domain.Payment{{
			ProviderRef:   "ps-1",
			AmountPesewas: 50000,
			Status:        domain.PaymentStatusSuccess,
			Method:        "",
			PaidAt:        &now,
		}},
		Status: status,
	}
}

func unpaidOrder(status domain.OrderStatus) *domain.Order {
	return &domain.Order{
		Ref:            "E25-TEST",
		DesignSnapshot: domain.DesignSnapshot{Name: "Blazer", PhotoPublicID: "", PricePesewas: 50000},
		Status:         status,
	}
}

// TestOrderTransitionTable pins down the full state machine: only the listed
// edges are legal, even for fully paid orders.
func TestOrderTransitionTable(t *testing.T) {
	t.Parallel()

	allStatuses := []domain.OrderStatus{
		domain.OrderStatusPendingPayment,
		domain.OrderStatusRequested,
		domain.OrderStatusQuoted,
		domain.OrderStatusPaymentLinkSent,
		domain.OrderStatusBooked,
		domain.OrderStatusInProduction,
		domain.OrderStatusReady,
		domain.OrderStatusFulfilled,
		domain.OrderStatusCancelled,
	}

	allowed := map[domain.OrderStatus][]domain.OrderStatus{
		domain.OrderStatusPendingPayment: {domain.OrderStatusBooked, domain.OrderStatusCancelled},
		domain.OrderStatusRequested:      {domain.OrderStatusQuoted, domain.OrderStatusCancelled},
		domain.OrderStatusQuoted: {
			domain.OrderStatusPaymentLinkSent,
			domain.OrderStatusBooked,
			domain.OrderStatusCancelled,
		},
		domain.OrderStatusPaymentLinkSent: {domain.OrderStatusBooked, domain.OrderStatusCancelled},
		domain.OrderStatusBooked:          {domain.OrderStatusInProduction, domain.OrderStatusCancelled},
		domain.OrderStatusInProduction:    {domain.OrderStatusReady, domain.OrderStatusCancelled},
		domain.OrderStatusReady:           {domain.OrderStatusFulfilled, domain.OrderStatusCancelled},
		domain.OrderStatusFulfilled:       {},
		domain.OrderStatusCancelled:       {},
	}

	for _, from := range allStatuses {
		for _, target := range allStatuses {
			order := paidOrder(from)

			legal := false

			for _, edge := range allowed[from] {
				if edge == target {
					legal = true
				}
			}

			_, err := order.Transition(target, "test", time.Now().UTC())
			if legal {
				require.NoError(t, err, "%s -> %s must be allowed", from, target)
				assert.Equal(t, target, order.Status)
			} else {
				require.ErrorIs(t, err, domain.ErrInvalidInput, "%s -> %s must be rejected", from, target)
				assert.Equal(t, from, order.Status, "rejected transition must not change status")
			}
		}
	}
}

func TestOrderTransition_UnpaidNeverEntersProduction(t *testing.T) {
	t.Parallel()

	// Even from booked — the only state with an in_production edge — an
	// unpaid order must not enter production.
	order := unpaidOrder(domain.OrderStatusBooked)

	_, err := order.Transition(domain.OrderStatusInProduction, "merchant", time.Now().UTC())
	require.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Equal(t, domain.OrderStatusBooked, order.Status)
}

func TestOrderTransition_UnknownStatusRejected(t *testing.T) {
	t.Parallel()

	order := paidOrder(domain.OrderStatusBooked)

	_, err := order.Transition(domain.OrderStatus("paid_lol"), "merchant", time.Now().UTC())
	require.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Equal(t, domain.OrderStatusBooked, order.Status)
	assert.Empty(t, order.StatusHistory, "rejected transitions must not pollute the audit trail")
}

func TestKnownOrderStatus(t *testing.T) {
	t.Parallel()

	assert.True(t, domain.KnownOrderStatus(domain.OrderStatusBooked))
	assert.True(t, domain.KnownOrderStatus(domain.OrderStatusCancelled))
	assert.False(t, domain.KnownOrderStatus(domain.OrderStatus("paid_lol")))
	assert.False(t, domain.KnownOrderStatus(domain.OrderStatus("")))
}

func TestMarkPaid_PreservesExpectedAmount(t *testing.T) {
	t.Parallel()

	order := unpaidOrder(domain.OrderStatusPendingPayment)
	order.Payments = []domain.Payment{{
		ProviderRef:   "E25-TEST",
		AmountPesewas: 50000,
		Status:        domain.PaymentStatusPending,
		Method:        "",
		PaidAt:        nil,
	}}

	prev, err := order.MarkPaid(domain.Payment{
		ProviderRef:   "E25-TEST",
		AmountPesewas: 49999,
	}, "payment_webhook", time.Now().UTC())
	require.NoError(t, err)

	assert.Equal(t, domain.OrderStatusPendingPayment, prev)
	assert.Equal(t, domain.OrderStatusBooked, order.Status)
	require.Len(t, order.Payments, 1)
	assert.Equal(t, int64(50000), order.Payments[0].AmountPesewas,
		"the originally recorded amount must never be overwritten by the webhook payload")
	assert.Equal(t, domain.PaymentStatusSuccess, order.Payments[0].Status)
	require.NotNil(t, order.Payments[0].PaidAt)

	lastChange := order.StatusHistory[len(order.StatusHistory)-1]
	assert.Equal(t, "payment_webhook", lastChange.By)
}
