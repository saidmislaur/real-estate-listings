package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"Flatly/internal/auth"
	"Flatly/internal/flat"
	"Flatly/internal/house"
	httprouter "Flatly/pkg/httproute"
)

type memoryUserRepo struct {
	mu      sync.Mutex
	nextID  int64
	byEmail map[string]*auth.User
}

func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{nextID: 1, byEmail: map[string]*auth.User{}}
}

func (r *memoryUserRepo) Create(_ context.Context, user *auth.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byEmail[user.Email]; exists {
		return auth.ErrEmailExists
	}

	copyUser := *user
	copyUser.ID = r.nextID
	copyUser.CreatedAt = time.Now().UTC()
	copyUser.UpdatedAt = copyUser.CreatedAt
	r.nextID++
	r.byEmail[user.Email] = &copyUser
	*user = copyUser
	return nil
}

func (r *memoryUserRepo) GetByEmail(_ context.Context, email string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.byEmail[email]
	if !ok {
		return nil, auth.ErrNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

type memoryStore struct {
	mu            sync.Mutex
	houses        map[int64]*house.House
	flats         map[int64]*flat.Flat
	subscriptions map[int64]map[string]struct{}
	nextFlatID    int64
}

type memoryHouseRepo struct {
	store *memoryStore
}

type memoryFlatRepo struct {
	store *memoryStore
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		houses:        map[int64]*house.House{},
		flats:         map[int64]*flat.Flat{},
		subscriptions: map[int64]map[string]struct{}{},
		nextFlatID:    1,
	}
}

func (r *memoryHouseRepo) Create(_ context.Context, item *house.House) error {
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.houses[item.ID]; exists {
		return house.ErrHouseExists
	}

	copyHouse := *item
	copyHouse.CreatedAt = time.Now().UTC()
	s.houses[item.ID] = &copyHouse
	*item = copyHouse
	return nil
}

func (r *memoryHouseRepo) GetByID(_ context.Context, id int64) (*house.House, error) {
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.houses[id]
	if !ok {
		return nil, house.ErrHouseNotFound
	}
	copyHouse := *item
	return &copyHouse, nil
}

func (r *memoryHouseRepo) Subscribe(_ context.Context, sub house.Subscription) error {
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subscriptions[sub.HouseID]; !ok {
		s.subscriptions[sub.HouseID] = map[string]struct{}{}
	}
	s.subscriptions[sub.HouseID][sub.Email] = struct{}{}
	return nil
}

func (r *memoryHouseRepo) ListSubscriberEmails(_ context.Context, houseID int64) ([]string, error) {
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]string, 0, len(s.subscriptions[houseID]))
	for email := range s.subscriptions[houseID] {
		result = append(result, email)
	}
	return result, nil
}

func (r *memoryFlatRepo) Create(_ context.Context, item *flat.Flat) error {
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()

	houseItem, ok := s.houses[item.HouseID]
	if !ok {
		return house.ErrHouseNotFound
	}

	for _, existing := range s.flats {
		if existing.HouseID == item.HouseID && existing.Number == item.Number {
			return flat.ErrFlatAlreadyExists
		}
	}

	now := time.Now().UTC()
	copyFlat := *item
	copyFlat.ID = s.nextFlatID
	copyFlat.CreatedAt = now
	copyFlat.UpdatedAt = now
	s.nextFlatID++
	s.flats[copyFlat.ID] = &copyFlat

	houseCopy := *houseItem
	houseCopy.LastFlatCreatedAt = &now
	s.houses[item.HouseID] = &houseCopy

	*item = copyFlat
	return nil
}

