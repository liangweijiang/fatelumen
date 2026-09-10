package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"fatelumen/backend/internal/bazi/displaydict"
	"fatelumen/backend/internal/llm/prompts"
	"fatelumen/backend/internal/model"
)

const (
	chapterValidatorVersion = "chapter-validator-v1"
	maxChapterOutputBytes   = 256 * 1024
	maxChapterJSONDepth     = 8
)

type validationSeverity string

const (
	validationError   validationSeverity = "error"
	validationWarning validationSeverity = "warning"
)

type chapterRuleResult struct {
	Code         string             `json:"code"`
	Name         string             `json:"name"`
	Stage        string             `json:"stage"`
	Severity     validationSeverity `json:"severity"`
	Passed       bool               `json:"passed"`
	Retryable    bool               `json:"retryable"`
	Expected     string             `json:"expected,omitempty"`
	Actual       string             `json:"actual,omitempty"`
	EvidenceRefs []string           `json:"evidence_refs,omitempty"`
	Message      string             `json:"message,omitempty"`
}

type chapterModuleOutput struct {
	No      int    `json:"序号"`
	Name    string `json:"名称"`
	Content string `json:"正文"`
}

type chapterOutput struct {
	Chapter string                `json:"章节"`
	Modules []chapterModuleOutput `json:"模块"`
}

type chapterValidation struct {
	ValidatorVersion string              `json:"validator_version"`
	Scope            string              `json:"scope"`
	Passed           bool                `json:"passed"`
	SchemaValid      bool                `json:"schema_valid"`
	Retryable        bool                `json:"retryable"`
	Code             string              `json:"code,omitempty"`
	Summary          string              `json:"summary,omitempty"`
	Errors           []string            `json:"errors"`
	Rules            []chapterRuleResult `json:"rules"`
	Parsed           chapterOutput       `json:"parsed"`
}

type chapterValidationFrozen struct {
	Glossary []displaydict.GlossaryEntry
	Facts    *model.InterpretationFacts
}

type chapterValidationContext struct {
	Definition prompts.ChapterDefinition
	Locale     string
	Raw        string
	Parsed     chapterOutput
	Frozen     chapterValidationFrozen
}

type ChapterValidator interface {
	Name() string
	Validate(*chapterValidationContext) []chapterRuleResult
}

type chapterValidatorFunc struct {
	name string
	fn   func(*chapterValidationContext) []chapterRuleResult
}

func (v chapterValidatorFunc) Name() string { return v.name }
func (v chapterValidatorFunc) Validate(ctx *chapterValidationContext) []chapterRuleResult {
	return v.fn(ctx)
}

func validateChapterOutput(def prompts.ChapterDefinition, locale, raw string, callErr error, frozen ...chapterValidationFrozen) chapterValidation {
	result := chapterValidation{ValidatorVersion: chapterValidatorVersion, Scope: "chapter", Errors: []string{}, Rules: []chapterRuleResult{}}
	if callErr != nil {
		rule := failedRule("PROVIDER_ERROR", "模型调用", "provider", true, "调用成功", callErr.Error(), nil, "模型调用失败")
		return finalizeChapterValidation(result, []chapterRuleResult{rule})
	}
	ctx := &chapterValidationContext{Definition: def, Locale: locale, Raw: raw}
	if len(frozen) > 0 {
		ctx.Frozen = frozen[0]
	}
	formatRules, parsed, formatOK := validateChapterEnvelope(raw)
	result.Rules = append(result.Rules, formatRules...)
	if !formatOK {
		return finalizeChapterValidation(result, result.Rules)
	}
	ctx.Parsed = parsed
	result.Parsed = parsed
	validators := []ChapterValidator{
		chapterValidatorFunc{name: "schema", fn: validateChapterSchema},
		chapterValidatorFunc{name: "completeness", fn: validateChapterCompleteness},
		chapterValidatorFunc{name: "protected_facts", fn: validateChapterProtectedFacts},
		chapterValidatorFunc{name: "language_terminology", fn: validateChapterLanguageAndTerms},
		chapterValidatorFunc{name: "product_boundary", fn: validateChapterProductBoundary},
	}
	for _, validator := range validators {
		result.Rules = append(result.Rules, validator.Validate(ctx)...)
	}
	result.SchemaValid = stagePassed(result.Rules, "format") && stagePassed(result.Rules, "schema")
	return finalizeChapterValidation(result, result.Rules)
}

