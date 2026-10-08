CREATE TABLE IF NOT EXISTS users (
  id                       INTEGER PRIMARY KEY AUTOINCREMENT,
  email                    TEXT NOT NULL, 
  password_hash            TEXT NOT NULL,
  role                     TEXT NOT NULL CHECK (role IN ('employer', 'applicant')),
  name                     TEXT NOT NULL DEFAULT '',
  company                  TEXT NOT NULL DEFAULT '',
  created_at               INTEGER NOT NULL, 
  UNIQUE (email, role)
);

CREATE TABLE IF NOT EXISTS sessions (
  token_hash               TEXT PRIMARY KEY,
  user_id                  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at               INTEGER NOT NULL, 
  created_at               INTEGER NOT NULL
); 

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

