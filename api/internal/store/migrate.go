package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	var ms []migration
	for _, e := range entries {
		prefix, _, ok := strings.Cut(e.Name(), "_")
		v, convErr := strconv.Atoi(prefix)
		if !ok || convErr != nil {
			return nil, fmt.Errorf("store: migration %q is not NNNN_name.sql", e.Name())
		}
		body, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, err
		}
		ms = append(ms, migration{version: v, name: e.Name(), sql: string(body)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].version < ms[j].version })
	for i, m := range ms {
		if m.version != i+1 {
			return nil, fmt.Errorf("store: migration %s breaks the 1..n sequence", m.name)
		}
	}
	return ms, nil
}

// migrate applies every embedded migration newer than meta.schema_version, one transaction each.
func migrate(ctx context.Context, w *sql.DB) error {
	ms, err := loadMigrations()
	if err != nil {
		return err
	}
	current, err := schemaVersion(ctx, w)
	if err != nil {
		return err
	}
	if current > len(ms) {
		return fmt.Errorf("store: database schema version %d is newer than this binary (%d)", current, len(ms))
	}
	for _, m := range ms[current:] {
		if err := apply(ctx, w, m); err != nil {
			return fmt.Errorf("store: migration %s: %w", m.name, err)
		}
	}
	return nil
}

func schemaVersion(ctx context.Context, w *sql.DB) (int, error) {
	var n int
	if err := w.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='meta'`).Scan(&n); err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, nil
	}
	var v string
	err := w.QueryRowContext(ctx, `SELECT value FROM meta WHERE key='schema_version'`).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(v)
}

func apply(ctx context.Context, w *sql.DB, m migration) error {
	tx, err := w.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO meta(key, value) VALUES ('schema_version', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		strconv.Itoa(m.version)); err != nil {
		return err
	}
	return tx.Commit()
}
