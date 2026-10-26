CREATE TABLE IF NOT EXISTS users (
  id                       BIGSERIAL PRIMARY KEY,
  email                    TEXT NOT NULL,
  password_hash            TEXT NOT NULL,
  role                     TEXT NOT NULL CHECK (role IN ('employer', 'applicant')),
  name                     TEXT NOT NULL DEFAULT '',
  company                  TEXT NOT NULL DEFAULT '',
  created_at               BIGINT NOT NULL,
  UNIQUE (email, role)
);

CREATE TABLE IF NOT EXISTS sessions (
  token_hash               TEXT PRIMARY KEY,
  user_id                  BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at               BIGINT NOT NULL,
  created_at               BIGINT NOT NULL
); 

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

