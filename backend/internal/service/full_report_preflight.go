package service

import (
	"fmt"
	"strings"
	"time"
)

type FullReportPreflightResult struct {
	ValidatorVersion string                       `json:"validator_version"`
	Passed           bool                         `json:"passed"`
	CheckedAt        time.Time                    `json:"checked_at"`
	Errors           []string                     `json:"errors"`
	Warnings         []string                     `json:"warnings"`
	ChapterCount     int                          `json:"chapter_count"`
	Chapters         []FullReportChapterPreflight `json:"chapters"`
	Groups           []FullReportPreflightGroup   `json:"groups"`
}

type FullReportPreflightGroup struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Logic   string `json:"logic"`
	Passed  bool   `json:"passed"`
	Summary string `json:"summary"`
}

type FullReportChapterPreflight struct {
	ChapterNo  int      `json:"chapter_no"`
	ChapterKey string   `json:"chapter_key"`
	Passed     bool     `json:"passed"`
	Errors     []string `json:"errors"`
	Warnings   []string `json:"warnings"`
}

// PreflightFullReport is the single automatic gate used immediately before
// freezing a report. A future admin diagnostic endpoint must call this same
// function rather than implement a second rule set.
func PreflightFullReport(locale string, runtime FullReportRuntimeConfig, plans []frozenChapterPlan) FullReportPreflightResult {
	result := FullReportPreflightResult{ValidatorVersion: "preflight-v2", CheckedAt: time.Now().UTC(), Errors: []string{}, Warnings: []string{}, ChapterCount: len(plans), Chapters: make([]FullReportChapterPreflight, 0, len(plans))}
	if locale != "zh" && locale != "en" && locale != "ja" && locale != "ko" {
		result.Errors = append(result.Errors, "unsupported locale")
	}
	if runtime.ChapterConcurrency < 1 || runtime.ChapterConcurrency > 10 {
		result.Errors = append(result.Errors, "chapter concurrency must be between 1 and 10")
	}
	if runtime.ChapterTimeout <= 0 {
		result.Errors = append(result.Errors, "chapter timeout must be positive")
	}
	if len(runtime.Routes) == 0 {
		result.Errors = append(result.Errors, "at least one enabled model route is required")
	}
	for i, route := range runtime.Routes {
		if route.RouteNo != uint8(i+1) || route.ProviderConfigID == 0 || route.ModelConfigID == 0 || strings.TrimSpace(route.ProviderCode) == "" || strings.TrimSpace(route.BaseURL) == "" || strings.TrimSpace(route.Model) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("model route %d is incomplete", i+1))
		}
		if route.MaxAttempts < 1 || route.MaxRetries < 0 || route.MaxAttempts != route.MaxRetries+1 {
			result.Errors = append(result.Errors, fmt.Sprintf("model route %d retry policy is invalid", i+1))
		}
		if route.TimeoutSeconds < 1 {
			result.Errors = append(result.Errors, fmt.Sprintf("model route %d timeout is invalid", i+1))
		}
	}
	if len(plans) != 10 {
		result.Errors = append(result.Errors, "report must contain exactly ten chapters")
	}
	seenNo, seenKey := map[int]struct{}{}, map[string]struct{}{}
	for _, plan := range plans {
		chapter := FullReportChapterPreflight{ChapterNo: plan.Definition.No, ChapterKey: plan.Definition.Key, Errors: []string{}, Warnings: []string{}}
		if _, exists := seenNo[plan.Definition.No]; exists {
			chapter.Errors = append(chapter.Errors, "duplicate chapter number")
		}
		seenNo[plan.Definition.No] = struct{}{}
		if _, exists := seenKey[plan.Definition.Key]; exists {
			chapter.Errors = append(chapter.Errors, "duplicate chapter key")
		}
		seenKey[plan.Definition.Key] = struct{}{}
		if plan.Definition.No < 1 || plan.Definition.No > 10 || strings.TrimSpace(plan.Definition.Key) == "" || strings.TrimSpace(plan.Definition.Name) == "" {
			chapter.Errors = append(chapter.Errors, "chapter identity is invalid")
		}
		if plan.Preview == nil {
			chapter.Errors = append(chapter.Errors, "prompt preview is missing")
		} else {
			if strings.TrimSpace(plan.Preview.CompleteInstruction) == "" || strings.TrimSpace(plan.Preview.ChapterInstruction) == "" {
				chapter.Errors = append(chapter.Errors, "chapter prompt is empty")
			}
			if strings.TrimSpace(plan.Preview.AdditiveInstruction) == "" {
				chapter.Errors = append(chapter.Errors, "locale instruction is empty")
			}
			if plan.Preview.OutputSchema == nil {
				chapter.Errors = append(chapter.Errors, "output schema is missing")
			}
			for _, required := range plan.Definition.RequiredFacts {
				if _, exists := plan.Preview.InputFacts[required]; !exists {
					chapter.Errors = append(chapter.Errors, fmt.Sprintf("required fact %s is missing", required))
				}
			}
			if locale != "zh" && len(plan.Preview.Glossary) == 0 {
				chapter.Warnings = append(chapter.Warnings, "terminology glossary is empty; provider may translate by context")
			}
		}
		if len(plan.Definition.Sections) == 0 {
			chapter.Errors = append(chapter.Errors, "chapter modules are empty")
		}
		if strings.TrimSpace(plan.PromptHash) == "" {
			chapter.Errors = append(chapter.Errors, "prompt hash is missing")
		}
		chapter.Passed = len(chapter.Errors) == 0
		result.Errors = append(result.Errors, chapter.Errors...)
		result.Warnings = append(result.Warnings, chapter.Warnings...)
		result.Chapters = append(result.Chapters, chapter)
	}
	result.Passed = len(result.Errors) == 0
	result.Groups = buildPreflightGroups(locale, runtime, plans, result)
	return result
}

