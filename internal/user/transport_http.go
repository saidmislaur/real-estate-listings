package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type HTTPHandler struct {
	srv    Service
	logger *zap.Logger
}

func NewHTTPHandler(srv Service, logger *zap.Logger) *HTTPHandler {
	return &HTTPHandler{
		srv:    srv,
		logger: logger.Named("user.http").With(zap.String("component", "user.http")),
	}
}

type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type APIErrorResponse struct {
	Error APIError
}

func (h *HTTPHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONDecodeError(w, err)
		return
	}

	u, err := h.srv.CreateUser(ctx, req)
	if err != nil {
		var ve ValidationError
		switch {
		case errors.As(err, &ve):
			writeValidationError(w, ve)

		case errors.Is(err, ErrEmailExists):
			writeAPIError(w, http.StatusConflict, "email_exists", ErrEmailExists.Error(), map[string]any{
				"field": "email",
			})

		case errors.Is(err, ErrInvalidRole):
			writeAPIError(w, http.StatusBadRequest, "invalid_role", ErrInvalidRole.Error(), map[string]any{
				"field": "role",
				"allowed": []string{
					string(RoleClient),
					string(RoleModerator),
				},
			})

		default:
			h.logger.Error(FailedCreate.Error(), zap.Any("error", err))
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		}
		return
	}

	resp := CreateUserResponse{
		ID:    u.ID,
		Email: u.Email,
		Role:  u.Role,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *HTTPHandler) HandleLoginUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONDecodeError(w, err)
		return
	}

	user, err := h.srv.LoginUser(ctx, LoginUserRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		var ve ValidationError
		switch {
		case errors.As(err, &ve):
			writeValidationError(w, ve)

		case errors.Is(err, ErrInvalidCredentials):
			writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Неверный email или пароль", nil)

		default:
			h.logger.Error("login handler error",
				zap.Error(err),
				zap.String("email", req.Email),
			)
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		}
		return
	}

	resp := LoginUserResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeValidationError(w http.ResponseWriter, ve ValidationError) {
	details := map[string]any{}
	if ve.Field != "" {
		details["field"] = ve.Field
	}
	writeAPIError(w, http.StatusBadRequest, "validation_error", ve.Message, details)
}

func decodeJSON(r *http.Request, dst any) error {
	const maxBody = 1 << 20 // 1MB

	// читаем тело ограниченно (без MaxBytesReader, т.к. нет ResponseWriter)
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(body) > maxBody {
		return fmt.Errorf("body too large")
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return fmt.Errorf("empty body")
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}

	// запрет на мусор после JSON
	if dec.More() {
		return fmt.Errorf("multiple json values")
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing data")
		}
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	writeJSON(w, status, APIErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func writeJSONDecodeError(w http.ResponseWriter, err error) {
	code := "invalid_json"
	msg := "Некорректный JSON в теле запроса"
	details := map[string]any{}

	if err == nil {
		writeAPIError(w, http.StatusBadRequest, code, msg, map[string]any{
			"hint": "Проверьте синтаксис JSON и названия полей",
		})
		return
	}

	// 1) пустое тело
	if err.Error() == "empty body" {
		writeAPIError(w, http.StatusBadRequest, "empty_body", "Пустое тело запроса", map[string]any{
			"hint": "Передайте JSON-объект в теле запроса",
		})
		return
	}

	// 2) слишком большое тело
	if err.Error() == "body too large" {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "request_body_too_large", "Слишком большое тело запроса", map[string]any{
			"max_bytes": 1 << 20,
		})
		return
	}

	// 3) SyntaxError
	var se *json.SyntaxError
	if errors.As(err, &se) {
		writeAPIError(w, http.StatusBadRequest, "invalid_json_syntax", "Ошибка синтаксиса JSON", map[string]any{
			"offset": se.Offset,
			"hint":   "Проверьте запятые, кавычки и скобки",
		})
		return
	}

	// 4) UnmarshalTypeError
	var te *json.UnmarshalTypeError
	if errors.As(err, &te) {
		details["field"] = te.Field
		details["expected"] = te.Type.String()
		details["got"] = te.Value
		writeAPIError(w, http.StatusBadRequest, "invalid_field_type", "Неверный тип поля в JSON", details)
		return
	}

	// 5) unknown field (строка от stdlib)
	// `json: unknown field "xxx"`
	const prefix = `json: unknown field "`
	s := err.Error()
	if strings.HasPrefix(s, prefix) && strings.HasSuffix(s, `"`) {
		field := strings.TrimSuffix(strings.TrimPrefix(s, prefix), `"`)
		writeAPIError(w, http.StatusBadRequest, "unknown_field", "Неизвестное поле в JSON", map[string]any{
			"field": field,
			"hint":  "Проверьте название поля (возможна опечатка)",
		})
		return
	}

	// 6) хвост/несколько JSON
	if s == "multiple json values" || s == "trailing data" {
		writeAPIError(w, http.StatusBadRequest, "trailing_data", "В теле запроса лишние данные после JSON", map[string]any{
			"hint": "Тело запроса должно содержать ровно один JSON-объект",
		})
		return
	}

	// fallback — не раскрываем внутренности, но даём подсказку
	writeAPIError(w, http.StatusBadRequest, code, msg, map[string]any{
		"hint": "Проверьте синтаксис JSON и названия полей",
	})
}
