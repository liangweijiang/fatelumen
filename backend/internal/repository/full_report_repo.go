package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fatelumen/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInsufficientCredits      = errors.New("insufficient credits")
	ErrFullReportTerminal       = errors.New("full report is terminal")
	ErrFullReportNotReady       = errors.New("full report is not ready to complete")
	ErrFullReportChapterCount   = errors.New("full report must contain exactly ten chapters")
	ErrFullReportImmutableWrite = errors.New("immutable full report payload already exists")
)

type FullReportRepo struct {
	db *gorm.DB
}

// ReportFilter is shared by the admin report list service and the new report
// repository. It intentionally contains metadata-only filters.
type ReportFilter struct {
	Status string
	Paid   *bool
	UserID uint64
}

func NewFullReportRepo(db *gorm.DB) *FullReportRepo {
	return &FullReportRepo{db: db}
}

func (r *FullReportRepo) Create(ctx context.Context, report *model.FullReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

// CreateWithCreditCharge atomically creates a pending report and consumes the
// user's report credits. Unlimited users retain their explicit back-office
// exemption, but their report is still marked paid so delivery is not gated.
func (r *FullReportRepo) CreateWithCreditCharge(ctx context.Context, report *model.FullReport, cost int) error {
	if cost <= 0 {
		return errors.New("full report credit cost must be positive")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, report.UserID).Error; err != nil {
			return err
		}
		report.Paid = true
		if user.Unlimited {
			report.PayMethod = "unlimited"
			return tx.Create(report).Error
		}
		if user.Credits < cost {
			return ErrInsufficientCredits
		}
		report.PayMethod = "credit"
		if err := tx.Create(report).Error; err != nil {
			return err
		}
		balance := user.Credits - cost
		if err := tx.Model(&model.User{}).Where("id = ?", user.ID).Update("credits", balance).Error; err != nil {
			return err
		}
		refID := report.ID
		return tx.Create(&model.CreditLedger{
			UserID: user.ID, Delta: -cost, BalanceAfter: balance,
			Reason: "consume_report", RefID: &refID, CreatedAt: time.Now().UTC(),
		}).Error
	})
}

// CancelPendingCreditCharge compensates the narrow failure window where the
// report transaction committed but its queue delivery could not be created.
// Generation-time terminal refunds are handled by the separate failure policy.
func (r *FullReportRepo) CancelPendingCreditCharge(ctx context.Context, reportID uint64, code, summary string, failedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportID).Error; err != nil {
			return err
		}
		if report.Status == model.FullReportStatusFailed {
			return nil
		}
		if report.Status != model.FullReportStatusPending {
			return ErrFullReportImmutableWrite
		}
		if report.PayMethod == "credit" {
			var debit model.CreditLedger
			if err := tx.Where("user_id = ? AND reason = ? AND ref_id = ?", report.UserID, "consume_report", report.ID).First(&debit).Error; err != nil {
				return err
			}
			var user model.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, report.UserID).Error; err != nil {
				return err
			}
			refID := report.ID
			balance := user.Credits - debit.Delta
			if err := tx.Model(&model.User{}).Where("id = ?", user.ID).Update("credits", balance).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.CreditLedger{UserID: user.ID, Delta: -debit.Delta, BalanceAfter: balance, Reason: "refund_report_creation", RefID: &refID, CreatedAt: failedAt}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&model.FullReport{}).Where("id = ?", report.ID).Updates(map[string]any{
			"paid": false, "status": model.FullReportStatusFailed, "current_stage": model.FullReportStatusFailed,
			"error_code": code, "error_summary": summary, "failed_at": failedAt, "completed_at": failedAt, "updated_at": failedAt,
		}).Error
	})
}

func (r *FullReportRepo) UpdateChartID(ctx context.Context, reportID, chartID uint64) error {
	res := r.db.WithContext(ctx).Model(&model.FullReport{}).
		Where("id = ? AND status = ?", reportID, model.FullReportStatusPreflighting).
		Updates(map[string]any{"chart_id": chartID, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrFullReportImmutableWrite
	}
	return nil
}

// BeginPreflight claims a pending report exactly once. Duplicate deliveries
// cannot start a second execution chain for the same report.
func (r *FullReportRepo) BeginPreflight(ctx context.Context, reportID uint64, startedAt time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.FullReport{}).
		Where("id = ? AND status = ?", reportID, model.FullReportStatusPending).
		Updates(map[string]any{
			"status": model.FullReportStatusPreflighting, "current_stage": model.FullReportStatusPreflighting,
			"started_at": startedAt, "updated_at": startedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrFullReportImmutableWrite
	}
	return nil
}

type FullReportCreateGraph struct {
	Report           *model.FullReport
	Snapshot         *model.FullReportExecutionSnapshot
	ExecutionPayload *model.FullReportExecutionPayload
	Chapters         []model.FullReportChapter
	ChapterPayloads  []model.FullReportChapterPayload
}

type FullReportFreezeExecution struct {
	ReportID           uint64
	ProviderChainKey   string
	ChapterConcurrency uint8
	Snapshot           *model.FullReportExecutionSnapshot
	ExecutionPayload   *model.FullReportExecutionPayload
	Chapters           []model.FullReportChapter
	ChapterPayloads    []model.FullReportChapterPayload
}

// FreezeExecution attaches the immutable execution graph to a pending report.
// It is intentionally separate from Create because the API returns report_id
// before the asynchronous deterministic calculation starts.
func (r *FullReportRepo) FreezeExecution(ctx context.Context, in FullReportFreezeExecution) error {
	if in.Snapshot == nil || in.ExecutionPayload == nil {
		return errors.New("full report execution graph is incomplete")
	}
	if len(in.Chapters) != 10 || len(in.ChapterPayloads) != 10 {
		return ErrFullReportChapterCount
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, in.ReportID).Error; err != nil {
			return err
		}
		if model.IsFullReportTerminalStatus(report.Status) {
			return ErrFullReportTerminal
		}
		if report.Status != model.FullReportStatusPreflighting {
			return ErrFullReportImmutableWrite
		}
		var count int64
		if err := tx.Model(&model.FullReportExecutionSnapshot{}).Where("report_id = ?", in.ReportID).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return ErrFullReportImmutableWrite
		}
		in.Snapshot.ReportID = in.ReportID
		if err := tx.Create(in.Snapshot).Error; err != nil {
			return err
		}
		in.ExecutionPayload.SnapshotID = in.Snapshot.ID
		if err := tx.Create(in.ExecutionPayload).Error; err != nil {
			return err
		}
		for i := range in.Chapters {
			in.Chapters[i].ReportID = in.ReportID
			if err := tx.Create(&in.Chapters[i]).Error; err != nil {
				return err
			}
			in.ChapterPayloads[i].ChapterID = in.Chapters[i].ID
			if err := tx.Create(&in.ChapterPayloads[i]).Error; err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		return tx.Model(&model.FullReport{}).Where("id = ? AND status = ?", in.ReportID, model.FullReportStatusPreflighting).Updates(map[string]any{
			"status":              model.FullReportStatusGenerating,
			"current_stage":       model.FullReportStatusGenerating,
			"provider_chain_key":  in.ProviderChainKey,
			"chapter_concurrency": in.ChapterConcurrency,
			"facts_hash":          in.Snapshot.FactsHash,
			"execution_hash":      in.Snapshot.ExecutionHash,
			"generating_at":       now,
			"updated_at":          now,
		}).Error
	})
}

