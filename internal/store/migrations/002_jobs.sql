CREATE TABLE jobs (
  id                  INTEGER PRIMARY KEY AUTOINCREMENT,
  employer_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  category            TEXT NOT NULL,
  role_id             TEXT NOT NULL,
  role_label          TEXT NOT NULL,
  employment_type     TEXT NOT NULL CHECK (employment_type IN ('full-time', 'part-time')),
  rate_centavos       INTEGER NOT NULL CHECK (rate_period IN ('hour', 'day', 'week', 'month')),
  rate_period         TEXT NOT NULL CHECK (rate_period IN ('hour', 'day', 'week', 'month')),
  city                TEXT NOT NULL,
  map_location        TEXT NOT NULL,
  maps_url            TEXT NOT NULL,
  lat                 REAL,
  lng                 REAL,
  timing              TEXT NOT NULL CHECK (timing IN ('now', 'later')),
  scheduled_at        INTEGER,
  crew_needed         INTEGER NOT NULL DEFAULT 1 CHECK (crew_needed BETWEEN 1 AND 20),
  accepted_count      INTEGER NOT NULL DEFAULT 0,
  status              TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'matching', 'accepted', 'active',            'completed', 'unfulfilled', 'cancelled')),
  search_attempts     INTEGER NOT NULL DEFAULT 0,
  search_expires_at   INTEGER,
  created_at          INTEGER NOT NULL,
  updated_at          INTEGER NOT NULL
);

CREATE INDEX idx_jobs_employer ON jobs(employer_id, status);
CREATE INDEX idx_jobs_search   ON jobs(status, search_expires_at);
CREATE INDEX idx_jobs_feed     ON jobs(status, city, role_id);

CREATE TABLE payments (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id      INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  user_id     INTEGER NOT NULL REFERENCES users(id),
  amount_centavos   INTEGER NOT NULL,
  method            TEXT NOT NULL CHECK (method IN ('gcash', 'maya', 'card')),
  status            TEXT NOT NULL,
  provider_ref      TEXT NOT NULL,
  created_at        INTEGER NOT NULL
);

CREATE TABLE job_assignments (
  job_id        INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  applicant_id  INTEGER NOT NULL REFERENCES users(id),
  accepted_at   INTEGER NOT NULL,
  ended_at      INTEGER,
  transit_status  TEXT NOT NULL DEFAULT 'none',
  PRIMARY KEY (job_id, applicant_id)
);

CREATE INDEX idx_assign_applicant ON job_assignments(applicant_id);
