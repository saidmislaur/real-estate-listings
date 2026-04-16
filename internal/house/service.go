package house

import (
	"context"
	"strings"

	"Flatly/internal/auth"
)

type Service struct {
	repo       Repository
	flatReader FlatReader
}

func NewService(repo Repository, flatReader FlatReader) *Service {
	return &Service{repo: repo, flatReader: flatReader}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*House, error) {
	if req.ID <= 0 {
		return nil, ErrInvalidHouseID
	}
	if strings.TrimSpace(req.Address) == "" {
		return nil, ErrInvalidAddress
	}
	if req.Year <= 0 {
		return nil, ErrInvalidYear
	}

	house := &House{
		ID:        req.ID,
		Address:   strings.TrimSpace(req.Address),
		Year:      req.Year,
		Developer: req.Developer,
	}
	if err := s.repo.Create(ctx, house); err != nil {
		return nil, err
	}
	return house, nil
}

func (s *Service) ListFlats(ctx context.Context, houseID int64, principal auth.Principal) (*HouseWithFlatsResponse, error) {
	if houseID <= 0 {
		return nil, ErrInvalidHouseID
	}

	house, err := s.repo.GetByID(ctx, houseID)
	if err != nil {
		return nil, err
	}

	flats, err := s.flatReader.ListByHouse(ctx, houseID, principal.Role == auth.RoleModerator)
	if err != nil {
		return nil, err
	}

	return &HouseWithFlatsResponse{
		House: *house,
		Flats: flats,
	}, nil
}

func (s *Service) Subscribe(ctx context.Context, houseID int64, principal auth.Principal, req SubscribeRequest) error {
	if houseID <= 0 {
		return ErrInvalidHouseID
	}
	if _, err := s.repo.GetByID(ctx, houseID); err != nil {
		return err
	}

	email := strings.TrimSpace(req.Email)
	if email == "" {
		email = strings.TrimSpace(principal.Email)
	}
	if email == "" {
		return ErrEmailRequired
	}

	return s.repo.Subscribe(ctx, Subscription{
		HouseID: houseID,
		UserID:  principal.UserID,
		Email:   email,
	})
}
