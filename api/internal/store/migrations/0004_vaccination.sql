ALTER TABLE kids ADD COLUMN vaccine_schema TEXT;

CREATE TABLE vaccine_doses (
    id             TEXT PRIMARY KEY,
    kid_id         TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    administration TEXT NOT NULL,
    given_on       TEXT NOT NULL CHECK (given_on GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND date(given_on) = given_on),
    antigen        TEXT NOT NULL,
    lot            TEXT CHECK (lot <> ''),
    product        TEXT CHECK (product <> ''),
    note           TEXT CHECK (note <> ''),
    created_by     TEXT NOT NULL REFERENCES parents (id),
    updated_by     TEXT NOT NULL REFERENCES parents (id),
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL,
    UNIQUE (administration, antigen)
) STRICT;

CREATE INDEX vaccine_doses_kid_date ON vaccine_doses (kid_id, given_on);
CREATE INDEX vaccine_doses_kid_administration ON vaccine_doses (kid_id, administration);

CREATE TABLE vaccine_dismissals (
    kid_id     TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    schema     TEXT NOT NULL,
    line       TEXT NOT NULL,
    reason     TEXT NOT NULL CHECK (reason IN ('not_needed', 'given_unknown')),
    created_by TEXT NOT NULL REFERENCES parents (id),
    created_at INTEGER NOT NULL,
    PRIMARY KEY (kid_id, schema, line)
) STRICT;
