-- One-time codes emailed for signup verification and password resets. Only a
-- hash of the code is stored; a newer code for the same purpose replaces it.
CREATE TABLE IF NOT EXISTS email_codes (
    email TEXT NOT NULL,
    purpose TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    expires_at BIGINT NOT NULL,
    created_at BIGINT NOT NULL,
    PRIMARY KEY (email, purpose)
);
