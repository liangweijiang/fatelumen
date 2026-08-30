package bazi

import (
	"container/list"
	"fmt"
	"math"
	"time"

	"fatelumen/backend/internal/bazi/annualcalendar"
	elementcalc "fatelumen/backend/internal/bazi/elementpower"
	strengthcalc "fatelumen/backend/internal/bazi/strength"
	strengthv2calc "fatelumen/backend/internal/bazi/strengthv2"
	tengodcalc "fatelumen/backend/internal/bazi/tengod"
	"fatelumen/backend/internal/model"

	"github.com/6tail/lunar-go/LunarUtil"
	"github.com/6tail/lunar-go/calendar"
)

const CalcVersion = "lunar-go v1.4.6"

type BirthInput struct {
	Gender          int8 // 0=female 1=male
	CalendarType    int8 // 0=solar(Gregorian) 1=lunar
	Year            int
	Month           int
	Day             int
	Hour            int // 0-23, -1 for unknown
	Minute          int // 0-59
	IsLeapMonth     bool
	Longitude       float64 // Deprecated: true solar time is calculated by birthchart.Engine.
	NormalizedSolar bool
	DayBoundaryRule string
	AnnualCalendar  []model.AnnualCalendarYear
}

func Calculate(in BirthInput) (*model.ChartData, error) {
	var solar *calendar.Solar
	var lunar *calendar.Lunar

	if in.NormalizedSolar || in.CalendarType == 0 {
		h := in.Hour
		if h < 0 {
			h = 0
		}
		solar = calendar.NewSolar(in.Year, in.Month, in.Day, h, in.Minute, 0)
		lunar = solar.GetLunar()
	} else {
		lunar = calendar.NewLunarFromYmd(in.Year, in.Month, in.Day)
		solar = lunar.GetSolar()
		h := in.Hour
		if h < 0 {
			h = 0
		}
		solar = calendar.NewSolar(solar.GetYear(), solar.GetMonth(), solar.GetDay(), h, in.Minute, 0)
		lunar = solar.GetLunar()
	}

	eightChar := calendar.NewEightChar(lunar)
	if in.DayBoundaryRule == "LATE_ZI_23" {
		eightChar.SetSect(1)
	} else {
		eightChar.SetSect(2)
	}

	hourUnknown := in.Hour < 0

	dayGan := eightChar.GetDayGan()
	dayWuXing := LunarUtil.WU_XING_GAN[dayGan]
	dayYinYang := "阳"
	if eightChar.GetDayGanIndex()%2 == 1 {
		dayYinYang = "阴"
	}

	yearStem := eightChar.GetYearGan()
	yearBranch := eightChar.GetYearZhi()
	monthStem := eightChar.GetMonthGan()
	monthBranch := eightChar.GetMonthZhi()
	dayStem := dayGan
	dayBranch := eightChar.GetDayZhi()
	timeStem := eightChar.GetTimeGan()
	timeBranch := eightChar.GetTimeZhi()

	yearHidden := eightChar.GetYearHideGan()
	monthHidden := eightChar.GetMonthHideGan()
	dayHidden := eightChar.GetDayHideGan()
	timeHidden := eightChar.GetTimeHideGan()

	yearTenGodHidden := listToStrings(eightChar.GetYearShiShenZhi())
	monthTenGodHidden := listToStrings(eightChar.GetMonthShiShenZhi())
	dayTenGodHidden := listToStrings(eightChar.GetDayShiShenZhi())
	timeTenGodHidden := listToStrings(eightChar.GetTimeShiShenZhi())

	chartData := &model.ChartData{
		Pillars: model.Pillars{
			Year: model.Pillar{
				Stem:          yearStem,
				Branch:        yearBranch,
				StemElement:   LunarUtil.WU_XING_GAN[yearStem],
				BranchElement: LunarUtil.WU_XING_ZHI[yearBranch],
				TenGodStem:    eightChar.GetYearShiShenGan(),
				TenGodHidden:  padStrings(yearTenGodHidden, len(yearHidden)),
				HiddenStems:   yearHidden,
				NaYin:         eightChar.GetYearNaYin(),
			},
			Month: model.Pillar{
				Stem:          monthStem,
				Branch:        monthBranch,
				StemElement:   LunarUtil.WU_XING_GAN[monthStem],
				BranchElement: LunarUtil.WU_XING_ZHI[monthBranch],
				TenGodStem:    eightChar.GetMonthShiShenGan(),
				TenGodHidden:  padStrings(monthTenGodHidden, len(monthHidden)),
				HiddenStems:   monthHidden,
				NaYin:         eightChar.GetMonthNaYin(),
			},
			Day: model.Pillar{
				Stem:          dayStem,
				Branch:        dayBranch,
				StemElement:   dayWuXing,
				BranchElement: LunarUtil.WU_XING_ZHI[dayBranch],
				TenGodStem:    "日主",
				TenGodHidden:  padStrings(dayTenGodHidden, len(dayHidden)),
				HiddenStems:   dayHidden,
				NaYin:         eightChar.GetDayNaYin(),
			},
			Hour: model.Pillar{
				Stem:          timeStem,
				Branch:        timeBranch,
				StemElement:   LunarUtil.WU_XING_GAN[timeStem],
				BranchElement: LunarUtil.WU_XING_ZHI[timeBranch],
				TenGodStem:    eightChar.GetTimeShiShenGan(),
				TenGodHidden:  padStrings(timeTenGodHidden, len(timeHidden)),
				HiddenStems:   timeHidden,
				NaYin:         eightChar.GetTimeNaYin(),
			},
		},
		DayMaster: model.DayMaster{
			Stem:    dayGan,
			Element: dayWuXing,
			YinYang: dayYinYang,
		},
		HourUnknown: hourUnknown,
		Meta: model.ChartMeta{
			SolarDate:   fmt.Sprintf("%04d-%02d-%02d %02d:%02d", solar.GetYear(), solar.GetMonth(), solar.GetDay(), solar.GetHour(), solar.GetMinute()),
			LunarDate:   fmt.Sprintf("%s年%s月%s日", lunar.GetYearInChinese(), lunar.GetMonthInChinese(), lunar.GetDayInChinese()),
			Zodiac:      lunar.GetYearShengXiao(),
			Gender:      genderLabel(in.Gender),
			CalcLib:     "lunar-go",
			CalcVersion: CalcVersion,
		},
	}

	// 五行计数
	fiveCount := computeFiveElementsCount(chartData)
	chartData.FiveElementsCount = fiveCount

	// 五行力量 V2 是独立、版本化的确定性计算。当前 V2-A 先输出原始、
	// 季节与基础生克力量；结构合化将在 V2-B 接入后移除 pending 警告。
	elementResult, err := elementcalc.Evaluate(elementcalc.Input{
		Year: elementcalc.Pillar{Stem: yearStem, Branch: yearBranch}, Month: elementcalc.Pillar{Stem: monthStem, Branch: monthBranch},
		Day: elementcalc.Pillar{Stem: dayStem, Branch: dayBranch}, Hour: elementcalc.Pillar{Stem: timeStem, Branch: timeBranch},
		SeasonProgress: seasonProgress(lunar),
		LongLifeStages: map[string]string{"year": eightChar.GetYearDiShi(), "month": eightChar.GetMonthDiShi(), "day": eightChar.GetDayDiShi(), "hour": eightChar.GetTimeDiShi()},
	}, elementcalc.RuleV2())
	if err != nil {
		return nil, fmt.Errorf("evaluate element power: %w", err)
	}
	chartData.ElementPower = toModelElementPower(elementResult)

	// 身强身弱是独立的版本化确定性算法，不参与喜用神推导。
	strengthResult, err := strengthcalc.Evaluate(strengthcalc.Input{
		Year:  strengthcalc.Pillar{Stem: yearStem, Branch: yearBranch},
		Month: strengthcalc.Pillar{Stem: monthStem, Branch: monthBranch},
		Day:   strengthcalc.Pillar{Stem: dayStem, Branch: dayBranch},
		Hour:  strengthcalc.Pillar{Stem: timeStem, Branch: timeBranch},
	}, strengthcalc.RuleV1())
	if err != nil {
		return nil, fmt.Errorf("evaluate day-master strength: %w", err)
	}
	chartData.Strength = toModelStrength(strengthResult)
	tenGodResult, err := tengodcalc.Evaluate(toTenGodInput(dayStem, dayWuXing, strengthResult))
	if err != nil {
		return nil, fmt.Errorf("evaluate ten-god structure: %w", err)
	}
	chartData.TenGodAnalysis = toModelTenGod(tenGodResult)
	tenGodEffective, err := tengodcalc.EvaluateEffective(tengodcalc.EffectiveInput{Raw: tenGodResult, ElementRatios: elementRatioMap(elementResult.EffectiveRatio)})
	if err != nil {
		return nil, fmt.Errorf("evaluate effective ten-god structure: %w", err)
	}
	chartData.TenGodEffective = toModelTenGod(tenGodEffective)
	strengthV2, err := strengthv2calc.Evaluate(toStrengthV2Input(dayWuXing, monthBranch, hourUnknown, elementResult, tenGodEffective))
	if err != nil {
		return nil, fmt.Errorf("evaluate day-master strength v2: %w", err)
	}
	chartData.StrengthV2 = toModelStrengthV2(strengthV2)
	if err := evaluatePreanalysis(chartData); err != nil {
		return nil, fmt.Errorf("evaluate deterministic preanalysis: %w", err)
	}

	// 大运
	gender := int(in.Gender)
	if gender == 0 {
		gender = 0
	} else {
		gender = 1
	}
	yun := eightChar.GetYun(gender)
	daYunList := yun.GetDaYun()
	luckCycles := make([]model.LuckCycle, 0)
	for _, dy := range daYunList {
		gz := dy.GetGanZhi()
		if gz == "" {
			continue
		}
		element := LunarUtil.WU_XING_GAN[string([]rune(gz)[0])]
		luckCycles = append(luckCycles, model.LuckCycle{
			GanZhi:    gz,
			StartAge:  dy.GetStartAge(),
			StartYear: dy.GetStartYear(),
			Element:   element,
		})
	}
	chartData.LuckCycles = luckCycles

	// 本年流年
	currentYear := time.Now().Year()
	calendarYears := in.AnnualCalendar
	if len(calendarYears) == 0 {
		calendarYears, _ = annualcalendar.Generate(currentYear, annualcalendar.ReportRangeYears)
	}
	annualFortunes := buildAnnualFortunes(daYunList, calendarYears)
	chartData.AnnualFortunes = annualFortunes
	if len(calendarYears) > 0 {
		chartData.AnnualFortuneRange = &model.AnnualFortuneRange{StartYear: calendarYears[0].Year, EndYear: calendarYears[len(calendarYears)-1].Year, Count: len(calendarYears), Source: annualcalendar.ReportRangeSource, DataVersion: calendarYears[0].DataVersion, SelectionRule: "calculation_year_to_plus_9"}
	}
	for _, dy := range daYunList {
		liuNianList := dy.GetLiuNian()
		for _, ln := range liuNianList {
			if ln.GetYear() == currentYear {
				gz := ln.GetGanZhi()
				chartData.CurrentYearFortune = &model.CurrentYearFortune{
					Year:    currentYear,
					Stem:    string([]rune(gz)[0:1]),
					Branch:  string([]rune(gz)[1:2]),
					Element: LunarUtil.WU_XING_GAN[string([]rune(gz)[0:1])],
				}
				break
			}
		}
		if chartData.CurrentYearFortune != nil {
			break
		}
	}

	return chartData, nil
}

