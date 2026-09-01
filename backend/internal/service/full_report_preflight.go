package service

import (
	"fmt"
	"strings"
	"time"
)

type FullReportPreflightResult struct {
	Passed       bool                         `json:"passed"`
	CheckedAt    time.Time                    `json:"checked_at"`
	Errors       []string                     `json:"errors"`
	Warnings     []string                     `json:"warnings"`
	ChapterCount int                          `json:"chapter_count"`
	Chapters     []FullReportChapterPreflight `json:"chapters"`
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
	result := FullReportPreflightResult{CheckedAt: time.Now().UTC(), Errors: []string{}, Warnings: []string{}, ChapterCount: len(plans), Chapters: make([]FullReportChapterPreflight, 0, len(plans))}
	if locale != "zh" && locale != "en" && locale != "ja" && locale != "ko" {
		result.Errors = append(result.Errors, "unsupported locale")
	}
	if runtime.ChapterConcurrency < 1 || runtime.ChapterConcurrency > 10 {
		result.Errors = append(result.Errors, "chapter concurrency must be between 1 and 10")
	}
	if runtime.MaxAttempts < 1 {
		result.Errors = append(result.Errors, "chapter max attempts must be positive")
	}
	if runtime.ChapterTimeout <= 0 {
		result.Errors = append(result.Errors, "chapter timeout must be positive")
	}
	if strings.TrimSpace(runtime.Provider) == "" || strings.TrimSpace(runtime.Model) == "" {
		result.Errors = append(result.Errors, "provider and model are required")
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
	return result
}
