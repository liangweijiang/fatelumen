package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
	hashutil "fatelumen/backend/internal/pkg/hash"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

const fullReportRuntimePolicyVersion = "full-report-runtime-v1"

type FullReportRuntimeConfig struct {
	ChapterConcurrency int           `json:"chapter_concurrency"`
	MaxAttempts        int           `json:"max_attempts"`
	ChapterTimeout     time.Duration `json:"chapter_timeout"`
	Provider           string        `json:"provider"`
	Model              string        `json:"model"`
}

type fullReportPayload struct {
	ReportID  uint64 `json:"report_id"`
	UserID    uint64 `json:"user_id"`
	ProfileID uint64 `json:"profile_id"`
	Locale    string `json:"locale"`
}

type chartSaver interface {
	FindByHash(hash string) (*model.Chart, error)
	Create(chart *model.Chart) error
}

type fullReportExecutor struct {
	profiles profileGetter
	charts   chartSaver
	reports  *repository.FullReportRepo
	provider llm.LLMProvider
	engine   birthchart.Engine
	runtime  FullReportRuntimeConfig
}

func NewFullReportExecutor(profiles *repository.ProfileRepo, charts *repository.ChartRepo, reports *repository.FullReportRepo, provider llm.LLMProvider, engine birthchart.Engine, runtime FullReportRuntimeConfig) job.JobHandler {
	if engine == nil {
		engine = birthchart.NewDefaultEngine()
	}
	if runtime.ChapterConcurrency < 1 || runtime.ChapterConcurrency > 10 {
		runtime.ChapterConcurrency = 3
	}
	if runtime.MaxAttempts < 1 {
		runtime.MaxAttempts = 2
	}
	if runtime.ChapterTimeout <= 0 {
		runtime.ChapterTimeout = 60 * time.Second
	}
	if runtime.Provider == "" && provider != nil {
		runtime.Provider = provider.Name()
	}
	return &fullReportExecutor{profiles: profiles, charts: charts, reports: reports, provider: provider, engine: engine, runtime: runtime}
}

