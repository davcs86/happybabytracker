package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

const symptomColumns = entryColumns + `, at, symptom, severity, other_text, temp_c_h, temp_f_h, temp_typed, method`

func scanSymptom(row rowScanner) (model.Symptom, error) {
	var (
		v                          model.Symptom
		at                         int64
		severity, otherText, typed sql.NullString
		method                     sql.NullString
		c, f                       sql.NullInt64
	)
	e, err := scanEntry(row, &at, &v.Symptom, &severity, &otherText, &c, &f, &typed, &method)
	if err != nil {
		return model.Symptom{}, err
	}
	v.Entry, v.At = e, unixTime(at)
	v.Severity, v.OtherText, v.Method = nullableString(severity), nullableString(otherText), nullableString(method)
	if c.Valid {
		v.Temperature = &units.Temperature{CHundredths: c.Int64, FHundredths: f.Int64, Typed: units.Unit(typed.String)}
	}
	return v, nil
}

// temperatureArgs are the three temperature columns of v; an absent temperature is three NULLs.
func temperatureArgs(v model.Symptom) []any {
	if t := v.Temperature; t != nil {
		return []any{t.CHundredths, t.FHundredths, string(t.Typed)}
	}
	return []any{nil, nil, nil}
}

// CreateSymptom stores v for kidID as actorID, who needs an active grant (else ErrNotFound); the caller validates the rest.
func (s *Store) CreateSymptom(ctx context.Context, kidID, actorID string, v model.Symptom, now time.Time) (model.Symptom, error) {
	v.ID, v.KidID, v.CreatedBy, v.UpdatedBy = model.NewID(), kidID, actorID, actorID
	v.CreatedAt, v.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	v.At = unixTime(v.At.Unix())
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		args := append([]any{v.ID, kidID, v.Note, actorID, actorID, v.CreatedAt.Unix(), v.UpdatedAt.Unix(), v.At.Unix(), v.Symptom, v.Severity, v.OtherText},
			append(temperatureArgs(v), v.Method)...)
		_, err := tx.ExecContext(ctx, `INSERT INTO symptoms(`+symptomColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, args...)
		return err
	})
	if err != nil {
		return model.Symptom{}, err
	}
	return v, nil
}

// Symptom is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Symptom(ctx context.Context, kidID, id string) (model.Symptom, error) {
	return scanSymptom(s.r.QueryRowContext(ctx, `SELECT `+symptomColumns+` FROM symptoms WHERE kid_id = ? AND id = ?`, kidID, id))
}

// ListSymptoms returns the kid's symptoms within q, newest first by (at, id); never nil.
func (s *Store) ListSymptoms(ctx context.Context, kidID string, q ListQuery) ([]model.Symptom, error) {
	return listEntries(ctx, s.r, `SELECT `+symptomColumns+` FROM symptoms`, "at", kidID, q, scanSymptom)
}

// UpdateSymptom replaces every editable field of v.ID, which must belong to kidID; actorID needs an active grant.
func (s *Store) UpdateSymptom(ctx context.Context, kidID, actorID string, v model.Symptom, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		args := append(append([]any{v.At.Unix(), v.Symptom, v.Severity, v.OtherText}, append(temperatureArgs(v), v.Method)...),
			v.Note, actorID, now.Unix(), kidID, v.ID)
		res, err := tx.ExecContext(ctx,
			`UPDATE symptoms SET at = ?, symptom = ?, severity = ?, other_text = ?, temp_c_h = ?, temp_f_h = ?, temp_typed = ?, method = ?,
			   note = ?, updated_by = ?, updated_at = ? WHERE kid_id = ? AND id = ?`, args...)
		return oneRow(res, err)
	})
}

// DeleteSymptom is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteSymptom(ctx context.Context, kidID, actorID, id string) error {
	return s.deleteEntry(ctx, "symptoms", kidID, actorID, id)
}
