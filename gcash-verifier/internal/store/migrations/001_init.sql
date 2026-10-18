-- Pending/paid GCash payments. Money is stored as integer centavos.
-- pay_centavos is the exact amount the payer must send (base + random 1..99
-- centavos). It is unique among pending payments, which is how an incoming
-- SMS is matched to a payment even when GCash masks the sender's name.
CREATE TABLE payments (
    id            TEXT PRIMARY KEY,
    label         TEXT NOT NULL DEFAULT '',
    base_centavos INTEGER NOT NULL,
    pay_centavos  INTEGER NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'paid', 'expired')),
    gcash_ref     TEXT,
    payer_number  TEXT,
    created_at    INTEGER NOT NULL,
    expires_at    INTEGER NOT NULL,
    paid_at       INTEGER
);

CREATE UNIQUE INDEX idx_payments_pending_amount
    ON payments (pay_centavos) WHERE status = 'pending';

-- A GCash reference number can only ever settle one payment.
CREATE UNIQUE INDEX idx_payments_ref
    ON payments (gcash_ref) WHERE gcash_ref IS NOT NULL;

-- Every webhook we receive, with what we decided to do with it.
-- Invaluable while testing: you can see the real SMS text and why it did or
-- did not match.
CREATE TABLE sms_log (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id       TEXT,
    sender           TEXT NOT NULL DEFAULT '',
    body             TEXT NOT NULL DEFAULT '',
    raw              TEXT NOT NULL DEFAULT '',
    sig_ok           INTEGER NOT NULL DEFAULT 0,
    amount_centavos  INTEGER,
    ref              TEXT,
    result           TEXT NOT NULL DEFAULT 'received',
    note             TEXT NOT NULL DEFAULT '',
    payment_id       TEXT,
    created_at       INTEGER NOT NULL
);

-- SMSGate retries failed deliveries; the same messageId must not be processed twice.
CREATE UNIQUE INDEX idx_sms_message_id
    ON sms_log (message_id) WHERE message_id IS NOT NULL;
