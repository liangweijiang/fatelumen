package repository

import (
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

// StatsRepo 数据看板统计查询层。
type StatsRepo struct {
	db *gorm.DB
}

func NewStatsRepo(db *gorm.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

// CountUsers 用户总数。
func (r *StatsRepo) CountUsers() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

// CountUsersSince 指定时间之后注册的用户数。
func (r *StatsRepo) CountUsersSince(since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("created_at >= ?", since).Count(&count).Error
	return count, err
}

// OrderAgg 订单聚合结果。
type OrderAgg struct {
	Total    int64
	ByStatus map[string]int64
}

// GetOrderAgg 订单总数 + 按状态分组计数。
func (r *StatsRepo) GetOrderAgg() (*OrderAgg, error) {
	var total int64
	if err := r.db.Model(&model.Order{}).Count(&total).Error; err != nil {
		return nil, err
	}

	type row struct {
		Status string
		Cnt    int64
	}
	var rows []row
	if err := r.db.Model(&model.Order{}).
		Select("status, count(*) as cnt").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	byStatus := make(map[string]int64, len(rows))
	for _, r := range rows {
		byStatus[r.Status] = r.Cnt
	}
	return &OrderAgg{Total: total, ByStatus: byStatus}, nil
}

// RevenueAgg 营收聚合结果（分）。
type RevenueAgg struct {
	TotalCents int64
	TodayCents int64
}

// GetRevenueAgg 总营收 + 今日营收（仅统计 status=paid 的已完成订单）。
func (r *StatsRepo) GetRevenueAgg(todayStart time.Time) (*RevenueAgg, error) {
	var ag RevenueAgg

	if err := r.db.Model(&model.Order{}).
		Where("status = ?", model.OrderStatusPaid).
		Select("COALESCE(SUM(amount_cents), 0)").
		Scan(&ag.TotalCents).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&model.Order{}).
		Where("status = ? AND created_at >= ?", model.OrderStatusPaid, todayStart).
		Select("COALESCE(SUM(amount_cents), 0)").
		Scan(&ag.TodayCents).Error; err != nil {
		return nil, err
	}

	return &ag, nil
}

// ReportAgg 报告聚合结果。
type ReportAgg struct {
	Total         int64
	ByStatus      map[string]int64
	UnlockedCount int64
}

// GetReportAgg 报告总数 + 按状态分组 + 已解锁数。
func (r *StatsRepo) GetReportAgg() (*ReportAgg, error) {
	var total int64
	if err := r.db.Model(&model.Report{}).Count(&total).Error; err != nil {
		return nil, err
	}

	type row struct {
		Status string
		Cnt    int64
	}
	var rows []row
	if err := r.db.Model(&model.Report{}).
		Select("status, count(*) as cnt").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	byStatus := make(map[string]int64, len(rows))
	for _, r := range rows {
		byStatus[r.Status] = r.Cnt
	}

	var unlocked int64
	if err := r.db.Model(&model.Report{}).Where("paid = ?", true).Count(&unlocked).Error; err != nil {
		return nil, err
	}

	return &ReportAgg{Total: total, ByStatus: byStatus, UnlockedCount: unlocked}, nil
}
