-- Documents the employer requires (comma-separated ids from jobs.RequirementDocs),
-- and the files the crew member attached when accepting ({"nbi":"/uploads/..."}).
ALTER TABLE jobs ADD COLUMN required_docs TEXT NOT NULL DEFAULT '';
ALTER TABLE job_assignments ADD COLUMN requirements TEXT NOT NULL DEFAULT '{}';