func (e *fullReportExecutor) Handle(ctx context.Context, j *job.Job) (result string, err error) {
	var payload fullReportPayload
	if err := json.Unmarshal(j.Payload, &payload); err != nil {
		logger.FromCtx(ctx).Error("full report payload parse failed", "err", err, "job_id", j.ID)
		return "", fmt.Errorf("parse full report payload: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}
		if failErr := e.reports.Fail(ctx, payload.ReportID, "generation_failed", truncateError(err), time.Now().UTC()); failErr != nil && !errors.Is(failErr, repository.ErrFullReportTerminal) {
			logger.FromCtx(ctx).Error("freeze failed full report failed", "err", failErr, "report_id", payload.ReportID)
		}
	}()

	report, err := e.reports.GetByID(ctx, payload.ReportID, payload.UserID)
	if err != nil {
		return "", fmt.Errorf("get full report: %w", err)
	}
	if report.Status == model.FullReportStatusCompleted {
		return report.PublicID, nil
	}
	if report.Status != model.FullReportStatusPending {
		// A non-pending report has already been claimed by this immutable
		// execution chain. Duplicate queue delivery is therefore a no-op.
		return report.PublicID, nil
	}
	if err := e.reports.BeginPreflight(ctx, report.ID, time.Now().UTC()); err != nil {
		return "", fmt.Errorf("claim full report execution: %w", err)
	}
	profile, err := e.profiles.FindByID(payload.ProfileID)
	if err != nil {
		return "", fmt.Errorf("find profile: %w", err)
	}
	if profile.UserID != payload.UserID {
		return "", fmt.Errorf("profile does not belong to report owner")
	}
	if e.provider == nil {
		return "", fmt.Errorf("full report provider is not configured")
	}
	input := birthchartInputFromProfile(profile)
	calculated, err := e.engine.Calculate(ctx, input)
	if err != nil {
		return "", fmt.Errorf("calculate birth chart: %w", err)
	}
	chartHash := BuildChartHash(profile.Gender, profile.CalendarType, int(profile.BirthYear), int(profile.BirthMonth), int(profile.BirthDay), int(profile.BirthHour), int(profile.BirthMinute), profile.IsLeapMonth == 1, calculated)
	chartID, err := e.persistChart(profile.ID, chartHash, calculated.Chart)
	if err != nil {
		return "", err
	}
	if err := e.reports.UpdateChartID(ctx, payload.ReportID, chartID); err != nil {
		return "", fmt.Errorf("set report chart: %w", err)
	}

	inputSnapshot, timeSnapshot, chartSnapshot, factsSnapshot, err := BuildDeterministicSnapshot(input, payload.Locale, chartHash, calculated)
	if err != nil {
		return "", fmt.Errorf("build deterministic snapshot: %w", err)
	}
	plans, err := e.buildPlans(payload.Locale, factsSnapshot)
	if err != nil {
		return "", fmt.Errorf("preflight report chapters: %w", err)
	}
	preflight := PreflightFullReport(payload.Locale, e.runtime, plans)
	if !preflight.Passed {
		return "", fmt.Errorf("preflight blocked: %s", strings.Join(preflight.Errors, "; "))
	}
	if err := e.freeze(ctx, payload.ReportID, calculated, inputSnapshot, timeSnapshot, chartSnapshot, factsSnapshot, plans, preflight); err != nil {
		return "", fmt.Errorf("freeze report execution: %w", err)
	}

	rows, err := e.reports.ListChaptersWithPayload(ctx, payload.ReportID)
	if err != nil {
		return "", fmt.Errorf("load frozen chapters: %w", err)
	}
	if len(rows) != 10 {
		return "", repository.ErrFullReportChapterCount
	}
	if err := e.runChapters(ctx, payload.ReportID, payload.Locale, rows); err != nil {
		return "", err
	}
	rows, err = e.reports.ListChaptersWithPayload(ctx, payload.ReportID)
	if err != nil {
		return "", err
	}
	content, err := assembleFullReportContent(payload.Locale, rows)
	if err != nil {
		return "", err
	}
	alignAnnualFortunes(&content, calculated.Chart.AnnualFortunes)
	contentHash, err := hashutil.CanonicalJSONSHA256(content)
	if err != nil {
		return "", err
	}
	completedAt := time.Now().UTC()
	if err := e.reports.Complete(ctx, payload.ReportID, &model.FullReportResult{Locale: payload.Locale, Content: content, ContentHash: contentHash, RenderVersion: "full-report-html-v1", CreatedAt: completedAt}, completedAt); err != nil {
		return "", fmt.Errorf("complete full report: %w", err)
	}
	return report.PublicID, nil
}

type frozenChapterPlan struct {
	Definition prompts.ChapterDefinition
	Preview    *prompts.ChapterPromptPreview
	PromptHash string
}

func (e *fullReportExecutor) buildPlans(locale string, facts model.InterpretationFacts) ([]frozenChapterPlan, error) {
	definitions := prompts.ChapterDefinitions()
	if len(definitions) != 10 {
		return nil, repository.ErrFullReportChapterCount
	}
	plans := make([]frozenChapterPlan, 0, len(definitions))
	for _, definition := range definitions {
		preview, err := prompts.BuildChapterPromptPreview(locale, definition.Key, facts)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(preview.CompleteInstruction) == "" || len(preview.Chapter.Sections) == 0 {
			return nil, fmt.Errorf("chapter %s has incomplete prompt contract", definition.Key)
		}
		h, err := hashutil.CanonicalJSONSHA256(preview.CompleteInstruction)
		if err != nil {
			return nil, err
		}
		plans = append(plans, frozenChapterPlan{Definition: definition, Preview: preview, PromptHash: h})
	}
	return plans, nil
}

func (e *fullReportExecutor) freeze(ctx context.Context, reportID uint64, calculated *birthchart.Result, input model.ReportInputSnapshot, tc model.TimeCalculationSnapshot, chart model.ChartSnapshot, facts model.InterpretationFacts, plans []frozenChapterPlan, preflight FullReportPreflightResult) error {
	planSnapshot := make([]map[string]any, 0, len(plans))
	chapters := make([]model.FullReportChapter, len(plans))
	payloads := make([]model.FullReportChapterPayload, len(plans))
	now := time.Now().UTC()
	for i, plan := range plans {
		planSnapshot = append(planSnapshot, map[string]any{"no": plan.Definition.No, "key": plan.Definition.Key, "prompt_hash": plan.PromptHash})
		glossary, _ := json.Marshal(plan.Preview.Glossary)
		schema, _ := json.Marshal(plan.Preview.OutputSchema)
		chapters[i] = model.FullReportChapter{ChapterNo: uint8(plan.Definition.No), ChapterKey: plan.Definition.Key, Title: plan.Definition.Name, Status: model.FullReportChapterStatusPending, PromptHash: plan.PromptHash, ValidationStatus: model.FullReportValidationStatusPending, CreatedAt: now, UpdatedAt: now}
		payloads[i] = model.FullReportChapterPayload{SemanticDigest: plan.Preview.SemanticDigest.Text(), LanguageInstruction: plan.Preview.AdditiveInstruction, TerminologySnapshot: model.JSONRaw(glossary), FinalPrompt: plan.Preview.CompleteInstruction, OutputSchema: model.JSONRaw(schema), CreatedAt: now}
	}
	runtimeSnapshot, _ := json.Marshal(e.runtime)
	preflightRaw, _ := json.Marshal(preflight)
	planRaw, _ := json.Marshal(planSnapshot)
	executionHash, err := hashutil.CanonicalJSONSHA256(map[string]any{"facts_hash": facts.FactsHash, "plans": planSnapshot, "runtime": e.runtime})
	if err != nil {
		return err
	}
	inputRaw, _ := json.Marshal(input)
	timeRaw, _ := json.Marshal(tc)
	chartRaw, _ := json.Marshal(chart)
	factsRaw, _ := json.Marshal(facts)
	versions := facts.Versions
	return e.reports.FreezeExecution(ctx, repository.FullReportFreezeExecution{
		ReportID:         reportID,
		Snapshot:         &model.FullReportExecutionSnapshot{ChartHash: chart.ChartHash, FactsHash: facts.FactsHash, ExecutionHash: executionHash, InputSchemaVersion: input.SchemaVersion, ChartSchemaVersion: chart.ChartSchemaVersion, FactsSchemaVersion: versions.FactsSchemaVersion, RuleSetVersion: versions.RuleSetVersion, PromptVersion: versions.PromptVersion, DictionaryVersion: "bazi-display-dictionary-v2", RuntimePolicyVersion: fullReportRuntimePolicyVersion, LocationDatabaseVersion: tc.LocationDatabaseVersion, TimezoneDatabaseVersion: tc.TimezoneDatabaseVersion, SolarAlgorithmVersion: tc.SolarAlgorithmVersion, LunarGoVersion: chart.LunarGoVersion, FrozenAt: now, CreatedAt: now},
		ExecutionPayload: &model.FullReportExecutionPayload{InputSnapshot: inputRaw, TimeCalculationSnapshot: timeRaw, ChartSnapshot: chartRaw, FactsSnapshot: factsRaw, PreflightResult: preflightRaw, ChapterPlanSnapshot: planRaw, RuntimeConfigSnapshot: runtimeSnapshot, CreatedAt: now},
		Chapters:         chapters, ChapterPayloads: payloads,
	})
}

func (e *fullReportExecutor) runChapters(ctx context.Context, reportID uint64, locale string, rows []repository.FullReportChapterWithPayload) error {
	sem := make(chan struct{}, e.runtime.ChapterConcurrency)
	errCh := make(chan error, len(rows))
	var wg sync.WaitGroup
	for _, row := range rows {
		row := row
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
			if err := e.runChapter(ctx, reportID, locale, row); err != nil {
				errCh <- fmt.Errorf("chapter %s: %w", row.Chapter.ChapterKey, err)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (e *fullReportExecutor) runChapter(ctx context.Context, reportID uint64, locale string, row repository.FullReportChapterWithPayload) error {
	definition, ok := prompts.ChapterByKey(row.Chapter.ChapterKey)
	if !ok {
		return fmt.Errorf("unknown chapter")
	}
	for n := 1; n <= e.runtime.MaxAttempts; n++ {
		started := time.Now().UTC()
		attempt := &model.FullReportAttempt{ReportID: reportID, ChapterID: row.Chapter.ID, AttemptNo: uint16(n), RouteNo: 1, Provider: e.runtime.Provider, Model: e.runtime.Model, Status: model.FullReportAttemptStatusRunning, ValidationStatus: model.FullReportValidationStatusPending, PromptHash: row.Chapter.PromptHash, TraceID: fmt.Sprintf("%d-%d-%d", reportID, row.Chapter.ID, n), StartedAt: started, CreatedAt: started}
		params, _ := json.Marshal(map[string]any{"temperature": 0.5, "max_tokens": definition.MaxTokens})
		attemptPayload := &model.FullReportAttemptPayload{RequestParameters: params, RequestPrompt: row.Payload.FinalPrompt, CreatedAt: started}
		if err := e.reports.AppendAttempt(ctx, attempt, attemptPayload); err != nil {
			return err
		}
		callCtx, cancel := context.WithTimeout(ctx, e.runtime.ChapterTimeout)
		raw, callErr := e.provider.GenerateJSON(callCtx, "你只能解释已提供的确定性事实，并严格返回JSON。", row.Payload.FinalPrompt, llm.WithMaxTokens(definition.MaxTokens), llm.WithTemperature(0.5))
		cancel()
		finished := time.Now().UTC()
		validation := validateChapterOutput(definition, locale, raw, callErr)
		validationRaw, _ := json.Marshal(validation)
		parsedRaw, _ := json.Marshal(validation.Parsed)
		schemaErrorsRaw, _ := json.Marshal(validation.Errors)
		outputHash := ""
		if raw != "" {
			outputHash, _ = hashutil.CanonicalJSONSHA256(raw)
		}
		status := model.FullReportAttemptStatusRejected
		if callErr != nil {
			status = model.FullReportAttemptStatusFailed
		} else if validation.Passed {
			status = model.FullReportAttemptStatusSucceeded
		}
		if err := e.reports.FinishAttempt(ctx, reportID, attempt.ID, repository.FullReportAttemptOutcome{Status: status, SchemaValid: validation.SchemaValid, ValidationStatus: validationStatus(validation.Passed), ErrorCode: validation.Code, ErrorSummary: validation.Summary, OutputHash: outputHash, DurationMS: finished.Sub(started).Milliseconds(), FinishedAt: finished, RawOutput: raw, ParsedOutput: parsedRaw, SchemaErrors: schemaErrorsRaw, ValidationResult: validationRaw}); err != nil {
			return err
		}
		if validation.Passed {
			return e.reports.FinishChapter(ctx, reportID, row.Chapter.ID, attempt.ID, raw, parsedRaw, validationRaw, outputHash, finished)
		}
	}
	return fmt.Errorf("validation failed after %d attempts", e.runtime.MaxAttempts)
}

type chapterOutput struct {
	Chapter string `json:"章节"`
	Modules []struct {
		No      int    `json:"序号"`
		Name    string `json:"名称"`
		Content string `json:"正文"`
	} `json:"模块"`
}

type chapterValidation struct {
	Passed      bool          `json:"passed"`
	SchemaValid bool          `json:"schema_valid"`
	Code        string        `json:"code,omitempty"`
	Summary     string        `json:"summary,omitempty"`
	Errors      []string      `json:"errors"`
	Parsed      chapterOutput `json:"parsed"`
}

func validateChapterOutput(def prompts.ChapterDefinition, locale, raw string, callErr error) chapterValidation {
	result := chapterValidation{Errors: []string{}}
	if callErr != nil {
		result.Code, result.Summary = "provider_error", callErr.Error()
		result.Errors = append(result.Errors, "provider call failed")
		return result
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.Contains(trimmed, "```") {
		result.Code, result.Summary = "invalid_json_envelope", "response is empty or wrapped in markdown"
		result.Errors = append(result.Errors, result.Summary)
		return result
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result.Parsed); err != nil {
		result.Code, result.Summary = "invalid_json", err.Error()
		result.Errors = append(result.Errors, "response is not valid JSON")
		return result
	}
	if err := ensureJSONEOF(decoder); err != nil {
		result.Code, result.Summary = "invalid_json_envelope", err.Error()
		result.Errors = append(result.Errors, "response contains content outside the JSON object")
		return result
	}
	result.SchemaValid = true
	if result.Parsed.Chapter != def.Name {
		result.Errors = append(result.Errors, "chapter name mismatch")
	}
	if len(result.Parsed.Modules) != len(def.Sections) {
		result.Errors = append(result.Errors, "module count mismatch")
	} else {
		seenNames := make(map[string]struct{}, len(result.Parsed.Modules))
		for i, section := range def.Sections {
			module := result.Parsed.Modules[i]
			if module.No != i+1 || module.Name != section.Name {
				result.Errors = append(result.Errors, fmt.Sprintf("module %d contract mismatch", i+1))
			}
			if len([]rune(strings.TrimSpace(module.Content))) < 20 {
				result.Errors = append(result.Errors, fmt.Sprintf("module %d content is too short", i+1))
			}
			if _, exists := seenNames[module.Name]; exists {
				result.Errors = append(result.Errors, fmt.Sprintf("module %d name is duplicated", i+1))
			}
			seenNames[module.Name] = struct{}{}
			if violatesReportProductBoundary(module.Content) {
				result.Errors = append(result.Errors, fmt.Sprintf("module %d violates product safety boundary", i+1))
			}
		}
	}
	if len(result.Errors) > 0 {
		result.Code, result.Summary = "chapter_validation_failed", strings.Join(result.Errors, "; ")
		return result
	}
	result.Passed = true
	return result
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func violatesReportProductBoundary(content string) bool {
	lower := strings.ToLower(content)
	for _, phrase := range []string{"保证发财", "必定发财", "一定患", "确诊", "寿命为", "死亡年份", "guaranteed profit", "certainly develop cancer", "exact death", "確実に儲", "死亡する年", "반드시 암", "사망 연도"} {
		if strings.Contains(lower, strings.ToLower(phrase)) {
			return true
		}
	}
	return false
}

func assembleFullReportContent(locale string, rows []repository.FullReportChapterWithPayload) (model.ReportContent, error) {
	content := model.ReportContent{Locale: locale, Chapters: make([]model.Chapter, 0, len(rows))}
	for _, row := range rows {
		var parsed chapterOutput
		if err := json.Unmarshal(row.Payload.FinalParsedOutput, &parsed); err != nil {
			return model.ReportContent{}, fmt.Errorf("parse frozen chapter %s: %w", row.Chapter.ChapterKey, err)
		}
		parts := make([]string, 0, len(parsed.Modules))
		for _, module := range parsed.Modules {
			parts = append(parts, module.Name+"\n"+module.Content)
		}
		body := strings.Join(parts, "\n\n")
		content.Chapters = append(content.Chapters, model.Chapter{No: int(row.Chapter.ChapterNo), Key: row.Chapter.ChapterKey, Title: row.Chapter.Title, Body: body})
		if content.SummaryLine == "" && len(parsed.Modules) > 0 {
			content.SummaryLine = parsed.Modules[0].Content
		}
	}
	return content, nil
}

func birthchartInputFromProfile(profile *model.BirthProfile) birthchart.Input {
	return birthchart.Input{Gender: profile.Gender, CalendarType: profile.CalendarType, Year: int(profile.BirthYear), Month: int(profile.BirthMonth), Day: int(profile.BirthDay), Hour: int(profile.BirthHour), Minute: int(profile.BirthMinute), IsLeapMonth: profile.IsLeapMonth == 1, Location: birthchart.LocationInput{CountryCode: profile.CountryCode, CountryName: profile.CountryName, RegionCode: profile.RegionCode, RegionName: profile.RegionName, City: profile.City, PlaceID: profile.PlaceID, DisplayName: profile.BirthPlace, Latitude: profile.Latitude, Longitude: profile.Longitude, TimezoneID: profile.Timezone, HasCoordinates: profile.HasCoordinates || profile.Latitude != 0 || profile.Longitude != 0}}
}

func (e *fullReportExecutor) persistChart(profileID uint64, chartHash string, data *model.ChartData) (uint64, error) {
	if existing, err := e.charts.FindByHash(chartHash); err == nil && existing != nil {
		return existing.ID, nil
	}
	chart := &model.Chart{ProfileID: profileID, ChartHash: chartHash, ChartData: *data, CreatedAt: time.Now().UTC()}
	if err := e.charts.Create(chart); err != nil {
		return 0, fmt.Errorf("create chart: %w", err)
	}
	return chart.ID, nil
}

func validationStatus(passed bool) string {
	if passed {
		return model.FullReportValidationStatusPassed
	}
	return model.FullReportValidationStatusFailed
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	value := err.Error()
	if len(value) > 500 {
		return value[:500]
	}
	return value
}
