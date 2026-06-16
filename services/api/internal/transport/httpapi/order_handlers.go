package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// --- DTOs --------------------------------------------------------------------

type orderDesignSnapshotDTO struct {
	Name          string `json:"name"`
	PhotoPublicID string `json:"photoPublicId"`
	PricePesewas  int64  `json:"pricePesewas"`
}

type orderCustomisationDTO struct {
	SizeMode     string            `json:"sizeMode"`
	BandLabel    string            `json:"bandLabel,omitempty"`
	Measurements map[string]string `json:"measurements,omitempty"`
	DesignChange string            `json:"designChange,omitempty"`
}

type orderQuoteDTO struct {
	PricePesewas int64  `json:"pricePesewas"`
	Timeline     string `json:"timeline"`
	Notes        string `json:"notes"`
}

type orderDeliveryDTO struct {
	Mode        string `json:"mode"`
	Area        string `json:"area,omitempty"`
	RatePesewas *int64 `json:"ratePesewas,omitempty"`
}

type orderPaymentDTO struct {
	ProviderRef   string     `json:"providerRef"`
	AmountPesewas int64      `json:"amountPesewas"`
	Status        string     `json:"status"`
	Method        string     `json:"method"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
}

type statusChangeDTO struct {
	Status string    `json:"status"`
	At     time.Time `json:"at"`
	By     string    `json:"by"`
}

type orderDTO struct {
	ID             string                 `json:"id"`
	Ref            string                 `json:"ref"`
	CustomerID     string                 `json:"customerId"`
	DesignID       string                 `json:"designId"`
	DesignSnapshot orderDesignSnapshotDTO `json:"designSnapshot"`
	Type           string                 `json:"type"`
	Customisation  orderCustomisationDTO  `json:"customisation"`
	Quote          orderQuoteDTO          `json:"quote"`
	Delivery       orderDeliveryDTO       `json:"delivery"`
	Payments       []orderPaymentDTO      `json:"payments"`
	Status         string                 `json:"status"`
	StatusHistory  []statusChangeDTO      `json:"statusHistory"`
	CustomerPhone  string                 `json:"customerPhone"`
	TotalPesewas   int64                  `json:"totalPesewas"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

func toOrderDTO(o *domain.Order) orderDTO {
	payments := make([]orderPaymentDTO, 0, len(o.Payments))
	for _, p := range o.Payments {
		payments = append(payments, orderPaymentDTO{
			ProviderRef:   p.ProviderRef,
			AmountPesewas: p.AmountPesewas,
			Status:        p.Status,
			Method:        p.Method,
			PaidAt:        p.PaidAt,
		})
	}

	history := make([]statusChangeDTO, 0, len(o.StatusHistory))
	for _, h := range o.StatusHistory {
		history = append(history, statusChangeDTO{
			Status: string(h.Status),
			At:     h.At,
			By:     h.By,
		})
	}

	return orderDTO{
		ID:         o.ID,
		Ref:        o.Ref,
		CustomerID: o.CustomerID,
		DesignID:   o.DesignID,
		DesignSnapshot: orderDesignSnapshotDTO{
			Name:          o.DesignSnapshot.Name,
			PhotoPublicID: o.DesignSnapshot.PhotoPublicID,
			PricePesewas:  o.DesignSnapshot.PricePesewas,
		},
		Type: string(o.Type),
		Customisation: orderCustomisationDTO{
			SizeMode:     o.Customisation.SizeMode,
			BandLabel:    o.Customisation.BandLabel,
			Measurements: o.Customisation.Measurements,
			DesignChange: o.Customisation.DesignChange,
		},
		Quote: orderQuoteDTO{
			PricePesewas: o.Quote.PricePesewas,
			Timeline:     o.Quote.Timeline,
			Notes:        o.Quote.Notes,
		},
		Delivery: orderDeliveryDTO{
			Mode:        o.Delivery.Mode,
			Area:        o.Delivery.Area,
			RatePesewas: o.Delivery.RatePesewas,
		},
		Payments:      payments,
		Status:        string(o.Status),
		StatusHistory: history,
		CustomerPhone: o.CustomerPhone,
		TotalPesewas:  o.TotalPesewas(),
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
	}
}

func toOrderDTOs(orders []domain.Order) []orderDTO {
	dtos := make([]orderDTO, 0, len(orders))
	for i := range orders {
		dtos = append(dtos, toOrderDTO(&orders[i]))
	}

	return dtos
}

type createOrderRequest struct {
	DesignID      string `json:"designId"`
	BandLabel     string `json:"bandLabel"`
	Delivery      string `json:"delivery"`
	CustomerPhone string `json:"customerPhone"`
	Email         string `json:"email"`
	Name          string `json:"name"`
}

type createOrderResponse struct {
	Order      orderDTO `json:"order"`
	PaymentURL string   `json:"paymentUrl"`
}

type createCustomRequest struct {
	DesignID      string            `json:"designId"`
	SizeMode      string            `json:"sizeMode"`
	Measurements  map[string]string `json:"measurements,omitempty"`
	BandLabel     string            `json:"bandLabel,omitempty"`
	DesignChange  string            `json:"designChange,omitempty"`
	Delivery      string            `json:"delivery"`
	CustomerPhone string            `json:"customerPhone"`
	Email         string            `json:"email"`
	Name          string            `json:"name"`
}

type createCustomResponse struct {
	Order orderDTO `json:"order"`
}

// CreateOrder handles POST /api/v1/orders. It creates a light account for the
// customer, initializes payment, and sets the session cookie so the checkout
// flow is authenticated.
func (h *Handlers) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if !decodeBody(w, r, &req) {
		return
	}

	result, err := h.orders.CreateStandardOrder(
		r.Context(),
		req.DesignID,
		req.BandLabel,
		req.Delivery,
		req.CustomerPhone,
		req.Email,
		req.Name,
	)

	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())

		return
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found", "design not found")

		return
	case err != nil:
		respondInternal(w, r, err)

		return
	}

	// Checkout is anonymous: we never mint a session from an unauthenticated,
	// body-supplied email (that would let anyone read another customer's orders
	// by submitting their address). Customers reach their orders via the
	// single-use magic link emailed to them.
	respondJSON(w, http.StatusCreated, createOrderResponse{
		Order:      toOrderDTO(result.Order),
		PaymentURL: result.PaymentURL,
	})
}

