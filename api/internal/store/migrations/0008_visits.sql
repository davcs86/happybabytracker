CREATE TABLE visits (
    id            TEXT PRIMARY KEY,
    kid_id        TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note          TEXT CHECK (note <> ''),
    created_by    TEXT NOT NULL REFERENCES parents (id),
    updated_by    TEXT NOT NULL REFERENCES parents (id),
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    starts_at     INTEGER NOT NULL,
    duration_min  INTEGER NOT NULL DEFAULT 30 CHECK (duration_min BETWEEN 5 AND 720),
    kind          TEXT NOT NULL CHECK (kind IN ('well_child', 'sick', 'specialist', 'dental', 'emergency', 'other')),
    provider      TEXT CHECK (provider <> ''),
    location      TEXT CHECK (location <> ''),
    outcome_notes TEXT CHECK (outcome_notes <> ''),
    cancelled_at  INTEGER,
    revision      INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0)
) STRICT;
CREATE INDEX visits_kid_starts ON visits (kid_id, starts_at, id);

ALTER TABLE growth_measurements ADD COLUMN visit_id TEXT REFERENCES visits (id) ON DELETE SET NULL;
ALTER TABLE vaccine_doses ADD COLUMN visit_id TEXT REFERENCES visits (id) ON DELETE SET NULL;
ALTER TABLE medications ADD COLUMN visit_id TEXT REFERENCES visits (id) ON DELETE SET NULL;
CREATE INDEX growth_measurements_visit ON growth_measurements (visit_id) WHERE visit_id IS NOT NULL;
CREATE INDEX vaccine_doses_visit ON vaccine_doses (visit_id) WHERE visit_id IS NOT NULL;
CREATE INDEX medications_visit ON medications (visit_id) WHERE visit_id IS NOT NULL;

CREATE TABLE kid_modules_new (
    kid_id     TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    module     TEXT NOT NULL CHECK (module IN (
        'bottles', 'nursing', 'pumping', 'diapers', 'sleep', 'feeding',
        'milestones', 'growth', 'vaccination', 'sickness', 'medication', 'visits')),
    enabled    INTEGER NOT NULL CHECK (enabled IN (0, 1)),
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (kid_id, module)
) STRICT;
INSERT INTO kid_modules_new (kid_id, module, enabled, updated_at)
SELECT kid_id, module, enabled, updated_at FROM kid_modules;
DROP TABLE kid_modules;
ALTER TABLE kid_modules_new RENAME TO kid_modules;
