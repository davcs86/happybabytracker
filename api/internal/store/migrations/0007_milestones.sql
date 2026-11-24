CREATE TABLE milestones_reached (
    id           TEXT PRIMARY KEY,
    kid_id       TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    catalog_item TEXT CHECK (catalog_item <> ''),
    custom_text  TEXT CHECK (custom_text <> ''),
    reached_on   TEXT NOT NULL CHECK (reached_on GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND date(reached_on) = reached_on),
    note         TEXT CHECK (note <> ''),
    created_by   TEXT NOT NULL REFERENCES parents (id),
    updated_by   TEXT NOT NULL REFERENCES parents (id),
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL,
    CHECK ((catalog_item IS NULL) <> (custom_text IS NULL))
) STRICT;
CREATE UNIQUE INDEX milestones_reached_kid_item ON milestones_reached (kid_id, catalog_item) WHERE catalog_item IS NOT NULL;
CREATE INDEX milestones_reached_kid_date ON milestones_reached (kid_id, reached_on, id);
