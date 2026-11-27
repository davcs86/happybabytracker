CREATE TABLE admins (
    id         TEXT PRIMARY KEY,
    username   TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE admin_identities (
    admin_id          TEXT NOT NULL REFERENCES admins (id) ON DELETE CASCADE,
    provider          TEXT NOT NULL,
    subject           TEXT NOT NULL,
    secret_hash       TEXT,
    secret_changed_at INTEGER,
    PRIMARY KEY (provider, subject)
) STRICT;

CREATE UNIQUE INDEX identities_one_oidc_per_issuer ON identities (parent_id, provider) WHERE provider <> 'local';
CREATE UNIQUE INDEX admin_identities_one_oidc_per_issuer ON admin_identities (admin_id, provider) WHERE provider <> 'local';

CREATE TABLE sessions_new (
    id           TEXT PRIMARY KEY,
    token_hash   BLOB NOT NULL UNIQUE,
    kind         TEXT NOT NULL CHECK (kind IN ('admin', 'parent')),
    parent_id    TEXT REFERENCES parents (id) ON DELETE CASCADE,
    admin_id     TEXT REFERENCES admins (id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    expires_at   INTEGER NOT NULL,
    device_label TEXT,
    reauth_at    INTEGER,
    CHECK ((kind = 'admin' AND admin_id IS NOT NULL AND parent_id IS NULL)
        OR (kind = 'parent' AND parent_id IS NOT NULL AND admin_id IS NULL))
) STRICT;
INSERT INTO sessions_new (id, token_hash, kind, parent_id, created_at, last_seen_at, expires_at, device_label)
    SELECT id, token_hash, kind, parent_id, created_at, last_seen_at, expires_at, device_label
    FROM sessions WHERE kind = 'parent';
DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

DELETE FROM meta WHERE key = 'admin_fingerprint';