// CreateCustomRequest handles POST /api/v1/orders/request. It creates a
// requested order for a custom size or design change, upserts the customer,
// and sets the session cookie.
func (h *Handlers) CreateCustomRequest(w http.ResponseWriter, r *http.Request) {
	var req createCustomRequest
	if !decodeBody(w, r, &req) {
		return
	}

	order, err := h.orders.CreateCustomRequest(
		r.Context(),
		req.DesignID,
		domain.Customisation{
			SizeMode:     req.SizeMode,
			Measurements: req.Measurements,
			BandLabel:    req.BandLabel,
			DesignChange: req.DesignChange,
		},
		req.Delivery,
		req.CustomerPhone,
		req.Email,
		req.Name,
	)

	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())

		return
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found", "design not found")

		return
	case err != nil:
		respondInternal(w, r, err)

		return
	}

	// Anonymous checkout — no session minted from a body-supplied email.
	respondJSON(w, http.StatusCreated, createCustomResponse{Order: toOrderDTO(order)})
}

// GetOrder handles GET /api/v1/orders/{ref}. Customers see their own orders;
// admins see any.
func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "sign in to continue")

		return
	}

	ref := chi.URLParam(r, "ref")

	order, err := h.orders.GetOrder(r.Context(), ref)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "not_found", "not found")

			return
		}

		respondInternal(w, r, err)

		return
	}

	if user.Role != domain.RoleAdmin && order.CustomerID != user.ID {
		respondError(w, http.StatusForbidden, "forbidden", "not your order")

		return
	}

	respondJSON(w, http.StatusOK, toOrderDTO(order))
}

// ListCustomerOrders handles GET /api/v1/orders.
func (h *Handlers) ListCustomerOrders(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "sign in to continue")

		return
	}

	orders, err := h.orders.ListCustomerOrders(r.Context(), user.ID)
	if err != nil {
		respondInternal(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, toOrderDTOs(orders))
}

