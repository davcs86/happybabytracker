// Package store is the SQLite persistence layer: one writer connection, a read-only pool.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	_ "modernc.org/sqlite"
)

const dbFile = "happybabytracker.db"

var (
	ErrNotFound     = errors.New("store: not found")
	ErrConflict     = errors.New("store: conflict")
	ErrLastGuardian = errors.New("store: last guardian")
)

type Store struct {
	w, r    *sql.DB
	cleanup string
}

// Open opens (creating if needed) dataDir/happybabytracker.db and applies pending migrations.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("store: create data dir: %w", err)
	}
	path := filepath.Join(dataDir, dbFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("store: create db file: %w", err)
	}
	f.Close()
	// auto_vacuum must precede journal_mode so it applies before any page is written.
	busy := constants.SQLiteBusyTimeout.Milliseconds()
	wdsn := fmt.Sprintf("file:%s?_txlock=immediate&_pragma=auto_vacuum(INCREMENTAL)&_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=secure_delete(ON)", path, busy)
	w, err := sql.Open("sqlite", wdsn)
	if err != nil {
		return nil, err
	}
	w.SetMaxOpenConns(1)
	if err := migrate(context.Background(), w); err != nil {
		w.Close()
		return nil, err
	}
	rdsn := fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(%d)&_pragma=foreign_keys(1)", path, busy)
	r, err := sql.Open("sqlite", rdsn)
	if err != nil {
		w.Close()
		return nil, err
	}
	r.SetMaxOpenConns(constants.ReadPoolMax)
	return &Store{w: w, r: r}, nil
}

// OpenMemory returns a throwaway store backed by a temp dir removed on Close.
func OpenMemory() (*Store, error) {
	dir, err := os.MkdirTemp("", "happybabytracker-store-")
	if err != nil {
		return nil, err
	}
	s, err := Open(dir)
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	s.cleanup = dir
	return s, nil
}

func (s *Store) Close() error {
	err := errors.Join(s.r.Close(), s.w.Close())
	if s.cleanup != "" {
		err = errors.Join(err, os.RemoveAll(s.cleanup))
	}
	return err
}

// Ping runs a trivial query on the read pool.
func (s *Store) Ping(ctx context.Context) error {
	var one int
	return s.r.QueryRowContext(ctx, `SELECT 1`).Scan(&one)
}
