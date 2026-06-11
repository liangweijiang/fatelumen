package service

import (
	"context"
	"fmt"
	"time"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// ---------- DTOs ----------

// AdminUserItem 用户列表项（脱敏，不含密码等敏感字段）。
type AdminUserItem struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminUserDetail 用户详情（脱敏）。
type AdminUserDetail struct {
	ID           uint64    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	Active       bool      `json:"active"`
	Credits      int       `json:"credits"`
	Locale       string    `json:"locale"`
	OrdersCount  int64     `json:"orders_count"`
	ReportsCount int64     `json:"reports_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AdminUsersPage 分页结果。
type AdminUsersPage struct {
	Items    []AdminUserItem `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// ---------- private interfaces ----------

type adminUserStore interface {
	ListUsers(keyword string, limit, offset int) ([]model.User, int64, error)
	GetUserByID(id uint64) (*model.User, error)
	SetUserActive(id uint64, active bool) error
	ClearCurrentToken(userID uint64) error
}

type adminCountStore interface {
	CountOrdersByUser(userID uint64) (int64, error)
	CountReportsByUser(userID uint64) (int64, error)
}

// ---------- AdminUserService ----------

type AdminUserService struct {
	userRepo adminUserStore
	countRepo adminCountStore
}

func NewAdminUserService(userRepo *repository.UserRepo, orderRepo *repository.OrderRepo, reportRepo *repository.ReportRepo) *AdminUserService {
	return &AdminUserService{
		userRepo:  userRepo,
		countRepo: &repoCountBridge{orders: orderRepo, reports: reportRepo},
	}
}

// ListUsers 分页搜索用户。
func (s *AdminUserService) ListUsers(ctx context.Context, keyword string, page, pageSize int) (*AdminUsersPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	users, total, err := s.userRepo.ListUsers(keyword, pageSize, offset)
	if err != nil {
		logger.FromCtx(ctx).Error("admin list users failed", "err", err)
		return nil, err
	}

	items := make([]AdminUserItem, len(users))
	for i, u := range users {
		items[i] = toAdminUserItem(u)
	}
	return &AdminUsersPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetUserDetail 获取用户详情（含订单/报告计数）。
func (s *AdminUserService) GetUserDetail(ctx context.Context, userID uint64) (*AdminUserDetail, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	ordersCount, _ := s.countRepo.CountOrdersByUser(userID)
	reportsCount, _ := s.countRepo.CountReportsByUser(userID)

	return &AdminUserDetail{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		Role:         user.Role,
		Active:       user.Active,
		Credits:      user.Credits,
		Locale:       user.Locale,
		OrdersCount:  ordersCount,
		ReportsCount: reportsCount,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}, nil
}

// SetUserActive 启用/禁用用户，带审计日志。禁用时同时清除令牌使其立即生效。
func (s *AdminUserService) SetUserActive(ctx context.Context, operatorID, targetUserID uint64, active bool) error {
	if err := s.userRepo.SetUserActive(targetUserID, active); err != nil {
		logger.FromCtx(ctx).Error("admin set user active failed",
			"err", err,
			"operator_id", operatorID,
			"target_user_id", targetUserID,
		)
		return err
	}
	if !active {
		if err := s.userRepo.ClearCurrentToken(targetUserID); err != nil {
			logger.FromCtx(ctx).Error("admin clear token on disable failed",
				"err", err,
				"operator_id", operatorID,
				"target_user_id", targetUserID,
			)
			// Non-fatal: active=false already written; token still invalid via active check
		}
	}
	logger.FromCtx(ctx).Info("admin set user active",
		"operator_id", operatorID,
		"target_user_id", targetUserID,
		"active", active,
	)
	return nil
}

// ---------- helpers ----------

// toAdminUserItem 将 model.User 转换为脱敏列表项。
func toAdminUserItem(u model.User) AdminUserItem {
	return AdminUserItem{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		Active:    u.Active,
		CreatedAt: u.CreatedAt,
	}
}

// ---------- count bridge ----------

type repoCountBridge struct {
	orders   interface {
		CountByUser(userID uint64) (int64, error)
	}
	reports  interface {
		CountByUser(userID uint64) (int64, error)
	}
}

func (b *repoCountBridge) CountOrdersByUser(userID uint64) (int64, error) {
	return b.orders.CountByUser(userID)
}

func (b *repoCountBridge) CountReportsByUser(userID uint64) (int64, error) {
	return b.reports.CountByUser(userID)
}
