package httpjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxBodySize = 1 << 20

type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

func Decode(r *http.Request, dst any) error {
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize+1))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(body) > maxBodySize {
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

func Write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	Write(w, status, ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func WriteDecodeError(w http.ResponseWriter, err error) {
	if err == nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Некорректный JSON в теле запроса", map[string]any{
			"hint": "Проверьте синтаксис JSON и названия полей",
		})
		return
	}

	switch err.Error() {
	case "empty body":
		WriteError(w, http.StatusBadRequest, "empty_body", "Пустое тело запроса", map[string]any{
			"hint": "Передайте JSON-объект в теле запроса",
		})
		return
	case "body too large":
		WriteError(w, http.StatusRequestEntityTooLarge, "request_body_too_large", "Слишком большое тело запроса", map[string]any{
			"max_bytes": maxBodySize,
		})
		return
	case "multiple json values", "trailing data":
		WriteError(w, http.StatusBadRequest, "trailing_data", "В теле запроса лишние данные после JSON", map[string]any{
			"hint": "Тело запроса должно содержать ровно один JSON-объект",
		})
		return
	}

	var se *json.SyntaxError
	if errors.As(err, &se) {
		WriteError(w, http.StatusBadRequest, "invalid_json_syntax", "Ошибка синтаксиса JSON", map[string]any{
			"offset": se.Offset,
			"hint":   "Проверьте запятые, кавычки и скобки",
		})
		return
	}

	var te *json.UnmarshalTypeError
	if errors.As(err, &te) {
		WriteError(w, http.StatusBadRequest, "invalid_field_type", "Неверный тип поля в JSON", map[string]any{
			"field":    te.Field,
			"expected": te.Type.String(),
			"got":      te.Value,
		})
		return
	}

	const prefix = `json: unknown field "`
	if strings.HasPrefix(err.Error(), prefix) && strings.HasSuffix(err.Error(), `"`) {
		field := strings.TrimSuffix(strings.TrimPrefix(err.Error(), prefix), `"`)
		WriteError(w, http.StatusBadRequest, "unknown_field", "Неизвестное поле в JSON", map[string]any{
			"field": field,
			"hint":  "Проверьте название поля",
		})
		return
	}

	WriteError(w, http.StatusBadRequest, "invalid_json", "Некорректный JSON в теле запроса", map[string]any{
		"hint": "Проверьте синтаксис JSON и названия полей",
	})
}
