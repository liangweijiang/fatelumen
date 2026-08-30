package prompts

import (
	"fmt"
	"strings"

	"fatelumen/backend/internal/bazi/displaydict"
	"fatelumen/backend/internal/model"
)

// renderCompactChapterFacts converts deterministic calculation facts into the
// short, readable input block expected by the authored Chinese chapter prompt.
// Calculation traces remain available separately and are not sent repeatedly.
func renderCompactChapterFacts(facts model.InterpretationFacts, keys []string) string {
	lines := []string{
		fmt.Sprintf("- 性别：%s", chartGender(facts)),
		fmt.Sprintf("- 八字：%s", fourPillars(facts.Chart.Data.Pillars)),
		fmt.Sprintf("- 日主：%s（%s、%s）", facts.Chart.Data.DayMaster.Stem, facts.Chart.Data.DayMaster.Element, facts.Chart.Data.DayMaster.YinYang),
	}
	for _, key := range keys {
		switch key {
		case "chart":
			// The common chart fields above are enough; do not repeat the chart snapshot.
		case "element_strength":
			lines = append(lines, "- 五行力量："+scoredFactsSummary(facts.ElementStrength))
		case "day_master_strength":
			lines = append(lines, fmt.Sprintf("- 身强弱：%s；评分%s；依据%s", readableCode(displaydict.ZHIn("strength", facts.DayMasterStrength.Level), "已判定"), optionalScore(facts.DayMasterStrength.Score), translatedList(facts.DayMasterStrength.Values)))
		case "ten_god_structure":
			lines = append(lines, "- 十神结构："+scoredFactsSummary(facts.TenGodStructure))
		case "stem_relations":
			lines = append(lines, "- 天干关系："+relationsSummary(facts.StemRelations))
		case "branch_relations":
			lines = append(lines, "- 地支关系："+relationsSummary(facts.BranchRelations))
		case "pattern_candidates":
			lines = append(lines, "- 格局候选："+scoredFactsSummary(facts.PatternCandidates))
		case "climate":
			if facts.Climate != nil {
				lines = append(lines, fmt.Sprintf("- 调候：温度%s（%.2f），湿度%s（%.2f），候选%s", displaydict.ZHIn("climate", facts.Climate.TemperatureLevel), facts.Climate.Temperature, displaydict.ZHIn("climate", facts.Climate.MoistureLevel), facts.Climate.Moisture, translatedList(facts.Climate.Candidates)))
			}
		case "disease":
			if facts.Disease != nil && facts.Disease.Primary != nil {
				lines = append(lines, fmt.Sprintf("- 五行偏颇提示：%s（%s）", readableCode(facts.Disease.Primary.Code, "已识别偏颇"), readableCode(facts.Disease.Primary.Level, "已评估")))
			}
		case "mediation":
			if facts.Mediation != nil {
				lines = append(lines, fmt.Sprintf("- 通关状态：流通评分 %.2f，共 %d 项冲突", facts.Mediation.FlowScore, len(facts.Mediation.Conflicts)))
			}
		case "useful_god":
			if facts.UsefulGod != nil {
				lines = append(lines, fmt.Sprintf("- 喜用神：首选%s，次选%s；喜%s；忌%s；可信度 %.0f%%", translatedValue(facts.UsefulGod.Primary), translatedValue(facts.UsefulGod.Secondary), translatedList(facts.UsefulGod.Favorable), translatedList(facts.UsefulGod.Taboo), facts.UsefulGod.Confidence*100))
			}
		case "favorable_elements":
			lines = append(lines, "- 喜忌候选："+translatedList(facts.FavorableElements.Values))
		case "luck_cycles":
			lines = append(lines, "- 大运："+luckCyclesSummary(facts.LuckCycles))
		case "annual_fortunes":
			lines = append(lines, "- 未来流年："+annualFortunesSummary(facts.AnnualFortunes))
		}
	}
	return strings.Join(uniqueLines(lines), "\n")
}

func composeAuthoredPrompt(source, compactFacts string) string {
	source = runtimeChapterPrompt(source)
	inputAt := strings.Index(source, "输入格式：")
	outputAt := strings.Index(source, "输出要求")
	if inputAt < 0 || outputAt <= inputAt {
		return source + "\n\n输入资料（系统已计算，直接引用）：\n" + compactFacts
	}
	return strings.TrimSpace(source[:inputAt]) + "\n\n输入资料（系统已计算，直接引用）：\n" + compactFacts + "\n\n" + strings.TrimSpace(source[outputAt:])
}

func fourPillars(p model.Pillars) string {
	return strings.Join([]string{p.Year.Stem + p.Year.Branch, p.Month.Stem + p.Month.Branch, p.Day.Stem + p.Day.Branch, p.Hour.Stem + p.Hour.Branch}, " ")
}

func chartGender(facts model.InterpretationFacts) string {
	if facts.Chart.Data.Meta.Gender != "" {
		return facts.Chart.Data.Meta.Gender
	}
	if facts.Input.Gender == 1 {
		return "乾·男"
	}
	return "坤·女"
}

func scoredFactsSummary(items []model.ScoredFact) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		part := readableCode(displaydict.ZHIn("fact", item.Code), "已计算项")
		if item.Level != "" {
			part += "=" + readableCode(displaydict.ZHIn("power_state", item.Level), "已评估")
		}
		if item.Score != nil {
			part += fmt.Sprintf("(%.2f)", *item.Score)
		}
		if len(item.Values) > 0 {
			part += "[" + translatedList(item.Values) + "]"
		}
		parts = append(parts, part)
	}
	return joinOrDash(parts)
}

func relationsSummary(items []model.RelationFact) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, strings.Join(item.Symbols, "、")+"·"+readableCode(displaydict.ZHIn("relation", item.Type), "既定关系"))
	}
	return joinOrDash(parts)
}

func luckCyclesSummary(items []model.LuckCycle) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%d岁起（%d年）%s", item.StartAge, item.StartYear, item.GanZhi))
	}
	return joinOrDash(parts)
}

func annualFortunesSummary(items []model.AnnualFortune) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%d年%s（大运%s）", item.Year, item.GanZhi, dash(item.LuckCycleGanZhi)))
	}
	return joinOrDash(parts)
}

func optionalScore(score *float64) string {
	if score == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f", *score)
}

func dash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "—"
	}
	return strings.Join(values, "；")
}

func translatedList(values []string) string {
	translated := make([]string, 0, len(values))
	for _, value := range values {
		translated = append(translated, readableCode(displaydict.ZH(value), "已计算"))
	}
	return joinOrDash(translated)
}

func translatedValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return readableCode(displaydict.ZH(value), "已计算")
}

func readableCode(value, fallback string) string {
	for _, char := range value {
		if char > 127 {
			return value
		}
	}
	return fallback
}

func uniqueLines(lines []string) []string {
	seen := make(map[string]struct{}, len(lines))
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if _, exists := seen[line]; exists {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	return out
}
