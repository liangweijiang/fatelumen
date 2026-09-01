-- Fresh full-report execution domain. It intentionally does not reuse the
-- legacy reports/report_fact_snapshots/report_llm_calls execution schema.
-- Legacy tables are removed only by the final cut-over migration after all
-- runtime readers have switched to these tables.

CREATE TABLE IF NOT EXISTS full_reports (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  public_id CHAR(26) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  profile_id BIGINT UNSIGNED NULL,
  chart_id BIGINT UNSIGNED NULL,
  order_id BIGINT UNSIGNED NULL,
  source_report_id BIGINT UNSIGNED NULL,
  locale VARCHAR(8) NOT NULL,
  pay_method VARCHAR(16) NOT NULL,
  paid TINYINT(1) NOT NULL DEFAULT 0,
  status VARCHAR(24) NOT NULL,
  current_stage VARCHAR(32) NOT NULL,
  chapter_total TINYINT UNSIGNED NOT NULL DEFAULT 10,
  chapter_succeeded TINYINT UNSIGNED NOT NULL DEFAULT 0,
  chapter_failed TINYINT UNSIGNED NOT NULL DEFAULT 0,
  provider_chain_key VARCHAR(64) NOT NULL,
  chapter_concurrency TINYINT UNSIGNED NOT NULL,
  facts_hash CHAR(64) NOT NULL,
  execution_hash CHAR(64) NOT NULL DEFAULT '',
  content_hash CHAR(64) NOT NULL DEFAULT '',
  error_code VARCHAR(64) NOT NULL DEFAULT '',
  error_summary VARCHAR(512) NOT NULL DEFAULT '',
  retention_policy VARCHAR(32) NOT NULL,
  started_at DATETIME(3) NULL,
  completed_at DATETIME(3) NULL,
  expires_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_reports_public_id (public_id),
  KEY idx_full_reports_user_created (user_id, created_at, id),
  KEY idx_full_reports_status_created (status, created_at, id),
  KEY idx_full_reports_expiry (expires_at, status, id),
  KEY idx_full_reports_facts_created (facts_hash, created_at, id),
  KEY idx_full_reports_profile (profile_id),
  KEY idx_full_reports_chart (chart_id),
  KEY idx_full_reports_order (order_id),
  KEY idx_full_reports_paid (paid),
  KEY idx_full_reports_source (source_report_id),
  CONSTRAINT chk_full_reports_chapter_total CHECK (chapter_total = 10),
  CONSTRAINT chk_full_reports_concurrency CHECK (chapter_concurrency BETWEEN 1 AND 10)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS full_report_execution_snapshots (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  report_id BIGINT UNSIGNED NOT NULL,
  chart_hash CHAR(64) NOT NULL,
  facts_hash CHAR(64) NOT NULL,
  execution_hash CHAR(64) NOT NULL,
  input_schema_version VARCHAR(64) NOT NULL,
  chart_schema_version VARCHAR(64) NOT NULL,
  facts_schema_version VARCHAR(64) NOT NULL,
  rule_set_version VARCHAR(64) NOT NULL,
  prompt_version VARCHAR(64) NOT NULL,
  dictionary_version VARCHAR(64) NOT NULL,
  runtime_policy_version VARCHAR(64) NOT NULL,
  location_database_version VARCHAR(64) NOT NULL,
  timezone_database_version VARCHAR(64) NOT NULL,
  solar_algorithm_version VARCHAR(64) NOT NULL,
  lunar_go_version VARCHAR(64) NOT NULL,
  frozen_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_report_execution_snapshot_report (report_id),
  CONSTRAINT fk_full_report_execution_snapshot_report FOREIGN KEY (report_id)
    REFERENCES full_reports (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS full_report_execution_payloads (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  snapshot_id BIGINT UNSIGNED NOT NULL,
  input_snapshot JSON NOT NULL,
  time_calculation_snapshot JSON NOT NULL,
  chart_snapshot JSON NOT NULL,
  facts_snapshot JSON NOT NULL,
  preflight_result JSON NOT NULL,
  chapter_plan_snapshot JSON NOT NULL,
  runtime_config_snapshot JSON NOT NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_report_execution_payload_snapshot (snapshot_id),
  CONSTRAINT fk_full_report_execution_payload_snapshot FOREIGN KEY (snapshot_id)
    REFERENCES full_report_execution_snapshots (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS full_report_chapters (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  report_id BIGINT UNSIGNED NOT NULL,
  chapter_no TINYINT UNSIGNED NOT NULL,
  chapter_key VARCHAR(64) NOT NULL,
  title VARCHAR(128) NOT NULL,
  status VARCHAR(24) NOT NULL,
  attempt_count SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  selected_attempt_id BIGINT UNSIGNED NULL,
  prompt_hash CHAR(64) NOT NULL,
  output_hash CHAR(64) NOT NULL DEFAULT '',
  schema_valid TINYINT(1) NOT NULL DEFAULT 0,
  validation_status VARCHAR(24) NOT NULL,
  error_code VARCHAR(64) NOT NULL DEFAULT '',
  error_summary VARCHAR(512) NOT NULL DEFAULT '',
  started_at DATETIME(3) NULL,
  completed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_report_chapter_no (report_id, chapter_no),
  UNIQUE KEY uk_full_report_chapter_key (report_id, chapter_key),
  KEY idx_full_report_chapters_status (status, report_id),
  KEY idx_full_report_chapters_selected_attempt (selected_attempt_id),
  CONSTRAINT chk_full_report_chapter_no CHECK (chapter_no BETWEEN 1 AND 10),
  CONSTRAINT fk_full_report_chapter_report FOREIGN KEY (report_id)
    REFERENCES full_reports (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS full_report_chapter_payloads (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  chapter_id BIGINT UNSIGNED NOT NULL,
  semantic_digest MEDIUMTEXT NOT NULL,
  language_instruction MEDIUMTEXT NOT NULL,
  terminology_snapshot JSON NOT NULL,
  final_prompt MEDIUMTEXT NOT NULL,
  output_schema JSON NOT NULL,
  final_raw_output MEDIUMTEXT NULL,
  final_parsed_output JSON NULL,
  validation_result JSON NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_report_chapter_payload_chapter (chapter_id),
  CONSTRAINT fk_full_report_chapter_payload_chapter FOREIGN KEY (chapter_id)
    REFERENCES full_report_chapters (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS full_report_attempts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  report_id BIGINT UNSIGNED NOT NULL,
  chapter_id BIGINT UNSIGNED NOT NULL,
  attempt_no SMALLINT UNSIGNED NOT NULL,
  route_no TINYINT UNSIGNED NOT NULL,
  provider VARCHAR(64) NOT NULL,
  model VARCHAR(128) NOT NULL,
  status VARCHAR(24) NOT NULL,
  schema_valid TINYINT(1) NOT NULL DEFAULT 0,
  validation_status VARCHAR(24) NOT NULL,
  error_code VARCHAR(64) NOT NULL DEFAULT '',
  error_summary VARCHAR(512) NOT NULL DEFAULT '',
  prompt_hash CHAR(64) NOT NULL,
  output_hash CHAR(64) NOT NULL DEFAULT '',
  prompt_tokens INT NOT NULL DEFAULT 0,
  completion_tokens INT NOT NULL DEFAULT 0,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  trace_id VARCHAR(64) NOT NULL,
  started_at DATETIME(3) NOT NULL,
  finished_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_report_attempt (chapter_id, attempt_no),
  KEY idx_full_report_attempt_report_chapter (report_id, chapter_id, attempt_no),
  KEY idx_full_report_attempt_trace (trace_id),
  KEY idx_full_report_attempt_status_started (status, started_at, id),
  CONSTRAINT fk_full_report_attempt_report FOREIGN KEY (report_id)
    REFERENCES full_reports (id) ON DELETE CASCADE,
  CONSTRAINT fk_full_report_attempt_chapter FOREIGN KEY (chapter_id)
    REFERENCES full_report_chapters (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS full_report_attempt_payloads (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  attempt_id BIGINT UNSIGNED NOT NULL,
  request_parameters JSON NOT NULL,
  request_prompt MEDIUMTEXT NOT NULL,
  raw_output MEDIUMTEXT NULL,
  parsed_output JSON NULL,
  schema_errors JSON NULL,
  validation_result JSON NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_report_attempt_payload_attempt (attempt_id),
  CONSTRAINT fk_full_report_attempt_payload_attempt FOREIGN KEY (attempt_id)
    REFERENCES full_report_attempts (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS full_report_results (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  report_id BIGINT UNSIGNED NOT NULL,
  locale VARCHAR(8) NOT NULL,
  content_json JSON NOT NULL,
  content_hash CHAR(64) NOT NULL,
  render_version VARCHAR(64) NOT NULL,
  pdf_storage_key VARCHAR(512) NOT NULL DEFAULT '',
  pdf_url VARCHAR(512) NOT NULL DEFAULT '',
  pdf_hash CHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_report_result_report (report_id),
  CONSTRAINT fk_full_report_result_report FOREIGN KEY (report_id)
    REFERENCES full_reports (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
