CREATE TABLE bottle_feed_sources (
    bottle_id     TEXT NOT NULL REFERENCES bottle_feeds (id) ON DELETE CASCADE,
    source        TEXT NOT NULL CHECK (source IN ('breast_milk', 'formula', 'cow_milk', 'other')),
    volume_ml     INTEGER NOT NULL,
    volume_floz_h INTEGER NOT NULL,
    volume_typed  TEXT NOT NULL CHECK (volume_typed IN ('ml', 'floz')),
    PRIMARY KEY (bottle_id, source)
) STRICT;

CREATE TABLE nursing_sessions_new (
    id                  TEXT PRIMARY KEY,
    kid_id              TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note                TEXT,
    created_by          TEXT NOT NULL REFERENCES parents (id),
    updated_by          TEXT NOT NULL REFERENCES parents (id),
    created_at          INTEGER NOT NULL,
    updated_at          INTEGER NOT NULL,
    started_at          INTEGER NOT NULL,
    ended_at            INTEGER,
    left_seconds        INTEGER NOT NULL DEFAULT 0 CHECK (left_seconds >= 0),
    right_seconds       INTEGER NOT NULL DEFAULT 0 CHECK (right_seconds >= 0),
    side                TEXT CHECK (side IN ('left', 'right')),
    side_since          INTEGER,
    weight_before_g     INTEGER,
    weight_before_oz_h  INTEGER,
    weight_before_typed TEXT CHECK (weight_before_typed IN ('kg', 'lb')),
    weight_after_g      INTEGER,
    weight_after_oz_h   INTEGER,
    weight_after_typed  TEXT CHECK (weight_after_typed IN ('kg', 'lb')),
    CHECK (ended_at IS NULL OR ended_at >= started_at),
    CHECK ((weight_before_g IS NULL) = (weight_before_oz_h IS NULL) AND (weight_before_g IS NULL) = (weight_before_typed IS NULL)),
    CHECK ((weight_after_g IS NULL) = (weight_after_oz_h IS NULL) AND (weight_after_g IS NULL) = (weight_after_typed IS NULL))
) STRICT;

INSERT INTO nursing_sessions_new (id, kid_id, note, created_by, updated_by, created_at, updated_at,
                                  started_at, ended_at, left_seconds, right_seconds, side, side_since)
SELECT id, kid_id, note, created_by, updated_by, created_at, updated_at,
       started_at, ended_at, left_seconds, right_seconds, side, side_since
FROM nursing_sessions;

DROP TABLE nursing_sessions;
ALTER TABLE nursing_sessions_new RENAME TO nursing_sessions;

CREATE INDEX nursing_sessions_kid_started ON nursing_sessions (kid_id, started_at);
CREATE UNIQUE INDEX nursing_one_running ON nursing_sessions (kid_id) WHERE ended_at IS NULL;
