
CREATE TABLE IF NOT EXISTS job_assignments (
    job_id         INTEGER NOT NULL
                   REFERENCES jobs(id) ON DELETE CASCADE,
    applicant_id   INTEGER NOT NULL
                   REFERENCES users(id) ON DELETE CASCADE,
    accepted_at    INTEGER NOT NULL DEFAULT (unixepoch()),
    ended_at       INTEGER,
    transit_status TEXT NOT NULL DEFAULT 'none'
                   CHECK (transit_status IN (
                       'none',
                       'en_route',
                       'arrived'
                   )),

    PRIMARY KEY (job_id, applicant_id),

    CHECK (ended_at IS NULL OR ended_at >= accepted_at)
);

CREATE INDEX IF NOT EXISTS idx_assign_applicant
    ON job_assignments(applicant_id);

