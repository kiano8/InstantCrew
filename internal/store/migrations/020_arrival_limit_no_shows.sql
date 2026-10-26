-- 1-hour arrival limit: set when the employer confirms the booking (or at the
-- scheduled start for later shifts). Crew who have not marked arrival by then
-- are recorded as no-shows and the shift is auto-cancelled.
ALTER TABLE job_assignments ADD COLUMN arrival_deadline_at BIGINT;
ALTER TABLE job_assignments DROP CONSTRAINT IF EXISTS job_assignments_transit_status_check;
ALTER TABLE job_assignments ADD CONSTRAINT job_assignments_transit_status_check
    CHECK (transit_status IN ('none', 'en_route', 'arrived', 'no_show'));

CREATE INDEX IF NOT EXISTS idx_assign_arrival_deadline
    ON job_assignments(arrival_deadline_at)
    WHERE ended_at IS NULL AND arrival_deadline_at IS NOT NULL;

-- No-show strikes and the penalties they escalate to.
ALTER TABLE applicant_profiles ADD COLUMN no_show_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE applicant_profiles ADD COLUMN deactivated_until BIGINT;
ALTER TABLE applicant_profiles ADD COLUMN served_temporary_deactivation BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE applicant_profiles ADD COLUMN is_permanently_deactivated BOOLEAN NOT NULL DEFAULT false;
