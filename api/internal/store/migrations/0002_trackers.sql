CREATE TABLE bottle_feeds (
    id              TEXT PRIMARY KEY,
    kid_id          TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note            TEXT,
    created_by      TEXT NOT NULL REFERENCES parents (id),
    updated_by      TEXT NOT NULL REFERENCES parents (id),
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    at              INTEGER NOT NULL,
    has_breast_milk INTEGER NOT NULL CHECK (has_breast_milk IN (0, 1)),
    has_formula     INTEGER NOT NULL CHECK (has_formula IN (0, 1)),
    has_cow_milk    INTEGER NOT NULL CHECK (has_cow_milk IN (0, 1)),
    has_other       INTEGER NOT NULL CHECK (has_other IN (0, 1)),
    volume_ml       INTEGER NOT NULL,
    volume_floz_h   INTEGER NOT NULL,
    volume_typed    TEXT NOT NULL CHECK (volume_typed IN ('ml', 'floz')),
    CHECK (has_breast_milk + has_formula + has_cow_milk + has_other >= 1)
) STRICT;

CREATE INDEX bottle_feeds_kid_at ON bottle_feeds (kid_id, at);

CREATE TABLE diapers (
    id         TEXT PRIMARY KEY,
    kid_id     TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note       TEXT,
    created_by TEXT NOT NULL REFERENCES parents (id),
    updated_by TEXT NOT NULL REFERENCES parents (id),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    at         INTEGER NOT NULL,
    kind       TEXT NOT NULL CHECK (kind IN ('wet', 'dirty', 'both', 'dry'))
) STRICT;

CREATE INDEX diapers_kid_at ON diapers (kid_id, at);

CREATE TABLE meals (
    id         TEXT PRIMARY KEY,
    kid_id     TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note       TEXT,
    created_by TEXT NOT NULL REFERENCES parents (id),
    updated_by TEXT NOT NULL REFERENCES parents (id),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    at         INTEGER NOT NULL,
    what       TEXT NOT NULL,
    reaction   TEXT
) STRICT;

CREATE INDEX meals_kid_at ON meals (kid_id, at);

CREATE TABLE pumping_sessions (
    id               TEXT PRIMARY KEY,
    kid_id           TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note             TEXT,
    created_by       TEXT NOT NULL REFERENCES parents (id),
    updated_by       TEXT NOT NULL REFERENCES parents (id),
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL,
    started_at       INTEGER NOT NULL,
    duration_seconds INTEGER NOT NULL,
    volume_ml        INTEGER NOT NULL,
    volume_floz_h    INTEGER NOT NULL,
    volume_typed     TEXT NOT NULL CHECK (volume_typed IN ('ml', 'floz')),
    side             TEXT NOT NULL CHECK (side IN ('left', 'right', 'both'))
) STRICT;

CREATE INDEX pumping_sessions_kid_started ON pumping_sessions (kid_id, started_at);

CREATE TABLE sleep_sessions (
    id         TEXT PRIMARY KEY,
    kid_id     TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note       TEXT,
    created_by TEXT NOT NULL REFERENCES parents (id),
    updated_by TEXT NOT NULL REFERENCES parents (id),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    started_at INTEGER NOT NULL,
    ended_at   INTEGER,
    CHECK (ended_at IS NULL OR ended_at >= started_at)
) STRICT;

CREATE INDEX sleep_sessions_kid_started ON sleep_sessions (kid_id, started_at);
CREATE UNIQUE INDEX sleep_one_running ON sleep_sessions (kid_id) WHERE ended_at IS NULL;

CREATE TABLE nursing_sessions (
    id            TEXT PRIMARY KEY,
    kid_id        TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note          TEXT,
    created_by    TEXT NOT NULL REFERENCES parents (id),
    updated_by    TEXT NOT NULL REFERENCES parents (id),
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    started_at    INTEGER NOT NULL,
    ended_at      INTEGER,
    left_seconds  INTEGER NOT NULL DEFAULT 0 CHECK (left_seconds >= 0),
    right_seconds INTEGER NOT NULL DEFAULT 0 CHECK (right_seconds >= 0),
    side          TEXT CHECK (side IN ('left', 'right')),
    side_since    INTEGER,
    CHECK (ended_at IS NULL OR ended_at >= started_at)
) STRICT;

CREATE INDEX nursing_sessions_kid_started ON nursing_sessions (kid_id, started_at);
CREATE UNIQUE INDEX nursing_one_running ON nursing_sessions (kid_id) WHERE ended_at IS NULL;

CREATE TABLE kid_modules (
    kid_id     TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    module     TEXT NOT NULL CHECK (module IN (
        'bottles', 'nursing', 'pumping', 'diapers', 'sleep', 'feeding',
        'milestones', 'growth', 'vaccination', 'sickness', 'medication')),
    enabled    INTEGER NOT NULL CHECK (enabled IN (0, 1)),
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (kid_id, module)
) STRICT;
