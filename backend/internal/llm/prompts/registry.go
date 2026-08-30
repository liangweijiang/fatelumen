package prompts

import (
	"encoding/json"
	"fmt"
	"strings"

	"fatelumen/backend/internal/model"
)

const ChapterRegistryVersion = "full-report-10-chapters-v2"

type LocaleSpec struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Instruction string `json:"instruction"`
}

type ChapterDefinition struct {
	No            int      `json:"no"`
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	Purpose       string   `json:"purpose"`
	RequiredFacts []string `json:"required_facts"`
	DefaultFacts  []string `json:"default_facts"`
	MaxTokens     int      `json:"max_tokens"`
}

type ChapterPromptPreview struct {
	RegistryVersion string            `json:"registry_version"`
	PromptVersion   string            `json:"prompt_version"`
	Locale          LocaleSpec        `json:"locale"`
	Chapter         ChapterDefinition `json:"chapter"`
	SystemPrompt    string            `json:"system_prompt"`
	UserPrompt      string            `json:"user_prompt"`
	InputFacts      map[string]any    `json:"input_facts"`
	OutputSchema    map[string]any    `json:"output_schema"`
	FactsHash       string            `json:"facts_hash"`
}

var localeRegistry = map[string]LocaleSpec{
	"zh": {Code: "zh", Name: "中文", Instruction: "Write every user-visible title and body directly in Simplified Chinese."},
	"en": {Code: "en", Name: "English", Instruction: "Write every user-visible title and body directly in English."},
	"ja": {Code: "ja", Name: "日本語", Instruction: "Write every user-visible title and body directly in natural Japanese."},
	"ko": {Code: "ko", Name: "한국어", Instruction: "Write every user-visible title and body directly in natural Korean."},
}

