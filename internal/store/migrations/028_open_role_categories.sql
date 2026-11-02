-- Categories are validated against the Go catalog (jobs.Catalog); a fixed list here
-- would need a migration for every new category.
ALTER TABLE job_roles DROP CONSTRAINT IF EXISTS job_roles_category_check;
