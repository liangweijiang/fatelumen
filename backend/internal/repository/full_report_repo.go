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

func (r *FullReportRepo) UpdateChartID(ctx context.Context, reportID, chartID uint64) error {
	res := r.db.WithContext(ctx).Model(&model.FullReport{}).
		Where("id = ? AND status NOT IN ?", reportID, []string{model.FullReportStatusCompleted, model.FullReportStatusFailed}).
		Updates(map[string]any{"chart_id": chartID, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrFullReportTerminal
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
	ReportID         uint64
	Snapshot         *model.FullReportExecutionSnapshot
	ExecutionPayload *model.FullReportExecutionPayload
	Chapters         []model.FullReportChapter
	ChapterPayloads  []model.FullReportChapterPayload
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
		return tx.Model(&model.FullReport{}).Where("id = ?", in.ReportID).Updates(map[string]any{
			"status":         model.FullReportStatusGenerating,
			"current_stage":  model.FullReportStatusGenerating,
			"facts_hash":     in.Snapshot.FactsHash,
			"execution_hash": in.Snapshot.ExecutionHash,
			"started_at":     time.Now().UTC(),
			"updated_at":     time.Now().UTC(),
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
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", reportID, userID).First(&report).Error
	return &report, err
}

func (r *FullReportRepo) GetResult(ctx context.Context, reportID uint64) (*model.FullReportResult, error) {
	var result model.FullReportResult
	err := r.db.WithContext(ctx).Where("report_id = ?", reportID).First(&result).Error
	return &result, err
}

func (r *FullReportRepo) UpdatePDF(ctx context.Context, reportID uint64, storageKey, pdfURL, pdfHash string) error {
	var report model.FullReport
	if err := r.db.WithContext(ctx).Where("id = ? AND status = ?", reportID, model.FullReportStatusCompleted).First(&report).Error; err != nil {
		return err
	}
	res := r.db.WithContext(ctx).Model(&model.FullReportResult{}).Where("report_id = ?", reportID).Updates(map[string]any{
		"pdf_storage_key": storageKey,
		"pdf_url":         pdfURL,
		"pdf_hash":        pdfHash,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *FullReportRepo) UnlockWithCredits(ctx context.Context, userID, reportID uint64, cost int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", reportID, userID).First(&report).Error; err != nil {
			return err
		}
		if report.Paid {
			return nil
		}
		if report.Status != model.FullReportStatusCompleted {
			return ErrFullReportNotReady
		}
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}
		if user.Credits < cost {
			return ErrInsufficientCredits
		}
		if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("credits", gorm.Expr("credits - ?", cost)).Error; err != nil {
			return err
		}
		balance := user.Credits - cost
		refID := reportID
		if err := tx.Create(&model.CreditLedger{UserID: userID, Delta: -cost, BalanceAfter: balance, Reason: "unlock_full_report", RefID: &refID, CreatedAt: time.Now().UTC()}).Error; err != nil {
			return err
		}
		return tx.Model(&model.FullReport{}).Where("id = ?", reportID).Updates(map[string]any{"paid": true, "pay_method": "credit", "updated_at": time.Now().UTC()}).Error
	})
}

func (r *FullReportRepo) MarkPaid(ctx context.Context, reportID, orderID uint64, payMethod string) error {
	return r.db.WithContext(ctx).Model(&model.FullReport{}).Where("id = ?", reportID).Updates(map[string]any{
		"paid": true, "order_id": orderID, "pay_method": payMethod, "updated_at": time.Now().UTC(),
	}).Error
}

func (r *FullReportRepo) AdminMarkPaid(ctx context.Context, reportID uint64, payMethod string) error {
	res := r.db.WithContext(ctx).Model(&model.FullReport{}).Where("id = ?", reportID).
		Updates(map[string]any{"paid": true, "pay_method": payMethod, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *FullReportRepo) AdminGetByID(ctx context.Context, reportID uint64) (*model.FullReport, error) {
	var report model.FullReport
	err := r.db.WithContext(ctx).First(&report, reportID).Error
	return &report, err
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

type FullReportCursor struct {
	CreatedAt time.Time
	ID        uint64
}

func (r *FullReportRepo) ListByUser(ctx context.Context, userID uint64, limit int, cursor *FullReportCursor) ([]model.FullReport, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if cursor != nil {
		q = q.Where("created_at < ? OR (created_at = ? AND id < ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}
	var rows []model.FullReport
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *FullReportRepo) ListByUserOffset(ctx context.Context, userID uint64, limit, offset int) ([]model.FullReport, error) {
	var rows []model.FullReport
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, err
}

func (r *FullReportRepo) CountByUser(userID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.FullReport{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

type FullReportExecutionTrace struct {
	Snapshot model.FullReportExecutionSnapshot
	Payload  model.FullReportExecutionPayload
}

func (r *FullReportRepo) AdminGetExecutionTrace(ctx context.Context, reportID uint64) (*FullReportExecutionTrace, error) {
	var snapshot model.FullReportExecutionSnapshot
	if err := r.db.WithContext(ctx).Where("report_id = ?", reportID).First(&snapshot).Error; err != nil {
		return nil, err
	}
	var payload model.FullReportExecutionPayload
	if err := r.db.WithContext(ctx).Where("snapshot_id = ?", snapshot.ID).First(&payload).Error; err != nil {
		return nil, err
	}
	return &FullReportExecutionTrace{Snapshot: snapshot, Payload: payload}, nil
}

type FullReportAttemptTrace struct {
	Attempt model.FullReportAttempt        `json:"attempt"`
	Payload model.FullReportAttemptPayload `json:"payload"`
	Chapter model.FullReportChapter        `json:"chapter"`
}

func (r *FullReportRepo) AdminListAttempts(ctx context.Context, reportID uint64, limit, offset int) ([]model.FullReportAttempt, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.FullReportAttempt{}).Where("report_id = ?", reportID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.FullReportAttempt
	err := q.Order("started_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *FullReportRepo) AdminGetAttemptTrace(ctx context.Context, reportID, attemptID uint64) (*FullReportAttemptTrace, error) {
	var attempt model.FullReportAttempt
	if err := r.db.WithContext(ctx).Where("id = ? AND report_id = ?", attemptID, reportID).First(&attempt).Error; err != nil {
		return nil, err
	}
	var payload model.FullReportAttemptPayload
	if err := r.db.WithContext(ctx).Where("attempt_id = ?", attempt.ID).First(&payload).Error; err != nil {
		return nil, err
	}
	var chapter model.FullReportChapter
	if err := r.db.WithContext(ctx).First(&chapter, attempt.ChapterID).Error; err != nil {
		return nil, err
	}
	return &FullReportAttemptTrace{Attempt: attempt, Payload: payload, Chapter: chapter}, nil
}

type FullReportChapterWithPayload struct {
	Chapter model.FullReportChapter
	Payload model.FullReportChapterPayload
}

type FullReportValidationTrace struct {
	Run     model.FullReportValidationRun     `json:"run"`
	Payload model.FullReportValidationPayload `json:"payload"`
}

func (r *FullReportRepo) AdminListValidationRuns(ctx context.Context, reportID uint64) ([]model.FullReportValidationRun, error) {
	var report model.FullReport
	if err := r.db.WithContext(ctx).Select("id").First(&report, reportID).Error; err != nil {
		return nil, err
	}
	var rows []model.FullReportValidationRun
	if err := r.db.WithContext(ctx).Where("report_id = ?", reportID).Order("round_no DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *FullReportRepo) AdminGetValidationTrace(ctx context.Context, reportID, validationID uint64) (*FullReportValidationTrace, error) {
	var run model.FullReportValidationRun
	if err := r.db.WithContext(ctx).Where("id = ? AND report_id = ?", validationID, reportID).First(&run).Error; err != nil {
		return nil, err
	}
	var payload model.FullReportValidationPayload
	if err := r.db.WithContext(ctx).Where("validation_run_id = ?", run.ID).First(&payload).Error; err != nil {
		return nil, err
	}
	return &FullReportValidationTrace{Run: run, Payload: payload}, nil
}

func (r *FullReportRepo) AdminListChapters(ctx context.Context, reportID uint64) ([]model.FullReportChapter, error) {
	var report model.FullReport
	if err := r.db.WithContext(ctx).Select("id").First(&report, reportID).Error; err != nil {
		return nil, err
	}
	var chapters []model.FullReportChapter
	if err := r.db.WithContext(ctx).Where("report_id = ?", reportID).Order("chapter_no ASC").Find(&chapters).Error; err != nil {
		return nil, err
	}
	return chapters, nil
}

func (r *FullReportRepo) AdminGetChapterTrace(ctx context.Context, reportID, chapterID uint64) (*FullReportChapterWithPayload, error) {
	var chapter model.FullReportChapter
	if err := r.db.WithContext(ctx).Where("id = ? AND report_id = ?", chapterID, reportID).First(&chapter).Error; err != nil {
		return nil, err
	}
	var payload model.FullReportChapterPayload
	if err := r.db.WithContext(ctx).Where("chapter_id = ?", chapter.ID).First(&payload).Error; err != nil {
		return nil, err
	}
	return &FullReportChapterWithPayload{Chapter: chapter, Payload: payload}, nil
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
		Updates(map[string]any{"status": model.FullReportStatusAssembling, "current_stage": model.FullReportStatusAssembling, "updated_at": at})
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
	PromptTokens     int
	CompletionTokens int
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
		updates := map[string]any{
			"status": outcome.Status, "schema_valid": outcome.SchemaValid, "validation_status": outcome.ValidationStatus,
			"error_code": outcome.ErrorCode, "error_summary": outcome.ErrorSummary, "output_hash": outcome.OutputHash,
			"prompt_tokens": outcome.PromptTokens, "completion_tokens": outcome.CompletionTokens,
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

// Complete atomically stores the final result and freezes the report. All ten
// chapters must already be successful and validated.
func (r *FullReportRepo) Complete(ctx context.Context, reportID uint64, result *model.FullReportResult, completedAt time.Time) error {
	if result == nil {
		return errors.New("full report result is nil")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var report model.FullReport
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, reportID).Error; err != nil {
			return err
		}
		if model.IsFullReportTerminalStatus(report.Status) {
			return ErrFullReportTerminal
		}
		var passed int64
		if err := tx.Model(&model.FullReportChapter{}).
			Where("report_id = ? AND status = ? AND schema_valid = ? AND validation_status = ?", reportID, model.FullReportChapterStatusSucceeded, true, model.FullReportValidationStatusPassed).
			Count(&passed).Error; err != nil {
			return err
		}
		if passed != 10 {
			return ErrFullReportNotReady
		}
		var aggregate model.FullReportValidationRun
		if err := tx.Where("report_id = ?", reportID).Order("round_no DESC").First(&aggregate).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFullReportNotReady
			}
			return err
		}
		if aggregate.Status != model.FullReportValidationStatusPassed {
			return ErrFullReportNotReady
		}
		result.ReportID = reportID
		if err := tx.Create(result).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"status":            model.FullReportStatusCompleted,
			"current_stage":     model.FullReportStatusCompleted,
			"chapter_succeeded": 10,
			"chapter_failed":    0,
			"content_hash":      result.ContentHash,
			"completed_at":      completedAt,
			"updated_at":        completedAt,
		}
		res := tx.Model(&model.FullReport{}).
			Where("id = ? AND status NOT IN ?", reportID, []string{model.FullReportStatusCompleted, model.FullReportStatusFailed}).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrFullReportTerminal
		}
		return nil
	})
}

func (r *FullReportRepo) Fail(ctx context.Context, reportID uint64, code, summary string, failedAt time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.FullReport{}).
		Where("id = ? AND status NOT IN ?", reportID, []string{model.FullReportStatusCompleted, model.FullReportStatusFailed}).
		Updates(map[string]any{
			"status":        model.FullReportStatusFailed,
			"current_stage": model.FullReportStatusFailed,
			"error_code":    code,
			"error_summary": summary,
			"completed_at":  failedAt,
			"updated_at":    failedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrFullReportTerminal
	}
	return nil
}
