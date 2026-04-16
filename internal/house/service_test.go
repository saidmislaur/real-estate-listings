package house

import (
	"context"
	"testing"
	"time"

	"Flatly/internal/auth"
)

type repoStub struct {
	house *House
}

func (r *repoStub) Create(context.Context, *House) error {
	return nil
}

func (r *repoStub) GetByID(context.Context, int64) (*House, error) {
	copyHouse := *r.house
	return &copyHouse, nil
}

func (r *repoStub) Subscribe(context.Context, Subscription) error {
	return nil
}

func (r *repoStub) ListSubscriberEmails(context.Context, int64) ([]string, error) {
	return nil, nil
}

func TestServiceListFlatsUsesRoleVisibility(t *testing.T) {
	now := time.Now().UTC()
	repo := &repoStub{
		house: &House{
			ID:        10,
			Address:   "Тестовая улица, 1",
			Year:      2022,
			CreatedAt: now,
		},
	}

	reader := &capturingFlatReader{
		items: []FlatSummary{{ID: 1, HouseID: 10, Status: "approved"}},
	}
	service := NewService(repo, reader)

	if _, err := service.ListFlats(context.Background(), 10, auth.Principal{Role: auth.RoleClient}); err != nil {
		t.Fatalf("ListFlats(client) error = %v", err)
	}
	if reader.includeAll {
		t.Fatalf("client request should not ask for all statuses")
	}

	if _, err := service.ListFlats(context.Background(), 10, auth.Principal{Role: auth.RoleModerator}); err != nil {
		t.Fatalf("ListFlats(moderator) error = %v", err)
	}
	if !reader.includeAll {
		t.Fatalf("moderator request should ask for all statuses")
	}
}

type capturingFlatReader struct {
	includeAll bool
	items      []FlatSummary
}

func (r *capturingFlatReader) ListByHouse(_ context.Context, _ int64, includeAll bool) ([]FlatSummary, error) {
	r.includeAll = includeAll
	items := make([]FlatSummary, len(r.items))
	copy(items, r.items)
	return items, nil
}
