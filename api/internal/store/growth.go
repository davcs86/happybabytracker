package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

const growthColumns = entryColumns + `, measured_on, weight_g, weight_oz_h, weight_typed, length_mm, length_in_h, length_typed, head_mm, head_in_h, head_typed, visit_id`

func scanGrowth(row rowScanner) (model.Growth, error) {
	var (
		g                      model.Growth
		wg, wo, lm, li, hm, hi sql.NullInt64
		wt, lt, ht, visit      sql.NullString
	)
	e, err := scanEntry(row, &g.MeasuredOn, &wg, &wo, &wt, &lm, &li, &lt, &hm, &hi, &ht, &visit)
	if err != nil {
		return model.Growth{}, err
	}
	g.Entry = e
	if wg.Valid {
		g.Weight = &units.Weight{Grams: wg.Int64, OzHundredths: wo.Int64, Typed: units.Unit(wt.String)}
	}
	g.Length, g.Head, g.VisitID = nullableLength(lm, li, lt), nullableLength(hm, hi, ht), nullableString(visit)
	return g, nil
}

func nullableLength(mm, in sql.NullInt64, typed sql.NullString) *units.Length {
	if !mm.Valid {
		return nil
	}
	return &units.Length{MM: mm.Int64, InHundredths: in.Int64, Typed: units.Unit(typed.String)}
}

// measureArgs are the nine measure columns of g, in growthColumns order; an absent measure is three NULLs.
func measureArgs(g model.Growth) []any {
	args := []any{nil, nil, nil}
	if w := g.Weight; w != nil {
		args = []any{w.Grams, w.OzHundredths, string(w.Typed)}
	}
	for _, l := range []*units.Length{g.Length, g.Head} {
		if l == nil {
			args = append(args, nil, nil, nil)
		} else {
			args = append(args, l.MM, l.InHundredths, string(l.Typed))
		}
	}
	return args
}

// CreateGrowth stores g for kidID as actorID, who needs an active grant (else ErrNotFound); ErrConflict when the kid
// already holds MaxGrowthPerKid measurements. The caller validates the rest.
func (s *Store) CreateGrowth(ctx context.Context, kidID, actorID string, g model.Growth, now time.Time) (model.Growth, error) {
	var out model.Growth
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		var err error
		out, err = createGrowthTx(ctx, tx, kidID, actorID, g, nil, now)
		return err
	})
	if err != nil {
		return model.Growth{}, err
	}
	return out, nil
}

// createGrowthTx is CreateGrowth inside tx, linked to visitID when set; the caller has asserted the actor's role.
func createGrowthTx(ctx context.Context, tx *sql.Tx, kidID, actorID string, g model.Growth, visitID *string, now time.Time) (model.Growth, error) {
	g.ID, g.KidID, g.CreatedBy, g.UpdatedBy, g.VisitID = model.NewID(), kidID, actorID, actorID, visitID
	g.CreatedAt, g.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM growth_measurements WHERE kid_id = ?`, kidID).Scan(&n); err != nil {
		return model.Growth{}, err
	}
	if n >= constants.MaxGrowthPerKid {
		return model.Growth{}, ErrConflict
	}
	args := append(append([]any{g.ID, kidID, g.Note, actorID, actorID, g.CreatedAt.Unix(), g.UpdatedAt.Unix(), g.MeasuredOn}, measureArgs(g)...), g.VisitID)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO growth_measurements(`+growthColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, args...); err != nil {
		return model.Growth{}, err
	}
	return g, nil
}

// Growth is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Growth(ctx context.Context, kidID, id string) (model.Growth, error) {
	return scanGrowth(s.r.QueryRowContext(ctx, `SELECT `+growthColumns+` FROM growth_measurements WHERE kid_id = ? AND id = ?`, kidID, id))
}

// ListGrowth returns all of the kid's measurements, newest first by (measured_on, id); never nil.
func (s *Store) ListGrowth(ctx context.Context, kidID string) ([]model.Growth, error) {
	return listGrowth(ctx, s.r, kidID)
}

func listGrowth(ctx context.Context, q querier, kidID string) ([]model.Growth, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT `+growthColumns+` FROM growth_measurements WHERE kid_id = ? ORDER BY measured_on DESC, id DESC LIMIT ?`,
		kidID, constants.MaxGrowthPerKid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Growth{}
	for rows.Next() {
		g, err := scanGrowth(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// UpdateGrowth replaces every editable field of g.ID, which must belong to kidID; actorID needs an active grant.
func (s *Store) UpdateGrowth(ctx context.Context, kidID, actorID string, g model.Growth, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		args := append(append([]any{g.MeasuredOn}, measureArgs(g)...), g.Note, actorID, now.Unix(), kidID, g.ID)
		res, err := tx.ExecContext(ctx,
			`UPDATE growth_measurements SET measured_on = ?, weight_g = ?, weight_oz_h = ?, weight_typed = ?,
			   length_mm = ?, length_in_h = ?, length_typed = ?, head_mm = ?, head_in_h = ?, head_typed = ?,
			   note = ?, updated_by = ?, updated_at = ?
			 WHERE kid_id = ? AND id = ?`, args...)
		return oneRow(res, err)
	})
}

// DeleteGrowth is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteGrowth(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "growth_measurements", kidID, actorID, id)
}
