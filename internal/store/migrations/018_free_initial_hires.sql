-- Every applicant starts with 5 free hires and pays nothing upfront.
-- New signups get their row at creation time (see store/history.go).
-- This backfills existing applicants who never paid and therefore have no pass row.
-- Applicants that already have a row (paid passes) are left untouched.
INSERT OR IGNORE INTO applicant_hiring_passes
    (applicant_id, total_hires, remaining_hires, used_hires, amount_centavos, batches_paid, updated_at)
SELECT id, 5, 5, 0, 0, 0, unixepoch() FROM users WHERE role = 'applicant';