func buildAnnualFortunes(daYunList []*calendar.DaYun, calendarYears []model.AnnualCalendarYear) []model.AnnualFortune {
	if len(calendarYears) == 0 {
		return nil
	}
	startYear, endYear := calendarYears[0].Year, calendarYears[len(calendarYears)-1].Year
	personal := make(map[int]model.AnnualFortune, len(calendarYears))

	for _, dy := range daYunList {
		if dy.GetEndYear() < startYear || dy.GetStartYear() > endYear {
			continue
		}
		for _, ln := range dy.GetLiuNian() {
			year := ln.GetYear()
			if year < startYear || year > endYear {
				continue
			}
			gz := ln.GetGanZhi()
			runes := []rune(gz)
			if len(runes) < 2 {
				continue
			}
			stem := string(runes[0])
			branch := string(runes[1])
			personal[year] = model.AnnualFortune{
				Year:               year,
				Age:                ln.GetAge(),
				GanZhi:             gz,
				Stem:               stem,
				Branch:             branch,
				Element:            LunarUtil.WU_XING_GAN[stem],
				LuckCycleGanZhi:    dy.GetGanZhi(),
				LuckCycleStartAge:  dy.GetStartAge(),
				LuckCycleStartYear: dy.GetStartYear(),
			}
		}
	}

	annualFortunes := make([]model.AnnualFortune, 0, len(calendarYears))
	for _, base := range calendarYears {
		item := personal[base.Year]
		item.Year, item.GanZhi, item.Stem, item.Branch, item.Element = base.Year, base.GanZhi, base.Stem, base.Branch, base.StemElement
		annualFortunes = append(annualFortunes, item)
	}
	return annualFortunes
}

