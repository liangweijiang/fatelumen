package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"fatelumen/backend/internal/bazi/displaydict"
	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/repository"
)

const reportValidatorVersion = "report-validator-v1"

type reportValidationRule struct {
	Code             string             `json:"code"`
	Name             string             `json:"name"`
	Severity         validationSeverity `json:"severity"`
	Passed           bool               `json:"passed"`
	Retryable        bool               `json:"retryable"`
	AffectedChapters []uint8            `json:"affected_chapters,omitempty"`
	Expected         string             `json:"expected,omitempty"`
	Actual           string             `json:"actual,omitempty"`
	EvidenceRefs     []string           `json:"evidence_refs,omitempty"`
	Message          string             `json:"message,omitempty"`
}

type reportValidation struct {
	ValidatorVersion string                 `json:"validator_version"`
	Scope            string                 `json:"scope"`
	Passed           bool                   `json:"passed"`
	Retryable        bool                   `json:"retryable"`
	Code             string                 `json:"code,omitempty"`
	Summary          string                 `json:"summary,omitempty"`
	AffectedChapters []uint8                `json:"affected_chapters"`
	Rules            []reportValidationRule `json:"rules"`
	ValidatedAt      time.Time              `json:"validated_at"`
}

func validateFullReport(locale string, facts model.InterpretationFacts, rows []repository.FullReportChapterWithPayload, now time.Time) reportValidation {
	result := reportValidation{ValidatorVersion: reportValidatorVersion, Scope: "report", Rules: []reportValidationRule{}, AffectedChapters: []uint8{}, ValidatedAt: now}
	definitions := prompts.ChapterDefinitions()
	result.Rules = append(result.Rules, validateReportChapterSet(definitions, rows)...)
	result.Rules = append(result.Rules, validateReportSelectedOutputs(locale, facts, rows)...)
	result.Rules = append(result.Rules, validateReportAnnualSequence(facts, rows)...)
	result.Rules = append(result.Rules, validateReportDuplicateContent(rows)...)

	affected := map[uint8]struct{}{}
	for _, rule := range result.Rules {
		if rule.Passed || rule.Severity != validationError {
			continue
		}
		result.Passed = false
		result.Retryable = result.Retryable || rule.Retryable
		for _, chapterNo := range rule.AffectedChapters {
			affected[chapterNo] = struct{}{}
		}
	}
	if len(result.Rules) > 0 {
		result.Passed = true
		for _, rule := range result.Rules {
			if !rule.Passed && rule.Severity == validationError {
				result.Passed = false
				break
			}
		}
	}
	for chapterNo := range affected {
		result.AffectedChapters = append(result.AffectedChapters, chapterNo)
	}
	sort.Slice(result.AffectedChapters, func(i, j int) bool { return result.AffectedChapters[i] < result.AffectedChapters[j] })
	if result.Passed {
		result.Code = "REPORT_VALID"
		result.Summary = "十章报告汇总校验通过"
		return result
	}
	result.Code = "REPORT_VALIDATION_FAILED"
	result.Summary = fmt.Sprintf("十章报告汇总校验失败，涉及%d章", len(result.AffectedChapters))
	return result
}

