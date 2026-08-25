package repository

import (
	"context"
	"strings"

	"fatelumen/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GeoRepo struct{ db *gorm.DB }

type AdminGeoCity struct {
	GeoNameID   uint64  `json:"id"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	RegionName  string  `json:"region_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func NewGeoRepo(db *gorm.DB) *GeoRepo { return &GeoRepo{db: db} }

func localeColumn(locale string) string {
	switch strings.ToLower(locale) {
	case "zh":
		return "name_zh"
	case "ja":
		return "name_ja"
	case "ko":
		return "name_ko"
	default:
		return "name_en"
	}
}

func (r *GeoRepo) Countries(query, locale string, page, size int) ([]model.GeoCountry, int64, error) {
	var rows []model.GeoCountry
	var total int64
	db := r.db.Model(&model.GeoCountry{})
	if q := strings.TrimSpace(query); q != "" {
		like := "%" + q + "%"
		db = db.Where("code = ? OR name_en LIKE ? OR name_zh LIKE ? OR name_ja LIKE ? OR name_ko LIKE ?", strings.ToUpper(q), like, like, like, like)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(localeColumn(locale) + " ASC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (r *GeoRepo) Cities(countryCode, query, locale string, page, size int) ([]model.GeoCity, int64, error) {
	var rows []model.GeoCity
	var total int64
	db := r.db.Model(&model.GeoCity{}).Where("country_code = ?", strings.ToUpper(countryCode))
	if q := strings.TrimSpace(query); q != "" {
		like := "%" + q + "%"
		db = db.Where("name_en LIKE ? OR name_zh LIKE ? OR name_ja LIKE ? OR name_ko LIKE ?", like, like, like, like)
		db = db.Order(clause.Expr{SQL: "CASE WHEN name_en = ? OR name_zh = ? OR name_ja = ? OR name_ko = ? THEN 0 WHEN name_en LIKE ? OR name_zh LIKE ? OR name_ja LIKE ? OR name_ko LIKE ? THEN 1 ELSE 2 END", Vars: []any{q, q, q, q, q + "%", q + "%", q + "%", q + "%"}, WithoutParentheses: true})
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(localeColumn(locale) + " ASC, geo_name_id ASC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (r *GeoRepo) City(id uint64) (*model.GeoCity, error) {
	var row model.GeoCity
	if err := r.db.First(&row, "geo_name_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GeoRepo) CityContext(ctx context.Context, id uint64) (*model.GeoCity, error) {
	var row model.GeoCity
	if err := r.db.WithContext(ctx).First(&row, "geo_name_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GeoRepo) CountryName(ctx context.Context, code, locale string) (string, error) {
	var row model.GeoCountry
	if err := r.db.WithContext(ctx).First(&row, "code = ?", strings.ToUpper(code)).Error; err != nil {
		return "", err
	}
	switch strings.ToLower(locale) {
	case "zh":
		return row.NameZH, nil
	case "ja":
		return row.NameJA, nil
	case "ko":
		return row.NameKO, nil
	default:
		return row.NameEN, nil
	}
}

func (r *GeoRepo) SearchCities(ctx context.Context, query, locale string, limit int) ([]model.GeoCity, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []model.GeoCity{}, nil
	}
	like := "%" + q + "%"
	var rows []model.GeoCity
	err := r.db.WithContext(ctx).Where("name_en LIKE ? OR name_zh LIKE ? OR name_ja LIKE ? OR name_ko LIKE ?", like, like, like, like).
		Order(localeColumn(locale) + " ASC, geo_name_id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *GeoRepo) AdminCities(ctx context.Context, query string, page, size int) ([]AdminGeoCity, int64, error) {
	base := r.db.WithContext(ctx).Table("geo_cities AS city").
		Joins("JOIN geo_countries AS country ON country.code = city.country_code")
	if q := strings.TrimSpace(query); q != "" {
		like := "%" + q + "%"
		base = base.Where("city.country_code = ? OR country.name_zh LIKE ? OR country.name_en LIKE ? OR city.name_zh LIKE ? OR city.name_en LIKE ?", strings.ToUpper(q), like, like, like, like)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AdminGeoCity
	err := base.Select("city.geo_name_id, city.country_code, CASE WHEN country.name_zh = '' THEN country.name_en ELSE country.name_zh END AS country_name, CASE WHEN city.name_zh = '' THEN city.name_en ELSE city.name_zh END AS region_name, city.latitude, city.longitude").
		Order("country_name ASC, region_name ASC, city.geo_name_id ASC").
		Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	return rows, total, err
}

func (r *GeoRepo) UpdateCoordinates(ctx context.Context, id uint64, latitude, longitude float64) (*model.GeoCity, error) {
	result := r.db.WithContext(ctx).Model(&model.GeoCity{}).Where("geo_name_id = ?", id).
		Updates(map[string]any{"latitude": latitude, "longitude": longitude})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row model.GeoCity
	if err := r.db.WithContext(ctx).First(&row, "geo_name_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
