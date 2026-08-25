package model

type GeoCountry struct {
	Code   string `gorm:"type:char(2);primaryKey" json:"code"`
	NameEN string `gorm:"type:varchar(128);not null;index" json:"name_en"`
	NameZH string `gorm:"type:varchar(128);not null" json:"name_zh"`
	NameJA string `gorm:"type:varchar(128);not null" json:"name_ja"`
	NameKO string `gorm:"type:varchar(128);not null" json:"name_ko"`
}

func (GeoCountry) TableName() string { return "geo_countries" }

type GeoCity struct {
	GeoNameID   uint64  `gorm:"primaryKey;autoIncrement:false" json:"id"`
	CountryCode string  `gorm:"type:char(2);not null;index:idx_geo_city_country_name,priority:1" json:"country_code"`
	Admin1Code  string  `gorm:"type:varchar(20);not null;default:''" json:"admin1_code"`
	Admin1Name  string  `gorm:"type:varchar(128);not null;default:''" json:"admin1_name"`
	NameEN      string  `gorm:"type:varchar(200);not null;index:idx_geo_city_country_name,priority:2" json:"name_en"`
	NameZH      string  `gorm:"type:varchar(200);not null" json:"name_zh"`
	NameJA      string  `gorm:"type:varchar(200);not null" json:"name_ja"`
	NameKO      string  `gorm:"type:varchar(200);not null" json:"name_ko"`
	Latitude    float64 `gorm:"type:decimal(10,7);not null" json:"latitude"`
	Longitude   float64 `gorm:"type:decimal(10,7);not null" json:"longitude"`
	TimezoneID  string  `gorm:"type:varchar(64);not null" json:"timezone_id"`
}

func (GeoCity) TableName() string { return "geo_cities" }
