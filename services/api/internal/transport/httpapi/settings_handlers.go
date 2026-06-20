package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

type deliveryRateDTO struct {
	Area        string `json:"area"`
	RatePesewas int64  `json:"ratePesewas"`
}

type settingsDTO struct {
	DepositPesewas  int64             `json:"depositPesewas"`
	WhatsAppNumber  string            `json:"whatsappNumber"`
	VisitLocation   string            `json:"visitLocation"`
	InstagramHandle string            `json:"instagramHandle"`
	ContactEmail    string            `json:"contactEmail"`
	DeliveryRates   []deliveryRateDTO `json:"deliveryRates"`
}

type publicSettingsDTO struct {
	settingsDTO

	// CloudName lets the storefront build Cloudinary photo URLs; it is
	// public information (it appears in every delivery URL).
	CloudName string `json:"cloudName"`
}

func toSettingsDTO(s *domain.Settings) settingsDTO {
	rates := make([]deliveryRateDTO, 0, len(s.DeliveryRates))
	for _, rate := range s.DeliveryRates {
		rates = append(rates, deliveryRateDTO{Area: rate.Area, RatePesewas: rate.RatePesewas})
	}

	return settingsDTO{
		DepositPesewas:  s.DepositPesewas,
		WhatsAppNumber:  s.WhatsAppNumber,
		VisitLocation:   s.VisitLocation,
		InstagramHandle: s.InstagramHandle,
		ContactEmail:    s.ContactEmail,
		DeliveryRates:   rates,
	}
}

func fromSettingsDTO(dto settingsDTO) *domain.Settings {
	rates := make([]domain.DeliveryRate, 0, len(dto.DeliveryRates))
	for _, rate := range dto.DeliveryRates {
		rates = append(rates, domain.DeliveryRate{Area: rate.Area, RatePesewas: rate.RatePesewas})
	}

	return &domain.Settings{
		DepositPesewas:  dto.DepositPesewas,
		WhatsAppNumber:  dto.WhatsAppNumber,
		VisitLocation:   dto.VisitLocation,
		InstagramHandle: dto.InstagramHandle,
		ContactEmail:    dto.ContactEmail,
		DeliveryRates:   rates,
	}
}

// GetSettings handles GET /api/v1/settings (public — the storefront needs
// the deposit amount, contact details, and the media cloud name).
func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.Get(r.Context())
	if err != nil {
		respondInternal(w, r, err)

		return
	}

	respondJSON(w, http.StatusOK, publicSettingsDTO{
		settingsDTO: toSettingsDTO(settings),
		CloudName:   h.cloudName,
	})
}

// UpdateSettings handles PUT /api/v1/admin/settings.
func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req settingsDTO

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "request body must be valid JSON")

		return
	}

	err = h.settings.Update(r.Context(), fromSettingsDTO(req))

	switch {
	case errors.Is(err, domain.ErrDuplicateArea):
		respondError(w, http.StatusConflict, "duplicate_area", err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
	case err != nil:
		respondInternal(w, r, err)
	default:
		respondJSON(w, http.StatusOK, req)
	}
}
