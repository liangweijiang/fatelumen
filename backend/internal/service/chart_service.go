package service

import (
	"context"
	"errors"
	"time"

	"fatelumen/backend/internal/bazi"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/hash"
	"fatelumen/backend/internal/repository"

	"gorm.io/gorm"
)

type ChartService struct {
	chartRepo   *repository.ChartRepo
	profileRepo *repository.ProfileRepo
}

func NewChartService(chartRepo *repository.ChartRepo, profileRepo *repository.ProfileRepo) *ChartService {
	return &ChartService{chartRepo: chartRepo, profileRepo: profileRepo}
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("profile not found")
		}
		return nil, err
	}
	if profile.UserID != userID {
		return nil, errors.New("profile not found")
	}

	isLeap := profile.IsLeapMonth == 1
	chartHash := hash.CalcChartHash(
		profile.Gender, profile.CalendarType,
		int(profile.BirthYear), int(profile.BirthMonth), int(profile.BirthDay),
		int(profile.BirthHour), int(profile.BirthMinute),
		isLeap, profile.Timezone,
	)

	existing, err := s.chartRepo.FindByHash(chartHash)
	if err == nil && existing != nil {
		return existing, nil
	}

	chartData, err := bazi.Calculate(bazi.BirthInput{
		Gender:       profile.Gender,
		CalendarType: profile.CalendarType,
		Year:         int(profile.BirthYear),
		Month:        int(profile.BirthMonth),
		Day:          int(profile.BirthDay),
		Hour:         int(profile.BirthHour),
		Minute:       int(profile.BirthMinute),
		IsLeapMonth:  isLeap,
		Longitude:    profile.Longitude,
	})
	if err != nil {
		return nil, err
	}

	chart := &model.Chart{
		ProfileID: profile.ID,
		ChartHash: chartHash,
		ChartData: *chartData,
		CreatedAt: time.Now(),
	}
	if err := s.chartRepo.Create(chart); err != nil {
		return nil, err
	}
	return chart, nil
}