// CreateGraph atomically creates one report, its frozen execution snapshot and
// exactly ten chapter plans. It never reads or writes the legacy report tables.
func (r *FullReportRepo) CreateGraph(ctx context.Context, graph FullReportCreateGraph) error {
	if graph.Report == nil || graph.Snapshot == nil || graph.ExecutionPayload == nil {
		return errors.New("full report graph is incomplete")
	}
	if len(graph.Chapters) != 10 || len(graph.ChapterPayloads) != 10 {
		return ErrFullReportChapterCount
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(graph.Report).Error; err != nil {
			return fmt.Errorf("create full report: %w", err)
		}
		graph.Snapshot.ReportID = graph.Report.ID
		if err := tx.Create(graph.Snapshot).Error; err != nil {
			return fmt.Errorf("create full report snapshot: %w", err)
		}
		graph.ExecutionPayload.SnapshotID = graph.Snapshot.ID
		if err := tx.Create(graph.ExecutionPayload).Error; err != nil {
			return fmt.Errorf("create full report execution payload: %w", err)
		}

		for i := range graph.Chapters {
			graph.Chapters[i].ReportID = graph.Report.ID
			if err := tx.Create(&graph.Chapters[i]).Error; err != nil {
				return fmt.Errorf("create full report chapter %d: %w", i+1, err)
			}
			graph.ChapterPayloads[i].ChapterID = graph.Chapters[i].ID
			if err := tx.Create(&graph.ChapterPayloads[i]).Error; err != nil {
				return fmt.Errorf("create full report chapter payload %d: %w", i+1, err)
			}
		}
		return nil
	})
}

func (r *FullReportRepo) GetByID(ctx context.Context, reportID, userID uint64) (*model.FullReport, error) {
	var report model.FullReport
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ? AND status <> ?", reportID, userID, model.FullReportStatusDeleting).First(&report).Error
	return &report, err
}

func (r *FullReportRepo) GetByInternalID(ctx context.Context, reportID uint64) (*model.FullReport, error) {
	var row model.FullReport
	if err := r.db.WithContext(ctx).First(&row, reportID).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListInterruptedExecutions pages through reports whose generation chain did
// not reach rendering or a terminal state. It is used at process startup when
// the configured queue is in-memory and therefore cannot retain deliveries.
func (r *FullReportRepo) ListInterruptedExecutions(ctx context.Context, afterID uint64, limit int) ([]model.FullReport, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var rows []model.FullReport
	err := r.db.WithContext(ctx).
		Where("id > ? AND status IN ?", afterID, []string{model.FullReportStatusPending, model.FullReportStatusPreflighting, model.FullReportStatusGenerating, model.FullReportStatusAssembling}).
		Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *FullReportRepo) GetExecutionPayload(ctx context.Context, reportID uint64) (*model.FullReportExecutionPayload, error) {
	var row model.FullReportExecutionPayload
	err := r.db.WithContext(ctx).Table("full_report_execution_payloads AS p").
		Select("p.*").Joins("JOIN full_report_execution_snapshots AS s ON s.id = p.snapshot_id").
		Where("s.report_id = ?", reportID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// RecoverInterruptedExecution closes attempts that were left running by a
// process crash. Their consumed attempt numbers remain immutable, allowing the
// executor to continue at the next frozen route budget.
func (r *FullReportRepo) RecoverInterruptedExecution(ctx context.Context, reportID uint64, recoveredAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportID).Error; err != nil {
			return err
		}
		if report.Status != model.FullReportStatusGenerating && report.Status != model.FullReportStatusAssembling {
			return ErrFullReportImmutableWrite
		}
		return tx.Model(&model.FullReportAttempt{}).
			Where("report_id = ? AND status = ?", reportID, model.FullReportAttemptStatusRunning).
			Updates(map[string]any{
				"status":            model.FullReportAttemptStatusFailed,
				"validation_status": model.FullReportValidationStatusFailed,
				"error_code":        "worker_interrupted",
				"error_summary":     "报告任务执行中断，已由恢复流程继续处理",
				"finished_at":       recoveredAt,
			}).Error
	})
}

func (r *FullReportRepo) GetResult(ctx context.Context, reportID uint64) (*model.FullReportResult, error) {
	var result model.FullReportResult
	err := r.db.WithContext(ctx).Where("report_id = ?", reportID).First(&result).Error
	return &result, err
}