var chapterRegistry = []ChapterDefinition{
	{1, "destiny_depth", "命格深析", "解释日主强弱、五行喜忌、十神结构、四柱宫位与关键应期。", []string{"chart", "day_master_strength"}, []string{"chart", "element_strength", "day_master_strength", "ten_god_structure", "stem_relations", "branch_relations", "pattern_candidates", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096},
	{2, "ten_gods_full", "十神全览", "逐项解释十神的出现、力量、位置及现实含义。", []string{"chart", "ten_god_structure"}, []string{"chart", "day_master_strength", "ten_god_structure", "stem_relations", "branch_relations"}, 4096},
	{3, "luck_cycle", "大运走势", "解释起运、当前大运、下一步大运、换运节点和人生节奏。", []string{"chart", "luck_cycles"}, []string{"chart", "day_master_strength", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096},
	{4, "ten_year_years", "未来十年流年", "严格解释后端给出的十年流年及月份干支，不自行推算。", []string{"chart", "luck_cycles", "annual_fortunes"}, []string{"chart", "day_master_strength", "ten_god_structure", "stem_relations", "branch_relations", "favorable_elements", "luck_cycles", "annual_fortunes"}, 8192},
	{5, "career_depth", "事业深析", "解释事业模式、职业方向、职场关系、阶段机会与风险。", []string{"chart", "ten_god_structure"}, []string{"chart", "day_master_strength", "ten_god_structure", "pattern_candidates", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096},
	{6, "wealth_depth", "财富格局", "解释财富结构、求财路径、现金流习惯和方向性风险管理。", []string{"chart", "ten_god_structure"}, []string{"chart", "day_master_strength", "ten_god_structure", "pattern_candidates", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096},
	{7, "love_depth", "情感姻缘", "解释夫妻宫、配偶星、关系模式、阶段信号和沟通建议。", []string{"chart", "branch_relations"}, []string{"chart", "ten_god_structure", "stem_relations", "branch_relations", "luck_cycles", "annual_fortunes"}, 4096},
	{8, "health_depth", "健康养生", "基于寒热燥湿和五行偏颇提供非诊断性的生活方式建议。", []string{"element_strength", "climate"}, []string{"chart", "element_strength", "climate", "disease", "mediation", "favorable_elements", "luck_cycles"}, 4096},
	{9, "element_tuning", "五行调候", "解释寒暖湿燥、调候需求、环境方向和大运变化。", []string{"element_strength", "climate", "useful_god"}, []string{"chart", "element_strength", "climate", "disease", "mediation", "useful_god", "favorable_elements", "luck_cycles"}, 4096},
	{10, "life_plan", "人生规划", "汇总阶段战略、关键窗口、风险地图与近期行动优先级。", []string{"chart", "luck_cycles", "annual_fortunes"}, []string{"chart", "element_strength", "day_master_strength", "ten_god_structure", "pattern_candidates", "climate", "useful_god", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096},
}

func SupportedLocales() []LocaleSpec {
	return []LocaleSpec{localeRegistry["zh"], localeRegistry["en"], localeRegistry["ja"], localeRegistry["ko"]}
}

func NormalizeLocale(locale string) (LocaleSpec, error) {
	spec, ok := localeRegistry[strings.ToLower(strings.TrimSpace(locale))]
	if !ok {
		return LocaleSpec{}, fmt.Errorf("unsupported locale %q", locale)
	}
	return spec, nil
}

func ChapterDefinitions() []ChapterDefinition {
	out := make([]ChapterDefinition, len(chapterRegistry))
	for i, chapter := range chapterRegistry {
		out[i] = chapter
		out[i].RequiredFacts = append([]string(nil), chapter.RequiredFacts...)
		out[i].DefaultFacts = append([]string(nil), chapter.DefaultFacts...)
	}
	return out
}

func ChapterByKey(key string) (ChapterDefinition, bool) {
	for _, chapter := range chapterRegistry {
		if chapter.Key == key {
			return chapter, true
		}
	}
	return ChapterDefinition{}, false
}

func BuildChapterPromptPreview(locale, chapterKey string, facts model.InterpretationFacts) (*ChapterPromptPreview, error) {
	localeSpec, err := NormalizeLocale(locale)
	if err != nil {
		return nil, err
	}
	chapter, ok := ChapterByKey(chapterKey)
	if !ok {
		return nil, fmt.Errorf("unknown chapter %q", chapterKey)
	}
	filtered, err := selectChapterFacts(facts, chapter.DefaultFacts)
	if err != nil {
		return nil, err
	}
	factJSON, err := json.Marshal(filtered)
	if err != nil {
		return nil, fmt.Errorf("marshal chapter facts: %w", err)
	}
	system := reportCommonRules + fmt.Sprintf(`
Target locale rule: %s
Chapter %d uses key %q and name %q.
Purpose: %s
Use only the supplied input_facts. Start with the conclusion, then explain the deterministic evidence in plain language.
Return exactly one chapter object matching the supplied schema.`, localeSpec.Instruction, chapter.No, chapter.Key, chapter.Name, chapter.Purpose)
	user := fmt.Sprintf("locale: %s\nchapter_key: %s\nfacts_hash: %s\ninput_facts: %s\n\nProduce STRICT JSON exactly as instructed.", localeSpec.Code, chapter.Key, facts.FactsHash, factJSON)
	schema := map[string]any{"type": "object", "required": []string{"chapters"}, "properties": map[string]any{"chapters": map[string]any{"type": "array", "minItems": 1, "maxItems": 1, "items": map[string]any{"type": "object", "required": []string{"no", "key", "title", "body"}, "properties": map[string]any{"no": map[string]any{"const": chapter.No}, "key": map[string]any{"const": chapter.Key}, "title": map[string]any{"type": "string"}, "body": map[string]any{"type": "string"}}}}}}
	return &ChapterPromptPreview{ChapterRegistryVersion, ReportPromptVersion, localeSpec, chapter, system, user, filtered, schema, facts.FactsHash}, nil
}

func selectChapterFacts(facts model.InterpretationFacts, keys []string) (map[string]any, error) {
	raw, err := json.Marshal(facts)
	if err != nil {
		return nil, fmt.Errorf("marshal facts: %w", err)
	}
	var all map[string]any
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("decode facts: %w", err)
	}
	selected := make(map[string]any, len(keys)+2)
	selected["facts_hash"] = facts.FactsHash
	selected["versions"] = all["versions"]
	for _, key := range keys {
		if value, ok := all[key]; ok {
			selected[key] = value
		}
	}
	return selected, nil
}
