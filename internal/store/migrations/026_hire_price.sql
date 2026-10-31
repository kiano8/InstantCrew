-- Crew buy hires one by one at ₱2 each and hold at most 5, so a hiring pass
-- receipt is ₱2 to ₱10 in steps of ₱2 (amount / 200 = hires bought).
ALTER TABLE applicant_hiring_pass_payments DROP CONSTRAINT applicant_hiring_pass_payments_amount_centavos_check;
ALTER TABLE applicant_hiring_pass_payments ADD CONSTRAINT applicant_hiring_pass_payments_amount_centavos_check
    CHECK (amount_centavos BETWEEN 200 AND 1000 AND amount_centavos % 200 = 0);
