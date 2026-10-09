
CREATE TABLE IF NOT EXISTS job_applications (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id       INTEGER NOT NULL
                 REFERENCES jobs(id) ON DELETE CASCADE,
    applicant_id INTEGER NOT NULL
                 REFERENCES applicant_profiles(user_id)
                 ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN (
                     'pending',
                     'accepted',
                     'rejected',
                     'withdrawn'
                 )),
    applied_at   INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at   INTEGER NOT NULL DEFAULT (unixepoch()),

    UNIQUE (job_id, applicant_id)
);

CREATE INDEX IF NOT EXISTS idx_applications_job
    ON job_applications(job_id, status);

CREATE INDEX IF NOT EXISTS idx_applications_applicant
    ON job_applications(applicant_id, status);

