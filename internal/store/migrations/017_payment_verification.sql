-- Track whether a hiring-pass payment has already credited 5 hires, and
-- keep a short reason when accounting rejects a receipt.
ALTER TABLE payments ADD COLUMN rejection_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE applicant_hiring_pass_payments ADD COLUMN credits_granted INTEGER NOT NULL DEFAULT 0;
ALTER TABLE applicant_hiring_pass_payments ADD COLUMN rejection_reason TEXT NOT NULL DEFAULT '';

-- Existing rows already granted hires at the moment of pay (pre-verification).
UPDATE applicant_hiring_pass_payments SET credits_granted = 1 WHERE credits_granted = 0;

CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status, created_at);
CREATE INDEX IF NOT EXISTS idx_pass_pay_status ON applicant_hiring_pass_payments(status, created_at);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
