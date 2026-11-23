package store

import (
	"context"
	"database/sql"
	"slices"
	"strings"
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

// orderSides returns sides in storage order; ok is false unless they are a valid set whose derived side is side.
// No sides is a simple session: nil.
func orderSides(side string, sides []model.PumpingSide) (ordered []model.PumpingSide, ok bool) {
	if len(sides) == 0 {
		return nil, true
	}
	if derived, valid := model.DerivedPumpingSide(sides); !valid || derived != side {
		return nil, false
	}
	for _, name := range model.PumpingSideNames {
		if i := slices.IndexFunc(sides, func(s model.PumpingSide) bool { return s.Side == name }); i >= 0 {
			ordered = append(ordered, sides[i])
		}
	}
	return ordered, true
}

// replaceSides makes sides, in order, the whole set stored for sessionID.
func replaceSides(ctx context.Context, tx *sql.Tx, sessionID string, sides []model.PumpingSide) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM pumping_session_sides WHERE session_id = ?`, sessionID); err != nil {
		return err
	}
	for _, side := range sides {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO pumping_session_sides(session_id, side, volume_ml, volume_floz_h, volume_typed) VALUES (?, ?, ?, ?, ?)`,
			sessionID, side.Side, side.Volume.ML, side.Volume.FlOzHundredths, string(side.Volume.Typed)); err != nil {
			return err
		}
	}
	return nil
}

// withSides sets Sides on each of sessions, read through r, with one query for the whole page.
func withSides(ctx context.Context, r querier, sessions []model.PumpingSession) ([]model.PumpingSession, error) {
	if len(sessions) == 0 {
		return sessions, nil
	}
	at := make(map[string]int, len(sessions))
	args := make([]any, len(sessions))
	for i, p := range sessions {
		at[p.ID], args[i] = i, p.ID
	}
	rows, err := r.QueryContext(ctx,
		`SELECT session_id, side, volume_ml, volume_floz_h, volume_typed FROM pumping_session_sides WHERE session_id IN (?`+
			strings.Repeat(`, ?`, len(args)-1)+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id, typed string
			side      model.PumpingSide
		)
		if err := rows.Scan(&id, &side.Side, &side.Volume.ML, &side.Volume.FlOzHundredths, &typed); err != nil {
			return nil, err
		}
		side.Volume.Typed = units.Unit(typed)
		i := at[id]
		sessions[i].Sides = append(sessions[i].Sides, side)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range sessions {
		slices.SortFunc(sessions[i].Sides, func(a, b model.PumpingSide) int {
			return slices.Index(model.PumpingSideNames, a.Side) - slices.Index(model.PumpingSideNames, b.Side)
		})
	}
	return sessions, nil
}

// CreatePumping stores p for kidID as actorID, who needs an active grant (else ErrNotFound); the caller validates the rest.
// p.Sides, when set, are valid and imply p.Side (else ErrInvalidSides).
func (s *Store) CreatePumping(ctx context.Context, kidID, actorID string, p model.PumpingSession, now time.Time) (model.PumpingSession, error) {
	sides, ok := orderSides(p.Side, p.Sides)
	if !ok {
		return model.PumpingSession{}, ErrInvalidSides
	}
	p.Sides = sides
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
		if err != nil {
			return err
		}
		return replaceSides(ctx, tx, p.ID, p.Sides)
	})
	if err != nil {
		return model.PumpingSession{}, err
	}
	return p, nil
}

// Pumping is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Pumping(ctx context.Context, kidID, id string) (model.PumpingSession, error) {
	var out model.PumpingSession
	err := s.inReadTx(ctx, func(q querier) error {
		p, err := scanPumping(q.QueryRowContext(ctx, `SELECT `+pumpingColumns+` FROM pumping_sessions WHERE kid_id = ? AND id = ?`, kidID, id))
		if err != nil {
			return err
		}
		one, err := withSides(ctx, q, []model.PumpingSession{p})
		if err == nil {
			out = one[0]
		}
		return err
	})
	return out, err
}

// ListPumping returns the kid's sessions within q, newest first by (started_at, id); never nil.
func (s *Store) ListPumping(ctx context.Context, kidID string, q ListQuery) ([]model.PumpingSession, error) {
	var out []model.PumpingSession
	err := s.inReadTx(ctx, func(r querier) error {
		rows, err := listEntries(ctx, r, `SELECT `+pumpingColumns+` FROM pumping_sessions`, "started_at", kidID, q, scanPumping)
		if err != nil {
			return err
		}
		out, err = withSides(ctx, r, rows)
		return err
	})
	return out, err
}

// UpdatePumping replaces every editable field of p.ID, which must belong to kidID, and its whole set of sides;
// actorID needs an active grant.
func (s *Store) UpdatePumping(ctx context.Context, kidID, actorID string, p model.PumpingSession, now time.Time) error {
	sides, ok := orderSides(p.Side, p.Sides)
	if !ok {
		return ErrInvalidSides
	}
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
		if err := oneRow(res, err); err != nil {
			return err
		}
		return replaceSides(ctx, tx, p.ID, sides)
	})
}

// DeletePumping is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeletePumping(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "pumping_sessions", kidID, actorID, id)
}
