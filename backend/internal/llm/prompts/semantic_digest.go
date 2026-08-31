package prompts

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"fatelumen/backend/internal/bazi/displaydict"
	"fatelumen/backend/internal/model"
)

const SemanticDigestVersion = "semantic-digest-v1"

type SemanticDigestItem struct {
	Fact   string `json:"fact"`
	Status string `json:"status"`
	Text   string `json:"text"`
}

type SemanticDigest struct {
	Version  string                 `json:"version"`
	Locale   string                 `json:"locale"`
	Items    []SemanticDigestItem   `json:"items"`
	Warnings []string               `json:"warnings,omitempty"`
	Coverage SemanticDigestCoverage `json:"coverage"`
}

type SemanticDigestCoverage struct {
	SelectedFacts    int      `json:"selected_facts"`
	CoveredFacts     int      `json:"covered_facts"`
	UnavailableFacts int      `json:"unavailable_facts"`
	PendingFacts     int      `json:"pending_facts"`
	CoverageRate     float64  `json:"coverage_rate"`
	MissingFacts     []string `json:"missing_facts,omitempty"`
	Ready            bool     `json:"ready"`
}

func buildSemanticDigest(chapterKey, locale string, facts model.InterpretationFacts, keys []string) (SemanticDigest, error) {
	if locale == "" {
		locale = "zh"
	}
	digest := SemanticDigest{Version: SemanticDigestVersion, Locale: locale}
	digest.Items = append(digest.Items,
		SemanticDigestItem{Fact: "gender", Status: "available", Text: "性别：" + chartGender(facts)},
		SemanticDigestItem{Fact: "pillars", Status: "available", Text: "八字：" + fourPillars(facts.Chart.Data.Pillars)},
		SemanticDigestItem{Fact: "day_master", Status: "available", Text: fmt.Sprintf("日主：%s（%s、%s）", facts.Chart.Data.DayMaster.Stem, facts.Chart.Data.DayMaster.Element, facts.Chart.Data.DayMaster.YinYang)},
	)
	for _, key := range keys {
		var items []SemanticDigestItem
		var warnings []string
		var err error
		switch key {
		case "chart":
			continue
		case "element_strength":
			items, warnings = summarizeElements(facts.ElementStrength)
		case "day_master_strength":
			items, err = summarizeStrength(facts.DayMasterStrength)
		case "ten_god_structure":
			items, warnings = summarizeTenGods(facts.TenGodStructure)
		case "stem_relations":
			items, warnings = summarizeRelations("天干关系", key, facts.StemRelations)
		case "branch_relations":
			items, warnings = summarizeRelations("地支关系", key, facts.BranchRelations)
		case "pattern_candidates":
			items, warnings = summarizePatterns(facts.PatternCandidates)
		case "climate":
			items = summarizeClimate(facts.Climate)
		case "disease":
			items = summarizeDisease(facts.Disease)
		case "mediation":
			items = summarizeMediation(facts.Mediation)
		case "useful_god":
			items = summarizeUsefulGod(facts.UsefulGod)
		case "favorable_elements":
			items = summarizeFavorable(facts.FavorableElements)
		case "luck_cycles":
			items = summarizeLuckCycles(chapterKey, facts.LuckCycles, referenceYear(facts))
		case "annual_fortunes":
			items = summarizeAnnualFortunes(chapterKey, facts.AnnualFortunes)
		default:
			warnings = []string{"未识别的章节事实：" + key}
		}
		if err != nil {
			return SemanticDigest{}, fmt.Errorf("summarize %s: %w", key, err)
		}
		if key != "chart" && len(items) == 0 {
			items = []SemanticDigestItem{{Fact: key, Status: "unavailable", Text: factDisplayName(key) + "：当前计算快照暂无可用数据。"}}
		}
		digest.Items = append(digest.Items, items...)
		digest.Warnings = append(digest.Warnings, warnings...)
	}
	digest.Items = uniqueDigestItems(digest.Items)
	digest.Coverage = calculateDigestCoverage(keys, digest.Items)
	return digest, nil
}

func (d SemanticDigest) Text() string {
	lines := make([]string, 0, len(d.Items))
	for _, item := range d.Items {
		if item.Status == "available" && item.Text != "" {
			lines = append(lines, "- "+item.Text)
		}
	}
	return strings.Join(lines, "\n")
}

