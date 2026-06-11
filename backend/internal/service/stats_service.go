package service

import (
	"context"
	"time"

	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// ---------- Stats DTOs ----------

type UserStats struct {
	Total    int64 `json:"total"`
	TodayNew int64 `json:"today_new"`
}

type OrderStats struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
}

type RevenueStats struct {
	TotalCents int64  `json:"total_cents"`
	TodayCents int64  `json:"today_cents"`
	Currency   string `json:"currency"`
}

type ReportStats struct {
	Total         int64            `json:"total"`
	ByStatus      map[string]int64 `json:"by_status"`
	UnlockedCount int64            `json:"unlocked_count"`
}

type Stats struct {
	Users   UserStats    `json:"users"`
	Orders  OrderStats   `json:"orders"`
	Revenue RevenueStats `json:"revenue"`
	Reports ReportStats  `json:"reports"`
}

// ---------- private interface (for test fakes) ----------

type statsStore interface {
	CountUsers() (int64, error)
	CountUsersSince(since time.Time) (int64, error)
	GetOrderAgg() (*repository.OrderAgg, error)
	GetRevenueAgg(todayStart time.Time) (*repository.RevenueAgg, error)
	GetReportAgg() (*repository.ReportAgg, error)
}

// ---------- StatsService ----------

type StatsService struct {
	repo statsStore
}

func NewStatsService(repo *repository.StatsRepo) *StatsService {
	return &StatsService{repo: repo}
}

// GetStats 组装看板统计数据。
func (s *StatsService) GetStats(ctx context.Context) (*Stats, error) {
	todayStart := time.Now().Truncate(24 * time.Hour)

	usersTotal, err := s.repo.CountUsers()
	if err != nil {
		logger.FromCtx(ctx).Error("stats: count users failed", "err", err)
		return nil, err
	}
	todayUsers, err := s.repo.CountUsersSince(todayStart)
	if err != nil {
		logger.FromCtx(ctx).Error("stats: count users since failed", "err", err)
		return nil, err
	}

	orderAgg, err := s.repo.GetOrderAgg()
	if err != nil {
		logger.FromCtx(ctx).Error("stats: get order agg failed", "err", err)
		return nil, err
	}

	revenueAgg, err := s.repo.GetRevenueAgg(todayStart)
	if err != nil {
		logger.FromCtx(ctx).Error("stats: get revenue agg failed", "err", err)
		return nil, err
	}

	reportAgg, err := s.repo.GetReportAgg()
	if err != nil {
		logger.FromCtx(ctx).Error("stats: get report agg failed", "err", err)
		return nil, err
	}

	return &Stats{
		Users: UserStats{
			Total:    usersTotal,
			TodayNew: todayUsers,
		},
		Orders: OrderStats{
			Total:    orderAgg.Total,
			ByStatus: orderAgg.ByStatus,
		},
		Revenue: RevenueStats{
			TotalCents: revenueAgg.TotalCents,
			TodayCents: revenueAgg.TodayCents,
			Currency:   "usd",
		},
		Reports: ReportStats{
			Total:         reportAgg.Total,
			ByStatus:      reportAgg.ByStatus,
			UnlockedCount: reportAgg.UnlockedCount,
		},
	}, nil
}
