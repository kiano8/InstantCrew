-- Add proof_image, verification timestamp, and verifier info to payments and hiring pass payments
ALTER TABLE payments ADD COLUMN proof_image TEXT NOT NULL DEFAULT '';
ALTER TABLE payments ADD COLUMN verified_at INTEGER;
ALTER TABLE payments ADD COLUMN verified_by TEXT NOT NULL DEFAULT '';

ALTER TABLE applicant_hiring_pass_payments ADD COLUMN proof_image TEXT NOT NULL DEFAULT '';
ALTER TABLE applicant_hiring_pass_payments ADD COLUMN status TEXT NOT NULL DEFAULT 'paid';
ALTER TABLE applicant_hiring_pass_payments ADD COLUMN verified_at INTEGER;
ALTER TABLE applicant_hiring_pass_payments ADD COLUMN verified_by TEXT NOT NULL DEFAULT '';
