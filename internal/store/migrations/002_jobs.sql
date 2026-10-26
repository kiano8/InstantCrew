CREATE TABLE jobs (
  id                  BIGSERIAL PRIMARY KEY,
  employer_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  category            TEXT NOT NULL,
  role_id             TEXT NOT NULL,
  role_label          TEXT NOT NULL,
  employment_type     TEXT NOT NULL CHECK (employment_type IN ('full-time', 'part-time')),
  rate_centavos       BIGINT NOT NULL CHECK (rate_period IN ('hour', 'day', 'week', 'month')),
  rate_period         TEXT NOT NULL CHECK (rate_period IN ('hour', 'day', 'week', 'month')),
  city                TEXT NOT NULL,
  map_location        TEXT NOT NULL,
  maps_url            TEXT NOT NULL,
  lat                 DOUBLE PRECISION,
  lng                 DOUBLE PRECISION,
  timing              TEXT NOT NULL CHECK (timing IN ('now', 'later')),
  scheduled_at        BIGINT,
  crew_needed         INTEGER NOT NULL DEFAULT 1 CHECK (crew_needed BETWEEN 1 AND 20),
  accepted_count      INTEGER NOT NULL DEFAULT 0,
  status              TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'matching', 'accepted', 'active',            'completed', 'unfulfilled', 'cancelled')),
  search_attempts     INTEGER NOT NULL DEFAULT 0,
  search_expires_at   BIGINT,
  created_at          BIGINT NOT NULL,
  updated_at          BIGINT NOT NULL
);

CREATE INDEX idx_jobs_employer ON jobs(employer_id, status);
CREATE INDEX idx_jobs_search   ON jobs(status, search_expires_at);
CREATE INDEX idx_jobs_feed     ON jobs(status, city, role_id);

CREATE TABLE payments (
  id          BIGSERIAL PRIMARY KEY,
  job_id      BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  user_id     BIGINT NOT NULL REFERENCES users(id),
  amount_centavos   BIGINT NOT NULL,
  method            TEXT NOT NULL CHECK (method IN ('gcash', 'maya', 'card')),
  status            TEXT NOT NULL,
  provider_ref      TEXT NOT NULL,
  created_at        BIGINT NOT NULL
);

CREATE TABLE job_assignments (
  job_id        BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  applicant_id  BIGINT NOT NULL REFERENCES users(id),
  accepted_at   BIGINT NOT NULL,
  ended_at      BIGINT,
  transit_status  TEXT NOT NULL DEFAULT 'none',
  PRIMARY KEY (job_id, applicant_id)
);

CREATE INDEX idx_assign_applicant ON job_assignments(applicant_id);
