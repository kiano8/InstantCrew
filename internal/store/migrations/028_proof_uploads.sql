-- Who uploaded each receipt, and whether a payment has already used it.
CREATE TABLE proof_uploads (
  path       TEXT PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at BIGINT NOT NULL,
  used_at    BIGINT
);
CREATE INDEX idx_proof_uploads_user ON proof_uploads(user_id, created_at);
