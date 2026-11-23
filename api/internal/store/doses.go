package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
)

const medicationDoseColumns = entryColumns + `, medication_id, given_at, amount`

func scanMedicationDose(row rowScanner) (model.MedicationDose, error) {
	var (
		v      model.MedicationDose
		given  int64
		amount sql.NullString
	)
	e, err := scanEntry(row, &v.MedicationID, &given, &amount)
	if err != nil {
		return model.MedicationDose{}, err
	}
	v.Entry, v.GivenAt, v.Amount = e, unixTime(given), nullableString(amount)
	return v, nil
}

// markDone marks at most one pending or expired reminder done and never deletes a row; it must not be replaced by
// rematerialise, which would drop undelivered rows. A done row the current timing does not produce is passed over.
func markDone(ctx context.Context, tx *sql.Tx, medicationID string, givenAt, now time.Time) error {
	v, err := scanMedication(tx.QueryRowContext(ctx, `SELECT `+medicationColumns+` FROM medications WHERE id = ?`, medicationID))
	if err != nil || v.Kind == schedule.KindNone {
		return err
	}
	loc, err := kidZone(ctx, tx, v.KidID)
	if err != nil {
		return err
	}
	current := timingOf(v).course()
	current.Active = true
	var (
		id, state string
		due       int64
	)
	for bound := givenAt.Add(constants.DoseDoneLead).Unix(); ; bound = due - 1 {
		err := tx.QueryRowContext(ctx, `SELECT id, due_at, state FROM reminders WHERE medication_id = ? AND due_at <= ? ORDER BY due_at DESC LIMIT 1`,
			medicationID, bound).Scan(&id, &due, &state)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		} else if err != nil {
			return err
		}
		if state != "done" || len(schedule.Occurrences(current, loc, unixTime(due), unixTime(due))) == 1 {
			break
		}
	}
	if state != "pending" && state != "expired" {
		return nil
	}
	if following, ok := schedule.Next(timingOf(v).course(), loc, unixTime(due)); ok && !givenAt.Before(following) {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE reminders SET state = 'done' WHERE id = ?`, id); err != nil {
		return err
	}
	return materialiseCourse(ctx, tx, medicationID, loc, now)
}

// CreateDose needs an active grant for actorID (else ErrNotFound); ErrInvalidMedication for a course that is not the
// kid's, ErrConflict at MaxMedicationDosesPerKid. The caller validates the rest.
func (s *Store) CreateDose(ctx context.Context, kidID, actorID string, v model.MedicationDose, now time.Time) (model.MedicationDose, error) {
	v.ID, v.KidID, v.CreatedBy, v.UpdatedBy = model.NewID(), kidID, actorID, actorID
	v.CreatedAt, v.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	v.GivenAt = unixTime(v.GivenAt.Unix())
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		var found bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM medications WHERE id = ? AND kid_id = ?)`, v.MedicationID, kidID).Scan(&found); err != nil {
			return err
		}
		if !found {
			return ErrInvalidMedication
		}
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM medication_doses WHERE kid_id = ?`, kidID).Scan(&n); err != nil {
			return err
		}
		if n >= constants.MaxMedicationDosesPerKid {
			return ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO medication_doses(`+medicationDoseColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			v.ID, kidID, v.Note, actorID, actorID, v.CreatedAt.Unix(), v.UpdatedAt.Unix(), v.MedicationID, v.GivenAt.Unix(), v.Amount); err != nil {
			return err
		}
		return markDone(ctx, tx, v.MedicationID, v.GivenAt, now)
	})
	if err != nil {
		return model.MedicationDose{}, err
	}
	return v, nil
}

// Dose is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Dose(ctx context.Context, kidID, id string) (model.MedicationDose, error) {
	return scanMedicationDose(s.r.QueryRowContext(ctx, `SELECT `+medicationDoseColumns+` FROM medication_doses WHERE kid_id = ? AND id = ?`, kidID, id))
}

// ListDoses returns the kid's doses within q, newest first by (given_at, id), of one course when medicationID is not
// empty; never nil.
func (s *Store) ListDoses(ctx context.Context, kidID, medicationID string, q ListQuery) ([]model.MedicationDose, error) {
	tail, args := pageTail("given_at", kidID, q)
	query := `SELECT ` + medicationDoseColumns + ` FROM medication_doses` + tail
	if medicationID != "" {
		query = `SELECT ` + medicationDoseColumns + ` FROM (SELECT * FROM medication_doses WHERE medication_id = ?)` + tail
		args = append([]any{medicationID}, args...)
	}
	rows, err := s.r.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.MedicationDose{}
	for rows.Next() {
		v, err := scanMedicationDose(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// UpdateDose replaces the editable fields of v.ID (never its course), which must belong to kidID; actorID needs an
// active grant. A new given_at marks its reminder done; the one the old time marked stays done.
func (s *Store) UpdateDose(ctx context.Context, kidID, actorID string, v model.MedicationDose, now time.Time) error {
	v.GivenAt = unixTime(v.GivenAt.Unix())
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		stored, err := scanMedicationDose(tx.QueryRowContext(ctx, `SELECT `+medicationDoseColumns+` FROM medication_doses WHERE kid_id = ? AND id = ?`, kidID, v.ID))
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE medication_doses SET given_at = ?, amount = ?, note = ?, updated_by = ?, updated_at = ? WHERE kid_id = ? AND id = ?`,
			v.GivenAt.Unix(), v.Amount, v.Note, actorID, now.Unix(), kidID, v.ID)
		if err := oneRow(res, err); err != nil {
			return err
		}
		if v.GivenAt.Equal(stored.GivenAt) {
			return nil
		}
		return markDone(ctx, tx, stored.MedicationID, v.GivenAt, now)
	})
}

// DeleteDose is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteDose(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "medication_doses", kidID, actorID, id)
}
