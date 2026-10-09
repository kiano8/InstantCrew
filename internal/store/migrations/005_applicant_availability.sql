
CREATE TABLE IF NOT EXISTS applicant_availability (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    applicant_id INTEGER NOT NULL
                 REFERENCES applicant_profiles(user_id)
                 ON DELETE CASCADE,
    starts_at    INTEGER NOT NULL,
    ends_at      INTEGER NOT NULL,
    status       TEXT NOT NULL DEFAULT 'available'
                 CHECK (status IN ('available', 'unavailable')),
    created_at   INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at   INTEGER NOT NULL DEFAULT (unixepoch()),

    CHECK (ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_availability_applicant
    ON applicant_availability(applicant_id, starts_at, ends_at);

