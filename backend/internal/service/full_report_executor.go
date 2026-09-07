package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"fatelumen/backend/internal/bazi/displaydict"
	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/job"
	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
	hashutil "fatelumen/backend/internal/pkg/hash"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

const fullReportRuntimePolicyVersion = "full-report-runtime-v2"

type FullReportRuntimeConfig struct {
	ChapterConcurrency int                    `json:"chapter_concurrency"`
	ChapterTimeout     time.Duration          `json:"chapter_timeout"`
	Routes             []FullReportModelRoute `json:"routes"`
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
	routes   FullReportRouteResolver
	engine   birthchart.Engine
	runtime  FullReportRuntimeConfig
}

func NewFullReportExecutor(profiles *repository.ProfileRepo, charts *repository.ChartRepo, reports *repository.FullReportRepo, routes FullReportRouteResolver, engine birthchart.Engine, runtime FullReportRuntimeConfig) job.JobHandler {
	if engine == nil {
		engine = birthchart.NewDefaultEngine()
	}
	if runtime.ChapterConcurrency < 1 || runtime.ChapterConcurrency > 10 {
		runtime.ChapterConcurrency = 3
	}
	if runtime.ChapterTimeout <= 0 {
		runtime.ChapterTimeout = 180 * time.Second
	}
	return &fullReportExecutor{profiles: profiles, charts: charts, reports: reports, routes: routes, engine: engine, runtime: runtime}
}

