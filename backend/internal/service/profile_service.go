package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"fatelumen/backend/internal/birthchart"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/repository"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrInvalidProfile  = errors.New("invalid birth profile")
)

// CreateProfileInput 创建出生档案的请求体。
type CreateProfileInput struct {
	DisplayName    string  `json:"display_name"`
	Gender         int8    `json:"gender"`
	CalendarType   int8    `json:"calendar_type"`
	BirthYear      int     `json:"birth_year"`
	BirthMonth     int     `json:"birth_month"`
	BirthDay       int     `json:"birth_day"`
	BirthHour      int     `json:"birth_hour"`
	BirthMinute    int     `json:"birth_minute"`
	IsLeapMonth    bool    `json:"is_leap_month"`
	BirthPlace     string  `json:"birth_place"`
	CountryCode    string  `json:"country_code"`
	CountryName    string  `json:"country_name"`
	RegionCode     string  `json:"region_code"`
	RegionName     string  `json:"region_name"`
	City           string  `json:"city"`
	PlaceID        string  `json:"place_id"`
	Timezone       string  `json:"timezone"`
	Longitude      float64 `json:"longitude"`
	Latitude       float64 `json:"latitude"`
	HasCoordinates bool    `json:"has_coordinates"`
}

// ProfileService 出生档案业务逻辑。
type ProfileService struct {
	repo      *repository.ProfileRepo
	locations birthchart.LocationResolver
}

func (s *ProfileService) SetLocationResolver(resolver birthchart.LocationResolver) {
	s.locations = resolver
}

func NewProfileService(repo *repository.ProfileRepo) *ProfileService {
	return &ProfileService{repo: repo}
}

// Create 创建出生档案。
func (s *ProfileService) Create(ctx context.Context, userID uint64, in CreateProfileInput) (*model.BirthProfile, error) {
	return s.create(ctx, userID, in, true)
}

// CreateReportSubject persists the submitted report subject without making it a
// personal profile. It remains available to the asynchronous report worker but
// is excluded from the user's profile list.
func (s *ProfileService) CreateReportSubject(ctx context.Context, userID uint64, in CreateProfileInput) (*model.BirthProfile, error) {
	return s.create(ctx, userID, in, false)
}

