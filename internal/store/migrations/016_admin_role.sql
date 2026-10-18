-- Update users table to allow 'admin' role
CREATE TABLE IF NOT EXISTS users_v2 (
  id                       INTEGER PRIMARY KEY AUTOINCREMENT,
  email                    TEXT NOT NULL, 
  password_hash            TEXT NOT NULL,
  role                     TEXT NOT NULL CHECK (role IN ('employer', 'applicant', 'admin')),
  name                     TEXT NOT NULL DEFAULT '',
  company                  TEXT NOT NULL DEFAULT '',
  created_at               INTEGER NOT NULL, 
  UNIQUE (email, role)
);

INSERT OR IGNORE INTO users_v2 (id, email, password_hash, role, name, company, created_at)
SELECT id, email, password_hash, role, name, company, created_at FROM users;

DROP TABLE users;

ALTER TABLE users_v2 RENAME TO users;