var ganZhiPattern = regexp.MustCompile(`[甲乙丙丁戊己庚辛壬癸][子丑寅卯辰巳午未申酉戌亥]`)
var yearPattern = regexp.MustCompile(`(?:19|20|21)[0-9]{2}`)
var dayMasterPattern = regexp.MustCompile(`日主(?:为|是|：|:)?\s*([甲乙丙丁戊己庚辛壬癸])`)
var usefulGodPattern = regexp.MustCompile(`(?:首选用神|第一用神|用神)(?:为|是|：|:)?\s*([木火土金水])`)
var strengthPattern = regexp.MustCompile(`(?:日主|命局)(?:为|是|属于|判定为|整体呈|：|:)?\s*(极强|身强|身偏强|中和|身偏弱|身弱|极弱)`)

func validateChapterProtectedFacts(ctx *chapterValidationContext) []chapterRuleResult {
	if ctx.Frozen.Facts == nil {
		return []chapterRuleResult{passedRule("FACT_PROTECTED_SKIPPED", "确定性事实核对", "fact_consistency")}
	}
	facts := ctx.Frozen.Facts
	content := moduleContents(ctx.Parsed.Modules)
	rules := []chapterRuleResult{}
	allowedGanZhi := map[string]struct{}{}
	for _, pillar := range []model.Pillar{facts.Chart.Data.Pillars.Year, facts.Chart.Data.Pillars.Month, facts.Chart.Data.Pillars.Day, facts.Chart.Data.Pillars.Hour} {
		if pillar.Stem != "" && pillar.Branch != "" {
			allowedGanZhi[pillar.Stem+pillar.Branch] = struct{}{}
		}
	}
	for _, cycle := range facts.LuckCycles {
		allowedGanZhi[cycle.GanZhi] = struct{}{}
	}
	for _, annual := range facts.AnnualFortunes {
		allowedGanZhi[annual.GanZhi] = struct{}{}
		if annual.LuckCycleGanZhi != "" {
			allowedGanZhi[annual.LuckCycleGanZhi] = struct{}{}
		}
	}
	seenGanZhi := map[string]struct{}{}
	for _, value := range ganZhiPattern.FindAllString(content, -1) {
		if _, seen := seenGanZhi[value]; seen {
			continue
		}
		seenGanZhi[value] = struct{}{}
		if _, ok := allowedGanZhi[value]; !ok {
			rules = append(rules, failedRule("FACT_GANZHI", "干支事实一致性", "fact_consistency", true, "冻结命盘、大运或流年中的干支", value, []string{"facts.chart", "facts.luck_cycles", "facts.annual_fortunes", "output.模块[*].正文"}, "正文出现冻结事实中不存在的干支"))
		}
	}
	allowedYears := map[string]struct{}{fmt.Sprintf("%d", facts.Input.Year): {}}
	for _, cycle := range facts.LuckCycles {
		if cycle.StartYear > 0 {
			allowedYears[fmt.Sprintf("%d", cycle.StartYear)] = struct{}{}
		}
	}
	for _, annual := range facts.AnnualFortunes {
		allowedYears[fmt.Sprintf("%d", annual.Year)] = struct{}{}
	}
	seenYears := map[string]struct{}{}
	for _, value := range yearPattern.FindAllString(content, -1) {
		if _, seen := seenYears[value]; seen {
			continue
		}
		seenYears[value] = struct{}{}
		if _, ok := allowedYears[value]; !ok {
			rules = append(rules, failedRule("FACT_ANNUAL_YEAR", "年份事实一致性", "fact_consistency", true, "冻结出生年、大运起始年或流年年份", value, []string{"facts.input.year", "facts.luck_cycles", "facts.annual_fortunes", "output.模块[*].正文"}, "正文出现冻结事实中不存在的年份"))
		}
	}
	if match := dayMasterPattern.FindStringSubmatch(content); len(match) == 2 && facts.Chart.Data.DayMaster.Stem != "" && match[1] != facts.Chart.Data.DayMaster.Stem {
		rules = append(rules, failedRule("FACT_DAY_MASTER", "日主事实一致性", "fact_consistency", true, facts.Chart.Data.DayMaster.Stem, match[1], []string{"facts.chart.data.day_master.stem", "output.模块[*].正文"}, "正文日主与冻结命盘不一致"))
	}
	if ctx.Locale == "zh" && facts.DayMasterStrength.Level != "" {
		expected := displaydict.ZHIn("strength", facts.DayMasterStrength.Level)
		if match := strengthPattern.FindStringSubmatch(content); len(match) == 2 && expected != "" && match[1] != expected {
			rules = append(rules, failedRule("FACT_STRENGTH", "身强弱事实一致性", "fact_consistency", true, expected, match[1], []string{"facts.day_master_strength.level", "output.模块[*].正文"}, "正文身强弱结论与冻结事实不一致"))
		}
	}
	if facts.UsefulGod != nil && facts.UsefulGod.Primary != "" {
		if match := usefulGodPattern.FindStringSubmatch(content); len(match) == 2 && match[1] != facts.UsefulGod.Primary {
			rules = append(rules, failedRule("FACT_USEFUL_GOD", "首选用神一致性", "fact_consistency", true, facts.UsefulGod.Primary, match[1], []string{"facts.useful_god.primary", "output.模块[*].正文"}, "正文首选用神与冻结事实不一致"))
		}
	}
	if len(rules) == 0 {
		rules = append(rules, passedRule("FACT_PROTECTED_VALID", "确定性事实核对", "fact_consistency"))
	}
	return rules
}

