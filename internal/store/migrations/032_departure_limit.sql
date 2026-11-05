-- Crew now have 30 minutes, not 1 hour, to tap On the Way. Bookings whose start is
-- still ahead move to start + 30 minutes; ones already counting down keep their deadline.
UPDATE job_assignments SET arrival_deadline_at = arrival_deadline_at - 1800
WHERE ended_at IS NULL AND transit_status = 'none'
  AND arrival_deadline_at > extract(epoch from now())::bigint + 3600;
