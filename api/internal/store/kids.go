package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

const kidColumns = `k.id, k.name, k.birth_date, k.sex, k.gestation_weeks, k.gestation_days, k.time_zone, k.growth_source, k.vaccine_schema, k.reminder_overdue_minutes`

func scanKid(row rowScanner, extra ...any) (model.Kid, error) {
	var (
		k           model.Kid
		weeks, days sql.NullInt64
	)
	dest := append([]any{&k.ID, &k.Name, &k.BirthDate, &k.Sex, &weeks, &days, &k.TimeZone, &k.GrowthSource, &k.VaccineSchema, &k.ReminderOverdueMinutes}, extra...)
	err := row.Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Kid{}, ErrNotFound
	}
	if err != nil {
		return model.Kid{}, err
	}
	k.GestationWeeks, k.GestationDays = nullableInt(weeks), nullableInt(days)
	return k, nil
}

func nullableInt(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// CreateKid stores the kid and makes the creator its guardian in one transaction; the caller validates k.
// An unknown or disabled creator is ErrNotFound and stores nothing.
func (s *Store) CreateKid(ctx context.Context, k model.Kid, creator string, now time.Time) (model.Kid, error) {
	k.ID = model.NewID()
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		var one int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM parents WHERE id = ? AND disabled = 0`, creator).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO kids(id, name, birth_date, sex, gestation_weeks, gestation_days, time_zone, growth_source, vaccine_schema, reminder_overdue_minutes, created_by, updated_by, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			k.ID, k.Name, k.BirthDate, k.Sex, k.GestationWeeks, k.GestationDays, k.TimeZone, k.GrowthSource, k.VaccineSchema, k.ReminderOverdueMinutes, creator, creator, now.Unix(), now.Unix()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES (?, ?, ?, ?)`,
			k.ID, creator, model.RoleGuardian, now.Unix()); err != nil {
			return err
		}
		return keepsActiveGuardian(ctx, tx, k.ID)
	})
	if err != nil {
		return model.Kid{}, err
	}
	return k, nil
}

// KidsFor returns the kids the parent holds any grant on, oldest first; never nil.
func (s *Store) KidsFor(ctx context.Context, parentID string) ([]model.KidWithRole, error) {
	rows, err := s.r.QueryContext(ctx,
		`SELECT `+kidColumns+`, a.role FROM kids k JOIN kid_access a ON a.kid_id = k.id
		 WHERE a.parent_id = ? ORDER BY k.created_at, k.id`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.KidWithRole{}
	for rows.Next() {
		var role model.Role
		k, err := scanKid(rows, &role)
		if err != nil {
			return nil, err
		}
		out = append(out, model.KidWithRole{Kid: k, Role: role})
	}
	return out, rows.Err()
}

func (s *Store) Kid(ctx context.Context, kidID string) (model.Kid, error) {
	return scanKid(s.r.QueryRowContext(ctx, `SELECT `+kidColumns+` FROM kids k WHERE k.id = ?`, kidID))
}

// UpdateKid replaces every profile field of k.ID; the caller validates the merged record. by must be a guardian.
// A changed time zone re-materialises the kid's courses in the same transaction.
func (s *Store) UpdateKid(ctx context.Context, k model.Kid, by string, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := actorIsGuardian(ctx, tx, k.ID, by); err != nil {
			return err
		}
		var zone string
		if err := tx.QueryRowContext(ctx, `SELECT time_zone FROM kids WHERE id = ?`, k.ID).Scan(&zone); errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE kids SET name = ?, birth_date = ?, sex = ?, gestation_weeks = ?, gestation_days = ?, time_zone = ?, growth_source = ?, vaccine_schema = ?, reminder_overdue_minutes = ?, updated_by = ?, updated_at = ?
			 WHERE id = ?`,
			k.Name, k.BirthDate, k.Sex, k.GestationWeeks, k.GestationDays, k.TimeZone, k.GrowthSource, k.VaccineSchema, k.ReminderOverdueMinutes, by, now.Unix(), k.ID)
		if err := oneRow(res, err); err != nil || zone == k.TimeZone {
			return err
		}
		return rematerialiseKid(ctx, tx, k.ID, k.TimeZone, now)
	})
}

// DeleteKid removes the kid; its grants go with it through ON DELETE CASCADE. actorID must be a guardian.
func (s *Store) DeleteKid(ctx context.Context, kidID, actorID string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := actorIsGuardian(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM kids WHERE id = ?`, kidID)
		return oneRow(res, err)
	})
}

// oneRow turns a write that touched no row into ErrNotFound.
func oneRow(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}
