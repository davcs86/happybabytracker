package store

import (
	"context"
	"database/sql"
	"errors"
)

// MetaGet reads a meta value; ok is false when the key is absent.
func (s *Store) MetaGet(ctx context.Context, key string) (value string, ok bool, err error) {
	err = s.r.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// MetaSet upserts a meta value.
func (s *Store) MetaSet(ctx context.Context, key, value string) error {
	_, err := s.w.ExecContext(ctx,
		`INSERT INTO meta(key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
