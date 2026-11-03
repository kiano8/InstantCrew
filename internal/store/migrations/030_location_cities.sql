-- Undo 029: matching is by exact city again, so put rows back on the plain city names.
-- Barangays ("Lahug, Cebu City") collapse to their city.
UPDATE jobs SET city = 'Cebu City' WHERE lower(trim(city)) ~ ', cebu city$';
UPDATE applicant_profiles SET city = 'Cebu City' WHERE lower(trim(city)) ~ ', cebu city$';

UPDATE jobs SET city = CASE lower(trim(city))
    WHEN 'cebu' THEN 'Cebu City'
    WHEN 'metro manila' THEN 'Manila'
    WHEN 'mandaue, cebu' THEN 'Mandaue'
    WHEN 'lapu-lapu, cebu' THEN 'Lapu-Lapu City'
    WHEN 'makati, metro manila' THEN 'Makati City'
    ELSE regexp_replace(city, ',\s*Metro Manila\s*$', '', 'i') END;

UPDATE applicant_profiles SET city = CASE lower(trim(city))
    WHEN 'cebu' THEN 'Cebu City'
    WHEN 'metro manila' THEN 'Manila'
    WHEN 'mandaue, cebu' THEN 'Mandaue'
    WHEN 'lapu-lapu, cebu' THEN 'Lapu-Lapu City'
    WHEN 'makati, metro manila' THEN 'Makati City'
    ELSE regexp_replace(city, ',\s*Metro Manila\s*$', '', 'i') END;
