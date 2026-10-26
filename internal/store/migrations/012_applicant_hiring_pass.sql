CREATE TABLE IF NOT EXISTS applicant_hiring_passes (
    applicant_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_hires INTEGER NOT NULL DEFAULT 0,
    remaining_hires INTEGER NOT NULL DEFAULT 0,
    used_hires INTEGER NOT NULL DEFAULT 0,
    amount_centavos BIGINT NOT NULL DEFAULT 0,
    batches_paid INTEGER NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from now())::BIGINT,
    CHECK (total_hires >= 0 AND remaining_hires >= 0 AND used_hires >= 0),
    CHECK (total_hires = remaining_hires + used_hires)
);

CREATE TABLE IF NOT EXISTS applicant_hiring_pass_payments (
    id BIGSERIAL PRIMARY KEY,
    applicant_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_centavos BIGINT NOT NULL CHECK (amount_centavos = 1000),
    method TEXT NOT NULL CHECK (method IN ('gcash', 'maya', 'card')),
    created_at BIGINT NOT NULL
);
