CREATE TABLE IF NOT EXISTS prompt_chapter_configs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  chapter_key VARCHAR(64) NOT NULL,
  fact_keys JSON NOT NULL,
  updated_by_admin_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_prompt_chapter_configs_key (chapter_key),
  KEY idx_prompt_chapter_configs_admin (updated_by_admin_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
