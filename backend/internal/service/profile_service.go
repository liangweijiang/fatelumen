package service

import (
	"context"
	"fmt"
	"time"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

// CreateProfileInput 创建出生档案的请求体。
type CreateProfileInput struct {
	DisplayName  string `json:"display_name"`
	Gender       int8   `json:"gender"`
	CalendarType int8   `json:"calendar_type"`
	BirthYear    int    `json:"birth_year"`
	BirthMonth   int    `json:"birth_month"`
	BirthDay     int    `json:"birth_day"`
	BirthHour    int    `json:"birth_hour"`
	BirthMinute  int    `json:"birth_minute"`
	IsLeapMonth  bool   `json:"is_leap_month"`
	BirthPlace   string `json:"birth_place"`
	Timezone     string `json:"timezone"`
	Longitude    float64 `json:"longitude"`
}

// ProfileService 出生档案业务逻辑。
type ProfileService struct {
	repo *repository.ProfileRepo
}

func NewProfileService(repo *repository.ProfileRepo) *ProfileService {
	return &ProfileService{repo: repo}
}

// Create 创建出生档案。
func (s *ProfileService) Create(ctx context.Context, userID uint64, in CreateProfileInput) (*model.BirthProfile, error) {
	isLeap := int8(0)
	if in.IsLeapMonth {
		isLeap = 1
	}
	minute := in.BirthMinute
	if in.CalendarType == 0 && in.BirthHour < 0 {
		minute = 0
	}
	if in.BirthYear < 1 || in.BirthYear > 9999 ||
		in.BirthMonth < 1 || in.BirthMonth > 12 ||
		in.BirthDay < 1 || in.BirthDay > 31 ||
		in.BirthHour < -1 || in.BirthHour > 23 ||
		in.BirthMinute < 0 || in.BirthMinute > 59 {
		logger.FromCtx(ctx).Warn("invalid birth profile input",
			"user_id", userID,
			"birth_year", in.BirthYear,
			"birth_month", in.BirthMonth,
			"birth_day", in.BirthDay,
			"birth_hour", in.BirthHour,
			"birth_minute", in.BirthMinute,
		)
		return nil, fmt.Errorf("invalid birth date fields")
	}
	profile := &model.BirthProfile{
		UserID:       userID,
		DisplayName:  in.DisplayName,
		Gender:       in.Gender,
		CalendarType: in.CalendarType,
		BirthYear:    int16(in.BirthYear),
		BirthMonth:   int8(in.BirthMonth),
		BirthDay:     int8(in.BirthDay),
		BirthHour:    int8(in.BirthHour),
		BirthMinute:  int8(minute),
		IsLeapMonth:  isLeap,
		BirthPlace:   in.BirthPlace,
		Timezone:     in.Timezone,
		Longitude:    in.Longitude,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.repo.Create(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// List 列出用户所有档案。
func (s *ProfileService) List(ctx context.Context, userID uint64) ([]model.BirthProfile, error) {
	return s.repo.ListByUserID(userID)
}

// Get 获取单个档案（需要校验归属）。
func (s *ProfileService) Get(ctx context.Context, userID, profileID uint64) (*model.BirthProfile, error) {
	profile, err := s.repo.FindByID(profileID)
	if err != nil {
		return nil, err
	}
	if profile.UserID != userID {
		return nil, nil // 不属于该用户，返回 nil
	}
	return profile, nil
}

// Delete 删除档案（需要校验归属）。
func (s *ProfileService) Delete(ctx context.Context, userID, profileID uint64) error {
	profile, err := s.repo.FindByID(profileID)
	if err != nil {
		return err
	}
	if profile.UserID != userID {
		return nil // 不属于该用户，静默
	}
	return s.repo.Delete(profileID)
}
