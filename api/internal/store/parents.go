package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

const providerLocal = "local"

const parentSelect = `SELECT p.id, p.username, p.display_name, p.disabled, p.unit_weight, p.unit_length, p.unit_volume, p.unit_temp, p.created_at, i.secret_hash, i.secret_changed_at
	FROM parents p LEFT JOIN identities i ON i.parent_id = p.id AND i.provider = '` + providerLocal + `' `

type rowScanner interface{ Scan(dest ...any) error }

func scanParent(row rowScanner) (model.Parent, string, error) {
	var (
		p       model.Parent
		hash    sql.NullString
		created int64
		changed sql.NullInt64
	)
	err := row.Scan(&p.ID, &p.Username, &p.DisplayName, &p.Disabled,
		&p.Units.Weight, &p.Units.Length, &p.Units.Volume, &p.Units.Temp, &created, &hash, &changed)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Parent{}, "", ErrNotFound
	}
	if err != nil {
		return model.Parent{}, "", err
	}
	p.CreatedAt = time.Unix(created, 0).UTC()
	p.HasPassword = hash.String != ""
	if changed.Valid {
		at := time.Unix(changed.Int64, 0).UTC()
		p.PasswordChangedAt = &at
	}
	return p, hash.String, nil
}

// ParentByUsername returns the parent and its local password hash, empty when no password is set.
func (s *Store) ParentByUsername(ctx context.Context, username string) (model.Parent, string, error) {
	return scanParent(s.r.QueryRowContext(ctx, parentSelect+`WHERE p.username = ?`, username))
}

func (s *Store) ParentByID(ctx context.Context, id string) (model.Parent, error) {
	p, _, err := scanParent(s.r.QueryRowContext(ctx, parentSelect+`WHERE p.id = ?`, id))
	return p, err
}

// CreateParent returns ErrConflict when a parent or the admin holds the username; the caller passes it normalised.
func (s *Store) CreateParent(ctx context.Context, username, displayName string, now time.Time) (model.Parent, error) {
	id := model.NewID()
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM admins WHERE username = ?`, username).Scan(new(int)); err == nil {
			return ErrConflict
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO parents(id, username, display_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			id, username, displayName, now.Unix(), now.Unix())
		var se *sqlite.Error
		if errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return ErrConflict
		}
		return err
	})
	if err != nil {
		return model.Parent{}, err
	}
	return s.ParentByID(ctx, id)
}

// ListParents returns every parent, by username; never nil.
func (s *Store) ListParents(ctx context.Context) ([]model.Parent, error) {
	rows, err := s.r.QueryContext(ctx, parentSelect+`ORDER BY p.username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Parent{}
	for rows.Next() {
		p, _, err := scanParent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LocalSecret returns the parent's local password hash; ErrNotFound when none is set.
func (s *Store) LocalSecret(ctx context.Context, parentID string) (string, error) {
	var hash sql.NullString
	err := s.r.QueryRowContext(ctx, `SELECT secret_hash FROM identities WHERE parent_id = ? AND provider = ?`, parentID, providerLocal).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && hash.String == "") {
		return "", ErrNotFound
	}
	return hash.String, err
}

// UpdateParentProfile replaces the display name and units; the caller validates them.
func (s *Store) UpdateParentProfile(ctx context.Context, id, displayName string, u model.Units, now time.Time) error {
	res, err := s.w.ExecContext(ctx,
		`UPDATE parents SET display_name = ?, unit_weight = ?, unit_length = ?, unit_volume = ?, unit_temp = ?, updated_at = ? WHERE id = ?`,
		displayName, u.Weight, u.Length, u.Volume, u.Temp, now.Unix(), id)
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

// SetPassword upserts the parent's local identity and deletes their sessions except keepSessionID ("" keeps none).
func (s *Store) SetPassword(ctx context.Context, parentID, secretHash, keepSessionID string, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		return setPassword(ctx, tx, parentID, secretHash, keepSessionID, now)
	})
}

func setPassword(ctx context.Context, tx *sql.Tx, parentID, secretHash, keepSessionID string, now time.Time) error {
	var username string
	err := tx.QueryRowContext(ctx, `SELECT username FROM parents WHERE id = ?`, parentID).Scan(&username)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO identities(parent_id, provider, subject, secret_hash, secret_changed_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(provider, subject) DO UPDATE SET secret_hash = excluded.secret_hash, secret_changed_at = excluded.secret_changed_at`,
		parentID, providerLocal, username, secretHash, now.Unix()); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE parent_id = ? AND id <> ?`, parentID, keepSessionID)
	return err
}

// SetParentDisabled flips the flag; disabling also deletes the parent's sessions, push devices and ntfy topic, and
// is ErrLastGuardian when it would leave any kid without an active guardian.
func (s *Store) SetParentDisabled(ctx context.Context, id string, disabled bool, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE parents SET disabled = ?, updated_at = ? WHERE id = ?`, disabled, now.Unix(), id)
		if err := oneRow(res, err); err != nil {
			return err
		}
		if !disabled {
			return nil
		}
		if err := guardsEveryKidWithOthers(ctx, tx, id); err != nil {
			return err
		}
		if err := deleteChannels(ctx, tx, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE parent_id = ?`, id)
		return err
	})
}

// inTx runs f in one write transaction and commits only when f returns nil.
func (s *Store) inTx(ctx context.Context, f func(*sql.Tx) error) error {
	tx, err := s.w.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := f(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// inReadTx runs f on one snapshot of the read pool, so reads that belong together never straddle a write.
func (s *Store) inReadTx(ctx context.Context, f func(q querier) error) error {
	tx, err := s.r.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return f(tx)
}
