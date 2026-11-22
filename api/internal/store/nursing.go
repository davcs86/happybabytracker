package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

const nursingColumns = entryColumns + `, started_at, ended_at, left_seconds, right_seconds, side, side_since`

func scanNursing(row rowScanner) (model.NursingSession, error) {
	var (
		n                  model.NursingSession
		startedAt          int64
		endedAt, sideSince sql.NullInt64
		side               sql.NullString
	)
	e, err := scanEntry(row, &startedAt, &endedAt, &n.LeftSeconds, &n.RightSeconds, &side, &sideSince)
	if err != nil {
		return model.NursingSession{}, err
	}
	n.Entry, n.StartedAt, n.EndedAt, n.SideSince = e, unixTime(startedAt), nullableTime(endedAt), nullableTime(sideSince)
	if side.Valid {
		n.Side = &side.String
	}
	return n, nil
}

func insertNursing(ctx context.Context, tx *sql.Tx, n model.NursingSession) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO nursing_sessions(`+nursingColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.KidID, n.Note, n.CreatedBy, n.UpdatedBy, n.CreatedAt.Unix(), n.UpdatedAt.Unix(),
		n.StartedAt.Unix(), nullableUnix(n.EndedAt), n.LeftSeconds, n.RightSeconds, n.Side, nullableUnix(n.SideSince))
	return runningConflict(err)
}

// CreateNursing stores n for kidID as actorID, who needs an active grant (else ErrNotFound). A nil EndedAt with a Side
// starts the timer from StartedAt (ErrAlreadyRunning when one runs); ErrInvalidSession when n breaks an invariant.
func (s *Store) CreateNursing(ctx context.Context, kidID, actorID string, n model.NursingSession, now time.Time) (model.NursingSession, error) {
	n.ID, n.KidID, n.CreatedBy, n.UpdatedBy = model.NewID(), kidID, actorID, actorID
	n.CreatedAt, n.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	n.StartedAt, n.EndedAt, n.SideSince = unixTime(n.StartedAt.Unix()), wholeSeconds(n.EndedAt), nil
	if n.EndedAt == nil {
		since := n.StartedAt
		n.SideSince = &since
	}
	if n.InvalidField() != "" {
		return model.NursingSession{}, ErrInvalidSession
	}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		if n.EndedAt == nil {
			if err := checkNoneRunning(ctx, tx, "nursing_sessions", kidID); err != nil {
				return err
			}
		}
		return insertNursing(ctx, tx, n)
	})
	if err != nil {
		return model.NursingSession{}, err
	}
	return n, nil
}

// Nursing is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Nursing(ctx context.Context, kidID, id string) (model.NursingSession, error) {
	return scanNursing(s.r.QueryRowContext(ctx, `SELECT `+nursingColumns+` FROM nursing_sessions WHERE kid_id = ? AND id = ?`, kidID, id))
}

// RunningNursing is the kid's running session, nil when none runs.
func (s *Store) RunningNursing(ctx context.Context, kidID string) (*model.NursingSession, error) {
	n, err := scanNursing(s.r.QueryRowContext(ctx, `SELECT `+nursingColumns+` FROM nursing_sessions WHERE kid_id = ? AND ended_at IS NULL`, kidID))
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// ListNursing returns the kid's sessions within q, newest first by (started_at, id); never nil.
func (s *Store) ListNursing(ctx context.Context, kidID string, q ListQuery) ([]model.NursingSession, error) {
	return listEntries(ctx, s.r, `SELECT `+nursingColumns+` FROM nursing_sessions`, "started_at", kidID, q, scanNursing)
}

// UpdateNursing reads the kid's session id inside the write transaction, applies edit (whose error aborts the write)
// and stores every timer column and the note; ErrInvalidSession when the result breaks an invariant or re-opens it.
func (s *Store) UpdateNursing(ctx context.Context, kidID, actorID, id string, now time.Time, edit func(model.NursingSession) (model.NursingSession, error)) (model.NursingSession, error) {
	var out model.NursingSession
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		stored, err := scanNursing(tx.QueryRowContext(ctx, `SELECT `+nursingColumns+` FROM nursing_sessions WHERE kid_id = ? AND id = ?`, kidID, id))
		if err != nil {
			return err
		}
		n, err := edit(stored)
		if err != nil {
			return err
		}
		n.Entry = model.Entry{
			ID: stored.ID, KidID: stored.KidID, CreatedBy: stored.CreatedBy, CreatedAt: stored.CreatedAt,
			UpdatedBy: actorID, UpdatedAt: unixTime(now.Unix()), Note: n.Note,
		}
		n.StartedAt, n.EndedAt, n.SideSince = unixTime(n.StartedAt.Unix()), wholeSeconds(n.EndedAt), wholeSeconds(n.SideSince)
		if n.InvalidField() != "" || (stored.EndedAt != nil && n.EndedAt == nil) {
			return ErrInvalidSession
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE nursing_sessions SET started_at = ?, ended_at = ?, left_seconds = ?, right_seconds = ?, side = ?, side_since = ?,
			   note = ?, updated_by = ?, updated_at = ?
			 WHERE kid_id = ? AND id = ?`,
			n.StartedAt.Unix(), nullableUnix(n.EndedAt), n.LeftSeconds, n.RightSeconds, n.Side, nullableUnix(n.SideSince),
			n.Note, actorID, n.UpdatedAt.Unix(), kidID, id)
		if err := oneRow(res, err); err != nil {
			return err
		}
		out = n
		return nil
	})
	if err != nil {
		return model.NursingSession{}, err
	}
	return out, nil
}

// SwitchNursingSide credits the side in use up to at and feeds side from at: ErrNotRunning when the session is
// finished, ErrInvalidSession when at precedes side_since or side is unknown.
func (s *Store) SwitchNursingSide(ctx context.Context, kidID, actorID, id, side string, at, now time.Time) (model.NursingSession, error) {
	at = unixTime(at.Unix())
	return s.UpdateNursing(ctx, kidID, actorID, id, now, func(n model.NursingSession) (model.NursingSession, error) {
		if n.EndedAt != nil {
			return n, ErrNotRunning
		}
		n, ok := n.Switched(side, at)
		if !ok {
			return n, ErrInvalidSession
		}
		return n, nil
	})
}

// EndNursing ends the running session id at at, crediting the side in use: ErrNotRunning when it is finished,
// ErrInvalidSession when at precedes side_since or the session would exceed MaxSessionDuration.
func (s *Store) EndNursing(ctx context.Context, kidID, actorID, id string, at, now time.Time) (model.NursingSession, error) {
	at = unixTime(at.Unix())
	return s.UpdateNursing(ctx, kidID, actorID, id, now, func(n model.NursingSession) (model.NursingSession, error) {
		if n.EndedAt != nil {
			return n, ErrNotRunning
		}
		n, ok := n.Ended(at)
		if !ok {
			return n, ErrInvalidSession
		}
		return n, nil
	})
}

// DeleteNursing is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteNursing(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "nursing_sessions", kidID, actorID, id)
}
