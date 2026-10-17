ALTER TABLE applicant_profiles ADD COLUMN rating_hundredths INTEGER NOT NULL DEFAULT 490 CHECK (rating_hundredths BETWEEN 100 AND 500);

CREATE TABLE applicant_offer_impressions (
    job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    applicant_id INTEGER NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    shown_at INTEGER NOT NULL DEFAULT (unixepoch()),
    resolved INTEGER NOT NULL DEFAULT 0 CHECK (resolved IN (0, 1)),
    PRIMARY KEY (job_id, applicant_id)
);

CREATE TABLE applicant_rating_events (
    job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    applicant_id INTEGER NOT NULL REFERENCES applicant_profiles(user_id) ON DELETE CASCADE,
    outcome TEXT NOT NULL CHECK (outcome IN ('accepted', 'declined')),
    PRIMARY KEY (job_id, applicant_id)
);
