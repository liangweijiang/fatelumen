package prompts

import (
	"encoding/json"
	"fmt"
	"strings"

	"fatelumen/backend/internal/bazi/displaydict"
	"fatelumen/backend/internal/model"
)

const ChapterRegistryVersion = "full-report-10-chapters-v2"

type LocaleSpec struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Instruction string `json:"instruction"`
}

type ChapterDefinition struct {
	No            int                 `json:"no"`
	Key           string              `json:"key"`
	Name          string              `json:"name"`
	Purpose       string              `json:"purpose"`
	RequiredFacts []string            `json:"required_facts"`
	DefaultFacts  []string            `json:"default_facts"`
	MaxTokens     int                 `json:"max_tokens"`
	Sections      []SectionDefinition `json:"sections"`
	SourcePrompt  string              `json:"source_prompt"`
}

type SectionDefinition struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type ChapterPromptPreview struct {
	RegistryVersion     string                      `json:"registry_version"`
	PromptVersion       string                      `json:"prompt_version"`
	Locale              LocaleSpec                  `json:"locale"`
	Chapter             ChapterDefinition           `json:"chapter"`
	SystemPrompt        string                      `json:"system_prompt"`
	UserPrompt          string                      `json:"user_prompt"`
	InputFacts          map[string]any              `json:"input_facts"`
	OutputSchema        map[string]any              `json:"output_schema"`
	FactsHash           string                      `json:"facts_hash"`
	DictionaryVersion   string                      `json:"dictionary_version"`
	SemanticDigest      SemanticDigest              `json:"semantic_digest"`
	ChapterInstruction  string                      `json:"chapter_instruction"`
	AdditiveInstruction string                      `json:"additive_instruction"`
	CompleteInstruction string                      `json:"complete_instruction"`
	TerminologyCoverage displaydict.Coverage        `json:"terminology_coverage"`
	PhraseCoverage      PhraseCoverage              `json:"phrase_coverage"`
	Glossary            []displaydict.GlossaryEntry `json:"glossary"`
}

var localeRegistry = map[string]LocaleSpec{
	"zh": {Code: "zh", Name: "中文", Instruction: "所有正文直接使用简体中文撰写。"},
	"en": {Code: "en", Name: "English", Instruction: "所有正文直接使用自然、专业的英文撰写，干支保留中文字符。"},
	"ja": {Code: "ja", Name: "日本語", Instruction: "所有正文直接使用自然、专业的日文撰写，干支保留中文字符。"},
	"ko": {Code: "ko", Name: "한국어", Instruction: "所有正文直接使用自然、专业的韩文撰写，干支保留中文字符。"},
}

