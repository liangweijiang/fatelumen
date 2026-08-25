CREATE TABLE IF NOT EXISTS geo_countries (
  code CHAR(2) PRIMARY KEY,
  name_en VARCHAR(128) NOT NULL,
  name_zh VARCHAR(128) NOT NULL,
  name_ja VARCHAR(128) NOT NULL,
  name_ko VARCHAR(128) NOT NULL,
  KEY idx_geo_countries_name_en (name_en)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS geo_cities (
  geo_name_id BIGINT UNSIGNED PRIMARY KEY,
  country_code CHAR(2) NOT NULL,
  admin1_code VARCHAR(20) NOT NULL DEFAULT '',
  admin1_name VARCHAR(128) NOT NULL DEFAULT '',
  name_en VARCHAR(200) NOT NULL,
  name_zh VARCHAR(200) NOT NULL,
  name_ja VARCHAR(200) NOT NULL,
  name_ko VARCHAR(200) NOT NULL,
  latitude DECIMAL(10,7) NOT NULL,
  longitude DECIMAL(10,7) NOT NULL,
  timezone_id VARCHAR(64) NOT NULL,
  KEY idx_geo_city_country_name (country_code, name_en),
  CONSTRAINT fk_geo_city_country FOREIGN KEY (country_code) REFERENCES geo_countries(code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