func (e *fullReportExecutor) Handle(ctx context.Context, j *job.Job) (result string, err error) {
	var payload fullReportPayload
	if err := json.Unmarshal(j.Payload, &payload); err != nil {
		logger.FromCtx(ctx).Error("full report payload parse failed", "err", err, "job_id", j.ID)
		return "", fmt.Errorf("parse full report payload: %w", err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.FromCtx(ctx).Error("full report worker panic recovered", "report_id", payload.ReportID, "panic_type", fmt.Sprintf("%T", recovered))
			err = fmt.Errorf("full report worker panic")
		}
		if err == nil {
			return
		}
		failureCtx := context.WithoutCancel(ctx)
		if failErr := e.reports.Fail(failureCtx, payload.ReportID, "generation_failed", truncateError(err), time.Now().UTC()); failErr != nil && !errors.Is(failErr, repository.ErrFullReportTerminal) {
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
	if e.routes == nil {
		return "", fmt.Errorf("full report route resolver is not configured")
	}
	resolvedRoutes, err := e.routes.Resolve(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve full report model routes: %w", err)
	}
	runtime := e.runtime
	runtime.Routes = frozenRoutes(resolvedRoutes)
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
	preflight := PreflightFullReport(payload.Locale, runtime, plans)
	if !preflight.Passed {
		return "", fmt.Errorf("preflight blocked: %s", strings.Join(preflight.Errors, "; "))
	}
	if err := e.freeze(ctx, payload.ReportID, calculated, inputSnapshot, timeSnapshot, chartSnapshot, factsSnapshot, plans, preflight, runtime); err != nil {
		return "", fmt.Errorf("freeze report execution: %w", err)
	}

	rows, err := e.reports.ListChaptersWithPayload(ctx, payload.ReportID)
	if err != nil {
		return "", fmt.Errorf("load frozen chapters: %w", err)
	}
	if len(rows) != 10 {
		return "", repository.ErrFullReportChapterCount
	}
	if err := e.runChapters(ctx, payload.ReportID, payload.Locale, factsSnapshot, rows, runtime, resolvedRoutes); err != nil {
		return "", err
	}
	rows, err = e.reports.ListChaptersWithPayload(ctx, payload.ReportID)
	if err != nil {
		return "", err
	}
	for {
		validationStarted := time.Now().UTC()
		if err := e.reports.BeginAssembling(ctx, payload.ReportID, validationStarted); err != nil {
			return "", fmt.Errorf("begin report assembling: %w", err)
		}
		aggregateValidation := validateFullReport(payload.Locale, factsSnapshot, rows, time.Now().UTC())
		aggregateRaw, marshalErr := json.Marshal(aggregateValidation)
		if marshalErr != nil {
			return "", fmt.Errorf("marshal aggregate validation: %w", marshalErr)
		}
		validationFinished := time.Now().UTC()
		validationRun := &model.FullReportValidationRun{
			ReportID: payload.ReportID, ValidatorVersion: aggregateValidation.ValidatorVersion,
			Status: validationStatus(aggregateValidation.Passed), Retryable: aggregateValidation.Retryable,
			AffectedChapters: uint8(len(aggregateValidation.AffectedChapters)), ErrorCode: aggregateValidation.Code,
			ErrorSummary: aggregateValidation.Summary, StartedAt: validationStarted, FinishedAt: &validationFinished, CreatedAt: validationStarted,
		}
		if err := e.reports.SaveAggregateValidation(ctx, validationRun, &model.FullReportValidationPayload{ValidationResult: aggregateRaw, CreatedAt: validationStarted}); err != nil {
			return "", fmt.Errorf("save aggregate validation: %w", err)
		}
		if aggregateValidation.Passed {
			break
		}
		retryRows := aggregateRetryRows(rows, aggregateValidation.AffectedChapters, totalRouteAttempts(runtime.Routes))
		if !aggregateValidation.Retryable || len(retryRows) == 0 {
			return "", fmt.Errorf("aggregate validation failed: %s", aggregateValidation.Summary)
		}
		retryNos := make([]uint8, 0, len(retryRows))
		for _, row := range retryRows {
			retryNos = append(retryNos, row.Chapter.ChapterNo)
		}
		if err := e.reports.PrepareAggregateRetry(ctx, payload.ReportID, retryNos, time.Now().UTC()); err != nil {
			return "", fmt.Errorf("prepare aggregate retry: %w", err)
		}
		if err := e.runChapters(ctx, payload.ReportID, payload.Locale, factsSnapshot, retryRows, runtime, resolvedRoutes); err != nil {
			return "", fmt.Errorf("aggregate retry: %w", err)
		}
		rows, err = e.reports.ListChaptersWithPayload(ctx, payload.ReportID)
		if err != nil {
			return "", fmt.Errorf("reload aggregate retry chapters: %w", err)
		}
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
	if err := e.reports.BeginRendering(ctx, payload.ReportID, time.Now().UTC()); err != nil {
		return "", fmt.Errorf("begin report rendering: %w", err)
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

func (e *fullReportExecutor) freeze(ctx context.Context, reportID uint64, calculated *birthchart.Result, input model.ReportInputSnapshot, tc model.TimeCalculationSnapshot, chart model.ChartSnapshot, facts model.InterpretationFacts, plans []frozenChapterPlan, preflight FullReportPreflightResult, runtime FullReportRuntimeConfig) error {
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
	runtimeSnapshot, _ := json.Marshal(runtime)
	preflightRaw, _ := json.Marshal(preflight)
	planRaw, _ := json.Marshal(planSnapshot)
	executionHash, err := hashutil.CanonicalJSONSHA256(map[string]any{"facts_hash": facts.FactsHash, "plans": planSnapshot, "runtime": runtime})
	if err != nil {
		return err
	}
	inputRaw, _ := json.Marshal(input)
	timeRaw, _ := json.Marshal(tc)
	chartRaw, _ := json.Marshal(chart)
	factsRaw, _ := json.Marshal(facts)
	versions := facts.Versions
	return e.reports.FreezeExecution(ctx, repository.FullReportFreezeExecution{
		ReportID:           reportID,
		ProviderChainKey:   executionHash[:16],
		ChapterConcurrency: uint8(runtime.ChapterConcurrency),
		Snapshot:           &model.FullReportExecutionSnapshot{ChartHash: chart.ChartHash, FactsHash: facts.FactsHash, ExecutionHash: executionHash, InputSchemaVersion: input.SchemaVersion, ChartSchemaVersion: chart.ChartSchemaVersion, FactsSchemaVersion: versions.FactsSchemaVersion, RuleSetVersion: versions.RuleSetVersion, PromptVersion: versions.PromptVersion, DictionaryVersion: "bazi-display-dictionary-v2", RuntimePolicyVersion: fullReportRuntimePolicyVersion, LocationDatabaseVersion: tc.LocationDatabaseVersion, TimezoneDatabaseVersion: tc.TimezoneDatabaseVersion, SolarAlgorithmVersion: tc.SolarAlgorithmVersion, LunarGoVersion: chart.LunarGoVersion, FrozenAt: now, CreatedAt: now},
		ExecutionPayload:   &model.FullReportExecutionPayload{InputSnapshot: inputRaw, TimeCalculationSnapshot: timeRaw, ChartSnapshot: chartRaw, FactsSnapshot: factsRaw, PreflightResult: preflightRaw, ChapterPlanSnapshot: planRaw, RuntimeConfigSnapshot: runtimeSnapshot, CreatedAt: now},
		Chapters:           chapters, ChapterPayloads: payloads,
	})
}

func (e *fullReportExecutor) runChapters(ctx context.Context, reportID uint64, locale string, facts model.InterpretationFacts, rows []repository.FullReportChapterWithPayload, runtime FullReportRuntimeConfig, routes []ResolvedFullReportRoute) error {
	sem := make(chan struct{}, runtime.ChapterConcurrency)
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
			if err := e.runChapter(ctx, reportID, locale, facts, row, routes); err != nil {
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

func (e *fullReportExecutor) runChapter(ctx context.Context, reportID uint64, locale string, facts model.InterpretationFacts, row repository.FullReportChapterWithPayload, routes []ResolvedFullReportRoute) error {
	definition, ok := prompts.ChapterByKey(row.Chapter.ChapterKey)
	if !ok {
		return fmt.Errorf("unknown chapter")
	}
	consumed := int(row.Chapter.AttemptCount)
	for {
		route, routeAttempt, ok := routeForConsumedAttempts(routes, consumed)
		if !ok {
			break
		}
		n := consumed + 1
		started := time.Now().UTC()
		traceID := logger.TraceIDFromCtx(ctx)
		if traceID == "" {
			traceID = fmt.Sprintf("%d-%d-%d", reportID, row.Chapter.ID, n)
		}
		attempt := &model.FullReportAttempt{ReportID: reportID, ChapterID: row.Chapter.ID, AttemptNo: uint16(n), RouteNo: route.Frozen.RouteNo, Provider: route.Frozen.ProviderCode, Model: route.Frozen.Model, Status: model.FullReportAttemptStatusRunning, ValidationStatus: model.FullReportValidationStatusPending, PromptHash: row.Chapter.PromptHash, TraceID: traceID, StartedAt: started, CreatedAt: started}
		params, _ := json.Marshal(map[string]any{"temperature": route.Frozen.Temperature, "max_tokens": definition.MaxTokens, "timeout_seconds": route.Frozen.TimeoutSeconds, "route_attempt": routeAttempt, "max_route_attempts": route.Frozen.MaxAttempts})
		attemptPayload := &model.FullReportAttemptPayload{RequestParameters: params, RequestPrompt: row.Payload.FinalPrompt, CreatedAt: started}
		if err := e.reports.AppendAttempt(ctx, attempt, attemptPayload); err != nil {
			return err
		}
		if routeAttempt > 1 {
			logger.FromCtx(ctx).Warn("retrying full report model route", "report_id", reportID, "chapter_id", row.Chapter.ID, "route_no", route.Frozen.RouteNo, "provider", route.Frozen.ProviderCode, "model", route.Frozen.Model, "route_attempt", routeAttempt, "max_route_attempts", route.Frozen.MaxAttempts)
		} else if route.Frozen.RouteNo > 1 {
			logger.FromCtx(ctx).Warn("switching full report model route", "report_id", reportID, "chapter_id", row.Chapter.ID, "route_no", route.Frozen.RouteNo, "provider", route.Frozen.ProviderCode, "model", route.Frozen.Model)
		}
		callCtx, cancel := context.WithTimeout(ctx, time.Duration(route.Frozen.TimeoutSeconds)*time.Second)
		generation, callErr := llm.GenerateJSONDetailed(callCtx, route.Provider, "你只能解释已提供的确定性事实，并严格返回JSON。", row.Payload.FinalPrompt, llm.WithMaxTokens(definition.MaxTokens), llm.WithTemperature(float32(route.Frozen.Temperature)))
		cancel()
		raw := generation.Content
		finished := time.Now().UTC()
		var outputSchema map[string]any
		_ = json.Unmarshal(row.Payload.OutputSchema, &outputSchema)
		var glossary []displaydict.GlossaryEntry
		_ = json.Unmarshal(row.Payload.TerminologySnapshot, &glossary)
		validation := validateChapterOutput(definition, locale, raw, callErr, chapterValidationFrozen{Glossary: glossary, Facts: &facts})
		outputHash := ""
		if raw != "" {
			outputHash, _ = hashutil.CanonicalJSONSHA256(raw)
		}
		status := model.FullReportAttemptStatusRejected
		if callErr != nil {
			status = model.FullReportAttemptStatusFailed
			callError := llm.ClassifyCallError(callErr)
			validation.Code = callError.Code
			validation.Summary = callError.Summary
		} else if validation.Passed {
			status = model.FullReportAttemptStatusSucceeded
		}
		validationRaw, _ := json.Marshal(validation)
		parsedRaw, _ := json.Marshal(validation.Parsed)
		schemaErrorsRaw, _ := json.Marshal(validation.Errors)
		if err := e.reports.FinishAttempt(ctx, reportID, attempt.ID, repository.FullReportAttemptOutcome{Status: status, SchemaValid: validation.SchemaValid, ValidationStatus: validationStatus(validation.Passed), ErrorCode: validation.Code, ErrorSummary: validation.Summary, OutputHash: outputHash, PromptTokens: generation.Usage.PromptTokens, CompletionTokens: generation.Usage.CompletionTokens, TotalTokens: generation.Usage.TotalTokens, DurationMS: finished.Sub(started).Milliseconds(), FinishedAt: finished, RawOutput: raw, ParsedOutput: parsedRaw, SchemaErrors: schemaErrorsRaw, ValidationResult: validationRaw}); err != nil {
			return err
		}
		if validation.Passed {
			return e.reports.FinishChapter(ctx, reportID, row.Chapter.ID, attempt.ID, raw, parsedRaw, validationRaw, outputHash, finished)
		}
		logger.FromCtx(ctx).Warn("full report model attempt rejected", "report_id", reportID, "chapter_id", row.Chapter.ID, "route_no", route.Frozen.RouteNo, "provider", route.Frozen.ProviderCode, "model", route.Frozen.Model, "route_attempt", routeAttempt, "max_route_attempts", route.Frozen.MaxAttempts, "validation_code", validation.Code)
		consumed++
	}
	return fmt.Errorf("all frozen model routes exhausted after %d attempts", consumed)
}

func aggregateRetryRows(rows []repository.FullReportChapterWithPayload, affected []uint8, maxAttempts int) []repository.FullReportChapterWithPayload {
	wanted := make(map[uint8]struct{}, len(affected))
	for _, chapterNo := range affected {
		wanted[chapterNo] = struct{}{}
	}
	result := make([]repository.FullReportChapterWithPayload, 0, len(wanted))
	for _, row := range rows {
		if _, ok := wanted[row.Chapter.ChapterNo]; ok && int(row.Chapter.AttemptCount) < maxAttempts {
			result = append(result, row)
		}
	}
	return result
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
