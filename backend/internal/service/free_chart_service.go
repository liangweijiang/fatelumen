package service

import (
	"context"
	"errors"
	"time"

	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
	"gorm.io/gorm"
)

var ErrFreeChartNotFound = errors.New("free chart not found")
var ErrInvalidBatchDelete = errors.New("invalid batch delete request")

type FreeChartInput struct {
	Gender       int8                     `json:"gender" binding:"oneof=0 1"`
	CalendarType int8                     `json:"calendar_type" binding:"oneof=0 1"`
	Year         int                      `json:"year" binding:"required"`
	Month        int                      `json:"month" binding:"required"`
	Day          int                      `json:"day" binding:"required"`
	Hour         int                      `json:"hour"`
	Minute       int                      `json:"minute"`
	IsLeapMonth  bool                     `json:"is_leap_month"`
	Location     birthchart.LocationInput `json:"location" binding:"required"`
}

type FreeChartResponse struct {
	ID                uint64                      `json:"id,omitempty"`
	ChartHash         string                      `json:"chart_hash"`
	Location          birthchart.ResolvedLocation `json:"location"`
	Pillars           model.Pillars               `json:"pillars"`
	DayMaster         model.DayMaster             `json:"day_master"`
	FiveElementsCount map[string]int              `json:"five_elements_count"`
	LunarDate         string                      `json:"lunar_date"`
	SolarDate         string                      `json:"solar_date"`
	Zodiac            string                      `json:"zodiac"`
	Gender            int8                        `json:"gender"`
	CalendarType      int8                        `json:"calendar_type"`
	BirthYear         int                         `json:"birth_year"`
	BirthMonth        int                         `json:"birth_month"`
	BirthDay          int                         `json:"birth_day"`
	BirthHour         int                         `json:"birth_hour"`
	BirthMinute       int                         `json:"birth_minute"`
	IsLeapMonth       bool                        `json:"is_leap_month"`
	TimeCalculation   model.TimeCalculationMeta   `json:"time_calculation"`
	CreatedAt         time.Time                   `json:"created_at,omitempty"`
}

type freeChartStore interface {
	Create(*model.FreeChartRecord) error
	ListByUser(userID uint64, page, pageSize int) ([]model.FreeChartRecord, int64, error)
	FindByIDAndUser(id, userID uint64) (*model.FreeChartRecord, error)
	DeleteByIDAndUser(id, userID uint64) (int64, error)
	BatchDeleteByUser(ids []uint64, userID uint64) (int64, error)
}

type FreeChartService struct {
	engine            birthchart.Engine
	locationValidator birthchart.LocationResolver
	repo              freeChartStore
}

// SetLocationValidator enables the free-chart boundary to reject forged
// country/region + coordinate combinations before the shared chart engine runs.
func (s *FreeChartService) SetLocationValidator(resolver birthchart.LocationResolver) {
	s.locationValidator = resolver
}

func NewFreeChartService(engine birthchart.Engine, repos ...*repository.FreeChartRepo) *FreeChartService {
	var repo freeChartStore
	if len(repos) > 0 {
		repo = repos[0]
	}
	return &FreeChartService{engine: engine, repo: repo}
}

func (s *FreeChartService) Calculate(ctx context.Context, in FreeChartInput) (*FreeChartResponse, error) {
	if s.locationValidator != nil && in.Location.HasCoordinates && hasNamedLocation(in.Location) {
		if s.locationValidator.Version() == "provided-location-v1" {
			return nil, birthchart.ErrLocationSearchUnavailable
		}
		if _, err := s.locationValidator.Resolve(ctx, in.Location); err != nil {
			return nil, err
		}
	}
	result, err := s.engine.Calculate(ctx, birthchart.Input{Gender: in.Gender, CalendarType: in.CalendarType, Year: in.Year, Month: in.Month, Day: in.Day, Hour: in.Hour, Minute: in.Minute, IsLeapMonth: in.IsLeapMonth, Location: in.Location})
	if err != nil {
		logger.FromCtx(ctx).Warn("free chart calculation failed", "err", err)
		return nil, err
	}
	hash := BuildChartHash(in.Gender, in.CalendarType, in.Year, in.Month, in.Day, in.Hour, in.Minute, in.IsLeapMonth, result)
	logger.FromCtx(ctx).Info("free chart calculated", "chart_hash", hash)
	return &FreeChartResponse{ChartHash: hash, Location: result.Location, Pillars: result.Chart.Pillars, DayMaster: result.Chart.DayMaster, FiveElementsCount: result.Chart.FiveElementsCount, LunarDate: result.Chart.Meta.LunarDate, SolarDate: result.Chart.Meta.SolarDate, Zodiac: result.Chart.Meta.Zodiac, Gender: in.Gender, CalendarType: in.CalendarType, BirthYear: in.Year, BirthMonth: in.Month, BirthDay: in.Day, BirthHour: in.Hour, BirthMinute: in.Minute, IsLeapMonth: in.IsLeapMonth, TimeCalculation: result.Chart.Meta.TimeCalculation}, nil
}

