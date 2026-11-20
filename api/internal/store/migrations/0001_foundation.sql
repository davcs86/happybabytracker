CREATE TABLE meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;

CREATE TABLE parents (
    id           TEXT PRIMARY KEY,
    username     TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    disabled     INTEGER NOT NULL DEFAULT 0,
    unit_weight  TEXT NOT NULL DEFAULT 'kg'   CHECK (unit_weight IN ('kg', 'lb')),
    unit_length  TEXT NOT NULL DEFAULT 'cm'   CHECK (unit_length IN ('cm', 'in')),
    unit_volume  TEXT NOT NULL DEFAULT 'ml'   CHECK (unit_volume IN ('ml', 'floz')),
    unit_temp    TEXT NOT NULL DEFAULT 'c'    CHECK (unit_temp IN ('c', 'f')),
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
) STRICT;

CREATE TABLE identities (
    parent_id   TEXT NOT NULL REFERENCES parents (id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,
    subject     TEXT NOT NULL,
    secret_hash TEXT,
    PRIMARY KEY (provider, subject)
) STRICT;

CREATE TABLE sessions (
    id           TEXT PRIMARY KEY,
    token_hash   BLOB NOT NULL UNIQUE,
    kind         TEXT NOT NULL CHECK (kind IN ('admin', 'parent')),
    parent_id    TEXT REFERENCES parents (id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    expires_at   INTEGER NOT NULL,
    CHECK ((kind = 'admin') = (parent_id IS NULL))
) STRICT;

CREATE TABLE one_time_tokens (
    token_hash BLOB PRIMARY KEY,
    kind       TEXT NOT NULL CHECK (kind IN ('invite', 'reset')),
    parent_id  TEXT NOT NULL REFERENCES parents (id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL,
    used_at    INTEGER
) STRICT;

CREATE TABLE kids (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    birth_date      TEXT NOT NULL,
    sex             TEXT NOT NULL CHECK (sex IN ('female', 'male')),
    gestation_weeks INTEGER,
    gestation_days  INTEGER,
    time_zone       TEXT NOT NULL,
    created_by      TEXT NOT NULL REFERENCES parents (id),
    updated_by      TEXT NOT NULL REFERENCES parents (id),
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
) STRICT;

CREATE TABLE kid_access (
    kid_id     TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    parent_id  TEXT NOT NULL REFERENCES parents (id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('guardian', 'caregiver')),
    created_at INTEGER NOT NULL,
    PRIMARY KEY (kid_id, parent_id)
) STRICT;

CREATE INDEX kid_access_parent ON kid_access (parent_id);