func summarizeElements(items []model.ScoredFact) ([]SemanticDigestItem, []string) {
	parts, warnings := []string{}, []string{}
	for _, item := range items {
		name := displaydict.ZHIn("fact", item.Code)
		level := displaydict.ZHIn("power_state", item.Level)
		if isInternalCode(name) || isInternalCode(level) {
			warnings = append(warnings, "五行力量存在未收录编码："+item.Code+"/"+item.Level)
			continue
		}
		if item.Score == nil {
			parts = append(parts, name+level)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s%s（%.2f）", name, level, *item.Score))
	}
	if len(parts) == 0 {
		return nil, warnings
	}
	return []SemanticDigestItem{{Fact: "element_strength", Status: "available", Text: "五行力量：" + strings.Join(parts, "；")}}, warnings
}

func summarizeStrength(f model.DayMasterStrengthFact) ([]SemanticDigestItem, error) {
	if strings.TrimSpace(f.Level) == "" {
		return []SemanticDigestItem{{Fact: "day_master_strength", Status: "unavailable", Text: "身强弱：当前计算快照未提供有效结论。"}}, nil
	}
	level := displaydict.ZHIn("strength", f.Level)
	if isInternalCode(level) {
		return nil, fmt.Errorf("unknown required strength level %q", f.Level)
	}
	pattern := ""
	if f.AnalysisV2 != nil {
		for _, candidate := range f.AnalysisV2.Patterns {
			if candidate.Matched {
				pattern = displaydict.ZH(candidate.Type)
				break
			}
		}
	}
	if pattern == "" && f.Analysis.Pattern != "" {
		pattern = displaydict.ZH(f.Analysis.Pattern)
	}
	text := "身强弱：日主" + level
	if f.Score != nil {
		text += fmt.Sprintf("，综合评分%.2f", *f.Score)
	}
	if pattern != "" && !isInternalCode(pattern) {
		text += "；按" + pattern + "分析"
	}
	return []SemanticDigestItem{{Fact: "day_master_strength", Status: "available", Text: text + "。"}}, nil
}

func summarizeTenGods(items []model.ScoredFact) ([]SemanticDigestItem, []string) {
	type ranked struct {
		name    string
		score   float64
		missing bool
	}
	rows, warnings := []ranked{}, []string{}
	for _, item := range items {
		if strings.HasPrefix(item.Code, "ten_god.category.") || item.Code == "ten_god.structure" {
			continue
		}
		name := displaydict.ZHIn("fact", item.Code)
		if isInternalCode(name) {
			warnings = append(warnings, "十神存在未收录编码："+item.Code)
			continue
		}
		score := 0.0
		if item.Score != nil {
			score = *item.Score
		}
		rows = append(rows, ranked{name, score, item.Level == "missing"})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].score > rows[j].score })
	main, missing := []string{}, []string{}
	for _, row := range rows {
		if row.missing {
			missing = append(missing, row.name)
			continue
		}
		if len(main) < 4 {
			main = append(main, row.name)
		}
	}
	parts := []string{}
	if len(main) > 0 {
		parts = append(parts, "主要十神为"+strings.Join(main, "、"))
	}
	if len(missing) > 0 {
		parts = append(parts, "原局不显"+strings.Join(missing, "、"))
	}
	if len(parts) == 0 {
		return nil, warnings
	}
	return []SemanticDigestItem{{Fact: "ten_god_structure", Status: "available", Text: "十神结构：" + strings.Join(parts, "；") + "。"}}, warnings
}

func summarizeRelations(label, fact string, items []model.RelationFact) ([]SemanticDigestItem, []string) {
	parts, warnings := []string{}, []string{}
	seen := map[string]bool{}
	for _, item := range items {
		relation := displaydict.ZHIn("relation", item.Type)
		if isInternalCode(relation) {
			warnings = append(warnings, label+"存在未收录编码："+item.Type)
			continue
		}
		part := strings.Join(item.Symbols, "、") + relation
		if !seen[part] {
			seen[part] = true
			parts = append(parts, part)
		}
		if len(parts) >= 6 {
			break
		}
	}
	if len(parts) == 0 {
		return nil, warnings
	}
	return []SemanticDigestItem{{Fact: fact, Status: "available", Text: label + "：" + strings.Join(parts, "；") + "。"}}, warnings
}

func summarizePatterns(items []model.ScoredFact) ([]SemanticDigestItem, []string) {
	matched, warnings := []string{}, []string{}
	for _, item := range items {
		if item.Level != "matched" {
			continue
		}
		name := displaydict.ZHIn("fact", item.Code)
		if isInternalCode(name) {
			warnings = append(warnings, "格局存在未收录编码："+item.Code)
			continue
		}
		matched = append(matched, name)
	}
	if len(matched) == 0 {
		return nil, warnings
	}
	return []SemanticDigestItem{{Fact: "pattern_candidates", Status: "available", Text: "格局：当前规则命中" + strings.Join(matched, "、") + "。"}}, warnings
}

