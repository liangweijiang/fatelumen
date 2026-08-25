package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/model"
	"gorm.io/gorm"
)

type unavailableLocationResolver struct{}

func (unavailableLocationResolver) Version() string { return "provided-location-v1" }
func (unavailableLocationResolver) Search(context.Context, string, string) ([]birthchart.LocationCandidate, error) {
	return nil, birthchart.ErrLocationSearchUnavailable
}
func (unavailableLocationResolver) Resolve(context.Context, birthchart.LocationInput) (*birthchart.ResolvedLocation, error) {
	return nil, birthchart.ErrLocationSearchUnavailable
}

type fakeFreeChartStore struct {
	records map[uint64]model.FreeChartRecord
	next    uint64
}

func (f *fakeFreeChartStore) Create(r *model.FreeChartRecord) error {
	f.next++
	r.ID = f.next
	f.records[r.ID] = *r
	return nil
}
func (f *fakeFreeChartStore) ListByUser(uid uint64, page, size int) ([]model.FreeChartRecord, int64, error) {
	out := []model.FreeChartRecord{}
	for _, r := range f.records {
		if r.UserID == uid {
			out = append(out, r)
		}
	}
	return out, int64(len(out)), nil
}
func (f *fakeFreeChartStore) FindByIDAndUser(id, uid uint64) (*model.FreeChartRecord, error) {
	r, ok := f.records[id]
	if !ok || r.UserID != uid {
		return nil, gorm.ErrRecordNotFound
	}
	return &r, nil
}
func (f *fakeFreeChartStore) DeleteByIDAndUser(id, uid uint64) (int64, error) {
	r, ok := f.records[id]
	if !ok || r.UserID != uid {
		return 0, nil
	}
	delete(f.records, id)
	return 1, nil
}
func (f *fakeFreeChartStore) BatchDeleteByUser(ids []uint64, uid uint64) (int64, error) {
	var n int64
	for _, id := range ids {
		r, ok := f.records[id]
		if ok && r.UserID == uid {
			delete(f.records, id)
			n++
		}
	}
	return n, nil
}

type fixedBirthChartEngine struct{ calls int }

func (e *fixedBirthChartEngine) Calculate(_ context.Context, _ birthchart.Input) (*birthchart.Result, error) {
	e.calls++
	return &birthchart.Result{
		Location:  birthchart.ResolvedLocation{CountryName: "China", RegionName: "Shanghai", Latitude: 31.23, Longitude: 121.47, TimezoneID: "Asia/Shanghai"},
		Timezone:  birthchart.TimezoneResult{HistoricalUTCOffset: 28800},
		SolarTime: birthchart.SolarTimeResult{AlgorithmVersion: "test", TrueSolarTime: time.Date(1990, 1, 1, 12, 0, 0, 0, time.UTC)},
		Mode:      birthchart.TrueSolarTime, DayBoundaryRule: birthchart.Midnight00, LocationVersion: "test-location",
		Chart: &model.ChartData{Pillars: model.Pillars{Year: model.Pillar{Stem: "庚", Branch: "午"}}, DayMaster: model.DayMaster{Stem: "甲"}, FiveElementsCount: map[string]int{"木": 1}, Meta: model.ChartMeta{SolarDate: "1990-01-01", LunarDate: "1989-12-05", Zodiac: "马"}},
	}, nil
}

func TestFreeChartHistoryOwnershipAndBatchDelete(t *testing.T) {
	store := &fakeFreeChartStore{records: map[uint64]model.FreeChartRecord{1: {ID: 1, UserID: 7}, 2: {ID: 2, UserID: 8}}}
	svc := &FreeChartService{engine: &fixedBirthChartEngine{}, repo: store}
	if _, err := svc.Get(context.Background(), 7, 2); err != ErrFreeChartNotFound {
		t.Fatalf("expected not found for other user, got %v", err)
	}
	deleted, err := svc.BatchDelete(context.Background(), 7, []uint64{1, 1, 2})
	if err != nil || deleted != 1 {
		t.Fatalf("unexpected batch delete: deleted=%d err=%v", deleted, err)
	}
	if _, ok := store.records[2]; !ok {
		t.Fatal("other user's record must remain")
	}
}

func TestFreeChartServiceReturnsCroppedResultWithoutPersistence(t *testing.T) {
	engine := &fixedBirthChartEngine{}
	svc := NewFreeChartService(engine)
	result, err := svc.Calculate(context.Background(), FreeChartInput{Gender: 1, CalendarType: 0, Year: 1990, Month: 1, Day: 1, Hour: 12, Location: birthchart.LocationInput{PlaceID: "1796236"}})
	if err != nil {
		t.Fatalf("calculate failed: %v", err)
	}
	if result.ChartHash == "" || result.Pillars.Year.Stem != "庚" || result.Location.TimezoneID != "Asia/Shanghai" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if engine.calls != 1 {
		t.Fatalf("expected one engine call, got %d", engine.calls)
	}
}

func TestFreeChartServiceFailsClosedWhenDualLocationCannotBeValidated(t *testing.T) {
	svc := NewFreeChartService(&fixedBirthChartEngine{})
	svc.SetLocationValidator(unavailableLocationResolver{})
	_, err := svc.Calculate(context.Background(), FreeChartInput{Gender: 1, CalendarType: 0, Year: 1990, Month: 1, Day: 1, Hour: 12, Location: birthchart.LocationInput{
		CountryCode: "CN", RegionCode: "HL", Latitude: 32, Longitude: 116, HasCoordinates: true,
	}})
	if !errors.Is(err, birthchart.ErrLocationSearchUnavailable) {
		t.Fatalf("expected fail-closed validation error, got %v", err)
	}
}

func TestFreeAndReportPathsProduceIdenticalHashAndPillars(t *testing.T) {
	engine := birthchart.NewDefaultEngine()
	input := FreeChartInput{Gender: 1, CalendarType: 0, Year: 1990, Month: 1, Day: 1, Hour: 12, Minute: 0, Location: birthchart.LocationInput{
		Latitude: 31.2304, Longitude: 121.4737, TimezoneID: "Asia/Shanghai", HasCoordinates: true,
	}}
	freeResult, err := NewFreeChartService(engine).Calculate(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	reportResult, err := engine.Calculate(context.Background(), birthchart.Input{Gender: input.Gender, CalendarType: input.CalendarType, Year: input.Year, Month: input.Month, Day: input.Day, Hour: input.Hour, Minute: input.Minute, Location: input.Location})
	if err != nil {
		t.Fatal(err)
	}
	reportHash := BuildChartHash(input.Gender, input.CalendarType, input.Year, input.Month, input.Day, input.Hour, input.Minute, false, reportResult)
	if freeResult.ChartHash != reportHash || !reflect.DeepEqual(freeResult.Pillars, reportResult.Chart.Pillars) {
		t.Fatalf("free/report mismatch: free_hash=%s report_hash=%s", freeResult.ChartHash, reportHash)
	}
}
