
CREATE TABLE IF NOT EXISTS job_roles (
    id        TEXT PRIMARY KEY,
    category  TEXT NOT NULL
              CHECK (category IN ('kitchen', 'delivery', 'helpers')),
    label     TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1
              CHECK (is_active IN (0, 1)),
    UNIQUE (category, label)
);

CREATE TABLE IF NOT EXISTS applicant_roles (
    applicant_id INTEGER NOT NULL
                  REFERENCES applicant_profiles(user_id)
                  ON DELETE CASCADE,
    role_id      TEXT NOT NULL
                  REFERENCES job_roles(id),
    is_primary   INTEGER NOT NULL DEFAULT 0
                  CHECK (is_primary IN (0, 1)),
    created_at   INTEGER NOT NULL DEFAULT (unixepoch()),

    PRIMARY KEY (applicant_id, role_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_applicant_primary_role
    ON applicant_roles(applicant_id)
    WHERE is_primary = 1;

CREATE INDEX IF NOT EXISTS idx_applicant_roles_role
    ON applicant_roles(role_id);

-- Starter role catalog
INSERT INTO job_roles (id, category, label) VALUES
    ('chef', 'kitchen', 'Chef'),
    ('sous-chef', 'kitchen', 'Sous Chef'),
    ('line-cook', 'kitchen', 'Line Cook'),
    ('prep-assistant', 'kitchen', 'Prep Assistant'),
    ('kitchen-helper', 'kitchen', 'Kitchen Helper'),
    ('delivery-rider', 'delivery', 'Delivery Rider'),
    ('delivery-driver', 'delivery', 'Delivery Driver'),
    ('general-helper', 'helpers', 'General Helper')
ON CONFLICT(id) DO NOTHING;

