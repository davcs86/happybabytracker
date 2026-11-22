package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

const mealColumns = entryColumns + `, at, what, reaction`

func scanMeal(row rowScanner) (model.Meal, error) {
	var (
		m        model.Meal
		at       int64
		reaction sql.NullString
	)
	e, err := scanEntry(row, &at, &m.What, &reaction)
	if err != nil {
		return model.Meal{}, err
	}
	m.Entry, m.At = e, unixTime(at)
	if reaction.Valid {
		m.Reaction = &reaction.String
	}
	return m, nil
}

// CreateMeal stores m for kidID as actorID, who needs an active grant (else ErrNotFound); the caller validates the rest.
func (s *Store) CreateMeal(ctx context.Context, kidID, actorID string, m model.Meal, now time.Time) (model.Meal, error) {
	m.ID, m.KidID, m.CreatedBy, m.UpdatedBy = model.NewID(), kidID, actorID, actorID
	m.CreatedAt, m.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	m.At = unixTime(m.At.Unix())
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO meals(`+mealColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			m.ID, kidID, m.Note, actorID, actorID, m.CreatedAt.Unix(), m.UpdatedAt.Unix(), m.At.Unix(), m.What, m.Reaction)
		return err
	})
	if err != nil {
		return model.Meal{}, err
	}
	return m, nil
}

// Meal is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Meal(ctx context.Context, kidID, id string) (model.Meal, error) {
	return scanMeal(s.r.QueryRowContext(ctx, `SELECT `+mealColumns+` FROM meals WHERE kid_id = ? AND id = ?`, kidID, id))
}

// ListMeals returns the kid's meals within q, newest first by (at, id); never nil.
func (s *Store) ListMeals(ctx context.Context, kidID string, q ListQuery) ([]model.Meal, error) {
	return listEntries(ctx, s.r, `SELECT `+mealColumns+` FROM meals`, "at", kidID, q, scanMeal)
}

// UpdateMeal replaces every editable field of m.ID, which must belong to kidID; actorID needs an active grant.
func (s *Store) UpdateMeal(ctx context.Context, kidID, actorID string, m model.Meal, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE meals SET at = ?, what = ?, reaction = ?, note = ?, updated_by = ?, updated_at = ? WHERE kid_id = ? AND id = ?`,
			m.At.Unix(), m.What, m.Reaction, m.Note, actorID, now.Unix(), kidID, m.ID)
		return oneRow(res, err)
	})
}

// DeleteMeal is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteMeal(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "meals", kidID, actorID, id)
}
