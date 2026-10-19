ALTER TABLE jobs ADD COLUMN current_batch INTEGER NOT NULL DEFAULT 1;

CREATE TABLE job_match_batch_members (
    job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    search_attempt INTEGER NOT NULL,
    batch_number INTEGER NOT NULL,
    slot INTEGER NOT NULL,
    applicant_id INTEGER NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    PRIMARY KEY (job_id, search_attempt, batch_number, applicant_id)
);

CREATE INDEX idx_job_match_batches
    ON job_match_batch_members(job_id, search_attempt, batch_number);