// HandlePaymentWebhook handles POST /api/v1/payments/webhook from Paystack.
func (h *Handlers) HandlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	signature := r.Header.Get("X-Paystack-Signature")
	if signature == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "missing signature")

		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "cannot read body")

		return
	}

	// Validate JSON shape without altering the raw bytes used for signature verification.
	if !json.Valid(raw) {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid JSON")

		return
	}

	err = h.orders.HandlePaymentWebhook(r.Context(), raw, signature)
	if err != nil {
		if errors.Is(err, domain.ErrWebhookInvalid) {
			respondError(w, http.StatusUnauthorized, "unauthorized", "invalid webhook")

			return
		}

		respondInternal(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// AdminListOrders handles GET /api/v1/admin/orders, paginated (page, pageSize),
// sorted for the inbox.
func (h *Handlers) AdminListOrders(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pageQuery(r)

	result, err := h.orders.ListAdminOrdersPaged(r.Context(), page, pageSize)
	if err != nil {
		respondInternal(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, pagedDTO[orderDTO]{
		Items:    toOrderDTOs(result.Items),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

// AdminGetOrder handles GET /api/v1/admin/orders/{ref}.
func (h *Handlers) AdminGetOrder(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	order, err := h.orders.GetAdminOrder(r.Context(), ref)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "not_found", "not found")

			return
		}

		respondInternal(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, toOrderDTO(order))
}

type updateQuoteRequest struct {
	PricePesewas int64  `json:"pricePesewas"`
	Timeline     string `json:"timeline"`
	Notes        string `json:"notes"`
}

// AdminUpdateQuote handles PUT /api/v1/admin/orders/{ref}/quote.
func (h *Handlers) AdminUpdateQuote(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	var req updateQuoteRequest
	if !decodeBody(w, r, &req) {
		return
	}

	err := h.orders.UpdateQuote(r.Context(), ref, domain.Quote{
		PricePesewas: req.PricePesewas,
		Timeline:     req.Timeline,
		Notes:        req.Notes,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "not_found", "not found")
		case errors.Is(err, domain.ErrInvalidInput):
			respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		case errors.Is(err, domain.ErrConflict):
			respondError(w, http.StatusConflict, "conflict", "the order changed underneath you, try again")
		default:
			respondInternal(w, r, err)
		}

		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type paymentLinkResponse struct {
	PaymentURL string `json:"paymentUrl"`
}

// AdminSendPaymentLink handles POST /api/v1/admin/orders/{ref}/payment-link.
func (h *Handlers) AdminSendPaymentLink(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	url, err := h.orders.SendPaymentLink(r.Context(), ref)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "not_found", "not found")
		case errors.Is(err, domain.ErrInvalidInput):
			respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		case errors.Is(err, domain.ErrConflict):
			respondError(w, http.StatusConflict, "conflict", "the order changed underneath you, try again")
		default:
			respondInternal(w, r, err)
		}

		return
	}

	respondJSON(w, http.StatusOK, paymentLinkResponse{PaymentURL: url})
}

type markPaidRequest struct {
	Note string `json:"note"`
}

// AdminMarkPaid handles POST /api/v1/admin/orders/{ref}/mark-paid.
func (h *Handlers) AdminMarkPaid(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	var req markPaidRequest
	if !decodeBody(w, r, &req) {
		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "sign in to continue")

		return
	}

	err := h.orders.MarkPaidManually(r.Context(), ref, req.Note, user.Email)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "not_found", "not found")
		case errors.Is(err, domain.ErrInvalidInput):
			respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		case errors.Is(err, domain.ErrConflict):
			respondError(w, http.StatusConflict, "conflict", "the order changed underneath you, try again")
		default:
			respondInternal(w, r, err)
		}

		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// AdminUpdateOrderStatus handles POST /api/v1/admin/orders/{ref}/status.
func (h *Handlers) AdminUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	var req updateStatusRequest
	if !decodeBody(w, r, &req) {
		return
	}

	status := domain.OrderStatus(req.Status)
	if status == "" {
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", "status is required")

		return
	}

	if !domain.KnownOrderStatus(status) {
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", "unknown status")

		return
	}

	user, ok := userFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "sign in to continue")

		return
	}

	err := h.orders.UpdateOrderStatus(r.Context(), ref, status, user.Email)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "not_found", "not found")
		case errors.Is(err, domain.ErrInvalidInput):
			respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		case errors.Is(err, domain.ErrConflict):
			respondError(w, http.StatusConflict, "conflict", "the order changed underneath you, try again")
		default:
			respondInternal(w, r, err)
		}

		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
