package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"fatelumen/backend/internal/model"
)

// ---------- fakes ----------
type fakeAdminUserStore struct {
	users     []model.User
	total     int64
	listErr   error
	keyword   string // captured
	getUser   *model.User
	getErr    error
	activeSet map[uint64]bool
	activeErr error
	unlimitedSet map[uint64]bool
	unlimitedErr error
	tokenCleared map[uint64]bool
	clearTokenErr error
}

func newFakeAdminUserStore() *fakeAdminUserStore {
	return &fakeAdminUserStore{
		activeSet: make(map[uint64]bool),
		unlimitedSet: make(map[uint64]bool),
		tokenCleared: make(map[uint64]bool),
	}
}

func (f *fakeAdminUserStore) ListUsers(keyword string, limit, offset int) ([]model.User, int64, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	f.keyword = keyword
	return f.users, f.total, nil
}


func (f *fakeAdminUserStore) GetUserByID(id uint64) (*model.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getUser, nil
}

func (f *fakeAdminUserStore) SetUserActive(id uint64, active bool) error {
	if f.activeErr != nil {
		return f.activeErr
	}
	f.activeSet[id] = active
	return nil
}

func (f *fakeAdminUserStore) ClearCurrentToken(userID uint64) error {
	if f.clearTokenErr != nil {
		return f.clearTokenErr
	}
	f.tokenCleared[userID] = true
	return nil
}

func (f *fakeAdminUserStore) SetUserUnlimited(id uint64, unlimited bool) error {
	if f.unlimitedErr != nil {
		return f.unlimitedErr
	}
	f.unlimitedSet[id] = unlimited
	return nil
}

type fakeAdminCountStore struct {
	orders  int64
	reports int64
	err     error
}

func (f *fakeAdminCountStore) CountOrdersByUser(userID uint64) (int64, error) {
	return f.orders, f.err
}

func (f *fakeAdminCountStore) CountReportsByUser(userID uint64) (int64, error) {
	return f.reports, f.err
}

// ---------- tests ----------

func TestAdminListUsers_Pagination(t *testing.T) {
	store := newFakeAdminUserStore()
	store.users = []model.User{
		{ID: 1, Email: "a@x.com", Name: "A", Role: model.RoleUser, Active: true},
		{ID: 2, Email: "b@x.com", Name: "B", Role: model.RoleUser, Active: true},
	}
	store.total = 50

	svc := &AdminUserService{userRepo: store, countRepo: &fakeAdminCountStore{}}

	page, err := svc.ListUsers(context.Background(), "", 2, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 50 {
		t.Errorf("total: want 50, got %d", page.Total)
	}
	if page.Page != 2 {
		t.Errorf("page: want 2, got %d", page.Page)
	}
	if page.PageSize != 20 {
		t.Errorf("pageSize: want 20, got %d", page.PageSize)
	}
	if len(page.Items) != 2 {
		t.Errorf("items: want 2, got %d", len(page.Items))
	}
}

func TestAdminListUsers_PageSizeCap(t *testing.T) {
	store := newFakeAdminUserStore()
	svc := &AdminUserService{userRepo: store, countRepo: &fakeAdminCountStore{}}

	// pageSize > 100 should be capped to 20 (default)
	page, err := svc.ListUsers(context.Background(), "", 1, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.PageSize != 20 {
		t.Errorf("expected capped to 20, got %d", page.PageSize)
	}
}

func TestAdminListUsers_KeywordPassthrough(t *testing.T) {
	store := newFakeAdminUserStore()
	svc := &AdminUserService{userRepo: store, countRepo: &fakeAdminCountStore{}}
	svc.ListUsers(context.Background(), "test@", 1, 20)

	if store.keyword != "test@" {
		t.Errorf("expected keyword 'test@', got '%s'", store.keyword)
	}
}

func TestAdminSetUserActive(t *testing.T) {
	store := newFakeAdminUserStore()
	svc := &AdminUserService{userRepo: store, countRepo: &fakeAdminCountStore{}}

	err := svc.SetUserActive(context.Background(), 1, 10, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v, ok := store.activeSet[10]; !ok || v != false {
		t.Error("expected SetUserActive(10, false) to be called")
	}
}

func TestAdminGetUserDetail(t *testing.T) {
	store := newFakeAdminUserStore()
	store.getUser = &model.User{
		ID:     10,
		Email:  "user@x.com",
		Name:   "Test",
		Role:   model.RoleUser,
		Active: true,
	}
	countStore := &fakeAdminCountStore{orders: 5, reports: 3}

	svc := &AdminUserService{userRepo: store, countRepo: countStore}

	detail, err := svc.GetUserDetail(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.OrdersCount != 5 {
		t.Errorf("orders: want 5, got %d", detail.OrdersCount)
	}
	if detail.ReportsCount != 3 {
		t.Errorf("reports: want 3, got %d", detail.ReportsCount)
	}
}

func TestAdminGetUserDetail_NotFound(t *testing.T) {
	store := newFakeAdminUserStore()
	store.getErr = errors.New("not found")
	svc := &AdminUserService{userRepo: store, countRepo: &fakeAdminCountStore{}}

	_, err := svc.GetUserDetail(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

// TestToAdminUserItem_NoSensitiveFields verifies the DTO has no password/salt fields.
func TestToAdminUserItem_NoSensitiveFields(t *testing.T) {
	u := model.User{
		ID:    1,
		Email: "user@x.com",
		Name:  "Test",
		Role:  model.RoleUser,
		Active: true,
	}
	item := toAdminUserItem(u)

	b, _ := json.Marshal(item)
	s := strings.ToLower(string(b))

	for _, forbidden := range []string{"password", "hash", "salt", "secret", "token"} {
		if strings.Contains(s, forbidden) {
			t.Errorf("AdminUserItem JSON contains forbidden field: %s", forbidden)
		}
	}
}

// ---------- SetUserActive token invalidation ----------

func TestSetUserActive_Disable_ClearsToken(t *testing.T) {
	store := newFakeAdminUserStore()
	svc := &AdminUserService{userRepo: store, countRepo: &fakeAdminCountStore{}}

	err := svc.SetUserActive(context.Background(), 1, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v, ok := store.activeSet[2]; !ok || v != false {
		t.Errorf("expected user 2 active=false, got %v", v)
	}
	if !store.tokenCleared[2] {
		t.Error("expected ClearCurrentToken called when disabling user")
	}
}

func TestSetUserActive_Enable_DoesNotClearToken(t *testing.T) {
	store := newFakeAdminUserStore()
	svc := &AdminUserService{userRepo: store, countRepo: &fakeAdminCountStore{}}

	err := svc.SetUserActive(context.Background(), 1, 3, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v, ok := store.activeSet[3]; !ok || v != true {
		t.Errorf("expected user 3 active=true, got %v", v)
	}
	if store.tokenCleared[3] {
		t.Error("token should NOT be cleared when enabling user")
	}
}
