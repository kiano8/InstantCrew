ALTER TABLE applicant_profiles ADD COLUMN IF NOT EXISTS no_show_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE applicant_profiles ADD COLUMN IF NOT EXISTS deactivated_until BIGINT;
ALTER TABLE applicant_profiles ADD COLUMN IF NOT EXISTS served_temporary_deactivation BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE applicant_profiles ADD COLUMN IF NOT EXISTS is_permanently_deactivated BOOLEAN NOT NULL DEFAULT false;

