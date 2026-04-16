package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type tokenPayload struct {
	UserID   int64 `json:"user_id,omitempty"`
	Email    string
	Role     Role `json:"role"`
	IssuedAt int64
}

type HMACTokenManager struct {
	secret []byte
}

func NewHMACTokenManager(secret string) *HMACTokenManager {
	return &HMACTokenManager{secret: []byte(secret)}
}

func (m *HMACTokenManager) Issue(principal Principal) (string, error) {
	payload := tokenPayload{
		UserID:   principal.UserID,
		Email:    principal.Email,
		Role:     principal.Role,
		IssuedAt: time.Now().Unix(),
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	sig := m.sign(rawPayload)
	return encode(rawPayload) + "." + encode(sig), nil
}

func (m *HMACTokenManager) Parse(token string) (Principal, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Principal{}, ErrUnauthorized
	}

	rawPayload, err := decode(parts[0])
	if err != nil {
		return Principal{}, ErrUnauthorized
	}

	signature, err := decode(parts[1])
	if err != nil {
		return Principal{}, ErrUnauthorized
	}

	expected := m.sign(rawPayload)
	if subtle.ConstantTimeCompare(signature, expected) != 1 {
		return Principal{}, ErrUnauthorized
	}

	var payload tokenPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return Principal{}, ErrUnauthorized
	}
	if !isValidRole(payload.Role) {
		return Principal{}, ErrUnauthorized
	}

	return Principal{
		UserID: payload.UserID,
		Email:  payload.Email,
		Role:   payload.Role,
	}, nil
}

func (m *HMACTokenManager) sign(payload []byte) []byte {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write(payload)
	return mac.Sum(nil)
}

func encode(src []byte) string {
	return base64.RawURLEncoding.EncodeToString(src)
}

func decode(src string) ([]byte, error) {
	dst, err := base64.RawURLEncoding.DecodeString(src)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	return dst, nil
}

var _ TokenManager = (*HMACTokenManager)(nil)

var ErrInvalidToken = errors.New("invalid token")
