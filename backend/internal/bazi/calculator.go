package bazi

import (
	"container/list"
	"fmt"
	"sort"
	"time"

	"fatelumen/backend/internal/model"

	"github.com/6tail/lunar-go/LunarUtil"
	"github.com/6tail/lunar-go/calendar"
)

const CalcVersion = "lunar-go v1.4.6"

type BirthInput struct {
	Gender       int8   // 0=female 1=male
	CalendarType int8   // 0=solar(Gregorian) 1=lunar
	Year         int
	Month        int
	Day          int
	Hour         int    // 0-23, -1 for unknown
	Minute       int    // 0-59
	IsLeapMonth  bool
	Longitude    float64 // 经度(东经为正)
}

func Calculate(in BirthInput) (*model.ChartData, error) {
	var solar *calendar.Solar
	var lunar *calendar.Lunar

	if in.CalendarType == 0 {
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

	// 身强身弱
	strength := computeStrength(dayWuXing, fiveCount)
	chartData.Strength = strength

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

func computeStrength(dayElement string, fiveCount map[string]int) model.Strength {
	gen := map[string]string{
		"木": "水",
		"火": "木",
		"土": "火",
		"金": "土",
		"水": "金",
	}
	sheng := gen[dayElement]

	sameCount := fiveCount[dayElement]
	producingCount := fiveCount[sheng]
	supportive := sameCount + producingCount

	level := "balanced"
	if supportive >= 5 {
		level = "strong"
	} else if supportive <= 2 {
		level = "weak"
	}

	allElements := []string{"木", "火", "土", "金", "水"}
	var favorable, unfavorable []string
	if level == "strong" || level == "balanced" {
		for _, e := range allElements {
			if e != dayElement && e != sheng {
				favorable = append(favorable, e)
			}
		}
		unfavorable = []string{dayElement, sheng}
	} else {
		favorable = []string{dayElement, sheng}
		for _, e := range allElements {
			if e != dayElement && e != sheng {
				unfavorable = append(unfavorable, e)
			}
		}
	}

	sort.Strings(favorable)
	sort.Strings(unfavorable)

	return model.Strength{
		Level:       level,
		Score:       supportive,
		Favorable:   favorable,
		Unfavorable: unfavorable,
	}
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
