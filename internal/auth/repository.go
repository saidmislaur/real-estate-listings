package auth

import (
	"context"
	"errors"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
}

var (
	ErrInvalidRole        = errors.New("некорректная роль")
	ErrInvalidEmail       = errors.New("некорректный email")
	ErrEmailRequired      = errors.New("email обязателен")
	ErrPasswordRequired   = errors.New("пароль обязателен")
	ErrUnauthorized       = errors.New("требуется авторизация")
	ErrForbidden          = errors.New("недостаточно прав")
	ErrEmailExists        = errors.New("пользователь с таким email уже существует")
	ErrInvalidCredentials = errors.New("неверные учетные данные")
	ErrNotFound           = errors.New("пользователь не найден")
)
