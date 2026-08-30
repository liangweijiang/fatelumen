package annualcalendar

import "testing"

func TestGenerateCompleteSexagenaryCycle(t *testing.T) {
	rows, err := Generate(2026, 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 60 || rows[0].GanZhi != "丙午" || rows[0].CycleIndex != 43 {
		t.Fatalf("unexpected first year: %+v", rows[0])
	}
	if rows[59].Year != 2085 || rows[59].GanZhi != "乙巳" {
		t.Fatalf("unexpected last year: %+v", rows[59])
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.GanZhi] {
			t.Fatalf("duplicate ganzhi in one cycle: %s", row.GanZhi)
		}
		seen[row.GanZhi] = true
	}
}
