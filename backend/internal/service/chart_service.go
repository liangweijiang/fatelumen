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

type ChartService struct {
	chartRepo   *repository.ChartRepo
	profileRepo *repository.ProfileRepo
	engine      birthchart.Engine
}

func NewChartService(chartRepo *repository.ChartRepo, profileRepo *repository.ProfileRepo, engines ...birthchart.Engine) *ChartService {
	var engine birthchart.Engine = birthchart.NewDefaultEngine()
	if len(engines) > 0 && engines[0] != nil {
		engine = engines[0]
	}
	return &ChartService{chartRepo: chartRepo, profileRepo: profileRepo, engine: engine}
}

type CreateChartInput struct {
	ProfileID uint64 `json:"profile_id"`
}

func (s *ChartService) GetByID(ctx context.Context, userID uint64, chartID uint64) (*model.Chart, error) {
	chart, err := s.chartRepo.FindByID(chartID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("chart not found")
		}
		return nil, err
	}
	profile, err := s.profileRepo.FindByID(chart.ProfileID)
	if err != nil || profile.UserID != userID {
		return nil, errors.New("chart not found")
	}
	return chart, nil
}

func (s *ChartService) Calculate(ctx context.Context, userID uint64, in CreateChartInput) (*model.Chart, error) {
	profile, err := s.profileRepo.FindByID(in.ProfileID)
	if err != nil {
		logger.FromCtx(ctx).Error("find profile for chart failed", "user_id", userID, "profile_id", in.ProfileID, "err", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("profile not found")
		}
		return nil, err
	}
	if profile.UserID != userID {
		return nil, errors.New("profile not found")
	}

	isLeap := profile.IsLeapMonth == 1
	result, err := s.engine.Calculate(ctx, birthchart.Input{
		Gender: profile.Gender, CalendarType: profile.CalendarType,
		Year: int(profile.BirthYear), Month: int(profile.BirthMonth), Day: int(profile.BirthDay),
		Hour: int(profile.BirthHour), Minute: int(profile.BirthMinute), IsLeapMonth: isLeap,
		Location: birthchart.LocationInput{
			CountryCode: profile.CountryCode, CountryName: profile.CountryName,
			RegionCode: profile.RegionCode, RegionName: profile.RegionName, City: profile.City,
			PlaceID: profile.PlaceID, DisplayName: profile.BirthPlace,
			Latitude: profile.Latitude, Longitude: profile.Longitude, TimezoneID: profile.Timezone,
			HasCoordinates: profile.HasCoordinates || profile.Longitude != 0 || profile.Latitude != 0,
		},
	})
	if err != nil {
		logger.FromCtx(ctx).Error("birth chart engine failed", "user_id", userID, "profile_id", profile.ID, "err", err)
		return nil, err
	}
	chartHash := BuildChartHash(profile.Gender, profile.CalendarType, int(profile.BirthYear), int(profile.BirthMonth), int(profile.BirthDay), int(profile.BirthHour), int(profile.BirthMinute), isLeap, result)
	existing, findErr := s.chartRepo.FindByHash(chartHash)
	if findErr == nil && existing != nil {
		return existing, nil
	}

	chart := &model.Chart{
		ProfileID: profile.ID,
		ChartHash: chartHash,
		ChartData: *result.Chart,
		CreatedAt: time.Now(),
	}
	if err := s.chartRepo.Create(chart); err != nil {
		logger.FromCtx(ctx).Error("create chart failed", "user_id", userID, "profile_id", profile.ID, "chart_hash", chartHash, "err", err)
		return nil, err
	}
	return chart, nil
}