func computeFiveElementsCount(cd *model.ChartData) map[string]int {
	counts := map[string]int{"木": 0, "火": 0, "土": 0, "金": 0, "水": 0}
	pillars := []model.Pillar{cd.Pillars.Year, cd.Pillars.Month, cd.Pillars.Day, cd.Pillars.Hour}
	for _, p := range pillars {
		counts[p.StemElement]++
		counts[p.BranchElement]++
		for _, hs := range p.HiddenStems {
			if e, ok := LunarUtil.WU_XING_GAN[hs]; ok {
				counts[e]++
			}
		}
	}
	return counts
}

func toModelStrength(in strengthcalc.Result) model.Strength {
	contributions := make([]model.StrengthContribution, 0, len(in.Contributions))
	for _, c := range in.Contributions {
		contributions = append(contributions, model.StrengthContribution{Code: c.Code, Source: c.Source, Position: c.Position, Symbol: c.Symbol, Element: c.Element, TenGod: c.TenGod, Category: c.Category, Score: c.Score, Adjustment: c.Adjustment})
	}
	relations := make([]model.StrengthRelation, 0, len(in.Relations))
	for _, r := range in.Relations {
		relations = append(relations, model.StrengthRelation{Code: r.Code, Type: r.Type, Positions: r.Positions, Symbols: r.Symbols, Element: r.Element, Score: r.Score, Transformed: r.Transformed, Reason: r.Reason})
	}
	return model.Strength{Level: in.Level, Score: int(math.Round(in.SupportRatio * 100)), Analysis: &model.StrengthAnalysis{
		RuleVersion: in.RuleVersion, DayElement: in.DayElement, MonthScore: in.MonthScore, SupportScore: in.SupportScore,
		RestraintScore: in.RestraintScore, SupportRatio: in.SupportRatio, RootLevel: in.RootLevel, Pattern: in.Pattern,
		PatternSubtype: in.PatternSubtype, FalseFollowing: in.FalseFollowing, Contributions: contributions, Relations: relations, Warnings: in.Warnings,
	}}
}