func summarizeClimate(v *model.ClimateAnalysis) []SemanticDigestItem {
	if v == nil {
		return nil
	}
	t := displaydict.ZHIn("climate", v.TemperatureLevel)
	m := displaydict.ZHIn("climate", v.MoistureLevel)
	return []SemanticDigestItem{{Fact: "climate", Status: "available", Text: "调候：" + t + "、" + m + "。"}}
}
func summarizeDisease(v *model.DiseaseAnalysis) []SemanticDigestItem {
	if v == nil || v.Primary == nil {
		return nil
	}
	elements := translatedList(v.Primary.CandidateElements)
	return []SemanticDigestItem{{Fact: "disease", Status: "available", Text: "五行偏颇：重点关注" + elements + "的平衡，仅作生活方式参考。"}}
}
func summarizeMediation(v *model.MediationAnalysis) []SemanticDigestItem {
	if v == nil {
		return nil
	}
	return []SemanticDigestItem{{Fact: "mediation", Status: "available", Text: fmt.Sprintf("通关：流通评分%.2f，识别%d组需调和关系。", v.FlowScore, len(v.Conflicts))}}
}
func summarizeUsefulGod(v *model.UsefulGodAnalysis) []SemanticDigestItem {
	if v == nil {
		return nil
	}
	if v.Primary == "" {
		return []SemanticDigestItem{{Fact: "useful_god", Status: "pending", Text: "喜用神：当前规则尚未形成稳定的首选结论。"}}
	}
	text := "喜用神：首选" + translatedValue(v.Primary)
	if v.Secondary != "" {
		text += "，次选" + translatedValue(v.Secondary)
	}
	if len(v.Taboo) > 0 {
		text += "；忌" + translatedList(v.Taboo)
	}
	return []SemanticDigestItem{{Fact: "useful_god", Status: "available", Text: text + "。"}}
}
func summarizeFavorable(v model.ScoredFact) []SemanticDigestItem {
	if len(v.Values) == 0 {
		return nil
	}
	return []SemanticDigestItem{{Fact: "favorable_elements", Status: "available", Text: "喜用候选元素：" + translatedList(v.Values) + "。"}}
}

func summarizeLuckCycles(chapter string, items []model.LuckCycle, year int) []SemanticDigestItem {
	if len(items) == 0 {
		return nil
	}
	selected := items
	if chapter != "luck_cycle" {
		index := 0
		for i, item := range items {
			if item.StartYear <= year {
				index = i
			}
		}
		end := index + 2
		if end > len(items) {
			end = len(items)
		}
		selected = items[index:end]
	}
	return []SemanticDigestItem{{Fact: "luck_cycles", Status: "available", Text: "大运：" + luckCyclesSummary(selected) + "。"}}
}
func summarizeAnnualFortunes(chapter string, items []model.AnnualFortune) []SemanticDigestItem {
	if len(items) == 0 {
		return nil
	}
	limit := 3
	if chapter == "ten_year_years" || chapter == "life_plan" {
		limit = 10
	}
	if limit > len(items) {
		limit = len(items)
	}
	return []SemanticDigestItem{{Fact: "annual_fortunes", Status: "available", Text: "流年：" + annualFortunesSummary(items[:limit]) + "。"}}
}
func referenceYear(f model.InterpretationFacts) int {
	if len(f.AnnualFortunes) > 0 {
		return f.AnnualFortunes[0].Year
	}
	return time.Now().Year()
}
func isInternalCode(value string) bool {
	if value == "" {
		return true
	}
	for _, r := range value {
		if r > 127 {
			return false
		}
	}
	return true
}
func uniqueDigestItems(items []SemanticDigestItem) []SemanticDigestItem {
	seen := map[string]bool{}
	out := make([]SemanticDigestItem, 0, len(items))
	for _, item := range items {
		key := item.Fact + "\x00" + item.Text
		if item.Text == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func calculateDigestCoverage(keys []string, items []SemanticDigestItem) SemanticDigestCoverage {
	selected := map[string]bool{}
	for _, key := range keys {
		selected[key] = true
	}
	statuses := map[string]string{"chart": "available"}
	for _, item := range items {
		if selected[item.Fact] {
			statuses[item.Fact] = item.Status
		}
	}
	coverage := SemanticDigestCoverage{SelectedFacts: len(selected), Ready: true}
	for key := range selected {
		switch statuses[key] {
		case "available":
			coverage.CoveredFacts++
		case "pending":
			coverage.PendingFacts++
			coverage.MissingFacts = append(coverage.MissingFacts, key)
			coverage.Ready = false
		default:
			coverage.UnavailableFacts++
			coverage.MissingFacts = append(coverage.MissingFacts, key)
			coverage.Ready = false
		}
	}
	sort.Strings(coverage.MissingFacts)
	if coverage.SelectedFacts > 0 {
		coverage.CoverageRate = float64(coverage.CoveredFacts) / float64(coverage.SelectedFacts)
	}
	return coverage
}

func factDisplayName(key string) string {
	names := map[string]string{
		"element_strength": "五行力量", "day_master_strength": "身强弱", "ten_god_structure": "十神结构",
		"stem_relations": "天干关系", "branch_relations": "地支关系", "pattern_candidates": "格局候选",
		"climate": "调候", "disease": "五行偏颇", "mediation": "通关", "useful_god": "喜用神",
		"favorable_elements": "喜用候选元素", "luck_cycles": "大运", "annual_fortunes": "流年",
	}
	if name := names[key]; name != "" {
		return name
	}
	return key
}
