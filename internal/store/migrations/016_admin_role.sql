-- Update users table to allow 'admin' role
CREATE TABLE IF NOT EXISTS users_v2 (
  id                       BIGSERIAL PRIMARY KEY,
  email                    TEXT NOT NULL,
  password_hash            TEXT NOT NULL,
  role                     TEXT NOT NULL CHECK (role IN ('employer', 'applicant', 'admin')),
  name                     TEXT NOT NULL DEFAULT '',
  company                  TEXT NOT NULL DEFAULT '',
  created_at               BIGINT NOT NULL,
  UNIQUE (email, role)
);

INSERT INTO users_v2 (id, email, password_hash, role, name, company, created_at)
SELECT id, email, password_hash, role, name, company, created_at FROM users
ON CONFLICT (email, role) DO NOTHING;

DROP TABLE users CASCADE;

ALTER TABLE users_v2 RENAME TO users;