func toTenGodInput(dayStem, dayElement string, in strengthcalc.Result) tengodcalc.Input {
	contributions := make([]tengodcalc.Contribution, 0, len(in.Contributions))
	for _, c := range in.Contributions {
		contributions = append(contributions, tengodcalc.Contribution{Code: c.Code, Source: c.Source, Position: c.Position, Symbol: c.Symbol, Element: c.Element, TenGod: c.TenGod, Category: c.Category, Score: c.Score})
	}
	relations := make([]tengodcalc.Relation, 0, len(in.Relations))
	for _, r := range in.Relations {
		relations = append(relations, tengodcalc.Relation{Code: r.Code, Type: r.Type, Element: r.Element, Reason: r.Reason, Positions: append([]string{}, r.Positions...), Symbols: append([]string{}, r.Symbols...), Score: r.Score, Transformed: r.Transformed})
	}
	return tengodcalc.Input{DayStem: dayStem, DayElement: dayElement, Contributions: contributions, Relations: relations}
}

func toModelTenGod(in tengodcalc.Result) *model.TenGodAnalysis {
	gods := make([]model.TenGodScore, 0, len(in.Gods))
	for _, x := range in.Gods {
		gods = append(gods, model.TenGodScore{Code: x.Code, Name: x.Name, Category: x.Category, RawScore: x.RawScore, EffectiveScore: x.EffectiveScore, Ratio: x.Ratio, Rank: x.Rank, Visible: x.Visible, Rooted: x.Rooted})
	}
	categories := make([]model.TenGodCategoryScore, 0, len(in.Categories))
	for _, x := range in.Categories {
		categories = append(categories, model.TenGodCategoryScore{Category: x.Category, RawScore: x.RawScore, RelationAdjustment: x.RelationAdjustment, EffectiveScore: x.EffectiveScore, Ratio: x.Ratio, Rank: x.Rank})
	}
	evidence := make([]model.TenGodEvidence, 0, len(in.Evidence))
	for _, x := range in.Evidence {
		evidence = append(evidence, model.TenGodEvidence{Code: x.Code, Source: x.Source, Position: x.Position, Symbols: append([]string{}, x.Symbols...), TenGod: x.TenGod, Category: x.Category, RawScore: x.RawScore, Adjustment: x.Adjustment, Reason: x.Reason})
	}
	return &model.TenGodAnalysis{RuleVersion: in.RuleVersion, DayStem: in.DayStem, DayElement: in.DayElement, TotalScore: in.TotalScore, DominantGods: append([]string{}, in.DominantGods...), SecondaryGods: append([]string{}, in.SecondaryGods...), MissingGods: append([]string{}, in.MissingGods...), VisibleGods: append([]string{}, in.VisibleGods...), RootedGods: append([]string{}, in.RootedGods...), Concentration: in.Concentration, Gods: gods, Categories: categories, Evidence: evidence, Warnings: append([]string{}, in.Warnings...)}
}

