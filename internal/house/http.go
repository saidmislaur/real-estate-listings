package house

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"Flatly/internal/auth"
	"Flatly/pkg/httpjson"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.WriteDecodeError(w, err)
		return
	}

	house, err := h.service.Create(r.Context(), req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	httpjson.Write(w, http.StatusCreated, house)
}

func (h *HTTPHandler) HandleGetByID(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, http.StatusUnauthorized, "unauthorized", auth.ErrUnauthorized.Error(), nil)
		return
	}

	houseID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.writeError(w, ErrInvalidHouseID)
		return
	}

	resp, err := h.service.ListFlats(r.Context(), houseID, principal)
	if err != nil {
		h.writeError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, http.StatusUnauthorized, "unauthorized", auth.ErrUnauthorized.Error(), nil)
		return
	}

	houseID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.writeError(w, ErrInvalidHouseID)
		return
	}

	var req SubscribeRequest
	if r.ContentLength != 0 {
		if err := httpjson.Decode(r, &req); err != nil {
			httpjson.WriteDecodeError(w, err)
			return
		}
	}

	if err := h.service.Subscribe(r.Context(), houseID, principal, req); err != nil {
		h.writeError(w, err)
		return
	}

	httpjson.Write(w, http.StatusAccepted, map[string]string{"status": "subscribed"})
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidHouseID):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_house_id", err.Error(), map[string]any{"field": "id"})
	case errors.Is(err, ErrInvalidAddress):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_address", err.Error(), map[string]any{"field": "address"})
	case errors.Is(err, ErrInvalidYear):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_year", err.Error(), map[string]any{"field": "year"})
	case errors.Is(err, ErrHouseExists):
		httpjson.WriteError(w, http.StatusConflict, "house_exists", err.Error(), nil)
	case errors.Is(err, ErrHouseNotFound):
		httpjson.WriteError(w, http.StatusNotFound, "house_not_found", err.Error(), nil)
	case errors.Is(err, ErrEmailRequired):
		httpjson.WriteError(w, http.StatusBadRequest, "email_required", err.Error(), map[string]any{"field": "email"})
	default:
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
	}
}