func buildPreflightGroups(locale string, runtime FullReportRuntimeConfig, plans []frozenChapterPlan, result FullReportPreflightResult) []FullReportPreflightGroup {
	failedBy := func(parts ...string) bool {
		for _, message := range result.Errors {
			for _, part := range parts {
				if strings.Contains(message, part) {
					return true
				}
			}
		}
		return false
	}
	groups := []FullReportPreflightGroup{
		{Code: "locale", Name: "目标语言配置", Logic: "目标语言必须为中文、英文、日文或韩文，每章必须生成对应语言附加指令；非中文术语表为空只记录警告。", Passed: !failedBy("unsupported locale", "locale instruction is empty"), Summary: fmt.Sprintf("目标语言 %s，检查 %d 章语言指令", locale, len(plans))},
		{Code: "runtime", Name: "报告执行配置", Logic: "十章并发数必须为1～10，单章调用超时必须为正数，执行参数随后随报告冻结。", Passed: !failedBy("chapter concurrency", "chapter timeout"), Summary: fmt.Sprintf("并发 %d，单章超时 %s", runtime.ChapterConcurrency, runtime.ChapterTimeout)},
		{Code: "routes", Name: "模型调用链", Logic: "至少存在一条启用路由；供应商、Base URL、模型、顺序、重试预算和超时必须完整有效。", Passed: !failedBy("model route", "enabled model route"), Summary: fmt.Sprintf("检查 %d 条冻结路由", len(runtime.Routes))},
		{Code: "chapters", Name: "十章编排计划", Logic: "报告必须正好包含十章；章节编号、Key和名称有效且不重复，每章至少定义一个输出模块。", Passed: !failedBy("exactly ten chapters", "duplicate chapter", "chapter identity", "chapter modules"), Summary: fmt.Sprintf("检查 %d 个章节的身份、顺序与模块", len(plans))},
		{Code: "prompts", Name: "Prompt完整性", Logic: "每章必须具备Prompt预览、章节指令、完整调用指令、语言附加指令和Prompt哈希。", Passed: !failedBy("prompt preview", "chapter prompt", "locale instruction", "prompt hash"), Summary: fmt.Sprintf("检查 %d 章冻结指令与哈希", len(plans))},
		{Code: "contract", Name: "数据与输出协议", Logic: "每章必须具备输出Schema、完整模块定义和全部必需前置事实；术语表缺失仅作警告，不阻断调用。", Passed: !failedBy("output schema", "required fact", "chapter modules"), Summary: fmt.Sprintf("检查 %d 章Schema、模块及必需事实", len(plans))},
	}
	return groups
}
