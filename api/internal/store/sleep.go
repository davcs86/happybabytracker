package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

const sleepColumns = entryColumns + `, started_at, ended_at`

func scanSleep(row rowScanner) (model.SleepSession, error) {
	var (
		v         model.SleepSession
		startedAt int64
		endedAt   sql.NullInt64
	)
	e, err := scanEntry(row, &startedAt, &endedAt)
	if err != nil {
		return model.SleepSession{}, err
	}
	v.Entry, v.StartedAt, v.EndedAt = e, unixTime(startedAt), nullableTime(endedAt)
	return v, nil
}

func insertSleep(ctx context.Context, tx *sql.Tx, v model.SleepSession) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO sleep_sessions(`+sleepColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.KidID, v.Note, v.CreatedBy, v.UpdatedBy, v.CreatedAt.Unix(), v.UpdatedAt.Unix(),
		v.StartedAt.Unix(), nullableUnix(v.EndedAt))
	return runningConflict(err)
}

// CreateSleep stores v for kidID as actorID, who needs an active grant (else ErrNotFound). A nil EndedAt starts the
// timer (ErrAlreadyRunning when one runs); ErrInvalidSession when v breaks a session invariant.
func (s *Store) CreateSleep(ctx context.Context, kidID, actorID string, v model.SleepSession, now time.Time) (model.SleepSession, error) {
	v.ID, v.KidID, v.CreatedBy, v.UpdatedBy = model.NewID(), kidID, actorID, actorID
	v.CreatedAt, v.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	v.StartedAt, v.EndedAt = unixTime(v.StartedAt.Unix()), wholeSeconds(v.EndedAt)
	if v.InvalidField() != "" {
		return model.SleepSession{}, ErrInvalidSession
	}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		if v.EndedAt == nil {
			if err := checkNoneRunning(ctx, tx, "sleep_sessions", kidID); err != nil {
				return err
			}
		}
		return insertSleep(ctx, tx, v)
	})
	if err != nil {
		return model.SleepSession{}, err
	}
	return v, nil
}

// Sleep is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Sleep(ctx context.Context, kidID, id string) (model.SleepSession, error) {
	return scanSleep(s.r.QueryRowContext(ctx, `SELECT `+sleepColumns+` FROM sleep_sessions WHERE kid_id = ? AND id = ?`, kidID, id))
}

// RunningSleep is the kid's running session, nil when none runs.
func (s *Store) RunningSleep(ctx context.Context, kidID string) (*model.SleepSession, error) {
	v, err := scanSleep(s.r.QueryRowContext(ctx, `SELECT `+sleepColumns+` FROM sleep_sessions WHERE kid_id = ? AND ended_at IS NULL`, kidID))
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListSleep returns the kid's sessions within q, newest first by (started_at, id); never nil.
func (s *Store) ListSleep(ctx context.Context, kidID string, q ListQuery) ([]model.SleepSession, error) {
	return listEntries(ctx, s.r, `SELECT `+sleepColumns+` FROM sleep_sessions`, "started_at", kidID, q, scanSleep)
}

// UpdateSleep reads the kid's session id inside the write transaction, applies edit (whose error aborts the write)
// and stores started_at, ended_at and note; ErrInvalidSession when the result breaks an invariant or re-opens it.
func (s *Store) UpdateSleep(ctx context.Context, kidID, actorID, id string, now time.Time, edit func(model.SleepSession) (model.SleepSession, error)) (model.SleepSession, error) {
	var out model.SleepSession
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		stored, err := scanSleep(tx.QueryRowContext(ctx, `SELECT `+sleepColumns+` FROM sleep_sessions WHERE kid_id = ? AND id = ?`, kidID, id))
		if err != nil {
			return err
		}
		v, err := edit(stored)
		if err != nil {
			return err
		}
		v.Entry = model.Entry{
			ID: stored.ID, KidID: stored.KidID, CreatedBy: stored.CreatedBy, CreatedAt: stored.CreatedAt,
			UpdatedBy: actorID, UpdatedAt: unixTime(now.Unix()), Note: v.Note,
		}
		v.StartedAt, v.EndedAt = unixTime(v.StartedAt.Unix()), wholeSeconds(v.EndedAt)
		if v.InvalidField() != "" || (stored.EndedAt != nil && v.EndedAt == nil) {
			return ErrInvalidSession
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE sleep_sessions SET started_at = ?, ended_at = ?, note = ?, updated_by = ?, updated_at = ? WHERE kid_id = ? AND id = ?`,
			v.StartedAt.Unix(), nullableUnix(v.EndedAt), v.Note, actorID, v.UpdatedAt.Unix(), kidID, id)
		if err := oneRow(res, err); err != nil {
			return err
		}
		out = v
		return nil
	})
	if err != nil {
		return model.SleepSession{}, err
	}
	return out, nil
}

// EndSleep ends the running session id at at: ErrNotRunning when it is finished, ErrInvalidSession when at precedes
// its start or the session would exceed MaxSessionDuration.
func (s *Store) EndSleep(ctx context.Context, kidID, actorID, id string, at, now time.Time) (model.SleepSession, error) {
	return s.UpdateSleep(ctx, kidID, actorID, id, now, func(v model.SleepSession) (model.SleepSession, error) {
		if v.EndedAt != nil {
			return v, ErrNotRunning
		}
		v.EndedAt = &at
		return v, nil
	})
}

// DeleteSleep is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteSleep(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "sleep_sessions", kidID, actorID, id)
}
