-- FateLumen Phase 2.6: structured birth location fields.
-- Apply once to MySQL 8.0 before deploying the shared birth chart engine.

ALTER TABLE birth_profiles
  MODIFY COLUMN timezone VARCHAR(64) NULL COMMENT 'IANA Timezone ID',
  MODIFY COLUMN longitude DECIMAL(9,6) NULL COMMENT 'east positive',
  ADD COLUMN country_code VARCHAR(8) NULL AFTER birth_place,
  ADD COLUMN country_name VARCHAR(96) NULL AFTER country_code,
  ADD COLUMN region_code VARCHAR(32) NULL AFTER country_name,
  ADD COLUMN region_name VARCHAR(96) NULL AFTER region_code,
  ADD COLUMN city VARCHAR(96) NULL AFTER region_name,
  ADD COLUMN place_id VARCHAR(128) NULL AFTER city,
  ADD COLUMN latitude DECIMAL(9,6) NULL COMMENT 'north positive' AFTER longitude,
  ADD COLUMN has_coordinates TINYINT(1) NOT NULL DEFAULT 0 AFTER latitude;

UPDATE birth_profiles
SET has_coordinates = 1
WHERE longitude <> 0 OR latitude <> 0;
