package repository

import (
	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
)

// ReportRepo 深度报告数据访问层。
type ReportRepo struct {
	db *gorm.DB
}

func NewReportRepo(db *gorm.DB) *ReportRepo {
	return &ReportRepo{db: db}
}

// Create 创建报告记录（状态=pending）。
func (r *ReportRepo) Create(report *model.Report) error {
	return r.db.Create(report).Error
}

// GetByID 按 ID 查找报告，带 userID 归属校验防越权。
func (r *ReportRepo) GetByID(id, userID uint64) (*model.Report, error) {
	var report model.Report
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// ListByUser 列出用户所有报告。
func (r *ReportRepo) ListByUser(userID uint64, limit, offset int) ([]model.Report, error) {
	var reports []model.Report
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reports).Error
	return reports, err
}

// UpdateStatus 更新报告状态，可选写入错误信息。
func (r *ReportRepo) UpdateStatus(id uint64, status string, errMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errMsg != "" {
		updates["error_msg"] = errMsg
	} else {
		updates["error_msg"] = ""
	}
	return r.db.Model(&model.Report{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateResult 写入报告内容与 PDF 地址（成功时调用）。
func (r *ReportRepo) UpdateResult(id uint64, content model.ReportContent, pdfURL string) error {
	return r.db.Model(&model.Report{}).Where("id = ?", id).Updates(map[string]interface{}{
		"content": content,
		"pdf_url": pdfURL,
	}).Error
}

// UpdateChartID 更新报告关联的排盘 ID（handler 排盘后回写）。
func (r *ReportRepo) UpdateChartID(id uint64, chartID uint64) error {
	return r.db.Model(&model.Report{}).Where("id = ?", id).Update("chart_id", chartID).Error
}
