package auth

import (
	"errors"
	"net/http"

	"Flatly/pkg/httpjson"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) HandleDummyLogin(w http.ResponseWriter, r *http.Request) {
	var req DummyLoginRequest
	if r.Method == http.MethodPost {
		if err := httpjson.Decode(r, &req); err != nil {
			httpjson.WriteDecodeError(w, err)
			return
		}
	}
	if req.Role == "" {
		req.Role = Role(r.URL.Query().Get("role"))
	}

	resp, err := h.service.DummyLogin(r.Context(), req.Role)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.WriteDecodeError(w, err)
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpjson.Write(w, http.StatusCreated, resp)
}

func (h *HTTPHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.WriteDecodeError(w, err)
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	httpjson.Write(w, http.StatusOK, resp)
}

func (h *HTTPHandler) writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRole):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_role", err.Error(), map[string]any{
			"allowed": []string{string(RoleClient), string(RoleModerator)},
		})
	case errors.Is(err, ErrEmailRequired):
		httpjson.WriteError(w, http.StatusBadRequest, "email_required", err.Error(), map[string]any{"field": "email"})
	case errors.Is(err, ErrInvalidEmail):
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_email", err.Error(), map[string]any{"field": "email"})
	case errors.Is(err, ErrPasswordRequired):
		httpjson.WriteError(w, http.StatusBadRequest, "password_required", err.Error(), map[string]any{"field": "password"})
	case errors.Is(err, ErrEmailExists):
		httpjson.WriteError(w, http.StatusConflict, "email_exists", err.Error(), map[string]any{"field": "email"})
	case errors.Is(err, ErrInvalidCredentials):
		httpjson.WriteError(w, http.StatusUnauthorized, "invalid_credentials", err.Error(), nil)
	default:
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
	}
}
