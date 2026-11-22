package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

const pumpingColumns = entryColumns + `, started_at, duration_seconds, volume_ml, volume_floz_h, volume_typed, side`

func scanPumping(row rowScanner) (model.PumpingSession, error) {
	var (
		p         model.PumpingSession
		startedAt int64
		typedUnit string
	)
	e, err := scanEntry(row, &startedAt, &p.DurationSeconds, &p.Volume.ML, &p.Volume.FlOzHundredths, &typedUnit, &p.Side)
	if err != nil {
		return model.PumpingSession{}, err
	}
	p.Entry, p.StartedAt, p.Volume.Typed = e, unixTime(startedAt), units.Unit(typedUnit)
	return p, nil
}

// CreatePumping stores p for kidID as actorID, who needs an active grant (else ErrNotFound); the caller validates the rest.
func (s *Store) CreatePumping(ctx context.Context, kidID, actorID string, p model.PumpingSession, now time.Time) (model.PumpingSession, error) {
	p.ID, p.KidID, p.CreatedBy, p.UpdatedBy = model.NewID(), kidID, actorID, actorID
	p.CreatedAt, p.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	p.StartedAt = unixTime(p.StartedAt.Unix())
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO pumping_sessions(`+pumpingColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.ID, kidID, p.Note, actorID, actorID, p.CreatedAt.Unix(), p.UpdatedAt.Unix(),
			p.StartedAt.Unix(), p.DurationSeconds, p.Volume.ML, p.Volume.FlOzHundredths, string(p.Volume.Typed), p.Side)
		return err
	})
	if err != nil {
		return model.PumpingSession{}, err
	}
	return p, nil
}

// Pumping is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Pumping(ctx context.Context, kidID, id string) (model.PumpingSession, error) {
	return scanPumping(s.r.QueryRowContext(ctx, `SELECT `+pumpingColumns+` FROM pumping_sessions WHERE kid_id = ? AND id = ?`, kidID, id))
}

// ListPumping returns the kid's sessions within q, newest first by (started_at, id); never nil.
func (s *Store) ListPumping(ctx context.Context, kidID string, q ListQuery) ([]model.PumpingSession, error) {
	return listEntries(ctx, s.r, `SELECT `+pumpingColumns+` FROM pumping_sessions`, "started_at", kidID, q, scanPumping)
}

// UpdatePumping replaces every editable field of p.ID, which must belong to kidID; actorID needs an active grant.
func (s *Store) UpdatePumping(ctx context.Context, kidID, actorID string, p model.PumpingSession, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE pumping_sessions SET started_at = ?, duration_seconds = ?, volume_ml = ?, volume_floz_h = ?, volume_typed = ?,
			   side = ?, note = ?, updated_by = ?, updated_at = ?
			 WHERE kid_id = ? AND id = ?`,
			p.StartedAt.Unix(), p.DurationSeconds, p.Volume.ML, p.Volume.FlOzHundredths, string(p.Volume.Typed),
			p.Side, p.Note, actorID, now.Unix(), kidID, p.ID)
		return oneRow(res, err)
	})
}

// DeletePumping is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeletePumping(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "pumping_sessions", kidID, actorID, id)
}
