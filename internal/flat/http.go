package flat

import (
	"errors"
	"net/http"

	"Flatly/internal/auth"
	"Flatly/internal/house"
	"Flatly/pkg/httpjson"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, http.StatusUnauthorized, "unauthorized", auth.ErrUnauthorized.Error(), nil)
		return
	}

	var req CreateRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.WriteDecodeError(w, err)
		return
	}

	flat, err := h.service.Create(r.Context(), principal, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	httpjson.Write(w, http.StatusCreated, flat)
}

func (h *HTTPHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, http.StatusUnauthorized, "unauthorized", auth.ErrUnauthorized.Error(), nil)
		return
	}

	var req UpdateRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.WriteDecodeError(w, err)
		return
	}

	flat, err := h.service.UpdateStatus(r.Context(), principal, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, flat)
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidHouseID):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_house_id", err.Error(), map[string]any{"field": "houseId"})
	case errors.Is(err, ErrInvalidFlatNumber):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_flat_number", err.Error(), map[string]any{"field": "number"})
	case errors.Is(err, ErrInvalidPrice):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_price", err.Error(), map[string]any{"field": "price"})
	case errors.Is(err, ErrInvalidRooms):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_rooms", err.Error(), map[string]any{"field": "rooms"})
	case errors.Is(err, ErrInvalidStatus):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_status", err.Error(), map[string]any{
			"field":   "status",
			"allowed": []string{string(StatusCreated), string(StatusApproved), string(StatusDeclined), string(StatusOnModeration)},
		})
	case errors.Is(err, ErrFlatAlreadyExists):
		httpjson.WriteError(w, http.StatusConflict, "flat_exists", err.Error(), nil)
	case errors.Is(err, ErrFlatNotFound):
		httpjson.WriteError(w, http.StatusNotFound, "flat_not_found", err.Error(), nil)
	case errors.Is(err, house.ErrHouseNotFound):
		httpjson.WriteError(w, http.StatusNotFound, "house_not_found", err.Error(), nil)
	case errors.Is(err, ErrModerationConflict):
		httpjson.WriteError(w, http.StatusConflict, "moderation_conflict", err.Error(), nil)
	default:
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
	}
}
