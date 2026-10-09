
CREATE TABLE IF NOT EXISTS applicant_profiles (
    user_id           INTEGER PRIMARY KEY
                      REFERENCES users(id) ON DELETE CASCADE,
    employment_type   TEXT NOT NULL
                      CHECK (employment_type IN ('full-time', 'part-time')),
    expected_rate     INTEGER NOT NULL
                      CHECK (expected_rate > 0),
    rate_period       TEXT NOT NULL
                      CHECK (rate_period IN ('hour', 'day', 'week', 'month')),
    city              TEXT NOT NULL,
    location_address  TEXT,
    maps_url          TEXT,
    lat               REAL,
    lng               REAL,
    availability_mode TEXT NOT NULL DEFAULT 'now'
                      CHECK (availability_mode IN ('now', 'later')),
    available_until   INTEGER,
    created_at        INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at        INTEGER NOT NULL DEFAULT (unixepoch()),

    CHECK (lat IS NULL OR lat BETWEEN -90 AND 90),
    CHECK (lng IS NULL OR lng BETWEEN -180 AND 180)
);

