CREATE TABLE IF NOT EXISTS full_report_render_jobs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    report_id BIGINT UNSIGNED NOT NULL,
    render_version VARCHAR(64) NOT NULL,
    status VARCHAR(24) NOT NULL,
    attempt_count SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    max_attempts SMALLINT UNSIGNED NOT NULL DEFAULT 3,
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_summary VARCHAR(512) NOT NULL DEFAULT '',
    started_at DATETIME(3) NULL,
    finished_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_full_report_render_version (report_id, render_version),
    KEY idx_full_report_render_jobs_report (report_id),
    KEY idx_full_report_render_status_updated (status, updated_at),
    CONSTRAINT fk_full_report_render_jobs_report FOREIGN KEY (report_id) REFERENCES full_reports(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