func hasNamedLocation(location birthchart.LocationInput) bool {
	return location.CountryCode != "" || location.CountryName != "" || location.RegionCode != "" || location.RegionName != "" || location.City != "" || location.PlaceID != ""
}

func (s *FreeChartService) Create(ctx context.Context, userID uint64, in FreeChartInput) (*FreeChartResponse, error) {
	result, err := s.Calculate(ctx, in)
	if err != nil {
		return nil, err
	}
	if s.repo == nil {
		return result, nil
	}
	record := &model.FreeChartRecord{UserID: userID, Gender: in.Gender, CalendarType: in.CalendarType, BirthYear: int16(in.Year), BirthMonth: int8(in.Month), BirthDay: int8(in.Day), BirthHour: int8(in.Hour), BirthMinute: int8(in.Minute), IsLeapMonth: in.IsLeapMonth, CountryCode: result.Location.CountryCode, CountryName: result.Location.CountryName, RegionCode: result.Location.RegionCode, RegionName: result.Location.RegionName, City: result.Location.City, PlaceID: result.Location.PlaceID, DisplayName: result.Location.DisplayName, TimezoneID: result.Location.TimezoneID, Latitude: result.Location.Latitude, Longitude: result.Location.Longitude, ChartHash: result.ChartHash, ChartData: model.ChartData{Pillars: result.Pillars, DayMaster: result.DayMaster, FiveElementsCount: result.FiveElementsCount, Meta: model.ChartMeta{SolarDate: result.SolarDate, LunarDate: result.LunarDate, Zodiac: result.Zodiac, TimeCalculation: result.TimeCalculation}}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := s.repo.Create(record); err != nil {
		logger.FromCtx(ctx).Error("create free chart history failed", "err", err, "user_id", userID, "chart_hash", result.ChartHash)
		return nil, err
	}
	result.ID = record.ID
	result.CreatedAt = record.CreatedAt
	return result, nil
}

type FreeChartPage struct {
	Items    []FreeChartResponse `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

func (s *FreeChartService) List(ctx context.Context, userID uint64, page, pageSize int) (*FreeChartPage, error) {
	records, total, err := s.repo.ListByUser(userID, page, pageSize)
	if err != nil {
		logger.FromCtx(ctx).Error("list free chart history failed", "err", err, "user_id", userID)
		return nil, err
	}
	items := make([]FreeChartResponse, 0, len(records))
	for i := range records {
		items = append(items, mapFreeChartRecord(&records[i]))
	}
	return &FreeChartPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}
func (s *FreeChartService) Get(ctx context.Context, userID, id uint64) (*FreeChartResponse, error) {
	record, err := s.repo.FindByIDAndUser(id, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFreeChartNotFound
	}
	if err != nil {
		logger.FromCtx(ctx).Error("get free chart history failed", "err", err, "user_id", userID, "free_chart_id", id)
		return nil, err
	}
	item := mapFreeChartRecord(record)
	return &item, nil
}
func (s *FreeChartService) Delete(ctx context.Context, userID, id uint64) error {
	affected, err := s.repo.DeleteByIDAndUser(id, userID)
	if err != nil {
		logger.FromCtx(ctx).Error("delete free chart history failed", "err", err, "user_id", userID, "free_chart_id", id)
		return err
	}
	if affected == 0 {
		return ErrFreeChartNotFound
	}
	return nil
}
func (s *FreeChartService) BatchDelete(ctx context.Context, userID uint64, ids []uint64) (int64, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return 0, ErrInvalidBatchDelete
	}
	unique := make([]uint64, 0, len(ids))
	seen := map[uint64]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}
	if len(unique) == 0 {
		return 0, ErrInvalidBatchDelete
	}
	affected, err := s.repo.BatchDeleteByUser(unique, userID)
	if err != nil {
		logger.FromCtx(ctx).Error("batch delete free chart history failed", "err", err, "user_id", userID, "count", len(unique))
		return 0, err
	}
	return affected, nil
}

func mapFreeChartRecord(r *model.FreeChartRecord) FreeChartResponse {
	return FreeChartResponse{ID: r.ID, ChartHash: r.ChartHash, Location: birthchart.ResolvedLocation{CountryCode: r.CountryCode, CountryName: r.CountryName, RegionCode: r.RegionCode, RegionName: r.RegionName, City: r.City, PlaceID: r.PlaceID, DisplayName: r.DisplayName, Latitude: r.Latitude, Longitude: r.Longitude, TimezoneID: r.TimezoneID}, Pillars: r.ChartData.Pillars, DayMaster: r.ChartData.DayMaster, FiveElementsCount: r.ChartData.FiveElementsCount, LunarDate: r.ChartData.Meta.LunarDate, SolarDate: r.ChartData.Meta.SolarDate, Zodiac: r.ChartData.Meta.Zodiac, Gender: r.Gender, CalendarType: r.CalendarType, BirthYear: int(r.BirthYear), BirthMonth: int(r.BirthMonth), BirthDay: int(r.BirthDay), BirthHour: int(r.BirthHour), BirthMinute: int(r.BirthMinute), IsLeapMonth: r.IsLeapMonth, TimeCalculation: r.ChartData.Meta.TimeCalculation, CreatedAt: r.CreatedAt}
}
