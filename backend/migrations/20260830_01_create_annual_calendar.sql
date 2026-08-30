CREATE TABLE IF NOT EXISTS annual_calendar_years (
  year INT NOT NULL PRIMARY KEY,
  gan_zhi VARCHAR(8) NOT NULL,
  stem VARCHAR(4) NOT NULL,
  branch VARCHAR(4) NOT NULL,
  stem_element VARCHAR(4) NOT NULL,
  branch_element VARCHAR(4) NOT NULL,
  stem_yin_yang VARCHAR(4) NOT NULL,
  branch_yin_yang VARCHAR(4) NOT NULL,
  zodiac VARCHAR(8) NOT NULL,
  cycle_index TINYINT UNSIGNED NOT NULL,
  data_version VARCHAR(64) NOT NULL,
  created_at DATETIME(3) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
