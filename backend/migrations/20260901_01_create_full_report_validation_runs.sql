CREATE TABLE IF NOT EXISTS full_report_validation_runs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  report_id BIGINT UNSIGNED NOT NULL,
  round_no SMALLINT UNSIGNED NOT NULL,
  validator_version VARCHAR(64) NOT NULL,
  status VARCHAR(24) NOT NULL,
  retryable TINYINT(1) NOT NULL DEFAULT 0,
  affected_chapters TINYINT UNSIGNED NOT NULL DEFAULT 0,
  error_code VARCHAR(64) NOT NULL DEFAULT '',
  error_summary VARCHAR(512) NOT NULL DEFAULT '',
  started_at DATETIME(3) NOT NULL,
  finished_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_report_validation_round (report_id, round_no),
  KEY idx_full_report_validation_status (status, created_at, id),
  CONSTRAINT fk_full_report_validation_report FOREIGN KEY (report_id)
    REFERENCES full_reports (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS full_report_validation_payloads (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  validation_run_id BIGINT UNSIGNED NOT NULL,
  validation_result JSON NOT NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_full_report_validation_payload_run (validation_run_id),
  CONSTRAINT fk_full_report_validation_payload_run FOREIGN KEY (validation_run_id)
    REFERENCES full_report_validation_runs (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
