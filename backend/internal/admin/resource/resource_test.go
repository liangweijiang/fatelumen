package resource

import (
	"context"
	"errors"
	"testing"

	"fatelumen/backend/internal/service"
)

type fakeUserListSvc struct {
	page        *service.AdminUsersPage
	listErr     error
	detail      *service.AdminUserDetail
	detailErr   error
	gotKeyword  string
	gotPage     int
	gotPageSize int
	gotUserID   uint64
}

func (f *fakeUserListSvc) ListUsers(ctx context.Context, keyword string, page, pageSize int) (*service.AdminUsersPage, error) {
	f.gotKeyword = keyword
	f.gotPage = page
	f.gotPageSize = pageSize
	return f.page, f.listErr
}

func (f *fakeUserListSvc) GetUserDetail(ctx context.Context, userID uint64) (*service.AdminUserDetail, error) {
	f.gotUserID = userID
	return f.detail, f.detailErr
}

func TestUsersResource_List_Passthrough(t *testing.T) {
	fake := &fakeUserListSvc{
		page: &service.AdminUsersPage{
			Items:    []service.AdminUserItem{{}, {}},
			Total:    42,
			Page:     2,
			PageSize: 10,
		},
	}
	res := NewUsersResource(fake)

	out, err := res.List(&AdminContext{}, ListQuery{Page: 2, PageSize: 10, Search: "alice"})
	if err != nil {
		t.Fatalf("List returned err: %v", err)
	}
	if out.Total != 42 {
		t.Errorf("Total: want 42, got %d", out.Total)
	}
	if out.Page != 2 {
		t.Errorf("Page: want 2, got %d", out.Page)
	}
	if out.PageSize != 10 {
		t.Errorf("PageSize: want 10, got %d", out.PageSize)
	}
	items, ok := out.Items.([]service.AdminUserItem)
	if !ok || len(items) != 2 {
		t.Errorf("Items not passed through, got %#v", out.Items)
	}
	if fake.gotKeyword != "alice" {
		t.Errorf("keyword passthrough: want alice, got %q", fake.gotKeyword)
	}
}

func TestUsersResource_List_DefaultsPaging(t *testing.T) {
	fake := &fakeUserListSvc{page: &service.AdminUsersPage{}}
	res := NewUsersResource(fake)

	if _, err := res.List(&AdminContext{}, ListQuery{Page: 0, PageSize: 0}); err != nil {
		t.Fatalf("List returned err: %v", err)
	}
	if fake.gotPage != 1 {
		t.Errorf("default page: want 1, got %d", fake.gotPage)
	}
	if fake.gotPageSize != 20 {
		t.Errorf("default pageSize: want 20, got %d", fake.gotPageSize)
	}
}

func TestUsersResource_Detail_Success(t *testing.T) {
	fake := &fakeUserListSvc{detail: &service.AdminUserDetail{ID: 7}}
	res := NewUsersResource(fake)

	out, err := res.Detail(&AdminContext{}, "7")
	if err != nil {
		t.Fatalf("Detail returned err: %v", err)
	}
	d, ok := out.(*service.AdminUserDetail)
	if !ok || d.ID != 7 {
		t.Errorf("Detail wrong result: %#v", out)
	}
	if fake.gotUserID != 7 {
		t.Errorf("parsed id: want 7, got %d", fake.gotUserID)
	}
}

func TestUsersResource_Detail_ParseFail(t *testing.T) {
	fake := &fakeUserListSvc{detail: &service.AdminUserDetail{}}
	res := NewUsersResource(fake)

	if _, err := res.Detail(&AdminContext{}, "not-a-number"); err == nil {
		t.Fatal("expected parse error for non-numeric id, got nil")
	}
	if fake.gotUserID != 0 {
		t.Errorf("service should not be called on parse failure, gotUserID=%d", fake.gotUserID)
	}
}

func TestUsersResource_Detail_SvcError(t *testing.T) {
	fake := &fakeUserListSvc{detailErr: errors.New("not found")}
	res := NewUsersResource(fake)

	if _, err := res.Detail(&AdminContext{}, "9"); err == nil {
		t.Fatal("expected service error to propagate, got nil")
	}
}
