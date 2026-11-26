ALTER TABLE identities ADD COLUMN secret_changed_at INTEGER;
ALTER TABLE sessions ADD COLUMN device_label TEXT;
