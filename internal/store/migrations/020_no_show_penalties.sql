ALTER TABLE applicant_profiles ADD COLUMN no_show_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE applicant_profiles ADD COLUMN deactivated_until INTEGER NOT NULL DEFAULT 0;
ALTER TABLE applicant_profiles ADD COLUMN served_temporary_deactivation INTEGER NOT NULL DEFAULT 0;
ALTER TABLE applicant_profiles ADD COLUMN is_permanently_deactivated INTEGER NOT NULL DEFAULT 0;