func validateChapterEnvelope(raw string) ([]chapterRuleResult, chapterOutput, bool) {
	rules := []chapterRuleResult{}
	if !utf8.ValidString(raw) {
		rules = append(rules, failedRule("FMT_ENCODING", "UTF-8编码", "format", true, "合法UTF-8", "非法编码", nil, "输出不是合法UTF-8"))
		return rules, chapterOutput{}, false
	}
	if len(raw) > maxChapterOutputBytes {
		rules = append(rules, failedRule("FMT_SIZE_LIMIT", "输出大小", "format", true, fmt.Sprintf("不超过%d字节", maxChapterOutputBytes), fmt.Sprintf("%d字节", len(raw)), nil, "输出超过章节大小限制"))
		return rules, chapterOutput{}, false
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		rules = append(rules, failedRule("FMT_EMPTY", "非空输出", "format", true, "单一JSON对象", "空输出", nil, "模型输出为空"))
		return rules, chapterOutput{}, false
	}
	if strings.Contains(trimmed, "```") {
		rules = append(rules, failedRule("FMT_MARKDOWN", "禁止Markdown包装", "format", true, "无代码块", "包含代码块", nil, "输出包含Markdown代码块"))
		return rules, chapterOutput{}, false
	}
	var generic any
	if err := json.Unmarshal([]byte(trimmed), &generic); err != nil {
		rules = append(rules, failedRule("FMT_JSON_INVALID", "合法JSON", "format", true, "合法JSON对象", err.Error(), nil, "输出不是合法JSON"))
		return rules, chapterOutput{}, false
	}
	if jsonDepth(generic) > maxChapterJSONDepth {
		rules = append(rules, failedRule("FMT_DEPTH_LIMIT", "JSON嵌套深度", "format", true, fmt.Sprintf("不超过%d层", maxChapterJSONDepth), fmt.Sprintf("%d层", jsonDepth(generic)), nil, "JSON嵌套过深"))
		return rules, chapterOutput{}, false
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	decoder.DisallowUnknownFields()
	var parsed chapterOutput
	if err := decoder.Decode(&parsed); err != nil {
		rules = append(rules, failedRule("FMT_UNKNOWN_FIELD", "字段白名单", "format", true, "仅包含约定字段", err.Error(), nil, "输出包含未知字段或字段类型错误"))
		return rules, chapterOutput{}, false
	}
	if err := ensureJSONEOF(decoder); err != nil {
		rules = append(rules, failedRule("FMT_SINGLE_JSON", "单一JSON对象", "format", true, "仅一个JSON对象", err.Error(), nil, "JSON对象之外存在额外内容"))
		return rules, chapterOutput{}, false
	}
	rules = append(rules, passedRule("FMT_VALID", "格式校验", "format"))
	return rules, parsed, true
}

func validateChapterSchema(ctx *chapterValidationContext) []chapterRuleResult {
	rules := []chapterRuleResult{}
	if ctx.Parsed.Chapter != ctx.Definition.Name {
		rules = append(rules, failedRule("SCH_CHAPTER_KEY", "章节名称", "schema", true, ctx.Definition.Name, ctx.Parsed.Chapter, []string{"output.章节"}, "章节名称与调用计划不一致"))
	}
	if len(ctx.Parsed.Modules) != len(ctx.Definition.Sections) {
		rules = append(rules, failedRule("SCH_MODULE_COUNT", "模块数量", "schema", true, fmt.Sprintf("%d", len(ctx.Definition.Sections)), fmt.Sprintf("%d", len(ctx.Parsed.Modules)), []string{"output.模块"}, "模块数量不符合章节协议"))
		return rules
	}
	seen := map[string]struct{}{}
	for i, section := range ctx.Definition.Sections {
		module := ctx.Parsed.Modules[i]
		path := fmt.Sprintf("output.模块[%d]", i)
		if module.No != i+1 {
			rules = append(rules, failedRule("SCH_MODULE_ORDER", "模块序号", "schema", true, fmt.Sprintf("%d", i+1), fmt.Sprintf("%d", module.No), []string{path + ".序号"}, "模块序号或顺序错误"))
		}
		if module.Name != section.Name {
			rules = append(rules, failedRule("SCH_MODULE_NAME", "模块名称", "schema", true, section.Name, module.Name, []string{path + ".名称"}, "模块名称与章节协议不一致"))
		}
		if _, ok := seen[module.Name]; ok {
			rules = append(rules, failedRule("SCH_DUPLICATE_MODULE", "模块唯一性", "schema", true, "模块名称唯一", module.Name, []string{path + ".名称"}, "模块名称重复"))
		}
		seen[module.Name] = struct{}{}
	}
	if len(rules) == 0 {
		rules = append(rules, passedRule("SCH_VALID", "Schema校验", "schema"))
	}
	return rules
}

func validateChapterCompleteness(ctx *chapterValidationContext) []chapterRuleResult {
	rules := []chapterRuleResult{}
	seenContent := map[string]int{}
	placeholders := []string{"待补充", "暂无分析", "无法判断", "信息不足", "placeholder", "to be completed", "insufficient information", "未定", "추후 보완"}
	for i, module := range ctx.Parsed.Modules {
		path := fmt.Sprintf("output.模块[%d].正文", i)
		content := strings.TrimSpace(module.Content)
		lower := strings.ToLower(content)
		for _, placeholder := range placeholders {
			if strings.Contains(lower, strings.ToLower(placeholder)) {
				rules = append(rules, failedRule("CMP_PLACEHOLDER", "禁止占位内容", "completeness", true, "完整正文", placeholder, []string{path}, "模块包含占位或拒答内容"))
				break
			}
		}
		normalized := normalizeForDuplicate(content)
		if previous, ok := seenContent[normalized]; normalized != "" && ok {
			rules = append(rules, failedRule("CMP_DUPLICATED", "模块内容去重", "completeness", true, "模块内容互不重复", fmt.Sprintf("与模块%d重复", previous+1), []string{path}, "模块正文重复"))
		} else {
			seenContent[normalized] = i
		}
	}
	if len(rules) == 0 {
		rules = append(rules, passedRule("CMP_VALID", "完整度校验", "completeness"))
	}
	return rules
}

func validateChapterLanguageAndTerms(ctx *chapterValidationContext) []chapterRuleResult {
	rules := []chapterRuleResult{}
	joined := moduleContents(ctx.Parsed.Modules)
	if !matchesTargetLanguage(ctx.Locale, joined) {
		rules = append(rules, failedRule("LANG_TARGET", "目标语言", "language_terminology", true, ctx.Locale, "正文主体语言不匹配", []string{"output.模块[*].正文"}, "正文未使用目标语言"))
	}
	if ctx.Locale != "zh" {
		for _, term := range ctx.Frozen.Glossary {
			if strings.Contains(joined, term.Source) && !strings.Contains(joined, term.Target) {
				rules = append(rules, failedRule("TERM_REQUIRED", "指定专业术语", "language_terminology", true, term.Target, term.Source, []string{"frozen.terminology_snapshot", "output.模块[*].正文"}, "使用了中文专业词但未采用指定译名"))
			}
		}
	}
	if len(rules) == 0 {
		rules = append(rules, passedRule("LANG_TERM_VALID", "语言与术语校验", "language_terminology"))
	}
	return rules
}

func validateChapterProductBoundary(ctx *chapterValidationContext) []chapterRuleResult {
	rules := []chapterRuleResult{}
	checks := []struct {
		code string
		name string
		re   *regexp.Regexp
	}{
		{"SAFE_LIFESPAN", "绝对寿命", regexp.MustCompile(`(?i)(死亡年份|寿命[为是]|exact death|death year|死亡する年|사망 연도)`)},
		{"SAFE_MEDICAL", "医学诊断", regexp.MustCompile(`(?i)(确诊|一定患|必患|diagnosed with|certainly develop|確実に発症|반드시.{0,8}(암|질환))`)},
		{"SAFE_INVESTMENT", "具体投资操作", regexp.MustCompile(`(?i)(立即买入|立即卖出|全仓|梭哈|buy now|sell now|all-in|今すぐ購入|전량 매수)`)},
		{"SAFE_PROFIT", "收益保证", regexp.MustCompile(`(?i)(保证发财|保证收益|稳赚|guaranteed profit|guaranteed return|必ず儲か|수익 보장)`)},
		{"SAFE_CERTAINTY", "绝对断言", regexp.MustCompile(`(?i)(百分之百|100%|绝对会|必定会|一定会|without exception|definitely will|間違いなく.{0,8}する|반드시.{0,8}할 것이다)`)},
		{"SAFE_PROMPT_LEAK", "内部指令泄露", regexp.MustCompile(`(?i)(system prompt|developer message|内部指令|系统提示词|プロンプト全文|시스템 프롬프트)`)},
		{"SAFE_PRIVATE_DATA", "隐私数据", regexp.MustCompile(`(?i)([a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}|(?:\+?86[- ]?)?1[3-9][0-9]{9})`)},
	}
	for i, module := range ctx.Parsed.Modules {
		for _, check := range checks {
			if matched := check.re.FindString(module.Content); matched != "" {
				rules = append(rules, failedRule(check.code, check.name, "product_boundary", true, "不出现受限内容", matched, []string{fmt.Sprintf("output.模块[%d].正文", i)}, "正文违反产品边界"))
			}
		}
	}
	if len(rules) == 0 {
		rules = append(rules, passedRule("SAFE_VALID", "产品边界校验", "product_boundary"))
	}
	return rules
}

func finalizeChapterValidation(result chapterValidation, rules []chapterRuleResult) chapterValidation {
	result.Rules = rules
	result.Passed = true
	result.Retryable = false
	for _, rule := range rules {
		if rule.Passed || rule.Severity == validationWarning {
			continue
		}
		result.Passed = false
		result.Retryable = result.Retryable || rule.Retryable
		result.Errors = append(result.Errors, rule.Code+": "+rule.Message)
		if result.Code == "" {
			result.Code = rule.Code
		}
	}
	if result.Passed {
		result.SchemaValid = stagePassed(rules, "format") && stagePassed(rules, "schema")
		return result
	}
	result.Summary = strings.Join(result.Errors, "; ")
	return result
}

func stagePassed(rules []chapterRuleResult, stage string) bool {
	for _, rule := range rules {
		if rule.Stage == stage && !rule.Passed && rule.Severity == validationError {
			return false
		}
	}
	return true
}

func passedRule(code, name, stage string) chapterRuleResult {
	return chapterRuleResult{Code: code, Name: name, Stage: stage, Severity: validationError, Passed: true}
}

func failedRule(code, name, stage string, retryable bool, expected, actual string, refs []string, message string) chapterRuleResult {
	return chapterRuleResult{Code: code, Name: name, Stage: stage, Severity: validationError, Passed: false, Retryable: retryable, Expected: expected, Actual: actual, EvidenceRefs: refs, Message: message}
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

func jsonDepth(value any) int {
	switch typed := value.(type) {
	case map[string]any:
		max := 1
		for _, child := range typed {
			if depth := 1 + jsonDepth(child); depth > max {
				max = depth
			}
		}
		return max
	case []any:
		max := 1
		for _, child := range typed {
			if depth := 1 + jsonDepth(child); depth > max {
				max = depth
			}
		}
		return max
	default:
		return 1
	}
}

func normalizeForDuplicate(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}

func moduleContents(modules []chapterModuleOutput) string {
	parts := make([]string, len(modules))
	for i := range modules {
		parts[i] = modules[i].Content
	}
	return strings.Join(parts, "\n")
}

func matchesTargetLanguage(locale, value string) bool {
	var latin, han, kana, hangul int
	for _, r := range value {
		switch {
		case unicode.In(r, unicode.Hangul):
			hangul++
		case unicode.In(r, unicode.Hiragana, unicode.Katakana):
			kana++
		case unicode.In(r, unicode.Han):
			han++
		case unicode.IsLetter(r) && r <= unicode.MaxLatin1:
			latin++
		}
	}
	switch locale {
	case "zh":
		return han >= 10 && kana == 0 && hangul == 0
	case "en":
		return latin >= 30 && latin > han*2 && kana == 0 && hangul == 0
	case "ja":
		return kana >= 5
	case "ko":
		return hangul >= 10
	default:
		return false
	}
}
