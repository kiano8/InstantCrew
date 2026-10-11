ALTER TABLE payments ADD COLUMN search_attempt INTEGER NOT NULL DEFAULT 1;

CREATE UNIQUE INDEX idx_payments_job_search_attempt
  ON payments(job_id, search_attempt);
