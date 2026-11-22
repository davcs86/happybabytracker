ALTER TABLE kids ADD COLUMN growth_source TEXT NOT NULL DEFAULT 'who' CHECK (growth_source IN ('who', 'cdc'));

CREATE TABLE growth_measurements (
    id           TEXT PRIMARY KEY,
    kid_id       TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    measured_on  TEXT NOT NULL,
    weight_g     INTEGER,
    weight_oz_h  INTEGER,
    weight_typed TEXT CHECK (weight_typed IN ('kg', 'lb')),
    length_mm    INTEGER,
    length_in_h  INTEGER,
    length_typed TEXT CHECK (length_typed IN ('cm', 'in')),
    head_mm      INTEGER,
    head_in_h    INTEGER,
    head_typed   TEXT CHECK (head_typed IN ('cm', 'in')),
    note         TEXT,
    created_by   TEXT NOT NULL REFERENCES parents (id),
    updated_by   TEXT NOT NULL REFERENCES parents (id),
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL,
    CHECK (weight_g IS NOT NULL OR length_mm IS NOT NULL OR head_mm IS NOT NULL),
    CHECK ((weight_g IS NULL) = (weight_oz_h IS NULL) AND (weight_g IS NULL) = (weight_typed IS NULL)),
    CHECK ((length_mm IS NULL) = (length_in_h IS NULL) AND (length_mm IS NULL) = (length_typed IS NULL)),
    CHECK ((head_mm IS NULL) = (head_in_h IS NULL) AND (head_mm IS NULL) = (head_typed IS NULL))
) STRICT;

CREATE INDEX growth_measurements_kid_date ON growth_measurements (kid_id, measured_on, id);
