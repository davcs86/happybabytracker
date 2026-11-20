package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

const kidColumns = `k.id, k.name, k.birth_date, k.sex, k.gestation_weeks, k.gestation_days, k.time_zone`

func scanKid(row rowScanner, extra ...any) (model.Kid, error) {
	var (
		k           model.Kid
		weeks, days sql.NullInt64
	)
	dest := append([]any{&k.ID, &k.Name, &k.BirthDate, &k.Sex, &weeks, &days, &k.TimeZone}, extra...)
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
func (s *Store) CreateKid(ctx context.Context, k model.Kid, creator string, now time.Time) (model.Kid, error) {
	k.ID = model.NewID()
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO kids(id, name, birth_date, sex, gestation_weeks, gestation_days, time_zone, created_by, updated_by, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			k.ID, k.Name, k.BirthDate, k.Sex, k.GestationWeeks, k.GestationDays, k.TimeZone, creator, creator, now.Unix(), now.Unix()); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES (?, ?, ?, ?)`,
			k.ID, creator, model.RoleGuardian, now.Unix())
		return err
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

// UpdateKid replaces every profile field of k.ID; the caller validates the merged record.
func (s *Store) UpdateKid(ctx context.Context, k model.Kid, by string, now time.Time) error {
	res, err := s.w.ExecContext(ctx,
		`UPDATE kids SET name = ?, birth_date = ?, sex = ?, gestation_weeks = ?, gestation_days = ?, time_zone = ?, updated_by = ?, updated_at = ?
		 WHERE id = ?`,
		k.Name, k.BirthDate, k.Sex, k.GestationWeeks, k.GestationDays, k.TimeZone, by, now.Unix(), k.ID)
	return oneRow(res, err)
}

// DeleteKid removes the kid; its grants go with it through ON DELETE CASCADE.
func (s *Store) DeleteKid(ctx context.Context, kidID string) error {
	res, err := s.w.ExecContext(ctx, `DELETE FROM kids WHERE id = ?`, kidID)
	return oneRow(res, err)
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
