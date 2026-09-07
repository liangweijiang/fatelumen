ALTER TABLE full_reports
  ADD COLUMN generating_at DATETIME(3) NULL AFTER started_at,
  ADD COLUMN assembling_at DATETIME(3) NULL AFTER generating_at,
  ADD COLUMN rendering_at DATETIME(3) NULL AFTER assembling_at,
  ADD COLUMN failed_at DATETIME(3) NULL AFTER rendering_at;
