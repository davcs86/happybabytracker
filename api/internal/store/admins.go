package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

// ErrAdminExists is CreateFirstAdmin when an admin is already stored.
var ErrAdminExists = errors.New("store: admin exists")

const adminSelect = `SELECT a.id, a.username, a.created_at, i.secret_hash,
	EXISTS (SELECT 1 FROM admin_identities o WHERE o.admin_id = a.id AND o.provider <> '` + providerLocal + `')
	FROM admins a LEFT JOIN admin_identities i ON i.admin_id = a.id AND i.provider = '` + providerLocal + `' `

func scanAdmin(row rowScanner) (model.Admin, string, error) {
	var (
		a       model.Admin
		created int64
		hash    sql.NullString
	)
	err := row.Scan(&a.ID, &a.Username, &created, &hash, &a.SSOLinked)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Admin{}, "", ErrNotFound
	}
	if err != nil {
		return model.Admin{}, "", err
	}
	a.CreatedAt = time.Unix(created, 0).UTC()
	a.HasPassword = hash.String != ""
	return a, hash.String, nil
}

func (s *Store) HasAdmin(ctx context.Context) (bool, error) {
	var ok bool
	err := s.r.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM admins)`).Scan(&ok)
	return ok, err
}

// CreateFirstAdmin is ErrAdminExists when an admin is stored and ErrConflict when a parent holds the username.
func (s *Store) CreateFirstAdmin(ctx context.Context, username, secretHash string, now time.Time) (model.Admin, error) {
	id := model.NewID()
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM admins LIMIT 1`).Scan(new(int)); err == nil {
			return ErrAdminExists
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM parents WHERE username = ?`, username).Scan(new(int)); err == nil {
			return ErrConflict
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO admins(id, username, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			id, username, now.Unix(), now.Unix()); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO admin_identities(admin_id, provider, subject, secret_hash, secret_changed_at) VALUES (?, ?, ?, ?, ?)`,
			id, providerLocal, username, secretHash, now.Unix())
		return err
	})
	if err != nil {
		return model.Admin{}, err
	}
	a, _, err := s.AdminByID(ctx, id)
	return a, err
}

// AdminByUsername returns the admin and its local password hash, empty when none is set.
func (s *Store) AdminByUsername(ctx context.Context, username string) (model.Admin, string, error) {
	return scanAdmin(s.r.QueryRowContext(ctx, adminSelect+`WHERE a.username = ?`, username))
}

// AdminByID returns the admin and its local password hash, empty when none is set.
func (s *Store) AdminByID(ctx context.Context, id string) (model.Admin, string, error) {
	return scanAdmin(s.r.QueryRowContext(ctx, adminSelect+`WHERE a.id = ?`, id))
}

// SetAdminPassword upserts the admin's local identity and deletes all of its sessions.
func (s *Store) SetAdminPassword(ctx context.Context, username, secretHash string, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		id, err := adminID(ctx, tx, username)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO admin_identities(admin_id, provider, subject, secret_hash, secret_changed_at) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT(provider, subject) DO UPDATE SET secret_hash = excluded.secret_hash, secret_changed_at = excluded.secret_changed_at`,
			id, providerLocal, username, secretHash, now.Unix()); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE admin_id = ?`, id)
		return err
	})
}

// UnlinkAdminSSO deletes the admin's non-local identities and all of its sessions.
func (s *Store) UnlinkAdminSSO(ctx context.Context, username string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		id, err := adminID(ctx, tx, username)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM admin_identities WHERE admin_id = ? AND provider <> ?`, id, providerLocal); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE admin_id = ?`, id)
		return err
	})
}

func adminID(ctx context.Context, tx *sql.Tx, username string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM admins WHERE username = ?`, username).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}
