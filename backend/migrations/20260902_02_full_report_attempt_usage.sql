ALTER TABLE full_report_attempts
  MODIFY COLUMN prompt_tokens INT NULL,
  MODIFY COLUMN completion_tokens INT NULL,
  ADD COLUMN total_tokens INT NULL AFTER completion_tokens;
