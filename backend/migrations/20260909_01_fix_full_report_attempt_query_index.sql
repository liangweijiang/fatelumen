ALTER TABLE full_report_attempts
  DROP INDEX idx_full_report_attempt_report_chapter,
  ADD INDEX idx_full_report_attempt_report_chapter (report_id, chapter_id, attempt_no DESC);
