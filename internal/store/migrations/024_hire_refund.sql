-- A hire spent on a booking the employer ended while the crew member was on the
-- way is given back; this marks those assignments so history doesn't count them.
ALTER TABLE job_assignments ADD COLUMN hire_refunded BOOLEAN NOT NULL DEFAULT false;
