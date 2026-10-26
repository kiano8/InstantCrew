ALTER TABLE applicant_profiles ADD COLUMN rating_hundredths INTEGER NOT NULL DEFAULT 490 CHECK (rating_hundredths BETWEEN 100 AND 500);

CREATE TABLE applicant_offer_impressions (
    job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    applicant_id BIGINT NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    shown_at BIGINT NOT NULL DEFAULT extract(epoch from now())::BIGINT,
    resolved BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (job_id, applicant_id)
);

CREATE TABLE applicant_rating_events (
    job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    applicant_id BIGINT NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    outcome TEXT NOT NULL CHECK (outcome IN ('accepted', 'declined')),
    PRIMARY KEY (job_id, applicant_id)
);
