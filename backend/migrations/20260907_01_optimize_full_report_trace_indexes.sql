-- Management trace queries use stable created_at/id ordering and retention
-- cleanup filters by expiry plus status. InnoDB appends the primary key to
-- secondary indexes, so these compact definitions also cover the id tie-break.
ALTER TABLE full_reports
  ADD INDEX idx_full_reports_created (created_at),
  ADD INDEX idx_full_reports_locale_created (locale, created_at),
  ADD INDEX idx_full_reports_expiry_status (expires_at, status);
