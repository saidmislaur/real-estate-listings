package house

import (
	"context"
	"errors"
)

type Repository interface {
	Create(ctx context.Context, house *House) error
	GetByID(ctx context.Context, id int64) (*House, error)
	Subscribe(ctx context.Context, subscription Subscription) error
	ListSubscriberEmails(ctx context.Context, houseID int64) ([]string, error)
}

type FlatReader interface {
	ListByHouse(ctx context.Context, houseID int64, includeAll bool) ([]FlatSummary, error)
}

type Subscription struct {
	HouseID int64
	UserID  int64
	Email   string
}

var (
	ErrInvalidHouseID = errors.New("некорректный номер дома")
	ErrInvalidAddress = errors.New("адрес обязателен")
	ErrInvalidYear    = errors.New("некорректный год постройки")
	ErrHouseExists    = errors.New("дом уже существует")
	ErrHouseNotFound  = errors.New("дом не найден")
	ErrEmailRequired  = errors.New("email обязателен для подписки")
)