func validateReportChapterSet(definitions []prompts.ChapterDefinition, rows []repository.FullReportChapterWithPayload) []reportValidationRule {
	if len(rows) != len(definitions) {
		return []reportValidationRule{failedReportRule("REPORT_CHAPTER_COUNT", "十章完整性", false, nil, strconv.Itoa(len(definitions)), strconv.Itoa(len(rows)), []string{"full_report_chapters"}, "报告章节数量不完整")}
	}
	rules := []reportValidationRule{}
	seenNo := map[uint8]struct{}{}
	seenKey := map[string]struct{}{}
	for i, definition := range definitions {
		row := rows[i]
		chapterNo := row.Chapter.ChapterNo
		if _, exists := seenNo[chapterNo]; exists {
			rules = append(rules, failedReportRule("REPORT_DUPLICATE_CHAPTER_NO", "章节编号唯一性", false, []uint8{chapterNo}, "唯一编号", strconv.Itoa(int(chapterNo)), []string{"full_report_chapters.chapter_no"}, "章节编号重复"))
		}
		seenNo[chapterNo] = struct{}{}
		if _, exists := seenKey[row.Chapter.ChapterKey]; exists {
			rules = append(rules, failedReportRule("REPORT_DUPLICATE_CHAPTER_KEY", "章节键唯一性", false, []uint8{chapterNo}, "唯一章节键", row.Chapter.ChapterKey, []string{"full_report_chapters.chapter_key"}, "章节键重复"))
		}
		seenKey[row.Chapter.ChapterKey] = struct{}{}
		if chapterNo != uint8(definition.No) || row.Chapter.ChapterKey != definition.Key {
			rules = append(rules, failedReportRule("REPORT_CHAPTER_ORDER", "章节顺序与身份", false, []uint8{chapterNo}, fmt.Sprintf("%d/%s", definition.No, definition.Key), fmt.Sprintf("%d/%s", chapterNo, row.Chapter.ChapterKey), []string{"full_report_chapters.chapter_no", "full_report_chapters.chapter_key"}, "章节顺序或章节键与冻结十章计划不一致"))
		}
	}
	if len(rules) == 0 {
		rules = append(rules, passedReportRule("REPORT_CHAPTER_SET_VALID", "十章完整、唯一且顺序正确"))
	}
	return rules
}

func validateReportSelectedOutputs(locale string, facts model.InterpretationFacts, rows []repository.FullReportChapterWithPayload) []reportValidationRule {
	rules := []reportValidationRule{}
	for _, row := range rows {
		chapterNo := row.Chapter.ChapterNo
		if row.Chapter.Status != model.FullReportChapterStatusSucceeded || row.Chapter.SelectedAttemptID == nil || !row.Chapter.SchemaValid || row.Chapter.ValidationStatus != model.FullReportValidationStatusPassed {
			rules = append(rules, failedReportRule("REPORT_CHAPTER_NOT_VALIDATED", "章节采用结果", true, []uint8{chapterNo}, "成功且六层校验通过", row.Chapter.Status+"/"+row.Chapter.ValidationStatus, []string{fmt.Sprintf("chapters.%d", chapterNo)}, "章节没有可用于汇总的最终采用结果"))
			continue
		}
		definition, ok := prompts.ChapterByKey(row.Chapter.ChapterKey)
		if !ok {
			rules = append(rules, failedReportRule("REPORT_UNKNOWN_CHAPTER", "章节注册信息", false, []uint8{chapterNo}, "已冻结注册章节", row.Chapter.ChapterKey, []string{fmt.Sprintf("chapters.%d.chapter_key", chapterNo)}, "章节键不在十章注册表中"))
			continue
		}
		var schema map[string]any
		var glossary []displaydict.GlossaryEntry
		_ = json.Unmarshal(row.Payload.OutputSchema, &schema)
		_ = json.Unmarshal(row.Payload.TerminologySnapshot, &glossary)
		validation := validateChapterOutput(definition, locale, row.Payload.FinalRawOutput, nil, chapterValidationFrozen{Glossary: glossary, Facts: &facts})
		if !validation.Passed {
			rules = append(rules, failedReportRule("REPORT_CHAPTER_REVALIDATION", "采用结果复核", true, []uint8{chapterNo}, "六层校验通过", validation.Summary, []string{fmt.Sprintf("chapters.%d.payload.final_raw_output", chapterNo)}, "冻结采用结果在汇总阶段复核失败"))
		}
	}
	if len(rules) == 0 {
		rules = append(rules, passedReportRule("REPORT_CONCLUSIONS_CONSISTENT", "关键结论与冻结事实跨章一致"))
	}
	return rules
}

