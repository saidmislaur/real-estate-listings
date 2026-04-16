package flat

import (
	"context"
	"errors"

	"Flatly/internal/house"
)

type Repository interface {
	Create(ctx context.Context, flat *Flat) error
	UpdateStatus(ctx context.Context, flatID int64, status Status, moderatorID int64) (*Flat, error)
	ListByHouse(ctx context.Context, houseID int64, includeAll bool) ([]house.FlatSummary, error)
}

var (
	ErrInvalidHouseID     = errors.New("некорректный номер дома")
	ErrInvalidFlatNumber  = errors.New("некорректный номер квартиры")
	ErrInvalidPrice       = errors.New("некорректная цена")
	ErrInvalidRooms       = errors.New("некорректное количество комнат")
	ErrInvalidStatus      = errors.New("некорректный статус модерации")
	ErrFlatNotFound       = errors.New("квартира не найдена")
	ErrModerationConflict = errors.New("квартира уже взята в модерацию или находится в неподходящем статусе")
	ErrFlatAlreadyExists  = errors.New("квартира с таким номером уже существует в доме")
)