var chapterRegistry = []ChapterDefinition{
	chapter(1, "destiny_depth", "命格深析", "解释日主强弱、五行喜忌、十神结构、四柱宫位与关键应期。", []string{"chart", "day_master_strength"}, []string{"chart", "element_strength", "day_master_strength", "ten_god_structure", "stem_relations", "branch_relations", "pattern_candidates", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096, sections("day_master_strength:日主强弱深层拆解", "element_logic:五行生克与喜忌逻辑", "ten_gods_mapping:十神组合与人生映射", "pillars_palaces:四柱宫位详解", "luck_analysis:大运推演", "key_periods:关键应期")),
	chapter(2, "ten_gods_full", "十神全览", "逐项解释十神的出现、力量、位置及现实含义。", []string{"chart", "ten_god_structure"}, []string{"chart", "day_master_strength", "ten_god_structure", "stem_relations", "branch_relations"}, 4096, sections("bi_jian:比肩", "jie_cai:劫财", "shi_shen:食神", "shang_guan:伤官", "zheng_cai:正财", "pian_cai:偏财", "zheng_guan:正官", "qi_sha:七杀", "zheng_yin:正印", "pian_yin:偏印")),
	chapter(3, "luck_cycle", "大运走势", "解释起运、当前大运、下一步大运、换运节点和人生节奏。", []string{"chart", "luck_cycles"}, []string{"chart", "day_master_strength", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096, sections("overview:起运与一生大运总览", "current:当前大运深度解读", "next:下一步大运展望", "transition:换运节点提醒", "advice:核心建议")),
	chapter(4, "ten_year_years", "未来十年流年", "严格解释后端给出的十年流年及月份干支，不自行推算。", []string{"chart", "luck_cycles", "annual_fortunes"}, []string{"chart", "day_master_strength", "ten_god_structure", "stem_relations", "branch_relations", "favorable_elements", "luck_cycles", "annual_fortunes"}, 8192, sections("overview:总运", "career_wealth:事业财运", "love_family:感情家庭", "health:健康", "key_months:关键月份", "advice:建议")),
	chapter(5, "career_depth", "事业深析", "解释事业模式、职业方向、职场关系、阶段机会与风险。", []string{"chart", "ten_god_structure"}, []string{"chart", "day_master_strength", "ten_god_structure", "pattern_candidates", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096, sections("overview:事业格局定性", "direction:职业方向指引", "wealth_logic:求财方式与逻辑", "key_periods:事业关键节点", "workplace:职场环境与人事", "entrepreneurship:创业与合伙时机", "risks:风险与避坑指南")),
	chapter(6, "wealth_depth", "财富格局", "解释财富结构、求财路径、现金流习惯和方向性风险管理。", []string{"chart", "ten_god_structure"}, []string{"chart", "day_master_strength", "ten_god_structure", "pattern_candidates", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096, sections("overview:财富总量定性", "wealth_stars:财星状态拆解", "path:求财方式与路径", "retention:财富流动与守成能力", "turning_points:财富转折点", "risks:破财风险与避坑", "allocation:财富配置建议")),
	chapter(7, "love_depth", "情感姻缘", "解释夫妻宫、配偶星、关系模式、阶段信号和沟通建议。", []string{"chart", "branch_relations"}, []string{"chart", "ten_god_structure", "stem_relations", "branch_relations", "luck_cycles", "annual_fortunes"}, 4096, sections("overview:情感格局定性", "partner:配偶画像", "spouse_palace:夫妻宫深度拆解", "timeline:感情时间线与关键节点", "relationship:相处模式与经营建议", "risks:风险与避坑指南", "current:当前感情指引")),
	chapter(8, "health_depth", "健康养生", "基于寒热燥湿和五行偏颇提供非诊断性的生活方式建议。", []string{"element_strength", "climate"}, []string{"chart", "element_strength", "climate", "disease", "mediation", "favorable_elements", "luck_cycles"}, 4096, sections("constitution:先天体质定性", "elements_organs:五行与五脏对应拆解", "risk_cycles:大运与健康风险期", "annual_health:流年健康应期", "daily:日常养生方向", "emotion:情绪与精神健康", "stage:阶段性养生重点")),
	chapter(9, "element_tuning", "五行调候", "解释寒暖湿燥、调候需求、环境方向和大运变化。", []string{"element_strength", "climate", "useful_god"}, []string{"chart", "element_strength", "climate", "disease", "mediation", "useful_god", "favorable_elements", "luck_cycles"}, 4096, sections("overview:调候定性", "climate_detail:寒暖湿燥拆解", "useful_element:调候用神分析", "health:调候对健康的影响", "emotion:调候对性格与情绪的影响", "environment:环境与方位调候", "luck_changes:大运调候变化")),
	chapter(10, "life_plan", "人生规划", "汇总阶段战略、关键窗口、风险地图与近期行动优先级。", []string{"chart", "luck_cycles", "annual_fortunes"}, []string{"chart", "element_strength", "day_master_strength", "ten_god_structure", "pattern_candidates", "climate", "useful_god", "favorable_elements", "luck_cycles", "annual_fortunes"}, 4096, sections("overview:人生总纲", "stages:分阶段战略", "windows:关键窗口期", "risks:风险地图", "career_wealth:财富与事业主线", "love_family:感情与家庭主线", "health:健康主线", "annual_action:年度行动纲领")),
}

func chapter(no int, key, name, purpose string, required, defaults []string, maxTokens int, items []SectionDefinition) ChapterDefinition {
	return ChapterDefinition{no, key, name, purpose, required, defaults, maxTokens, items, chapterTemplatesZH[key]}
}
func sections(items ...string) []SectionDefinition {
	out := make([]SectionDefinition, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, ":", 2)
		out = append(out, SectionDefinition{parts[0], parts[1]})
	}
	return out
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
		out[i].Sections = append([]SectionDefinition(nil), chapter.Sections...)
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
	chapter, ok := ChapterByKey(chapterKey)
	if !ok {
		return nil, fmt.Errorf("unknown chapter %q", chapterKey)
	}
	return BuildChapterPromptPreviewWithFacts(locale, chapterKey, chapter.DefaultFacts, facts)
}

// BuildChapterPromptPreviewWithFacts builds a preview from an administrator-selected
// fact allow-list. Required facts are always enforced by the backend registry.
func BuildChapterPromptPreviewWithFacts(locale, chapterKey string, factKeys []string, facts model.InterpretationFacts) (*ChapterPromptPreview, error) {
	localeSpec, err := NormalizeLocale(locale)
	if err != nil {
		return nil, err
	}
	chapter, ok := ChapterByKey(chapterKey)
	if !ok {
		return nil, fmt.Errorf("unknown chapter %q", chapterKey)
	}
	selected, err := ResolveChapterFactKeys(chapterKey, factKeys)
	if err != nil {
		return nil, err
	}
	filtered, err := selectChapterFacts(facts, selected)
	if err != nil {
		return nil, err
	}
	system := "你只能解读系统提供的确定性事实，不得重新排盘，不得修改或自行推算干支、大运、流年及月份干支。"
	digest, err := buildSemanticDigest(chapterKey, localeSpec.Code, facts, selected)
	if err != nil {
		return nil, err
	}
	chapterInstruction := composeAuthoredPrompt(chapter.SourcePrompt, digest.Text())
	languageInstruction, glossary := buildLanguageInstruction(localeSpec, chapterInstruction)
	additiveInstruction := languageInstruction
	user := chapterInstruction + "\n\n" + languageInstruction + "\n\n" + jsonOutputRequirement(chapter)
	completeInstruction := system + "\n\n" + user
	sectionNames := make([]string, 0, len(chapter.Sections))
	for _, section := range chapter.Sections {
		sectionNames = append(sectionNames, section.Name)
	}
	schema := map[string]any{"type": "object", "required": []string{"章节", "模块"}, "properties": map[string]any{"章节": map[string]any{"const": chapter.Name}, "模块": map[string]any{"type": "array", "minItems": len(chapter.Sections), "maxItems": len(chapter.Sections), "items": map[string]any{"type": "object", "required": []string{"序号", "名称", "正文"}, "properties": map[string]any{"序号": map[string]any{"type": "integer", "minimum": 1, "maximum": len(chapter.Sections)}, "名称": map[string]any{"type": "string", "enum": sectionNames}, "正文": map[string]any{"type": "string", "minLength": 1}}}}}}
	return &ChapterPromptPreview{ChapterRegistryVersion, ReportPromptVersion, localeSpec, chapter, system, user, filtered, schema, facts.FactsHash, displaydict.Version, digest, chapterInstruction, additiveInstruction, completeInstruction, displaydict.LocaleCoverage(localeSpec.Code), phraseCoverage(localeSpec.Code), glossary}, nil
}

func jsonOutputRequirement(chapter ChapterDefinition) string {
	var builder strings.Builder
	builder.WriteString("输出必须是一个合法 JSON 对象，禁止输出 Markdown、代码块或 JSON 之外的文字。模块数量、序号、名称和顺序必须与下列结构完全一致，每个正文都必须填写：\n{\n  \"章节\": \"")
	builder.WriteString(chapter.Name)
	builder.WriteString("\",\n  \"模块\": [\n")
	for index, section := range chapter.Sections {
		if index > 0 {
			builder.WriteString(",\n")
		}
		builder.WriteString(fmt.Sprintf("    {\"序号\": %d, \"名称\": \"%s\", \"正文\": \"...\"}", index+1, section.Name))
	}
	builder.WriteString("\n  ]\n}")
	return builder.String()
}

// runtimeChapterPrompt keeps the source document available for traceability,
// while excluding its illustrative sample and author TODOs from a real call.
func runtimeChapterPrompt(source string) string {
	end := len(source)
	for _, marker := range []string{"\n示例输入：", "\nTODO："} {
		if index := strings.Index(source, marker); index >= 0 && index < end {
			end = index
		}
	}
	return strings.TrimSpace(source[:end])
}

// ResolveChapterFactKeys validates configurable keys and appends mandatory keys.
func ResolveChapterFactKeys(chapterKey string, factKeys []string) ([]string, error) {
	chapter, ok := ChapterByKey(chapterKey)
	if !ok {
		return nil, fmt.Errorf("unknown chapter %q", chapterKey)
	}
	allowed := make(map[string]struct{})
	for _, definition := range chapterRegistry {
		for _, key := range definition.DefaultFacts {
			allowed[key] = struct{}{}
		}
	}
	selected := make([]string, 0, len(factKeys)+len(chapter.RequiredFacts))
	seen := make(map[string]struct{})
	for _, key := range append(append([]string(nil), factKeys...), chapter.RequiredFacts...) {
		if _, ok := allowed[key]; !ok {
			return nil, fmt.Errorf("unsupported fact key %q", key)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		selected = append(selected, key)
	}
	return selected, nil
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
