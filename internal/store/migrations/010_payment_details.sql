ALTER TABLE payments RENAME TO payments_old;

CREATE TABLE payments (
  id                BIGSERIAL PRIMARY KEY,
  job_id            BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  user_id           BIGINT NOT NULL REFERENCES users(id),
  amount_centavos   BIGINT NOT NULL,
  method            TEXT NOT NULL CHECK (method IN ('cash', 'gcash', 'maya', 'card')),
  details           TEXT NOT NULL DEFAULT '',
  status            TEXT NOT NULL,
  provider_ref      TEXT NOT NULL,
  created_at        BIGINT NOT NULL
);

INSERT INTO payments (id, job_id, user_id, amount_centavos, method, details, status, provider_ref, created_at)
SELECT id, job_id, user_id, amount_centavos, method, '', status, provider_ref, created_at FROM payments_old;

DROP TABLE payments_old;
