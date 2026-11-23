ALTER TABLE kids ADD COLUMN reminder_overdue_minutes INTEGER NOT NULL DEFAULT 60 CHECK (reminder_overdue_minutes BETWEEN 5 AND 1440);

CREATE TABLE symptoms (
    id         TEXT PRIMARY KEY,
    kid_id     TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note       TEXT,
    created_by TEXT NOT NULL REFERENCES parents (id),
    updated_by TEXT NOT NULL REFERENCES parents (id),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    at         INTEGER NOT NULL,
    symptom    TEXT NOT NULL CHECK (symptom IN ('congestion', 'cough', 'fever', 'vomiting', 'diarrhea', 'constipation',
        'rash', 'ear_pain', 'sore_throat', 'poor_appetite', 'unusually_sleepy', 'other')),
    severity   TEXT CHECK (severity IN ('mild', 'moderate', 'severe')),
    other_text TEXT,
    temp_c_h   INTEGER,
    temp_f_h   INTEGER,
    temp_typed TEXT CHECK (temp_typed IN ('c', 'f')),
    method     TEXT CHECK (method IN ('oral', 'rectal', 'axillary', 'ear', 'forehead', 'other')),
    CHECK ((temp_c_h IS NULL) = (temp_f_h IS NULL) AND (temp_c_h IS NULL) = (temp_typed IS NULL)),
    CHECK ((symptom = 'fever') = (temp_c_h IS NOT NULL AND method IS NOT NULL)),
    CHECK ((temp_c_h IS NULL) = (method IS NULL)),
    CHECK ((symptom = 'other') = (other_text IS NOT NULL))
) STRICT;

CREATE INDEX symptoms_kid_at ON symptoms (kid_id, at);

CREATE TABLE medications (
    id             TEXT PRIMARY KEY,
    kid_id         TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note           TEXT,
    created_by     TEXT NOT NULL REFERENCES parents (id),
    updated_by     TEXT NOT NULL REFERENCES parents (id),
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL,
    name           TEXT NOT NULL,
    dose           TEXT NOT NULL,
    schedule_kind  TEXT NOT NULL CHECK (schedule_kind IN ('none', 'times', 'every')),
    schedule_times TEXT,
    every_hours    INTEGER CHECK (every_hours BETWEEN 1 AND 168),
    start_at       INTEGER NOT NULL,
    end_on         TEXT,
    active         INTEGER NOT NULL CHECK (active IN (0, 1)),
    CHECK ((schedule_kind = 'times') = (schedule_times IS NOT NULL)),
    CHECK ((schedule_kind = 'every') = (every_hours IS NOT NULL))
) STRICT;

CREATE INDEX medications_kid_start ON medications (kid_id, start_at);

CREATE TABLE medication_doses (
    id            TEXT PRIMARY KEY,
    kid_id        TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    note          TEXT,
    created_by    TEXT NOT NULL REFERENCES parents (id),
    updated_by    TEXT NOT NULL REFERENCES parents (id),
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    medication_id TEXT NOT NULL REFERENCES medications (id) ON DELETE CASCADE,
    given_at      INTEGER NOT NULL,
    amount        TEXT
) STRICT;

CREATE INDEX medication_doses_kid_given ON medication_doses (kid_id, given_at);
CREATE INDEX medication_doses_medication_given ON medication_doses (medication_id, given_at);

CREATE TABLE reminders (
    id            TEXT PRIMARY KEY,
    kid_id        TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    medication_id TEXT NOT NULL REFERENCES medications (id) ON DELETE CASCADE,
    due_at        INTEGER NOT NULL,
    state         TEXT NOT NULL CHECK (state IN ('pending', 'done', 'expired', 'cancelled')),
    created_at    INTEGER NOT NULL,
    UNIQUE (medication_id, due_at)
) STRICT;

CREATE INDEX reminders_state_due ON reminders (state, due_at);

CREATE TABLE reminder_deliveries (
    id              TEXT PRIMARY KEY,
    reminder_id     TEXT NOT NULL REFERENCES reminders (id) ON DELETE CASCADE,
    channel         TEXT NOT NULL CHECK (channel IN ('push', 'ntfy')),
    target_id       TEXT NOT NULL,
    parent_id       TEXT NOT NULL REFERENCES parents (id) ON DELETE CASCADE,
    state           TEXT NOT NULL CHECK (state IN ('pending', 'sending', 'sent', 'gone', 'failed', 'expired', 'unconfirmed', 'cancelled')),
    attempts        INTEGER NOT NULL DEFAULT 0,
    next_attempt_at INTEGER NOT NULL,
    settled_at      INTEGER,
    UNIQUE (reminder_id, channel, target_id)
) STRICT;

CREATE INDEX reminder_deliveries_state_settled ON reminder_deliveries (state, settled_at);

CREATE TABLE push_subscriptions (
    id         TEXT PRIMARY KEY,
    parent_id  TEXT NOT NULL REFERENCES parents (id) ON DELETE CASCADE,
    endpoint   TEXT NOT NULL UNIQUE,
    p256dh     TEXT NOT NULL,
    auth       TEXT NOT NULL,
    label      TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE push_kid_optins (
    subscription_id TEXT NOT NULL REFERENCES push_subscriptions (id) ON DELETE CASCADE,
    kid_id          TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    PRIMARY KEY (subscription_id, kid_id)
) STRICT;

CREATE TABLE ntfy_topics (
    parent_id  TEXT PRIMARY KEY REFERENCES parents (id) ON DELETE CASCADE,
    topic      TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL
) STRICT;

CREATE TABLE ntfy_kid_optins (
    parent_id TEXT NOT NULL REFERENCES ntfy_topics (parent_id) ON DELETE CASCADE,
    kid_id    TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    PRIMARY KEY (parent_id, kid_id)
) STRICT;
