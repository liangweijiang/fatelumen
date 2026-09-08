ALTER TABLE jobs
    ADD COLUMN lane VARCHAR(32) NOT NULL DEFAULT 'default' AFTER type,
    ADD INDEX idx_jobs_lane_status_created (lane, status, created_at);

UPDATE jobs
SET lane = CASE
    WHEN type = 'full_report_pdf_v1' THEN 'pdf-render'
    WHEN type = 'full_report_v2' THEN 'report-generation'
    ELSE 'default'
END;
