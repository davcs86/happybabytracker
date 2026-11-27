// Package store is the SQLite persistence layer: one writer connection, a read-only pool.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	_ "modernc.org/sqlite"
)

const dbFile = "happybabytracker.db"

var (
	ErrNotFound     = errors.New("store: not found")
	ErrConflict     = errors.New("store: conflict")
	ErrDuplicate    = errors.New("store: duplicate")
	ErrLastGuardian = errors.New("store: last guardian")
	ErrForbidden    = errors.New("store: forbidden")
	// ErrNotLinked is an OIDC pair that names no principal, or names a disabled parent.
	ErrNotLinked = errors.New("store: not linked")
	// ErrAlreadyLinked is an OIDC pair held by anyone, or a principal that already has an identity for the issuer.
	ErrAlreadyLinked = errors.New("store: already linked")
	// ErrInvalidContents is a bottle whose contents are empty, repeat a name or name an unknown one.
	ErrInvalidContents = errors.New("store: invalid bottle contents")
	// ErrInvalidSides is a pumping session whose sides repeat a name, name an unknown one or imply another Side.
	ErrInvalidSides = errors.New("store: invalid pumping sides")
	// ErrInvalidAntigens is an administration whose antigen list is empty or repeats one.
	ErrInvalidAntigens = errors.New("store: invalid antigens")
	// ErrInvalidSchedule is a course whose schedule the engine refuses.
	ErrInvalidSchedule = errors.New("store: invalid schedule")
	// ErrInvalidMedication is a dose naming a course that is not the kid's.
	ErrInvalidMedication = errors.New("store: invalid medication")
	// ErrAlreadyRunning is a timer start while the kid already has one running in that table.
	ErrAlreadyRunning = errors.New("store: already running")
	// ErrNotRunning is an end or side switch on a finished session.
	ErrNotRunning = errors.New("store: not running")
	// ErrInvalidSession is a timer write that breaks a session invariant (model's InvalidField, or a re-opened session).
	ErrInvalidSession = errors.New("store: invalid session")
	// ErrSummaryRange is a Summaries range longer than constants.SummaryMaxDays days.
	ErrSummaryRange = errors.New("store: summary range too long")
	// ErrHasOutcome is a cancel, or a move into the future, of a visit with notes or linked rows (visits spec 5.3).
	ErrHasOutcome = errors.New("store: visit has an outcome")
	// ErrNotPast is an outcome on a visit that is upcoming or cancelled.
	ErrNotPast = errors.New("store: visit is not past")
	// ErrStale is an outcome sent with a revision the visit no longer has.
	ErrStale = errors.New("store: visit revision is stale")
	// ErrParentNotFound is an ErrNotFound naming the grant's target, not the kid or the actor.
	ErrParentNotFound = fmt.Errorf("%w: parent", ErrNotFound)
)

// Cursor names the last row of a page, by unix seconds and id; the next page is strictly older.
type Cursor struct {
	At int64
	ID string
}

// ListQuery selects a kid's rows with a time in [From, To), newest first; Limit is at least 1.
type ListQuery struct {
	From, To time.Time
	Limit    int
	After    *Cursor
}

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