func (r *memoryFlatRepo) UpdateStatus(_ context.Context, flatID int64, status flat.Status, moderatorID int64) (*flat.Flat, error) {
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.flats[flatID]
	if !ok {
		return nil, flat.ErrFlatNotFound
	}

	switch status {
	case flat.StatusOnModeration:
		if item.Status != flat.StatusCreated || item.ModeratorID != nil {
			return nil, flat.ErrModerationConflict
		}
		item.Status = status
		item.ModeratorID = &moderatorID
	case flat.StatusApproved, flat.StatusDeclined:
		if item.Status != flat.StatusOnModeration || item.ModeratorID == nil || *item.ModeratorID != moderatorID {
			return nil, flat.ErrModerationConflict
		}
		item.Status = status
	default:
		return nil, flat.ErrInvalidStatus
	}

	item.UpdatedAt = time.Now().UTC()
	copyFlat := *item
	return &copyFlat, nil
}

func (r *memoryFlatRepo) ListByHouse(_ context.Context, houseID int64, includeAll bool) ([]house.FlatSummary, error) {
	s := r.store
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]house.FlatSummary, 0)
	for _, item := range s.flats {
		if item.HouseID != houseID {
			continue
		}
		if !includeAll && item.Status != flat.StatusApproved {
			continue
		}

		var moderatorID int64
		if item.ModeratorID != nil {
			moderatorID = *item.ModeratorID
		}

		result = append(result, house.FlatSummary{
			ID:          item.ID,
			HouseID:     item.HouseID,
			Number:      item.Number,
			Price:       item.Price,
			Rooms:       item.Rooms,
			Status:      string(item.Status),
			CreatedAt:   item.CreatedAt,
			CreatedBy:   item.CreatedBy,
			ModeratorID: moderatorID,
		})
	}

	return result, nil
}

type noopNotifier struct{}

func (noopNotifier) NotifyNewFlat(context.Context, int64, flat.Flat) {}

func newTestServer() http.Handler {
	store := newMemoryStore()
	houseRepo := &memoryHouseRepo{store: store}
	flatRepo := &memoryFlatRepo{store: store}
	userRepo := newMemoryUserRepo()
	tokenManager := auth.NewHMACTokenManager("test-secret")

	authService := auth.NewService(userRepo, tokenManager)
	houseService := house.NewService(houseRepo, flatRepo)
	flatService := flat.NewService(flatRepo, noopNotifier{})

	return httprouter.NewRouter(
		auth.NewHTTPHandler(authService),
		house.NewHTTPHandler(houseService),
		flat.NewHTTPHandler(flatService),
		authService,
	)
}

func doJSON(t *testing.T, server http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)
	return rr
}

func readToken(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("dummy login status = %d, body = %s", rr.Code, rr.Body.String())
	}

	var resp auth.AuthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal auth response: %v", err)
	}
	return resp.Token
}

func TestPublishFlatFlow(t *testing.T) {
	server := newTestServer()

	moderatorToken := readToken(t, doJSON(t, server, http.MethodPost, "/dummyLogin", "", map[string]any{"role": "moderator"}))
	clientToken := readToken(t, doJSON(t, server, http.MethodPost, "/dummyLogin", "", map[string]any{"role": "client"}))

	houseResp := doJSON(t, server, http.MethodPost, "/house/create", moderatorToken, map[string]any{
		"id":      1001,
		"address": "Москва, Пушкина 1",
		"year":    2018,
	})
	if houseResp.Code != http.StatusCreated {
		t.Fatalf("create house status = %d, body = %s", houseResp.Code, houseResp.Body.String())
	}

	flatResp := doJSON(t, server, http.MethodPost, "/flat/create", clientToken, map[string]any{
		"houseId": 1001,
		"number":  7,
		"price":   15000000,
		"rooms":   2,
	})
	if flatResp.Code != http.StatusCreated {
		t.Fatalf("create flat status = %d, body = %s", flatResp.Code, flatResp.Body.String())
	}

	var created flat.Flat
	if err := json.Unmarshal(flatResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created flat: %v", err)
	}
	if created.Status != flat.StatusCreated {
		t.Fatalf("flat status = %q, want %q", created.Status, flat.StatusCreated)
	}

	modListResp := doJSON(t, server, http.MethodGet, "/house/1001", moderatorToken, nil)
	if modListResp.Code != http.StatusOK {
		t.Fatalf("moderator house status = %d, body = %s", modListResp.Code, modListResp.Body.String())
	}

	var modList house.HouseWithFlatsResponse
	if err := json.Unmarshal(modListResp.Body.Bytes(), &modList); err != nil {
		t.Fatalf("unmarshal moderator list: %v", err)
	}
	if modList.House.LastFlatCreatedAt == nil {
		t.Fatalf("expected house lastFlatCreatedAt to be set")
	}
	if got := len(modList.Flats); got != 1 {
		t.Fatalf("moderator flats count = %d, want 1", got)
	}

	clientListResp := doJSON(t, server, http.MethodGet, "/house/1001", clientToken, nil)
	if clientListResp.Code != http.StatusOK {
		t.Fatalf("client house status = %d, body = %s", clientListResp.Code, clientListResp.Body.String())
	}

	var clientList house.HouseWithFlatsResponse
	if err := json.Unmarshal(clientListResp.Body.Bytes(), &clientList); err != nil {
		t.Fatalf("unmarshal client list: %v", err)
	}
	if got := len(clientList.Flats); got != 0 {
		t.Fatalf("client flats count = %d, want 0", got)
	}
}