func (s *ProfileService) create(ctx context.Context, userID uint64, in CreateProfileInput, saved bool) (*model.BirthProfile, error) {
	if err := s.resolveLocation(ctx, &in); err != nil {
		logger.FromCtx(ctx).Warn("resolve birth profile location failed", "user_id", userID, "err", err)
		return nil, err
	}
	if err := validateProfileInput(in); err != nil {
		logger.FromCtx(ctx).Warn("invalid birth profile input", "user_id", userID, "err", err)
		return nil, err
	}
	isLeap := int8(0)
	if in.IsLeapMonth {
		isLeap = 1
	}
	minute := in.BirthMinute
	if in.CalendarType == 0 && in.BirthHour < 0 {
		minute = 0
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
		CountryCode:  in.CountryCode, CountryName: in.CountryName,
		RegionCode: in.RegionCode, RegionName: in.RegionName, City: in.City, PlaceID: in.PlaceID,
		Timezone:       in.Timezone,
		Longitude:      in.Longitude,
		Latitude:       in.Latitude,
		HasCoordinates: in.HasCoordinates || in.Longitude != 0 || in.Latitude != 0,
		Saved:          saved,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.repo.Create(profile); err != nil {
		logger.FromCtx(ctx).Error("create birth profile failed", "user_id", userID, "saved", saved, "err", err)
		return nil, err
	}
	return profile, nil
}

// Update explicitly replaces a saved profile owned by the current user.
func (s *ProfileService) Update(ctx context.Context, userID, profileID uint64, in CreateProfileInput) (*model.BirthProfile, error) {
	if err := s.resolveLocation(ctx, &in); err != nil {
		logger.FromCtx(ctx).Warn("resolve birth profile update location failed", "user_id", userID, "profile_id", profileID, "err", err)
		return nil, err
	}
	if err := validateProfileInput(in); err != nil {
		logger.FromCtx(ctx).Warn("invalid birth profile update", "user_id", userID, "profile_id", profileID, "err", err)
		return nil, err
	}
	if _, err := s.repo.FindSavedByIDAndUserID(profileID, userID); err != nil {
		logger.FromCtx(ctx).Warn("birth profile update target not found", "user_id", userID, "profile_id", profileID, "err", err)
		return nil, ErrProfileNotFound
	}
	isLeap := int8(0)
	if in.IsLeapMonth {
		isLeap = 1
	}
	minute := in.BirthMinute
	if in.BirthHour < 0 {
		minute = 0
	}
	updates := map[string]interface{}{
		"display_name": in.DisplayName, "gender": in.Gender, "calendar_type": in.CalendarType,
		"birth_year": in.BirthYear, "birth_month": in.BirthMonth, "birth_day": in.BirthDay,
		"birth_hour": in.BirthHour, "birth_minute": minute, "is_leap_month": isLeap,
		"birth_place": in.BirthPlace, "country_code": in.CountryCode, "country_name": in.CountryName,
		"region_code": in.RegionCode, "region_name": in.RegionName, "city": in.City, "place_id": in.PlaceID,
		"timezone": in.Timezone, "longitude": in.Longitude, "latitude": in.Latitude,
		"has_coordinates": in.HasCoordinates || in.Longitude != 0 || in.Latitude != 0,
		"updated_at":      time.Now(),
	}
	if err := s.repo.UpdateOwnedSaved(profileID, userID, updates); err != nil {
		logger.FromCtx(ctx).Error("update birth profile failed", "user_id", userID, "profile_id", profileID, "err", err)
		return nil, err
	}
	profile, err := s.repo.FindSavedByIDAndUserID(profileID, userID)
	if err != nil {
		logger.FromCtx(ctx).Error("reload updated birth profile failed", "user_id", userID, "profile_id", profileID, "err", err)
		return nil, err
	}
	return profile, nil
}

func (s *ProfileService) resolveLocation(ctx context.Context, in *CreateProfileInput) error {
	if s.locations == nil {
		return nil
	}
	resolved, err := s.locations.Resolve(ctx, birthchart.LocationInput{
		CountryCode: in.CountryCode, CountryName: in.CountryName,
		RegionCode: in.RegionCode, RegionName: in.RegionName, City: in.City,
		PlaceID: in.PlaceID, DisplayName: in.BirthPlace,
		TimezoneID: in.Timezone, Longitude: in.Longitude, Latitude: in.Latitude,
		HasCoordinates: in.HasCoordinates,
	})
	if err != nil {
		return err
	}
	in.CountryCode, in.CountryName = resolved.CountryCode, resolved.CountryName
	in.RegionCode, in.RegionName, in.City = resolved.RegionCode, resolved.RegionName, resolved.City
	in.PlaceID, in.BirthPlace = resolved.PlaceID, resolved.DisplayName
	in.Timezone, in.Longitude, in.Latitude = resolved.TimezoneID, resolved.Longitude, resolved.Latitude
	in.HasCoordinates = true
	return nil
}

func validateProfileInput(in CreateProfileInput) error {
	if in.Gender != 0 && in.Gender != 1 {
		return fmt.Errorf("%w: gender must be 0 or 1", ErrInvalidProfile)
	}
	if in.CalendarType != 0 && in.CalendarType != 1 {
		return fmt.Errorf("%w: calendar_type must be 0 or 1", ErrInvalidProfile)
	}
	if in.BirthYear < 1 || in.BirthYear > 9999 || in.BirthMonth < 1 || in.BirthMonth > 12 ||
		in.BirthHour < -1 || in.BirthHour > 23 || in.BirthMinute < 0 || in.BirthMinute > 59 {
		return fmt.Errorf("%w: invalid birth date fields", ErrInvalidProfile)
	}
	if in.CalendarType == 0 {
		date := time.Date(in.BirthYear, time.Month(in.BirthMonth), in.BirthDay, 0, 0, 0, 0, time.UTC)
		if date.Year() != in.BirthYear || int(date.Month()) != in.BirthMonth || date.Day() != in.BirthDay {
			return fmt.Errorf("%w: invalid solar date", ErrInvalidProfile)
		}
	} else if in.BirthDay < 1 || in.BirthDay > 30 {
		return fmt.Errorf("%w: invalid lunar date", ErrInvalidProfile)
	}
	if strings.TrimSpace(in.Timezone) == "" {
		return fmt.Errorf("%w: timezone is required", ErrInvalidProfile)
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return fmt.Errorf("%w: invalid timezone", ErrInvalidProfile)
	}
	if in.Longitude < -180 || in.Longitude > 180 {
		return fmt.Errorf("%w: invalid longitude", ErrInvalidProfile)
	}
	if in.Latitude < -90 || in.Latitude > 90 {
		return fmt.Errorf("%w: invalid latitude", ErrInvalidProfile)
	}
	return nil
}

// List 列出用户所有档案。
func (s *ProfileService) List(ctx context.Context, userID uint64) ([]model.BirthProfile, error) {
	profiles, err := s.repo.ListByUserID(userID)
	if err != nil {
		logger.FromCtx(ctx).Error("list birth profiles failed", "user_id", userID, "err", err)
	}
	return profiles, err
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