func toModelElementPower(in elementcalc.Result) *model.ElementPowerAnalysis {
	contributions := make([]model.ElementPowerContribution, 0, len(in.Contributions))
	for _, x := range in.Contributions {
		contributions = append(contributions, model.ElementPowerContribution{Code: x.Code, Position: x.Position, Symbol: x.Symbol, Element: x.Element, HiddenLevel: x.HiddenLevel, BaseWeight: x.BaseWeight, HiddenRatio: x.HiddenRatio, RawPower: x.RawPower, SeasonCoefficient: x.SeasonCoefficient, VisibilityMultiplier: x.VisibilityMultiplier, SeasonalPower: x.SeasonalPower})
	}
	roots := make([]model.ElementRootEvidence, 0, len(in.Roots))
	for _, x := range in.Roots {
		roots = append(roots, model.ElementRootEvidence{Position: x.Position, Branch: x.Branch, Stem: x.Stem, Level: x.Level, HiddenRatio: x.HiddenRatio, Quality: x.Quality, Power: x.Power})
	}
	interactions := make([]model.ElementInteractionEvidence, 0, len(in.Interactions))
	for _, x := range in.Interactions {
		interactions = append(interactions, model.ElementInteractionEvidence{Iteration: x.Iteration, Type: x.Type, SourceElement: x.SourceElement, TargetElement: x.TargetElement, SourceBefore: x.SourceBefore, TargetBefore: x.TargetBefore, Amount: x.Amount, Efficiency: x.Efficiency, ContactFactor: x.ContactFactor, RatioFactor: x.RatioFactor})
	}
	structures := make([]model.ElementStructureEvidence, 0, len(in.Structures))
	for _, x := range in.Structures {
		structures = append(structures, model.ElementStructureEvidence{Code: x.Code, Type: x.Type, TargetElement: x.TargetElement, State: x.State, Reason: x.Reason, Positions: append([]string{}, x.Positions...), Symbols: append([]string{}, x.Symbols...), Confidence: x.Confidence, TransferRate: x.TransferRate, Before: toModelElementVector(x.Before), After: toModelElementVector(x.After)})
	}
	return &model.ElementPowerAnalysis{RuleVersion: in.RuleVersion, Season: model.ElementSeasonAnalysis{Branch: in.Season.Branch, NextBranch: in.Season.NextBranch, Progress: in.Season.Progress, Coefficients: toModelElementVector(in.Season.Coefficients)}, RawPower: toModelElementVector(in.RawPower), SeasonalPower: toModelElementVector(in.SeasonalPower), EffectivePower: toModelElementVector(in.EffectivePower), EffectiveRatio: toModelElementVector(in.EffectiveRatio), Contributions: contributions, Roots: roots, RootPower: in.RootPower, Interactions: interactions, Structures: structures, Warnings: append([]string{}, in.Warnings...)}
}

func toModelElementVector(v elementcalc.Vector) model.ElementPowerVector {
	return model.ElementPowerVector{Wood: v.Wood, Fire: v.Fire, Earth: v.Earth, Metal: v.Metal, Water: v.Water}
}

func elementRatioMap(v elementcalc.Vector) map[string]float64 {
	return map[string]float64{"木": v.Wood, "火": v.Fire, "土": v.Earth, "金": v.Metal, "水": v.Water}
}

