-- After a contract ends, the employer rates the crew member and the crew member
-- rates the employer, 1-5 stars. stars is NULL when the reviewer chose to skip.
CREATE TABLE job_reviews (
    job_id      BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    reviewer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stars       SMALLINT CHECK (stars BETWEEN 1 AND 5),
    created_at  BIGINT NOT NULL,
    PRIMARY KEY (job_id, reviewer_id)
);
CREATE INDEX idx_job_reviews_reviewee ON job_reviews(reviewee_id);

-- Contracts that ended before ratings existed are not asked about.
INSERT INTO job_reviews(job_id, reviewer_id, reviewee_id, stars, created_at)
SELECT j.id, j.employer_id, a.applicant_id, NULL, extract(epoch from now())::bigint
FROM jobs j JOIN job_assignments a ON a.job_id=j.id
WHERE j.status='completed' AND a.arrival_deadline_at IS NOT NULL AND a.transit_status <> 'no_show'
ON CONFLICT DO NOTHING;
INSERT INTO job_reviews(job_id, reviewer_id, reviewee_id, stars, created_at)
SELECT j.id, a.applicant_id, j.employer_id, NULL, extract(epoch from now())::bigint
FROM jobs j JOIN job_assignments a ON a.job_id=j.id
WHERE j.status='completed' AND a.arrival_deadline_at IS NOT NULL AND a.transit_status <> 'no_show'
ON CONFLICT DO NOTHING;
