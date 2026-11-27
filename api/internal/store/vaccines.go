package store

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

const doseColumns = `id, kid_id, administration, given_on, antigen, lot, product, note, created_by, updated_by, created_at, updated_at, visit_id`

func scanDose(row rowScanner) (model.VaccineDose, error) {
	var (
		d                         model.VaccineDose
		lot, product, note, visit sql.NullString
		created, updated          int64
	)
	if err := row.Scan(&d.ID, &d.KidID, &d.Administration, &d.GivenOn, &d.Antigen, &lot, &product, &note,
		&d.CreatedBy, &d.UpdatedBy, &created, &updated, &visit); err != nil {
		return model.VaccineDose{}, err
	}
	d.Lot, d.Product, d.Note, d.VisitID = nullableString(lot), nullableString(product), nullableString(note), nullableString(visit)
	d.CreatedAt, d.UpdatedAt = unixTime(created), unixTime(updated)
	return d, nil
}

func nullableString(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func queryDoses(ctx context.Context, r querier, tail string, args ...any) ([]model.VaccineDose, error) {
	rows, err := r.QueryContext(ctx, `SELECT `+doseColumns+` FROM vaccine_doses `+tail, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.VaccineDose{}
	for rows.Next() {
		d, err := scanDose(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// groupDoses folds rows ordered by administration (then antigen) into administrations, keeping that order.
func groupDoses(doses []model.VaccineDose) []model.Administration {
	out := []model.Administration{}
	for _, d := range doses {
		if n := len(out); n > 0 && out[n-1].ID == d.Administration {
			out[n-1].Antigens = append(out[n-1].Antigens, d.Antigen)
			continue
		}
		out = append(out, model.Administration{ID: d.Administration, KidID: d.KidID, GivenOn: d.GivenOn,
			Antigens: []string{d.Antigen}, Lot: d.Lot, Product: d.Product, Note: d.Note, VisitID: d.VisitID,
			CreatedBy: d.CreatedBy, UpdatedBy: d.UpdatedBy, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt})
	}
	return out
}

// canonicalAntigens is the sorted set, or false when it is empty or repeats an antigen.
func canonicalAntigens(antigens []string) ([]string, bool) {
	set := slices.Sorted(slices.Values(antigens))
	if len(set) == 0 || len(slices.Compact(slices.Clone(set))) != len(set) {
		return nil, false
	}
	return set, true
}

func insertDoses(ctx context.Context, tx *sql.Tx, a model.Administration, antigens []string) error {
	for _, antigen := range antigens {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO vaccine_doses(`+doseColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			model.NewID(), a.KidID, a.ID, a.GivenOn, antigen, a.Lot, a.Product, a.Note,
			a.CreatedBy, a.UpdatedBy, a.CreatedAt.Unix(), a.UpdatedAt.Unix(), a.VisitID); err != nil {
			return err
		}
	}
	return nil
}

// CreateAdministration stores one row per antigen of a for kidID as actorID, who needs an active grant (else
// ErrNotFound); ErrConflict when the kid already holds MaxVaccineAdministrationsPerKid. The caller validates the rest.
func (s *Store) CreateAdministration(ctx context.Context, kidID, actorID string, a model.Administration, now time.Time) (model.Administration, error) {
	if _, ok := canonicalAntigens(a.Antigens); !ok {
		return model.Administration{}, ErrInvalidAntigens
	}
	var out model.Administration
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		var err error
		out, err = createAdministrationTx(ctx, tx, kidID, actorID, a, nil, now)
		return err
	})
	if err != nil {
		return model.Administration{}, err
	}
	return out, nil
}

// createAdministrationTx is CreateAdministration inside tx, every row linked to visitID when set; the caller has
// asserted the actor's role.
func createAdministrationTx(ctx context.Context, tx *sql.Tx, kidID, actorID string, a model.Administration, visitID *string,
	now time.Time) (model.Administration, error) {
	set, ok := canonicalAntigens(a.Antigens)
	if !ok {
		return model.Administration{}, ErrInvalidAntigens
	}
	a.ID, a.KidID, a.Antigens, a.CreatedBy, a.UpdatedBy, a.VisitID = model.NewID(), kidID, set, actorID, actorID, visitID
	a.CreatedAt, a.UpdatedAt = unixTime(now.Unix()), unixTime(now.Unix())
	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT administration) FROM vaccine_doses WHERE kid_id = ?`, kidID).Scan(&n); err != nil {
		return model.Administration{}, err
	}
	if n >= constants.MaxVaccineAdministrationsPerKid {
		return model.Administration{}, ErrConflict
	}
	if err := insertDoses(ctx, tx, a, set); err != nil {
		return model.Administration{}, err
	}
	return a, nil
}

// Administration is ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) Administration(ctx context.Context, kidID, id string) (model.Administration, error) {
	doses, err := queryDoses(ctx, s.r, `WHERE kid_id = ? AND administration = ? ORDER BY antigen`, kidID, id)
	if err != nil {
		return model.Administration{}, err
	}
	if len(doses) == 0 {
		return model.Administration{}, ErrNotFound
	}
	return groupDoses(doses)[0], nil
}

// ListAdministrations returns all of the kid's administrations, newest first by (given_on, id); never nil.
func (s *Store) ListAdministrations(ctx context.Context, kidID string) ([]model.Administration, error) {
	doses, err := queryDoses(ctx, s.r,
		`WHERE kid_id = ? ORDER BY given_on DESC, administration DESC, antigen`, kidID)
	if err != nil {
		return nil, err
	}
	return groupDoses(doses), nil
}

// Doses returns every stored row of the kid, oldest first by (given_on, administration, antigen); never nil.
func (s *Store) Doses(ctx context.Context, kidID string) ([]model.VaccineDose, error) {
	return queryDoses(ctx, s.r, `WHERE kid_id = ? ORDER BY given_on, administration, antigen`, kidID)
}

// UpdateAdministration replaces every editable field and the antigen set of a.ID, which must belong to kidID; kept
// antigens keep their row ids. actorID needs an active grant.
func (s *Store) UpdateAdministration(ctx context.Context, kidID, actorID string, a model.Administration, now time.Time) error {
	set, ok := canonicalAntigens(a.Antigens)
	if !ok {
		return ErrInvalidAntigens
	}
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx,
			`SELECT antigen, created_by, created_at, visit_id FROM vaccine_doses WHERE kid_id = ? AND administration = ?`, kidID, a.ID)
		if err != nil {
			return err
		}
		var (
			stored    []string
			createdAt int64
			visit     sql.NullString
		)
		for rows.Next() {
			var antigen string
			if err := rows.Scan(&antigen, &a.CreatedBy, &createdAt, &visit); err != nil {
				rows.Close()
				return err
			}
			stored = append(stored, antigen)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(stored) == 0 {
			return ErrNotFound
		}
		for _, antigen := range stored {
			if !slices.Contains(set, antigen) {
				if _, err := tx.ExecContext(ctx,
					`DELETE FROM vaccine_doses WHERE kid_id = ? AND administration = ? AND antigen = ?`, kidID, a.ID, antigen); err != nil {
					return err
				}
			}
		}
		var added []string
		for _, antigen := range set {
			if !slices.Contains(stored, antigen) {
				added = append(added, antigen)
			}
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE vaccine_doses SET given_on = ?, lot = ?, product = ?, note = ?, updated_by = ?, updated_at = ?
			 WHERE kid_id = ? AND administration = ?`,
			a.GivenOn, a.Lot, a.Product, a.Note, actorID, now.Unix(), kidID, a.ID); err != nil {
			return err
		}
		a.KidID, a.UpdatedBy, a.CreatedAt, a.UpdatedAt = kidID, actorID, unixTime(createdAt), unixTime(now.Unix())
		a.VisitID = nullableString(visit)
		return insertDoses(ctx, tx, a, added)
	})
}

// DeleteAdministration removes every row of id; ErrNotFound for an id that is absent or belongs to another kid.
func (s *Store) DeleteAdministration(ctx context.Context, kidID, actorID, id string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM vaccine_doses WHERE kid_id = ? AND administration = ?`, kidID, id)
		return oneRow(res, err)
	})
}

// SetDismissal dismisses line of schema for the kid, replacing an earlier reason. actorID must be a guardian
// (ErrNotFound without an active grant, ErrForbidden as a caregiver).
func (s *Store) SetDismissal(ctx context.Context, kidID, actorID, schema, line, reason string, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := actorIsGuardian(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO vaccine_dismissals(kid_id, schema, line, reason, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(kid_id, schema, line) DO UPDATE SET reason = excluded.reason, created_by = excluded.created_by,
			   created_at = excluded.created_at`,
			kidID, schema, line, reason, actorID, now.Unix())
		return err
	})
}

// ClearDismissal undoes a dismissal of line in schema, a no-op when there is none; actorID must be a guardian.
func (s *Store) ClearDismissal(ctx context.Context, kidID, actorID, schema, line string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := actorIsGuardian(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`DELETE FROM vaccine_dismissals WHERE kid_id = ? AND schema = ? AND line = ?`, kidID, schema, line)
		return err
	})
}

// Dismissals maps each dismissed line of the kid's schema to its reason; never nil.
func (s *Store) Dismissals(ctx context.Context, kidID, schema string) (map[string]string, error) {
	return dismissals(ctx, s.r, kidID, schema)
}

func dismissals(ctx context.Context, q querier, kidID, schema string) (map[string]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT line, reason FROM vaccine_dismissals WHERE kid_id = ? AND schema = ?`, kidID, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var line, reason string
		if err := rows.Scan(&line, &reason); err != nil {
			return nil, err
		}
		out[line] = reason
	}
	return out, rows.Err()
}

// DismissLines dismisses every line in one transaction, leaving an already dismissed line as it is, and returns
// how many it inserted (0 when the transaction fails). actorID must be a guardian.
func (s *Store) DismissLines(ctx context.Context, kidID, actorID, schema string, lines []string, reason string, now time.Time) (int, error) {
	var inserted int64
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := actorIsGuardian(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		inserted = 0
		for _, line := range lines {
			res, err := tx.ExecContext(ctx,
				`INSERT INTO vaccine_dismissals(kid_id, schema, line, reason, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?)
				 ON CONFLICT(kid_id, schema, line) DO NOTHING`,
				kidID, schema, line, reason, actorID, now.Unix())
			if err != nil {
				return err
			}
			n, err := res.RowsAffected()
			if err != nil {
				return err
			}
			inserted += n
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int(inserted), nil
}