func (r *FullReportRepo) AdminGetByID(ctx context.Context, reportID uint64) (*model.FullReport, error) {
	var report model.FullReport
	err := r.db.WithContext(ctx).Where("id = ? AND status <> ?", reportID, model.FullReportStatusDeleting).First(&report).Error
	return &report, err
}

func (r *FullReportRepo) AdminGetResult(ctx context.Context, reportID uint64) (*model.FullReportResult, error) {
	var result model.FullReportResult
	err := r.db.WithContext(ctx).Model(&model.FullReportResult{}).Select("full_report_results.*").
		Joins("JOIN full_reports AS r ON r.id = full_report_results.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_results.report_id = ?", reportID).First(&result).Error
	return &result, err
}

func (r *FullReportRepo) AdminList(ctx context.Context, status string, paid *bool, userID uint64, limit, offset int) ([]model.FullReport, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.FullReport{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if paid != nil {
		q = q.Where("paid = ?", *paid)
	}
	if userID != 0 {
		q = q.Where("user_id = ?", userID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.FullReport
	err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

type AdminFullReportListFilter struct {
	Status      string
	Locale      string
	UserID      uint64
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Cursor      *FullReportCursor
}

type AdminFullReportListItem struct {
	ID               uint64     `json:"id"`
	PublicID         string     `json:"public_id"`
	UserID           uint64     `json:"user_id"`
	ProfileID        *uint64    `json:"profile_id,omitempty"`
	ProfileName      string     `json:"profile_name"`
	Locale           string     `json:"locale"`
	Status           string     `json:"status"`
	CurrentStage     string     `json:"current_stage"`
	ChapterTotal     uint8      `json:"chapter_total"`
	ChapterSucceeded uint8      `json:"chapter_succeeded"`
	ChapterFailed    uint8      `json:"chapter_failed"`
	ErrorCode        string     `json:"error_code,omitempty"`
	ErrorSummary     string     `json:"error_summary,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

func (r *FullReportRepo) AdminListPage(ctx context.Context, filter AdminFullReportListFilter, limit int) ([]AdminFullReportListItem, bool, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Table("full_reports AS r").
		Select(`r.id, r.public_id, r.user_id, r.profile_id, COALESCE(p.display_name, '') AS profile_name,
			r.locale, r.status, r.current_stage, r.chapter_total, r.chapter_succeeded, r.chapter_failed,
			r.error_code, r.error_summary, r.started_at, r.completed_at, r.created_at`).
		Joins("LEFT JOIN birth_profiles AS p ON p.id = r.profile_id")
	if filter.Status != "" {
		q = q.Where("r.status = ?", filter.Status)
	}
	if filter.Locale != "" {
		q = q.Where("r.locale = ?", filter.Locale)
	}
	if filter.UserID != 0 {
		q = q.Where("r.user_id = ?", filter.UserID)
	}
	if filter.CreatedFrom != nil {
		q = q.Where("r.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		q = q.Where("r.created_at < ?", *filter.CreatedTo)
	}
	if filter.Cursor != nil {
		q = q.Where("r.created_at < ? OR (r.created_at = ? AND r.id < ?)", filter.Cursor.CreatedAt, filter.Cursor.CreatedAt, filter.Cursor.ID)
	}
	var rows []AdminFullReportListItem
	if err := q.Order("r.created_at DESC, r.id DESC").Limit(limit + 1).Scan(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return rows, hasMore, nil
}

type FullReportCursor struct {
	CreatedAt time.Time
	ID        uint64
}

func (r *FullReportRepo) ListByUser(ctx context.Context, userID uint64, limit int, cursor *FullReportCursor) ([]model.FullReport, error) {
	q := r.db.WithContext(ctx).Where("user_id = ? AND status <> ?", userID, model.FullReportStatusDeleting)
	if cursor != nil {
		q = q.Where("created_at < ? OR (created_at = ? AND id < ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}
	var rows []model.FullReport
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *FullReportRepo) ListByUserOffset(ctx context.Context, userID uint64, limit, offset int) ([]model.FullReport, error) {
	var rows []model.FullReport
	err := r.db.WithContext(ctx).Where("user_id = ? AND status <> ?", userID, model.FullReportStatusDeleting).
		Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, err
}

func (r *FullReportRepo) CountByUser(userID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.FullReport{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

type FullReportExecutionTrace struct {
	Snapshot   model.FullReportExecutionSnapshot `json:"snapshot"`
	Payload    model.FullReportExecutionPayload  `json:"payload"`
	ModelStats []FullReportModelStats            `json:"model_stats"`
}

type FullReportModelStats struct {
	RouteNo            uint8  `json:"route_no"`
	Provider           string `json:"provider"`
	Model              string `json:"model"`
	AttemptCount       int64  `json:"attempt_count"`
	SucceededCount     int64  `json:"succeeded_count"`
	FailedCount        int64  `json:"failed_count"`
	UsageReportedCount int64  `json:"usage_reported_count"`
	PromptTokens       *int64 `json:"prompt_tokens"`
	CompletionTokens   *int64 `json:"completion_tokens"`
	TotalTokens        *int64 `json:"total_tokens"`
	DurationMS         int64  `json:"duration_ms"`
}

type FullReportCreditSettlement struct {
	Status   string               `json:"status"`
	Charged  int                  `json:"charged"`
	Refunded int                  `json:"refunded"`
	Net      int                  `json:"net"`
	Entries  []model.CreditLedger `json:"entries"`
}

func (r *FullReportRepo) AdminCreditSettlement(ctx context.Context, report *model.FullReport) (*FullReportCreditSettlement, error) {
	settlement := &FullReportCreditSettlement{Status: "not_charged", Entries: make([]model.CreditLedger, 0)}
	if report.PayMethod == "unlimited" {
		settlement.Status = "exempt"
		return settlement, nil
	}
	if err := r.db.WithContext(ctx).
		Where("ref_id = ? AND reason IN ?", report.ID, []string{"consume_report", "refund_report", "refund_report_creation"}).
		Order("id ASC").Find(&settlement.Entries).Error; err != nil {
		return nil, err
	}
	for _, entry := range settlement.Entries {
		switch entry.Reason {
		case "consume_report":
			if entry.Delta < 0 {
				settlement.Charged += -entry.Delta
			}
		case "refund_report", "refund_report_creation":
			if entry.Delta > 0 {
				settlement.Refunded += entry.Delta
			}
		}
	}
	settlement.Net = settlement.Charged - settlement.Refunded
	if settlement.Refunded > 0 {
		settlement.Status = "refunded"
	} else if settlement.Charged > 0 {
		settlement.Status = "charged"
	}
	return settlement, nil
}

func (r *FullReportRepo) AdminGetExecutionTrace(ctx context.Context, reportID uint64) (*FullReportExecutionTrace, error) {
	var snapshot model.FullReportExecutionSnapshot
	if err := r.db.WithContext(ctx).Model(&model.FullReportExecutionSnapshot{}).
		Select("full_report_execution_snapshots.*").
		Joins("JOIN full_reports AS r ON r.id = full_report_execution_snapshots.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_execution_snapshots.report_id = ?", reportID).First(&snapshot).Error; err != nil {
		return nil, err
	}
	var payload model.FullReportExecutionPayload
	if err := r.db.WithContext(ctx).Where("snapshot_id = ?", snapshot.ID).First(&payload).Error; err != nil {
		return nil, err
	}
	return &FullReportExecutionTrace{Snapshot: snapshot, Payload: payload}, nil
}

// AdminGetExecutionSection projects exactly one frozen JSON field so large
// execution payloads are never loaded as a whole for section views.
func (r *FullReportRepo) AdminGetExecutionSection(ctx context.Context, reportID uint64, column string) (model.JSONRaw, error) {
	allowed := map[string]bool{
		"input_snapshot": true, "time_calculation_snapshot": true,
		"chart_snapshot": true, "facts_snapshot": true, "preflight_result": true,
	}
	if !allowed[column] {
		return nil, errors.New("unsupported execution payload section")
	}
	var projection struct {
		Value model.JSONRaw `gorm:"column:value"`
	}
	result := r.db.WithContext(ctx).Table("full_report_execution_payloads AS p").
		Select("p."+column+" AS value").
		Joins("JOIN full_report_execution_snapshots AS s ON s.id = p.snapshot_id").
		Joins("JOIN full_reports AS r ON r.id = s.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("s.report_id = ?", reportID).Scan(&projection)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return projection.Value, nil
}

func (r *FullReportRepo) AdminAttemptStats(ctx context.Context, reportID uint64) ([]FullReportModelStats, error) {
	stats := make([]FullReportModelStats, 0)
	err := r.db.WithContext(ctx).Model(&model.FullReportAttempt{}).
		Select(`full_report_attempts.route_no AS route_no, full_report_attempts.provider AS provider, full_report_attempts.model AS model,
			COUNT(*) AS attempt_count,
			SUM(CASE WHEN full_report_attempts.status = ? THEN 1 ELSE 0 END) AS succeeded_count,
			SUM(CASE WHEN full_report_attempts.status IN (?, ?) THEN 1 ELSE 0 END) AS failed_count,
			SUM(CASE WHEN full_report_attempts.total_tokens IS NOT NULL THEN 1 ELSE 0 END) AS usage_reported_count,
			SUM(full_report_attempts.prompt_tokens) AS prompt_tokens, SUM(full_report_attempts.completion_tokens) AS completion_tokens,
			SUM(full_report_attempts.total_tokens) AS total_tokens, SUM(full_report_attempts.duration_ms) AS duration_ms`,
			model.FullReportAttemptStatusSucceeded, model.FullReportAttemptStatusFailed, model.FullReportAttemptStatusRejected).
		Joins("JOIN full_reports AS r ON r.id = full_report_attempts.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_attempts.report_id = ?", reportID).
		Group("full_report_attempts.route_no, full_report_attempts.provider, full_report_attempts.model").
		Order("full_report_attempts.route_no ASC").Scan(&stats).Error
	return stats, err
}

type FullReportAttemptTrace struct {
	Attempt model.FullReportAttempt        `json:"attempt"`
	Payload model.FullReportAttemptPayload `json:"payload"`
	Chapter model.FullReportChapter        `json:"chapter"`
}

func (r *FullReportRepo) AdminListAttempts(ctx context.Context, reportID, chapterID uint64, limit, offset int) ([]model.FullReportAttempt, int64, error) {
	if _, err := r.AdminGetByID(ctx, reportID); err != nil {
		return nil, 0, err
	}
	q := r.db.WithContext(ctx).Model(&model.FullReportAttempt{}).
		Joins("JOIN full_reports AS r ON r.id = full_report_attempts.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_attempts.report_id = ?", reportID)
	if chapterID != 0 {
		q = q.Where("full_report_attempts.chapter_id = ?", chapterID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.FullReportAttempt
	err := q.Select("full_report_attempts.*").Order("full_report_attempts.attempt_no DESC, full_report_attempts.id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *FullReportRepo) AdminGetAttemptValidation(ctx context.Context, reportID, attemptID uint64) (model.JSONRaw, error) {
	var projection struct {
		Value model.JSONRaw `gorm:"column:value"`
	}
	result := r.db.WithContext(ctx).Table("full_report_attempt_payloads AS p").
		Select("p.validation_result AS value").
		Joins("JOIN full_report_attempts AS a ON a.id = p.attempt_id").
		Joins("JOIN full_reports AS r ON r.id = a.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("a.id = ? AND a.report_id = ?", attemptID, reportID).Scan(&projection)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	return projection.Value, nil
}

func (r *FullReportRepo) AdminGetAttemptTrace(ctx context.Context, reportID, attemptID uint64) (*FullReportAttemptTrace, error) {
	var attempt model.FullReportAttempt
	if err := r.db.WithContext(ctx).Model(&model.FullReportAttempt{}).Select("full_report_attempts.*").
		Joins("JOIN full_reports AS r ON r.id = full_report_attempts.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_attempts.id = ? AND full_report_attempts.report_id = ?", attemptID, reportID).First(&attempt).Error; err != nil {
		return nil, err
	}
	var payload model.FullReportAttemptPayload
	if err := r.db.WithContext(ctx).Select("id", "attempt_id", "request_parameters", "request_prompt", "raw_output", "parsed_output", "schema_errors", "created_at").Where("attempt_id = ?", attempt.ID).First(&payload).Error; err != nil {
		return nil, err
	}
	var chapter model.FullReportChapter
	if err := r.db.WithContext(ctx).First(&chapter, attempt.ChapterID).Error; err != nil {
		return nil, err
	}
	return &FullReportAttemptTrace{Attempt: attempt, Payload: payload, Chapter: chapter}, nil
}

type FullReportChapterWithPayload struct {
	Chapter model.FullReportChapter        `json:"chapter"`
	Payload model.FullReportChapterPayload `json:"payload"`
}

type FullReportValidationTrace struct {
	Run     model.FullReportValidationRun     `json:"run"`
	Payload model.FullReportValidationPayload `json:"payload"`
}

func (r *FullReportRepo) AdminListValidationRuns(ctx context.Context, reportID uint64) ([]model.FullReportValidationRun, error) {
	var rows []model.FullReportValidationRun
	if err := r.db.WithContext(ctx).Model(&model.FullReportValidationRun{}).Select("full_report_validation_runs.*").
		Joins("JOIN full_reports AS r ON r.id = full_report_validation_runs.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_validation_runs.report_id = ?", reportID).Order("full_report_validation_runs.round_no DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		if _, err := r.AdminGetByID(ctx, reportID); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (r *FullReportRepo) AdminGetValidationTrace(ctx context.Context, reportID, validationID uint64) (*FullReportValidationTrace, error) {
	var run model.FullReportValidationRun
	if err := r.db.WithContext(ctx).Model(&model.FullReportValidationRun{}).Select("full_report_validation_runs.*").
		Joins("JOIN full_reports AS r ON r.id = full_report_validation_runs.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_validation_runs.id = ? AND full_report_validation_runs.report_id = ?", validationID, reportID).First(&run).Error; err != nil {
		return nil, err
	}
	var payload model.FullReportValidationPayload
	if err := r.db.WithContext(ctx).Where("validation_run_id = ?", run.ID).First(&payload).Error; err != nil {
		return nil, err
	}
	return &FullReportValidationTrace{Run: run, Payload: payload}, nil
}

func (r *FullReportRepo) AdminListChapters(ctx context.Context, reportID uint64) ([]model.FullReportChapter, error) {
	var chapters []model.FullReportChapter
	if err := r.db.WithContext(ctx).Model(&model.FullReportChapter{}).Select("full_report_chapters.*").
		Joins("JOIN full_reports AS r ON r.id = full_report_chapters.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_chapters.report_id = ?", reportID).Order("full_report_chapters.chapter_no ASC").Find(&chapters).Error; err != nil {
		return nil, err
	}
	if len(chapters) == 0 {
		if _, err := r.AdminGetByID(ctx, reportID); err != nil {
			return nil, err
		}
	}
	return chapters, nil
}

func (r *FullReportRepo) AdminGetChapterTrace(ctx context.Context, reportID, chapterID uint64) (*FullReportChapterWithPayload, error) {
	var chapter model.FullReportChapter
	if err := r.db.WithContext(ctx).Model(&model.FullReportChapter{}).Select("full_report_chapters.*").
		Joins("JOIN full_reports AS r ON r.id = full_report_chapters.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_chapters.id = ? AND full_report_chapters.report_id = ?", chapterID, reportID).First(&chapter).Error; err != nil {
		return nil, err
	}
	var payload model.FullReportChapterPayload
	if err := r.db.WithContext(ctx).Where("chapter_id = ?", chapter.ID).First(&payload).Error; err != nil {
		return nil, err
	}
	return &FullReportChapterWithPayload{Chapter: chapter, Payload: payload}, nil
}

type FullReportChapterArtifact struct {
	Chapter model.FullReportChapter `json:"chapter"`
	Type    string                  `json:"type"`
	Value   any                     `json:"value"`
}

// AdminGetChapterArtifact selects one large field at a time. Generated
// artifacts prefer the immutable selected attempt and fall back to legacy
// chapter payload copies for historical reports.
func (r *FullReportRepo) AdminGetChapterArtifact(ctx context.Context, reportID, chapterID uint64, artifact string) (*FullReportChapterArtifact, error) {
	var chapter model.FullReportChapter
	if err := r.db.WithContext(ctx).Model(&model.FullReportChapter{}).Select("full_report_chapters.*").
		Joins("JOIN full_reports AS r ON r.id = full_report_chapters.report_id AND r.status <> ?", model.FullReportStatusDeleting).
		Where("full_report_chapters.id = ? AND full_report_chapters.report_id = ?", chapterID, reportID).First(&chapter).Error; err != nil {
		return nil, err
	}
	switch artifact {
	case "content", "validation":
		var projection struct {
			Value model.JSONRaw `gorm:"column:value"`
		}
		attemptColumn := map[string]string{"content": "parsed_output", "raw_output": "raw_output", "validation": "validation_result"}[artifact]
		if chapter.SelectedAttemptID != nil {
			if err := r.db.WithContext(ctx).Table("full_report_attempt_payloads").Select(attemptColumn+" AS value").Where("attempt_id = ?", *chapter.SelectedAttemptID).Scan(&projection).Error; err != nil {
				return nil, err
			}
		}
		if len(projection.Value) == 0 || string(projection.Value) == "null" {
			legacyColumn := map[string]string{"content": "final_parsed_output", "raw_output": "final_raw_output", "validation": "validation_result"}[artifact]
			if err := r.db.WithContext(ctx).Table("full_report_chapter_payloads").Select(legacyColumn+" AS value").Where("chapter_id = ?", chapterID).Scan(&projection).Error; err != nil {
				return nil, err
			}
		}
		return &FullReportChapterArtifact{Chapter: chapter, Type: artifact, Value: projection.Value}, nil
	case "raw_output":
		var value string
		if chapter.SelectedAttemptID != nil {
			if err := r.db.WithContext(ctx).Table("full_report_attempt_payloads").Select("raw_output").Where("attempt_id = ?", *chapter.SelectedAttemptID).Scan(&value).Error; err != nil {
				return nil, err
			}
		}
		if value == "" {
			if err := r.db.WithContext(ctx).Table("full_report_chapter_payloads").Select("final_raw_output").Where("chapter_id = ?", chapterID).Scan(&value).Error; err != nil {
				return nil, err
			}
		}
		return &FullReportChapterArtifact{Chapter: chapter, Type: artifact, Value: value}, nil
	case "prompt":
		var value string
		if err := r.db.WithContext(ctx).Table("full_report_chapter_payloads").Select("final_prompt").Where("chapter_id = ?", chapterID).Scan(&value).Error; err != nil {
			return nil, err
		}
		return &FullReportChapterArtifact{Chapter: chapter, Type: artifact, Value: value}, nil
	case "terminology":
		var projection struct {
			Value model.JSONRaw `gorm:"column:value"`
		}
		if err := r.db.WithContext(ctx).Table("full_report_chapter_payloads").Select("terminology_snapshot AS value").Where("chapter_id = ?", chapterID).Scan(&projection).Error; err != nil {
			return nil, err
		}
		return &FullReportChapterArtifact{Chapter: chapter, Type: artifact, Value: projection.Value}, nil
	default:
		return nil, errors.New("unsupported chapter artifact")
	}
}

// ListChaptersWithPayload uses two bounded queries, not one query per chapter.
func (r *FullReportRepo) ListChaptersWithPayload(ctx context.Context, reportID uint64) ([]FullReportChapterWithPayload, error) {
	var chapters []model.FullReportChapter
	if err := r.db.WithContext(ctx).Where("report_id = ?", reportID).Order("chapter_no ASC").Find(&chapters).Error; err != nil {
		return nil, err
	}
	if len(chapters) == 0 {
		return []FullReportChapterWithPayload{}, nil
	}
	ids := make([]uint64, len(chapters))
	for i := range chapters {
		ids[i] = chapters[i].ID
	}
	var payloads []model.FullReportChapterPayload
	if err := r.db.WithContext(ctx).Where("chapter_id IN ?", ids).Find(&payloads).Error; err != nil {
		return nil, err
	}
	byChapter := make(map[uint64]model.FullReportChapterPayload, len(payloads))
	for _, payload := range payloads {
		byChapter[payload.ChapterID] = payload
	}
	rows := make([]FullReportChapterWithPayload, len(chapters))
	for i, chapter := range chapters {
		rows[i] = FullReportChapterWithPayload{Chapter: chapter, Payload: byChapter[chapter.ID]}
	}
	return rows, nil
}

// BeginAssembling advances the report only after all chapter workers have
// returned. Report-level validation runs in this explicit state.
func (r *FullReportRepo) BeginAssembling(ctx context.Context, reportID uint64, at time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.FullReport{}).
		Where("id = ? AND status = ?", reportID, model.FullReportStatusGenerating).
		Updates(map[string]any{"status": model.FullReportStatusAssembling, "current_stage": model.FullReportStatusAssembling, "assembling_at": at, "updated_at": at})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrFullReportImmutableWrite
	}
	return nil
}

// SaveAggregateValidation appends one immutable report-level validation round.
// Failed rounds remain queryable after the report reaches a terminal state.
func (r *FullReportRepo) SaveAggregateValidation(ctx context.Context, run *model.FullReportValidationRun, payload *model.FullReportValidationPayload) error {
	if run == nil || payload == nil {
		return errors.New("full report aggregate validation is incomplete")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, run.ReportID).Error; err != nil {
			return err
		}
		if model.IsFullReportTerminalStatus(report.Status) {
			return ErrFullReportTerminal
		}
		if report.Status != model.FullReportStatusAssembling {
			return ErrFullReportImmutableWrite
		}
		if run.RoundNo == 0 {
			var latest uint16
			if err := tx.Model(&model.FullReportValidationRun{}).Where("report_id = ?", run.ReportID).Select("COALESCE(MAX(round_no), 0)").Scan(&latest).Error; err != nil {
				return err
			}
			run.RoundNo = latest + 1
		}
		if err := tx.Create(run).Error; err != nil {
			return err
		}
		payload.ValidationRunID = run.ID
		return tx.Create(payload).Error
	})
}

// PrepareAggregateRetry moves only the chapters identified by a retryable
// report-level validation failure back to pending. Previous attempts and the
// failed aggregate validation round remain immutable and queryable.
func (r *FullReportRepo) PrepareAggregateRetry(ctx context.Context, reportID uint64, chapterNos []uint8, at time.Time) error {
	if len(chapterNos) == 0 {
		return errors.New("aggregate retry has no affected chapters")
	}
	chapterNoValues := make([]int, len(chapterNos))
	for i, chapterNo := range chapterNos {
		chapterNoValues[i] = int(chapterNo)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportID).Error; err != nil {
			return err
		}
		if model.IsFullReportTerminalStatus(report.Status) {
			return ErrFullReportTerminal
		}
		if report.Status != model.FullReportStatusAssembling {
			return ErrFullReportImmutableWrite
		}
		var chapters []model.FullReportChapter
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("report_id = ? AND chapter_no IN ?", reportID, chapterNoValues).
			Find(&chapters).Error; err != nil {
			return err
		}
		if len(chapters) != len(chapterNos) {
			return ErrFullReportChapterCount
		}
		chapterIDs := make([]uint64, 0, len(chapters))
		for _, chapter := range chapters {
			if chapter.Status != model.FullReportChapterStatusSucceeded || chapter.SelectedAttemptID == nil {
				return ErrFullReportImmutableWrite
			}
			chapterIDs = append(chapterIDs, chapter.ID)
		}
		if err := tx.Model(&model.FullReportChapterPayload{}).Where("chapter_id IN ?", chapterIDs).Updates(map[string]any{
			"final_raw_output": "", "final_parsed_output": nil, "validation_result": nil,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.FullReportChapter{}).Where("id IN ?", chapterIDs).Updates(map[string]any{
			"status": model.FullReportChapterStatusPending, "selected_attempt_id": nil,
			"output_hash": "", "schema_valid": false, "validation_status": model.FullReportValidationStatusPending,
			"error_code": "", "error_summary": "", "completed_at": nil, "updated_at": at,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&model.FullReport{}).Where("id = ?", reportID).Updates(map[string]any{
			"status": model.FullReportStatusGenerating, "current_stage": model.FullReportStatusGenerating,
			"chapter_succeeded": gorm.Expr("chapter_succeeded - ?", len(chapters)), "updated_at": at,
		}).Error
	})
}

// AppendAttempt appends one immutable model request. A terminal report cannot
// receive new attempts.
func (r *FullReportRepo) AppendAttempt(ctx context.Context, attempt *model.FullReportAttempt, payload *model.FullReportAttemptPayload) error {
	if attempt == nil || payload == nil {
		return errors.New("full report attempt is incomplete")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, attempt.ReportID).Error; err != nil {
			return err
		}
		if model.IsFullReportTerminalStatus(report.Status) {
			return ErrFullReportTerminal
		}
		if report.Status != model.FullReportStatusGenerating {
			return ErrFullReportImmutableWrite
		}
		var chapter model.FullReportChapter
		if err := tx.Where("id = ? AND report_id = ?", attempt.ChapterID, attempt.ReportID).First(&chapter).Error; err != nil {
			return err
		}
		if err := tx.Create(attempt).Error; err != nil {
			return err
		}
		payload.AttemptID = attempt.ID
		if err := tx.Create(payload).Error; err != nil {
			return err
		}
		return tx.Model(&model.FullReportChapter{}).
			Where("id = ? AND report_id = ?", chapter.ID, report.ID).
			UpdateColumn("attempt_count", gorm.Expr("attempt_count + 1")).Error
	})
}

type FullReportAttemptOutcome struct {
	Status           string
	SchemaValid      bool
	ValidationStatus string
	ErrorCode        string
	ErrorSummary     string
	OutputHash       string
	PromptTokens     *int
	CompletionTokens *int
	TotalTokens      *int
	DurationMS       int64
	FinishedAt       time.Time
	RawOutput        string
	ParsedOutput     model.JSONRaw
	SchemaErrors     model.JSONRaw
	ValidationResult model.JSONRaw
}

func (r *FullReportRepo) FinishAttempt(ctx context.Context, reportID, attemptID uint64, outcome FullReportAttemptOutcome) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportID).Error; err != nil {
			return err
		}
		if model.IsFullReportTerminalStatus(report.Status) {
			return ErrFullReportTerminal
		}
		if report.Status != model.FullReportStatusGenerating {
			return ErrFullReportImmutableWrite
		}
		updates := map[string]any{
			"status": outcome.Status, "schema_valid": outcome.SchemaValid, "validation_status": outcome.ValidationStatus,
			"error_code": outcome.ErrorCode, "error_summary": outcome.ErrorSummary, "output_hash": outcome.OutputHash,
			"prompt_tokens": outcome.PromptTokens, "completion_tokens": outcome.CompletionTokens, "total_tokens": outcome.TotalTokens,
			"duration_ms": outcome.DurationMS, "finished_at": outcome.FinishedAt,
		}
		res := tx.Model(&model.FullReportAttempt{}).Where("id = ? AND report_id = ?", attemptID, reportID).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&model.FullReportAttemptPayload{}).Where("attempt_id = ?", attemptID).Updates(map[string]any{
			"raw_output": outcome.RawOutput, "parsed_output": outcome.ParsedOutput,
			"schema_errors": outcome.SchemaErrors, "validation_result": outcome.ValidationResult,
		}).Error
	})
}

func (r *FullReportRepo) FinishChapter(ctx context.Context, reportID, chapterID, attemptID uint64, raw string, parsed, validation model.JSONRaw, outputHash string, finishedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportID).Error; err != nil {
			return err
		}
		if model.IsFullReportTerminalStatus(report.Status) {
			return ErrFullReportTerminal
		}
		if report.Status != model.FullReportStatusGenerating {
			return ErrFullReportImmutableWrite
		}
		if err := tx.Model(&model.FullReportChapterPayload{}).Where("chapter_id = ?", chapterID).Updates(map[string]any{
			"final_raw_output": raw, "final_parsed_output": parsed, "validation_result": validation,
		}).Error; err != nil {
			return err
		}
		res := tx.Model(&model.FullReportChapter{}).Where("id = ? AND report_id = ? AND status NOT IN ?", chapterID, reportID, []string{model.FullReportChapterStatusSucceeded, model.FullReportChapterStatusFailed}).Updates(map[string]any{
			"status": model.FullReportChapterStatusSucceeded, "selected_attempt_id": attemptID,
			"output_hash": outputHash, "schema_valid": true, "validation_status": model.FullReportValidationStatusPassed,
			"completed_at": finishedAt, "updated_at": finishedAt,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrFullReportImmutableWrite
		}
		return tx.Model(&model.FullReport{}).Where("id = ?", reportID).Updates(map[string]any{
			"chapter_succeeded": gorm.Expr("chapter_succeeded + 1"), "updated_at": finishedAt,
		}).Error
	})
}

// PrepareRendering atomically stores the assembled immutable content, advances
// the report to rendering and creates the durable PDF task.
func (r *FullReportRepo) PrepareRendering(ctx context.Context, reportID uint64, result *model.FullReportResult, renderVersion string, maxAttempts uint16, at time.Time) (*model.FullReportRenderJob, error) {
	if result == nil {
		return nil, errors.New("full report result is nil")
	}
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	var task model.FullReportRenderJob
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportID).Error; err != nil {
			return err
		}
		if report.Status != model.FullReportStatusAssembling {
			return ErrFullReportImmutableWrite
		}
		var passed int64
		if err := tx.Model(&model.FullReportChapter{}).Where("report_id = ? AND status = ? AND schema_valid = ? AND validation_status = ?", reportID, model.FullReportChapterStatusSucceeded, true, model.FullReportValidationStatusPassed).Count(&passed).Error; err != nil {
			return err
		}
		if passed != 10 {
			return ErrFullReportNotReady
		}
		var aggregate model.FullReportValidationRun
		if err := tx.Where("report_id = ?", reportID).Order("round_no DESC").First(&aggregate).Error; err != nil || aggregate.Status != model.FullReportValidationStatusPassed {
			return ErrFullReportNotReady
		}
		result.ReportID = reportID
		result.RenderVersion = renderVersion
		result.CreatedAt = at
		if err := tx.Create(result).Error; err != nil {
			return err
		}
		task = model.FullReportRenderJob{ReportID: reportID, RenderVersion: renderVersion, Status: model.FullReportRenderJobStatusQueued, MaxAttempts: maxAttempts, CreatedAt: at, UpdatedAt: at}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		res := tx.Model(&model.FullReport{}).Where("id = ? AND status = ?", reportID, model.FullReportStatusAssembling).Updates(map[string]any{
			"status": model.FullReportStatusRendering, "current_stage": model.FullReportStatusRendering,
			"rendering_at": at, "content_hash": result.ContentHash, "updated_at": at,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrFullReportImmutableWrite
		}
		return nil
	})
	return &task, err
}

// CompleteRendering freezes the report only after the PDF bytes have been
// uploaded and their raw SHA-256 has been persisted.
func (r *FullReportRepo) CompleteRendering(ctx context.Context, reportID uint64, storageKey, url, pdfHash string, completedAt time.Time) error {
	if storageKey == "" || url == "" || len(pdfHash) != 64 {
		return ErrFullReportNotReady
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportID).Error; err != nil {
			return err
		}
		if report.Status == model.FullReportStatusCompleted {
			return nil
		}
		if report.Status != model.FullReportStatusRendering {
			return ErrFullReportImmutableWrite
		}
		res := tx.Model(&model.FullReportResult{}).Where("report_id = ?", reportID).Updates(map[string]any{"pdf_storage_key": storageKey, "pdf_url": url, "pdf_hash": pdfHash})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrFullReportNotReady
		}
		res = tx.Model(&model.FullReport{}).Where("id = ? AND status = ?", reportID, model.FullReportStatusRendering).Updates(map[string]any{
			"status": model.FullReportStatusCompleted, "current_stage": model.FullReportStatusCompleted,
			"chapter_succeeded": 10, "chapter_failed": 0, "completed_at": completedAt, "updated_at": completedAt,
		})
		return res.Error
	})
}

func (r *FullReportRepo) Fail(ctx context.Context, reportID uint64, code, summary string, failedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportID).Error; err != nil {
			return err
		}
		if model.IsFullReportTerminalStatus(report.Status) {
			return ErrFullReportTerminal
		}

		refunded := false
		if report.Paid && report.PayMethod == "credit" {
			var debit model.CreditLedger
			if err := tx.Where("user_id = ? AND reason = ? AND ref_id = ?", report.UserID, "consume_report", report.ID).First(&debit).Error; err != nil {
				return fmt.Errorf("find report credit debit: %w", err)
			}
			if debit.Delta >= 0 {
				return errors.New("report credit debit has invalid delta")
			}
			var user model.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, report.UserID).Error; err != nil {
				return err
			}
			balance := user.Credits - debit.Delta
			if err := tx.Model(&model.User{}).Where("id = ?", user.ID).Update("credits", balance).Error; err != nil {
				return err
			}
			refID := report.ID
			if err := tx.Create(&model.CreditLedger{
				UserID: user.ID, Delta: -debit.Delta, BalanceAfter: balance,
				Reason: "refund_report", RefID: &refID, CreatedAt: failedAt,
			}).Error; err != nil {
				return err
			}
			refunded = true
		}

		// A failed report is terminal. Converge every unfinished chapter in the
		// same transaction so list/detail views never expose pending work under
		// a terminal report. Existing successes and chapter-specific failures
		// remain immutable.
		if err := tx.Model(&model.FullReportChapter{}).
			Where("report_id = ? AND status NOT IN ?", reportID, []string{model.FullReportChapterStatusSucceeded, model.FullReportChapterStatusFailed}).
			Updates(map[string]any{
				"status":            model.FullReportChapterStatusFailed,
				"validation_status": model.FullReportValidationStatusFailed,
				"error_code":        code,
				"error_summary":     summary,
				"completed_at":      failedAt,
				"updated_at":        failedAt,
			}).Error; err != nil {
			return err
		}

		var succeeded, failed int64
		if err := tx.Model(&model.FullReportChapter{}).Where("report_id = ? AND status = ?", reportID, model.FullReportChapterStatusSucceeded).Count(&succeeded).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.FullReportChapter{}).Where("report_id = ? AND status = ?", reportID, model.FullReportChapterStatusFailed).Count(&failed).Error; err != nil {
			return err
		}

		updates := map[string]any{
			"status":            model.FullReportStatusFailed,
			"current_stage":     model.FullReportStatusFailed,
			"chapter_succeeded": succeeded,
			"chapter_failed":    failed,
			"error_code":        code,
			"error_summary":     summary,
			"failed_at":         failedAt,
			"completed_at":      failedAt,
			"updated_at":        failedAt,
		}
		if refunded {
			updates["paid"] = false
		}
		res := tx.Model(&model.FullReport{}).Where("id = ?", reportID).Updates(updates)
		return res.Error
	})
}
