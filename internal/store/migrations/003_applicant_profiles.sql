
CREATE TABLE IF NOT EXISTS applicant_profiles (
    user_id           BIGINT PRIMARY KEY
                      REFERENCES users(id) ON DELETE CASCADE,
    employment_type   TEXT NOT NULL
                      CHECK (employment_type IN ('full-time', 'part-time')),
    expected_rate     BIGINT NOT NULL
                      CHECK (expected_rate > 0),
    rate_period       TEXT NOT NULL
                      CHECK (rate_period IN ('hour', 'day', 'week', 'month')),
    city              TEXT NOT NULL,
    location_address  TEXT,
    maps_url          TEXT,
    lat               DOUBLE PRECISION,
    lng               DOUBLE PRECISION,
    availability_mode TEXT NOT NULL DEFAULT 'now'
                      CHECK (availability_mode IN ('now', 'later')),
    available_until   BIGINT,
    created_at        BIGINT NOT NULL DEFAULT extract(epoch from now())::BIGINT,
    updated_at        BIGINT NOT NULL DEFAULT extract(epoch from now())::BIGINT,

    CHECK (lat IS NULL OR lat BETWEEN -90 AND 90),
    CHECK (lng IS NULL OR lng BETWEEN -180 AND 180)
);

