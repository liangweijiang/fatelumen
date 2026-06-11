package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"fatelumen/backend/internal/repository"
)

type fakeStatsRepo struct {
	usersTotal   int64
	usersToday   int64
	orderAgg     *repository.OrderAgg
	revenueAgg   *repository.RevenueAgg
	reportAgg    *repository.ReportAgg
	err          error
}

func (f *fakeStatsRepo) CountUsers() (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.usersTotal, nil
}

func (f *fakeStatsRepo) CountUsersSince(since time.Time) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.usersToday, nil
}

func (f *fakeStatsRepo) GetOrderAgg() (*repository.OrderAgg, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.orderAgg, nil
}

func (f *fakeStatsRepo) GetRevenueAgg(todayStart time.Time) (*repository.RevenueAgg, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.revenueAgg, nil
}

func (f *fakeStatsRepo) GetReportAgg() (*repository.ReportAgg, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.reportAgg, nil
}

func TestGetStats_Success(t *testing.T) {
	repo := &fakeStatsRepo{
		usersTotal: 100,
		usersToday: 5,
		orderAgg: &repository.OrderAgg{
			Total: 50,
			ByStatus: map[string]int64{
				"created": 10, "pending": 15, "paid": 20, "failed": 3, "refunded": 2,
			},
		},
		revenueAgg: &repository.RevenueAgg{
			TotalCents: 199900,
			TodayCents: 9980,
		},
		reportAgg: &repository.ReportAgg{
			Total: 30,
			ByStatus: map[string]int64{
				"pending": 5, "processing": 3, "done": 20, "failed": 2,
			},
			UnlockedCount: 20,
		},
	}
	svc := &StatsService{repo: repo}

	stats, err := svc.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.Users.Total != 100 {
		t.Errorf("users total: want 100, got %d", stats.Users.Total)
	}
	if stats.Users.TodayNew != 5 {
		t.Errorf("users today: want 5, got %d", stats.Users.TodayNew)
	}

	if stats.Orders.Total != 50 {
		t.Errorf("orders total: want 50, got %d", stats.Orders.Total)
	}
	if stats.Orders.ByStatus["paid"] != 20 {
		t.Errorf("orders byStatus paid: want 20, got %d", stats.Orders.ByStatus["paid"])
	}
	if stats.Orders.ByStatus["created"] != 10 {
		t.Errorf("orders byStatus created: want 10, got %d", stats.Orders.ByStatus["created"])
	}

	if stats.Revenue.TotalCents != 199900 {
		t.Errorf("revenue total: want 199900, got %d", stats.Revenue.TotalCents)
	}
	if stats.Revenue.TodayCents != 9980 {
		t.Errorf("revenue today: want 9980, got %d", stats.Revenue.TodayCents)
	}
	if stats.Revenue.Currency != "usd" {
		t.Errorf("currency: want usd, got %s", stats.Revenue.Currency)
	}

	if stats.Reports.Total != 30 {
		t.Errorf("reports total: want 30, got %d", stats.Reports.Total)
	}
	if stats.Reports.ByStatus["done"] != 20 {
		t.Errorf("reports byStatus done: want 20, got %d", stats.Reports.ByStatus["done"])
	}
	if stats.Reports.UnlockedCount != 20 {
		t.Errorf("reports unlocked: want 20, got %d", stats.Reports.UnlockedCount)
	}
}

func TestGetStats_Error(t *testing.T) {
	repo := &fakeStatsRepo{err: errors.New("db down")}
	svc := &StatsService{repo: repo}

	_, err := svc.GetStats(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetStats_EmptyData(t *testing.T) {
	repo := &fakeStatsRepo{
		orderAgg:  &repository.OrderAgg{Total: 0, ByStatus: map[string]int64{}},
		revenueAgg: &repository.RevenueAgg{},
		reportAgg: &repository.ReportAgg{Total: 0, ByStatus: map[string]int64{}},
	}
	svc := &StatsService{repo: repo}

	stats, err := svc.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Orders.Total != 0 {
		t.Errorf("expected 0 orders")
	}
	if stats.Revenue.TotalCents != 0 {
		t.Errorf("expected 0 revenue")
	}
	if stats.Reports.Total != 0 {
		t.Errorf("expected 0 reports")
	}
}