func toStrengthV2Input(dayElement, monthBranch string, hourUnknown bool, power elementcalc.Result, gods tengodcalc.Result) strengthv2calc.Input {
	c := strengthv2calc.CategoryPower{}
	for _, x := range gods.Categories {
		switch x.Category {
		case "peer":
			c.Peer = x.EffectiveScore
		case "resource":
			c.Resource = x.EffectiveScore
		case "output":
			c.Output = x.EffectiveScore
		case "wealth":
			c.Wealth = x.EffectiveScore
		case "officer":
			c.Officer = x.EffectiveScore
		}
	}
	structures := make([]strengthv2calc.Structure, 0, len(power.Structures))
	for _, x := range power.Structures {
		structures = append(structures, strengthv2calc.Structure{Code: x.Code, Type: x.Type, TargetElement: x.TargetElement, State: x.State, Confidence: x.Confidence})
	}
	coeff := elementRatioMap(power.Season.Coefficients)[dayElement]
	return strengthv2calc.Input{DayElement: dayElement, MonthBranch: monthBranch, SeasonCoefficient: coeff, SeasonProgress: power.Season.Progress, RootPower: power.RootPower, Categories: c, ElementRatios: elementRatioMap(power.EffectiveRatio), Structures: structures, HourUnknown: hourUnknown}
}

func toModelStrengthV2(in strengthv2calc.Result) *model.StrengthV2Analysis {
	patterns := make([]model.StrengthPatternCandidate, 0, len(in.Patterns))
	for _, x := range in.Patterns {
		patterns = append(patterns, model.StrengthPatternCandidate{Type: x.Type, Subtype: x.Subtype, Alternative: x.Alternative, Matched: x.Matched, Confidence: x.Confidence, Evidence: append([]string{}, x.Evidence...), RejectedBy: append([]string{}, x.RejectedBy...)})
	}
	trace := make([]model.StrengthV2Trace, 0, len(in.Trace))
	for _, x := range in.Trace {
		values := map[string]float64{}
		for k, v := range x.Values {
			values[k] = v
		}
		trace = append(trace, model.StrengthV2Trace{Rule: x.Rule, Result: x.Result, Reason: x.Reason, Score: x.Score, Values: values})
	}
	return &model.StrengthV2Analysis{RuleVersion: in.RuleVersion, Level: in.Level, BaseScore: in.BaseScore, Score: in.Score, Confidence: in.Confidence, Support: in.Support, Pressure: in.Pressure, SupportRatio: in.SupportRatio, DeLing: in.DeLing, DeDi: in.DeDi, DeShi: in.DeShi, RootPower: in.RootPower, Patterns: patterns, Trace: trace, Warnings: append([]string{}, in.Warnings...)}
}

func seasonProgress(lunar *calendar.Lunar) float64 {
	prev, next := lunar.GetPrevJie(), lunar.GetNextJie()
	if prev == nil || next == nil || prev.GetSolar() == nil || next.GetSolar() == nil {
		return 0
	}
	toTime := func(s *calendar.Solar) time.Time {
		return time.Date(s.GetYear(), time.Month(s.GetMonth()), s.GetDay(), s.GetHour(), s.GetMinute(), s.GetSecond(), 0, time.UTC)
	}
	start, end := toTime(prev.GetSolar()), toTime(next.GetSolar())
	current := time.Date(lunar.GetSolar().GetYear(), time.Month(lunar.GetSolar().GetMonth()), lunar.GetSolar().GetDay(), lunar.GetSolar().GetHour(), lunar.GetSolar().GetMinute(), lunar.GetSolar().GetSecond(), 0, time.UTC)
	if !end.After(start) {
		return 0
	}
	p := current.Sub(start).Seconds() / end.Sub(start).Seconds()
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

func genderLabel(g int8) string {
	if g == 1 {
		return "男"
	}
	return "女"
}

func listToStrings(l *list.List) []string {
	if l == nil {
		return nil
	}
	var result []string
	for e := l.Front(); e != nil; e = e.Next() {
		if s, ok := e.Value.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func padStrings(s []string, targetLen int) []string {
	if len(s) >= targetLen {
		return s[:targetLen]
	}
	result := make([]string, targetLen)
	copy(result, s)
	return result
}
