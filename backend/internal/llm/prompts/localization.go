package prompts

import (
	"fmt"
	"strings"

	"fatelumen/backend/internal/bazi/displaydict"
)

// PhraseCoverage tracks reviewed domain sentence templates separately from
// individual terms. A translated noun is not enough to make a professional
// Bazi sentence safe for production.
type PhraseCoverage struct {
	Locale   string  `json:"locale"`
	Total    int     `json:"total"`
	Approved int     `json:"approved"`
	Draft    int     `json:"draft"`
	Missing  int     `json:"missing"`
	Rate     float64 `json:"rate"`
	Ready    bool    `json:"ready"`
}

var semanticPhraseKeys = []string{
	"profile.gender", "chart.pillars", "chart.day_master", "power.elements",
	"strength.result", "ten_gods.structure", "relations.stems", "relations.branches",
	"pattern.result", "climate.result", "disease.result", "mediation.result",
	"useful_god.result", "favorable_elements.result", "luck_cycles.summary", "annual_fortunes.summary",
}

func phraseCoverage(locale string) PhraseCoverage {
	locale = strings.ToLower(strings.TrimSpace(locale))
	coverage := PhraseCoverage{Locale: locale, Total: len(semanticPhraseKeys)}
	// The current semantic digest was authored and reviewed in Chinese. Other
	// locales remain explicitly missing until their domain templates are reviewed.
	if locale == "zh" {
		coverage.Approved = coverage.Total
	} else {
		coverage.Missing = coverage.Total
	}
	if coverage.Total > 0 {
		coverage.Rate = float64(coverage.Approved) / float64(coverage.Total)
	}
	coverage.Ready = coverage.Approved == coverage.Total
	return coverage
}

func buildLanguageInstruction(locale LocaleSpec, chapterText string) (string, []displaydict.GlossaryEntry) {
	if locale.Code == "zh" {
		return "【语言要求】\n所有正文使用自然、准确的简体中文。命理专业名称使用系统提供的中文标准名称，JSON 字段和模块 key 保持程序约定。", nil
	}
	glossary := displaydict.GlossaryForText(chapterText, locale.Code)
	var builder strings.Builder
	builder.WriteString("【目标语言要求】\n")
	builder.WriteString(locale.Instruction)
	builder.WriteString("\n命理专业名称必须优先使用下列指定译名；未列出的专业词根据上下文进行专业、自然的翻译，不得因此拒绝输出。\n同一术语在本章和整份报告中必须保持一致。甲、乙、子、丑等干支符号保留中文原文。JSON 字段和模块 key 不得翻译。")
	if len(glossary) > 0 {
		builder.WriteString("\n\n【本章指定术语】\n")
		for _, item := range glossary {
			builder.WriteString(fmt.Sprintf("- %s → %s\n", item.Source, item.Target))
		}
	}
	return strings.TrimSpace(builder.String()), glossary
}
