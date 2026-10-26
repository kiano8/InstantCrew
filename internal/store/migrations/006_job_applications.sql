
CREATE TABLE IF NOT EXISTS job_applications (
    id           BIGSERIAL PRIMARY KEY,
    job_id       BIGINT NOT NULL
                 REFERENCES jobs(id) ON DELETE CASCADE,
    applicant_id BIGINT NOT NULL
                 REFERENCES applicant_profiles(user_id)
                 ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN (
                     'pending',
                     'accepted',
                     'rejected',
                     'withdrawn'
                 )),
    applied_at   BIGINT NOT NULL DEFAULT extract(epoch from now())::BIGINT,
    updated_at   BIGINT NOT NULL DEFAULT extract(epoch from now())::BIGINT,

    UNIQUE (job_id, applicant_id)
);

CREATE INDEX IF NOT EXISTS idx_applications_job
    ON job_applications(job_id, status);

CREATE INDEX IF NOT EXISTS idx_applications_applicant
    ON job_applications(applicant_id, status);

