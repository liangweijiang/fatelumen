package repository

import (
	"context"
	"time"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"

	"gorm.io/gorm"
)

// AuditRepo 后台操作审计落库。
type AuditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// Write 写一条审计记录。失败只记日志不阻断主流程(审计不应拖垮业务)。
func (r *AuditRepo) Write(ctx context.Context, e model.AdminAuditLog) {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	if err := r.db.WithContext(ctx).Create(&e).Error; err != nil {
		logger.FromCtx(ctx).Error("write audit log failed",
			"err", err,
			"action", e.Action,
			"resource", e.Resource,
			"resource_id", e.ResourceID,
		)
	}
}

// ListAudit 审计日志分页查询(供后台审计资源用)。
func (r *AuditRepo) ListAudit(ctx context.Context, resource string, limit, offset int) ([]model.AdminAuditLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.AdminAuditLog{})
	if resource != "" {
		q = q.Where("resource = ?", resource)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.AdminAuditLog
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
