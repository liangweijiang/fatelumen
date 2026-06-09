package bazi

import (
	"reflect"
	"testing"

	"fatelumen/backend/internal/model"
)

func TestCalculate_Case1(t *testing.T) {
	in := BirthInput{
		Gender:       1,
		CalendarType: 0,
		Year:         1990,
		Month:        8,
		Day:          15,
		Hour:         14,
		Minute:       30,
		IsLeapMonth:  false,
		Longitude:    0,
	}

	chart, err := Calculate(in)
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if chart == nil {
		t.Fatal("Calculate returned nil chart")
	}

	// 验证四柱 (按 lunar-go 实际计算结果)
	if chart.Pillars.Year.Stem != "庚" || chart.Pillars.Year.Branch != "午" {
		t.Errorf("Year pillar: expected 庚午, got %s%s", chart.Pillars.Year.Stem, chart.Pillars.Year.Branch)
	}
	if chart.Pillars.Month.Stem != "甲" || chart.Pillars.Month.Branch != "申" {
		t.Errorf("Month pillar: expected 甲申, got %s%s", chart.Pillars.Month.Stem, chart.Pillars.Month.Branch)
	}
	if chart.Pillars.Day.Stem != "壬" || chart.Pillars.Day.Branch != "子" {
		t.Errorf("Day pillar: expected 壬子, got %s%s", chart.Pillars.Day.Stem, chart.Pillars.Day.Branch)
	}
	if chart.Pillars.Hour.Stem != "丁" || chart.Pillars.Hour.Branch != "未" {
		t.Errorf("Hour pillar: expected 丁未, got %s%s", chart.Pillars.Hour.Stem, chart.Pillars.Hour.Branch)
	}

	// 验证日主
	if chart.DayMaster.Stem != "壬" {
		t.Errorf("DayMaster stem: expected 壬, got %s", chart.DayMaster.Stem)
	}
	if chart.DayMaster.Element != "水" {
		t.Errorf("DayMaster element: expected 水, got %s", chart.DayMaster.Element)
	}

	// 验证纳音
	if chart.Pillars.Year.NaYin != "路旁土" {
		t.Errorf("Year NaYin: expected 路旁土, got %s", chart.Pillars.Year.NaYin)
	}
	if chart.Pillars.Month.NaYin != "泉中水" {
		t.Errorf("Month NaYin: expected 泉中水, got %s", chart.Pillars.Month.NaYin)
	}
	if chart.Pillars.Day.NaYin != "桑柘木" {
		t.Errorf("Day NaYin: expected 桑柘木, got %s", chart.Pillars.Day.NaYin)
	}
	if chart.Pillars.Hour.NaYin != "天河水" {
		t.Errorf("Hour NaYin: expected 天河水, got %s", chart.Pillars.Hour.NaYin)
	}

	// 验证五行计数非空
	if len(chart.FiveElementsCount) != 5 {
		t.Errorf("FiveElementsCount should have 5 entries, got %d", len(chart.FiveElementsCount))
	}
	for _, e := range []string{"木", "火", "土", "金", "水"} {
		if _, ok := chart.FiveElementsCount[e]; !ok {
			t.Errorf("FiveElementsCount missing key: %s", e)
		}
	}

	// 验证身强身弱判定
	if chart.Strength.Level == "" {
		t.Error("Strength.Level should not be empty")
	}
	if len(chart.Strength.Favorable) == 0 {
		t.Error("Strength.Favorable should not be empty")
	}
	if len(chart.Strength.Unfavorable) == 0 {
		t.Error("Strength.Unfavorable should not be empty")
	}

	// 验证大运
	if len(chart.LuckCycles) == 0 {
		t.Error("LuckCycles should not be empty")
	}
	for _, lc := range chart.LuckCycles {
		if lc.GanZhi == "" || lc.StartAge == 0 || lc.StartYear == 0 {
			t.Errorf("Invalid LuckCycle: %+v", lc)
		}
	}

	// 验证元信息
	if chart.Meta.CalcLib != "lunar-go" {
		t.Errorf("CalcLib: expected lunar-go, got %s", chart.Meta.CalcLib)
	}
}

