package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// --- DTOs --------------------------------------------------------------------

type slotDTO struct {
	ID        string    `json:"id"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func toSlotDTO(s domain.Slot) slotDTO {
	return slotDTO{
		ID:        s.ID,
		Start:     s.Start,
		End:       s.End,
		Status:    string(s.Status),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func toSlotDTOs(slots []domain.Slot) []slotDTO {
	dtos := make([]slotDTO, 0, len(slots))
	for _, s := range slots {
		dtos = append(dtos, toSlotDTO(s))
	}

	return dtos
}

type visitDTO struct {
	ID               string    `json:"id"`
	OrderID          string    `json:"orderId"`
	SlotID           string    `json:"slotId"`
	DepositPaymentID string    `json:"depositPaymentId"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func toVisitDTO(visit domain.Visit) visitDTO {
	return visitDTO{
		ID:               visit.ID,
		OrderID:          visit.OrderID,
		SlotID:           visit.SlotID,
		DepositPaymentID: visit.DepositPaymentID,
		Status:           string(visit.Status),
		CreatedAt:        visit.CreatedAt,
		UpdatedAt:        visit.UpdatedAt,
	}
}

func toVisitDTOs(visits []domain.Visit) []visitDTO {
	dtos := make([]visitDTO, 0, len(visits))
	for _, visit := range visits {
		dtos = append(dtos, toVisitDTO(visit))
	}

	return dtos
}

type bookSlotRequest struct {
	DesignID string `json:"designId,omitempty"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
}

type bookSlotResponse struct {
	Visit      visitDTO `json:"visit"`
	Order      orderDTO `json:"order"`
	PaymentURL string   `json:"paymentUrl"`
	User       userDTO  `json:"user"`
}

type createSlotRequest struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type rescheduleVisitRequest struct {
	NewSlotID string `json:"newSlotId"`
}

// --- public storefront endpoints ---------------------------------------------

// ListOpenSlots handles GET /api/v1/slots.
func (h *Handlers) ListOpenSlots(w http.ResponseWriter, r *http.Request) {
	filter := domain.SlotFilter{Status: domain.SlotStatusOpen}

	if from := r.URL.Query().Get("from"); from != "" {
		parsed, err := time.Parse(time.RFC3339, from)
		if err == nil {
			filter.After = parsed
		}
	}

	if to := r.URL.Query().Get("to"); to != "" {
		parsed, err := time.Parse(time.RFC3339, to)
		if err == nil {
			filter.Before = parsed
		}
	}

	slots, err := h.slots.ListSlots(r.Context(), filter)
	if err != nil {
		respondSlotError(w, err)

		return
	}

	respondJSON(w, http.StatusOK, toSlotDTOs(slots))
}

// BookSlot handles POST /api/v1/slots/{id}/book.
func (h *Handlers) BookSlot(w http.ResponseWriter, r *http.Request) {
	var req bookSlotRequest
	if !decodeBody(w, r, &req) {
		return
	}

	result, err := h.visits.BookSlot(
		r.Context(),
		chi.URLParam(r, "id"),
		req.DesignID,
		req.Email,
		req.Name,
		req.Phone,
	)

	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())

		return
	case errors.Is(err, domain.ErrSlotNotFound):
		respondError(w, http.StatusNotFound, "not_found", "slot not found")

		return
	case errors.Is(err, domain.ErrSlotUnavailable):
		respondError(w, http.StatusConflict, "slot_unavailable", "this slot is no longer available")

		return
	case err != nil:
		respondError(w, http.StatusInternalServerError, "internal", "something went wrong")

		return
	}

	sessionToken, err := h.auth.CreateSession(r.Context(), result.User.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal", "something went wrong")

		return
	}

	h.setSessionCookie(w, sessionToken, int(sessionCookieMaxAge.Seconds()))

	respondJSON(w, http.StatusCreated, bookSlotResponse{
		Visit:      toVisitDTO(*result.Visit),
		Order:      toOrderDTO(result.Order),
		PaymentURL: result.PaymentURL,
		User:       h.toUserDTO(result.User),
	})
}

// --- admin endpoints ---------------------------------------------------------

// AdminListSlots handles GET /api/v1/admin/slots.
func (h *Handlers) AdminListSlots(w http.ResponseWriter, r *http.Request) {
	slots, err := h.slots.ListSlots(r.Context(), domain.SlotFilter{})
	if err != nil {
		respondSlotError(w, err)

		return
	}

	respondJSON(w, http.StatusOK, toSlotDTOs(slots))
}

// AdminCreateSlot handles POST /api/v1/admin/slots.
func (h *Handlers) AdminCreateSlot(w http.ResponseWriter, r *http.Request) {
	var req createSlotRequest
	if !decodeBody(w, r, &req) {
		return
	}

	slot, err := h.slots.CreateSlot(r.Context(), req.Start, req.End)
	if err != nil {
		respondSlotError(w, err)

		return
	}

	respondJSON(w, http.StatusCreated, toSlotDTO(*slot))
}

// AdminCloseSlot handles POST /api/v1/admin/slots/{id}/close.
func (h *Handlers) AdminCloseSlot(w http.ResponseWriter, r *http.Request) {
	err := h.slots.CloseSlot(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondSlotError(w, err)

		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// AdminReopenSlot handles POST /api/v1/admin/slots/{id}/reopen.
func (h *Handlers) AdminReopenSlot(w http.ResponseWriter, r *http.Request) {
	err := h.slots.ReopenSlot(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondSlotError(w, err)

		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// AdminListVisits handles GET /api/v1/admin/visits.
func (h *Handlers) AdminListVisits(w http.ResponseWriter, r *http.Request) {
	visits, err := h.visits.ListVisits(r.Context(), domain.VisitFilter{})
	if err != nil {
		respondSlotError(w, err)

		return
	}

	respondJSON(w, http.StatusOK, toVisitDTOs(visits))
}

// AdminRescheduleVisit handles POST /api/v1/admin/visits/{id}/reschedule.
func (h *Handlers) AdminRescheduleVisit(w http.ResponseWriter, r *http.Request) {
	var req rescheduleVisitRequest
	if !decodeBody(w, r, &req) {
		return
	}

	visit, err := h.visits.RescheduleVisit(r.Context(), chi.URLParam(r, "id"), req.NewSlotID)
	if err != nil {
		respondSlotError(w, err)

		return
	}

	respondJSON(w, http.StatusOK, toVisitDTO(*visit))
}

// AdminCancelVisit handles POST /api/v1/admin/visits/{id}/cancel.
func (h *Handlers) AdminCancelVisit(w http.ResponseWriter, r *http.Request) {
	visit, err := h.visits.CancelVisit(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		respondSlotError(w, err)

		return
	}

	respondJSON(w, http.StatusOK, toVisitDTO(*visit))
}

func respondSlotError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrNotFound),
		errors.Is(err, domain.ErrSlotNotFound),
		errors.Is(err, domain.ErrVisitNotFound):
		respondError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, domain.ErrSlotUnavailable):
		respondError(w, http.StatusConflict, "slot_unavailable", "slot is not available")
	case errors.Is(err, domain.ErrVisitAlreadyCancelled):
		respondError(w, http.StatusConflict, "already_cancelled", "visit already cancelled")
	default:
		respondError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
