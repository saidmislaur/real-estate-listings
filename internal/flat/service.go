package flat

import (
	"context"

	"Flatly/internal/auth"
)

type Notifier interface {
	NotifyNewFlat(ctx context.Context, houseID int64, flat Flat)
}

type Service struct {
	repo     Repository
	notifier Notifier
}

func NewService(repo Repository, notifier Notifier) *Service {
	return &Service{repo: repo, notifier: notifier}
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, req CreateRequest) (*Flat, error) {
	if req.HouseID <= 0 {
		return nil, ErrInvalidHouseID
	}
	if req.Number <= 0 {
		return nil, ErrInvalidFlatNumber
	}
	if req.Price <= 0 {
		return nil, ErrInvalidPrice
	}
	if req.Rooms <= 0 {
		return nil, ErrInvalidRooms
	}

	flat := &Flat{
		HouseID:   req.HouseID,
		Number:    req.Number,
		Price:     req.Price,
		Rooms:     req.Rooms,
		Status:    StatusCreated,
		CreatedBy: principal.UserID,
	}

	if err := s.repo.Create(ctx, flat); err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.NotifyNewFlat(ctx, flat.HouseID, *flat)
	}

	return flat, nil
}

func (s *Service) UpdateStatus(ctx context.Context, principal auth.Principal, req UpdateRequest) (*Flat, error) {
	if req.ID <= 0 {
		return nil, ErrFlatNotFound
	}
	if !isValidStatus(req.Status) || req.Status == StatusCreated {
		return nil, ErrInvalidStatus
	}

	return s.repo.UpdateStatus(ctx, req.ID, req.Status, principal.UserID)
}

func isValidStatus(status Status) bool {
	return status == StatusCreated || status == StatusApproved || status == StatusDeclined || status == StatusOnModeration
}