func TestCalculate_Deterministic(t *testing.T) {
	in := BirthInput{
		Gender:       1,
		CalendarType: 0,
		Year:         1990,
		Month:        8,
		Day:          15,
		Hour:         14,
		Minute:       30,
		IsLeapMonth:  false,
		Longitude:    0,
	}

	c1, err := Calculate(in)
	if err != nil {
		t.Fatalf("First Calculate returned error: %v", err)
	}

	c2, err := Calculate(in)
	if err != nil {
		t.Fatalf("Second Calculate returned error: %v", err)
	}

	// 相同输入应该得到相同结果
	if !reflect.DeepEqual(c1.Pillars.Year, c2.Pillars.Year) {
		t.Error("Year pillar mismatch between runs")
	}
	if !reflect.DeepEqual(c1.Pillars.Month, c2.Pillars.Month) {
		t.Error("Month pillar mismatch between runs")
	}
	if !reflect.DeepEqual(c1.Pillars.Day, c2.Pillars.Day) {
		t.Error("Day pillar mismatch between runs")
	}
	if !reflect.DeepEqual(c1.Pillars.Hour, c2.Pillars.Hour) {
		t.Error("Hour pillar mismatch between runs")
	}
	if !reflect.DeepEqual(c1.DayMaster, c2.DayMaster) {
		t.Error("DayMaster mismatch between runs")
	}
	if len(c1.LuckCycles) != len(c2.LuckCycles) {
		t.Errorf("LuckCycles length mismatch: %d vs %d", len(c1.LuckCycles), len(c2.LuckCycles))
	}
	if c1.Strength.Level != c2.Strength.Level {
		t.Error("Strength.Level mismatch between runs")
	}
}

func TestCalculate_FemaleLunarInput(t *testing.T) {
	in := BirthInput{
		Gender:       0, // female
		CalendarType: 1, // lunar
		Year:         2000,
		Month:        1,
		Day:          1,
		Hour:         8,
		Minute:       0,
		IsLeapMonth:  false,
	}

	chart, err := Calculate(in)
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if chart == nil {
		t.Fatal("Calculate returned nil chart")
	}
	if chart.DayMaster.Stem == "" {
		t.Error("DayMaster stem should not be empty")
	}
	if chart.Meta.Gender != "女" {
		t.Errorf("Gender should be 女, got %s", chart.Meta.Gender)
	}
}

func TestCalculate_HourUnknown(t *testing.T) {
	in := BirthInput{
		Gender:       1,
		CalendarType: 0,
		Year:         1990,
		Month:        8,
		Day:          15,
		Hour:         -1,
		Minute:       0,
		IsLeapMonth:  false,
	}

	chart, err := Calculate(in)
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if chart == nil {
		t.Fatal("Calculate returned nil chart")
	}
	if !chart.HourUnknown {
		t.Error("HourUnknown should be true when hour is -1")
	}
}

func TestChartData_ModelConversion(t *testing.T) {
	in := BirthInput{
		Gender:       1,
		CalendarType: 0,
		Year:         1990,
		Month:        8,
		Day:          15,
		Hour:         14,
		Minute:       30,
		IsLeapMonth:  false,
	}

	chartData, err := Calculate(in)
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}

	chart := model.Chart{
		ProfileID: 1,
		ChartHash: "test",
		ChartData: *chartData,
	}

	val, err := chart.ChartData.Value()
	if err != nil {
		t.Fatalf("ChartData.Value() error: %v", err)
	}

	var decoded model.ChartData
	if err := decoded.Scan(val.([]byte)); err != nil {
		t.Fatalf("ChartData.Scan() error: %v", err)
	}

	if decoded.Pillars.Year.Stem != chartData.Pillars.Year.Stem {
		t.Error("JSON round-trip mismatch for year stem")
	}
}