func validateReportAnnualSequence(facts model.InterpretationFacts, rows []repository.FullReportChapterWithPayload) []reportValidationRule {
	chapterNo := uint8(4)
	if len(facts.AnnualFortunes) != 10 {
		return []reportValidationRule{failedReportRule("REPORT_ANNUAL_COUNT", "十年流年数量", false, []uint8{chapterNo}, "10", strconv.Itoa(len(facts.AnnualFortunes)), []string{"facts.annual_fortunes"}, "冻结流年不是完整十年")}
	}
	for i := 1; i < len(facts.AnnualFortunes); i++ {
		if facts.AnnualFortunes[i].Year != facts.AnnualFortunes[i-1].Year+1 {
			return []reportValidationRule{failedReportRule("REPORT_ANNUAL_CONTINUITY", "十年流年连续性", false, []uint8{chapterNo}, "连续十个公历年", fmt.Sprintf("%d后为%d", facts.AnnualFortunes[i-1].Year, facts.AnnualFortunes[i].Year), []string{"facts.annual_fortunes"}, "冻结流年年份不连续")}
		}
	}
	var body string
	for _, row := range rows {
		if row.Chapter.ChapterKey != "ten_year_years" {
			continue
		}
		var parsed chapterOutput
		if json.Unmarshal(row.Payload.FinalParsedOutput, &parsed) == nil {
			body = moduleContents(parsed.Modules)
		}
		chapterNo = row.Chapter.ChapterNo
		break
	}
	missing := []string{}
	for _, annual := range facts.AnnualFortunes {
		if !strings.Contains(body, strconv.Itoa(annual.Year)) || (annual.GanZhi != "" && !strings.Contains(body, annual.GanZhi)) {
			missing = append(missing, fmt.Sprintf("%d%s", annual.Year, annual.GanZhi))
		}
	}
	if len(missing) > 0 {
		return []reportValidationRule{failedReportRule("REPORT_ANNUAL_COVERAGE", "十年流年覆盖", true, []uint8{chapterNo}, "十个年份及对应干支全部出现", strings.Join(missing, "、"), []string{"facts.annual_fortunes", "chapters.4.payload.final_parsed_output"}, "未来十年章节遗漏年份或对应干支")}
	}
	return []reportValidationRule{passedReportRule("REPORT_ANNUAL_VALID", "十年流年连续且完整覆盖")}
}

func validateReportDuplicateContent(rows []repository.FullReportChapterWithPayload) []reportValidationRule {
	seen := map[string]uint8{}
	for _, row := range rows {
		var parsed chapterOutput
		if json.Unmarshal(row.Payload.FinalParsedOutput, &parsed) != nil {
			continue
		}
		for _, module := range parsed.Modules {
			normalized := normalizeForDuplicate(module.Content)
			if len([]rune(normalized)) < 20 {
				continue
			}
			if previous, exists := seen[normalized]; exists {
				return []reportValidationRule{failedReportRule("REPORT_DUPLICATE_CONTENT", "跨章正文重复", true, []uint8{previous, row.Chapter.ChapterNo}, "不同章节正文不完全重复", module.Name, []string{fmt.Sprintf("chapters.%d", previous), fmt.Sprintf("chapters.%d", row.Chapter.ChapterNo)}, "不同章节出现完全相同的模块正文")}
			}
			seen[normalized] = row.Chapter.ChapterNo
		}
	}
	return []reportValidationRule{passedReportRule("REPORT_CONTENT_DISTINCT", "跨章正文无完全重复模块")}
}

func passedReportRule(code, name string) reportValidationRule {
	return reportValidationRule{Code: code, Name: name, Severity: validationError, Passed: true}
}

func failedReportRule(code, name string, retryable bool, affected []uint8, expected, actual string, refs []string, message string) reportValidationRule {
	return reportValidationRule{Code: code, Name: name, Severity: validationError, Passed: false, Retryable: retryable, AffectedChapters: affected, Expected: expected, Actual: actual, EvidenceRefs: refs, Message: message}
}
