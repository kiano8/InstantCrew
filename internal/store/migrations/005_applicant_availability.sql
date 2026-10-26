
CREATE TABLE IF NOT EXISTS applicant_availability (
    id           BIGSERIAL PRIMARY KEY,
    applicant_id BIGINT NOT NULL
                 REFERENCES applicant_profiles(user_id)
                 ON DELETE CASCADE,
    starts_at    BIGINT NOT NULL,
    ends_at      BIGINT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'available'
                 CHECK (status IN ('available', 'unavailable')),
    created_at   BIGINT NOT NULL DEFAULT extract(epoch from now())::BIGINT,
    updated_at   BIGINT NOT NULL DEFAULT extract(epoch from now())::BIGINT,

    CHECK (ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_availability_applicant
    ON applicant_availability(applicant_id, starts_at, ends_at);

