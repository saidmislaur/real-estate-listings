package user

import (
	"context"
	"errors"
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetPasswordHashByEmail(ctx context.Context, email string) (string, error)
}

var (
	ErrEmailExists = errors.New("пользователь с таким email уже существует")
	ErrNotFound    = errors.New("пользователь не найден")

	ErrRequiredEmail = errors.New("поле email обязательно для заполнения")
	ErrInvalidEmail  = errors.New("поле email некорректно")

	ErrInvalidPasswordHash  = errors.New("неверный пароль")
	ErrRequiredPasswordHash = errors.New("пароль обязателен")

	ErrInvalidRole = errors.New("некорректная роль")

	FailedCreate = errors.New("ошибка создания пользователя")

	ErrInvalidCredentials = errors.New("неверные учетные данные")
)
