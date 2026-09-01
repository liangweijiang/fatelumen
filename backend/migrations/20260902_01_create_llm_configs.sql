CREATE TABLE IF NOT EXISTS llm_provider_configs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code VARCHAR(64) NOT NULL,
  name VARCHAR(100) NOT NULL,
  base_url VARCHAR(500) NOT NULL,
  api_key_ciphertext TEXT NOT NULL,
  api_key_hint VARCHAR(32) NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_llm_provider_configs_code (code),
  KEY idx_llm_provider_configs_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS llm_model_configs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  provider_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(120) NOT NULL,
  model_id VARCHAR(200) NOT NULL,
  priority INT NOT NULL DEFAULT 100,
  max_retries INT NOT NULL DEFAULT 3,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_llm_model_provider_model (provider_id, model_id),
  KEY idx_llm_model_configs_priority (priority),
  KEY idx_llm_model_configs_enabled (enabled),
  CONSTRAINT fk_llm_models_provider FOREIGN KEY (provider_id)
    REFERENCES llm_provider_configs(id) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
