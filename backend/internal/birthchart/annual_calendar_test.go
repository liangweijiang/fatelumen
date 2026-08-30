package birthchart

import (
	"context"
	"testing"
	"time"

	"fatelumen/backend/internal/bazi/annualcalendar"
	"fatelumen/backend/internal/model"
)

type annualCalendarStub struct {
	rows []model.AnnualCalendarYear
	err  error
}

func (s annualCalendarStub) Range(context.Context, int, int) ([]model.AnnualCalendarYear, error) {
	return s.rows, s.err
}

func TestLunarGoCalculatorUsesPublicCalendarAndPinsRange(t *testing.T) {
	start := time.Now().Year()
	rows, err := annualcalendar.Generate(start, 10)
	if err != nil {
		t.Fatal(err)
	}
	chart, err := (LunarGoCalculator{AnnualCalendar: annualCalendarStub{rows: rows}}).Calculate(context.Background(), CalculatorInput{Gender: 1, TrueSolarTime: time.Date(1990, 1, 1, 12, 0, 0, 0, time.UTC), DayBoundaryRule: Midnight00})
	if err != nil {
		t.Fatal(err)
	}
	if len(chart.AnnualFortunes) != 10 || chart.AnnualFortunes[0].Year != start || chart.AnnualFortunes[9].Year != start+9 {
		t.Fatalf("unexpected report range: %+v", chart.AnnualFortunes)
	}
	if chart.AnnualFortuneRange == nil || chart.AnnualFortuneRange.Source != annualcalendar.ReportRangeSource || chart.AnnualFortuneRange.Count != 10 {
		t.Fatalf("range trace was not pinned: %+v", chart.AnnualFortuneRange)
	}
}

func TestLunarGoCalculatorRejectsIncompletePublicCalendar(t *testing.T) {
	rows, _ := annualcalendar.Generate(time.Now().Year(), 9)
	_, err := (LunarGoCalculator{AnnualCalendar: annualCalendarStub{rows: rows}}).Calculate(context.Background(), CalculatorInput{Gender: 1, TrueSolarTime: time.Now(), DayBoundaryRule: Midnight00})
	if err == nil {
		t.Fatal("expected incomplete calendar to be rejected")
	}
}
