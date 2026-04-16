package flat

import (
	"context"
	"testing"
	"time"

	"Flatly/internal/auth"
	"Flatly/internal/house"
)

type repoStub struct {
	created *Flat
}

func (r *repoStub) Create(_ context.Context, item *Flat) error {
	now := time.Now().UTC()
	item.ID = 101
	item.CreatedAt = now
	item.UpdatedAt = now
	r.created = item
	return nil
}

func (r *repoStub) UpdateStatus(_ context.Context, _ int64, _ Status, _ int64) (*Flat, error) {
	return nil, nil
}

func (r *repoStub) ListByHouse(_ context.Context, _ int64, _ bool) ([]house.FlatSummary, error) {
	return nil, nil
}

type notifierStub struct {
	called bool
	house  int64
	flat   Flat
}

func (n *notifierStub) NotifyNewFlat(_ context.Context, houseID int64, item Flat) {
	n.called = true
	n.house = houseID
	n.flat = item
}

func TestServiceCreatePublishesFlatAndNotifies(t *testing.T) {
	repo := &repoStub{}
	notifier := &notifierStub{}
	service := NewService(repo, notifier)

	principal := auth.Principal{UserID: 77, Role: auth.RoleClient}
	resp, err := service.Create(context.Background(), principal, CreateRequest{
		HouseID: 12,
		Number:  45,
		Price:   9_000_000,
		Rooms:   3,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if resp.Status != StatusCreated {
		t.Fatalf("status = %q, want %q", resp.Status, StatusCreated)
	}
	if resp.CreatedBy != 77 {
		t.Fatalf("createdBy = %d, want 77", resp.CreatedBy)
	}
	if repo.created == nil || repo.created.Number != 45 {
		t.Fatalf("repository Create() was not called with expected flat")
	}
	if !notifier.called {
		t.Fatalf("notifier was not called")
	}
	if notifier.house != 12 {
		t.Fatalf("notifier house = %d, want 12", notifier.house)
	}
}