func TestHouseListingFilteredByRole(t *testing.T) {
	server := newTestServer()

	moderatorToken := readToken(t, doJSON(t, server, http.MethodPost, "/dummyLogin", "", map[string]any{"role": "moderator"}))
	clientToken := readToken(t, doJSON(t, server, http.MethodPost, "/dummyLogin", "", map[string]any{"role": "client"}))

	createHouseResp := doJSON(t, server, http.MethodPost, "/house/create", moderatorToken, map[string]any{
		"id":      2002,
		"address": "Санкт-Петербург, Невский 10",
		"year":    2020,
	})
	if createHouseResp.Code != http.StatusCreated {
		t.Fatalf("create house status = %d, body = %s", createHouseResp.Code, createHouseResp.Body.String())
	}

	var flatIDs []int64
	for idx := 0; idx < 2; idx++ {
		resp := doJSON(t, server, http.MethodPost, "/flat/create", clientToken, map[string]any{
			"houseId": 2002,
			"number":  idx + 1,
			"price":   10000000 + idx,
			"rooms":   2,
		})
		if resp.Code != http.StatusCreated {
			t.Fatalf("create flat %d status = %d, body = %s", idx, resp.Code, resp.Body.String())
		}
		var created flat.Flat
		if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
			t.Fatalf("unmarshal created flat %d: %v", idx, err)
		}
		flatIDs = append(flatIDs, created.ID)
	}

	for _, status := range []flat.Status{flat.StatusOnModeration, flat.StatusApproved} {
		resp := doJSON(t, server, http.MethodPost, "/flat/update", moderatorToken, map[string]any{
			"id":     flatIDs[0],
			"status": status,
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("update flat to %s failed: %d, body = %s", status, resp.Code, resp.Body.String())
		}
	}

	resp := doJSON(t, server, http.MethodGet, "/house/2002", clientToken, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("client house list status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var clientView house.HouseWithFlatsResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &clientView); err != nil {
		t.Fatalf("unmarshal client view: %v", err)
	}
	if len(clientView.Flats) != 1 {
		t.Fatalf("client visible flats = %d, want 1", len(clientView.Flats))
	}
	if clientView.Flats[0].Status != string(flat.StatusApproved) {
		t.Fatalf("client sees status = %s, want approved", clientView.Flats[0].Status)
	}

	resp = doJSON(t, server, http.MethodGet, "/house/"+strconv.FormatInt(2002, 10), moderatorToken, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("moderator house list status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var moderatorView house.HouseWithFlatsResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &moderatorView); err != nil {
		t.Fatalf("unmarshal moderator view: %v", err)
	}
	if len(moderatorView.Flats) != 2 {
		t.Fatalf("moderator visible flats = %d, want 2", len(moderatorView.Flats))
	}
}
