ALTER TABLE visits ADD COLUMN provider_npi TEXT CHECK (provider_npi IS NULL OR
    (provider IS NOT NULL AND length(provider_npi) = 10 AND provider_npi NOT GLOB '*[^0-9]*'));
