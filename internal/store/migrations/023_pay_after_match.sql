-- Crew searches no longer run in timed tries/attempts/batches: a search stays open
-- until a crew member accepts or the employer cancels, and the search fee is paid
-- after a crew member accepts.
ALTER TABLE jobs DROP COLUMN current_try;
ALTER TABLE jobs DROP COLUMN max_tries;
ALTER TABLE jobs DROP COLUMN current_attempt;
ALTER TABLE jobs DROP COLUMN max_attempts;
ALTER TABLE jobs DROP COLUMN current_batch;
ALTER TABLE jobs DROP COLUMN search_expires_at;
DROP TABLE job_match_batch_members;

-- ₱50 per crew member the employer ends while they are on the way. It is added
-- to the next search fee and cleared when that fee is verified.
ALTER TABLE users ADD COLUMN search_penalty_centavos BIGINT NOT NULL DEFAULT 0;
-- The part of a fee that was penalty, so verifying it clears exactly that much.
ALTER TABLE payments ADD COLUMN penalty_centavos BIGINT NOT NULL DEFAULT 0;
