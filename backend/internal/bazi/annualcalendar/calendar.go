package annualcalendar

import (
	"fmt"
	"time"

	"fatelumen/backend/internal/model"
)

const (
	DataVersion       = "sexagenary-calendar-v1"
	PublicRangeYears  = 60
	ReportRangeYears  = 10
	ReportRangeSource = "annual_calendar_years"
)

var (
	stems          = []string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"}
	branches       = []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	stemElements   = []string{"木", "木", "火", "火", "土", "土", "金", "金", "水", "水"}
	branchElements = []string{"水", "土", "木", "木", "土", "火", "火", "土", "金", "金", "土", "水"}
	zodiacs        = []string{"鼠", "牛", "虎", "兔", "龙", "蛇", "马", "羊", "猴", "鸡", "狗", "猪"}
)

// Generate returns immutable, person-independent year facts. 4 CE is a 甲子 year.
func Generate(startYear, count int) ([]model.AnnualCalendarYear, error) {
	if startYear < 1 || count < 1 || count > 600 {
		return nil, fmt.Errorf("invalid annual calendar range: start=%d count=%d", startYear, count)
	}
	rows := make([]model.AnnualCalendarYear, 0, count)
	for year := startYear; year < startYear+count; year++ {
		cycleZero := positiveMod(year-4, 60)
		stemIndex, branchIndex := cycleZero%10, cycleZero%12
		rows = append(rows, model.AnnualCalendarYear{
			Year: year, GanZhi: stems[stemIndex] + branches[branchIndex], Stem: stems[stemIndex], Branch: branches[branchIndex],
			StemElement: stemElements[stemIndex], BranchElement: branchElements[branchIndex],
			StemYinYang: yinYang(stemIndex), BranchYinYang: yinYang(branchIndex), Zodiac: zodiacs[branchIndex],
			CycleIndex: cycleZero + 1, DataVersion: DataVersion, CreatedAt: time.Now().UTC(),
		})
	}
	return rows, nil
}

func positiveMod(value, divisor int) int {
	result := value % divisor
	if result < 0 {
		return result + divisor
	}
	return result
}

func yinYang(index int) string {
	if index%2 == 0 {
		return "阳"
	}
	return "阴"
}
