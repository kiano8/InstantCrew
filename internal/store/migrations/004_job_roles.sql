
CREATE TABLE IF NOT EXISTS job_roles (
    id        TEXT PRIMARY KEY,
    category  TEXT NOT NULL
              CHECK (category IN ('kitchen', 'delivery', 'helpers')),
    label     TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    UNIQUE (category, label)
);

CREATE TABLE IF NOT EXISTS applicant_roles (
    applicant_id BIGINT NOT NULL
                  REFERENCES applicant_profiles(user_id)
                  ON DELETE CASCADE,
    role_id      TEXT NOT NULL
                  REFERENCES job_roles(id),
    is_primary   BOOLEAN NOT NULL DEFAULT false,
    created_at   BIGINT NOT NULL DEFAULT extract(epoch from now())::BIGINT,

    PRIMARY KEY (applicant_id, role_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_applicant_primary_role
    ON applicant_roles(applicant_id)
    WHERE is_primary = true;

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

