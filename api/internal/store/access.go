package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

// RoleFor returns ErrNotFound alike for an unknown kid and for a kid the parent holds no grant on.
func (s *Store) RoleFor(ctx context.Context, kidID, parentID string) (model.Role, error) {
	var role model.Role
	err := s.r.QueryRowContext(ctx, `SELECT role FROM kid_access WHERE kid_id = ? AND parent_id = ?`, kidID, parentID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

// AccessList returns the kid's grants, guardians first, then by username; never nil.
func (s *Store) AccessList(ctx context.Context, kidID string) ([]model.Grant, error) {
	rows, err := s.r.QueryContext(ctx,
		`SELECT p.id, p.username, p.display_name, a.role, p.disabled FROM kid_access a JOIN parents p ON p.id = a.parent_id
		 WHERE a.kid_id = ? ORDER BY CASE a.role WHEN ? THEN 0 ELSE 1 END, p.username`, kidID, model.RoleGuardian)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Grant{}
	for rows.Next() {
		var g model.Grant
		if err := rows.Scan(&g.ParentID, &g.Username, &g.DisplayName, &g.Role, &g.Disabled); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// SetAccess grants or changes a role. actorID must be a guardian (ErrNotFound without a grant, ErrForbidden as
// a caregiver); ErrParentNotFound for an unknown or disabled target; ErrLastGuardian on a downgrade.
func (s *Store) SetAccess(ctx context.Context, kidID, actorID, username string, role model.Role, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := actorIsGuardian(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		var parentID string
		err := tx.QueryRowContext(ctx, `SELECT id FROM parents WHERE username = ? AND disabled = 0`, username).Scan(&parentID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrParentNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES (?, ?, ?, ?)
			 ON CONFLICT(kid_id, parent_id) DO UPDATE SET role = excluded.role`,
			kidID, parentID, role, now.Unix()); err != nil {
			return err
		}
		return keepsActiveGuardian(ctx, tx, kidID)
	})
}

// RemoveAccess needs an active grant for actorID (ErrNotFound) and, to remove someone else, the
// guardian role (ErrForbidden); a target without a grant is ErrNotFound. The target's push and ntfy opt-ins for the kid go too.
func (s *Store) RemoveAccess(ctx context.Context, kidID, actorID, parentID string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		role, err := actorRole(ctx, tx, kidID, actorID)
		if err != nil {
			return err
		}
		if actorID != parentID && role != model.RoleGuardian {
			return ErrForbidden
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM kid_access WHERE kid_id = ? AND parent_id = ?`, kidID, parentID)
		if err := oneRow(res, err); err != nil {
			return err
		}
		if err := deletePushOptins(ctx, tx, kidID, parentID); err != nil {
			return err
		}
		if err := deleteNtfyOptin(ctx, tx, kidID, parentID); err != nil {
			return err
		}
		return keepsActiveGuardian(ctx, tx, kidID)
	})
}

// actorRole re-asserts on tx what the middleware checked on the read pool, which may be stale by now.
// A disabled actor keeps the grant row but counts as having none: ErrNotFound.
func actorRole(ctx context.Context, tx *sql.Tx, kidID, actorID string) (model.Role, error) {
	var role model.Role
	err := tx.QueryRowContext(ctx,
		`SELECT a.role FROM kid_access a JOIN parents p ON p.id = a.parent_id
		 WHERE a.kid_id = ? AND a.parent_id = ? AND p.disabled = 0`, kidID, actorID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

func actorIsGuardian(ctx context.Context, tx *sql.Tx, kidID, actorID string) error {
	role, err := actorRole(ctx, tx, kidID, actorID)
	if err != nil {
		return err
	}
	if role != model.RoleGuardian {
		return ErrForbidden
	}
	return nil
}

// keepsActiveGuardian must read through tx: a count on the read pool would let two writers both pass.
func keepsActiveGuardian(ctx context.Context, tx *sql.Tx, kidID string) error {
	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM kid_access a JOIN parents p ON p.id = a.parent_id
		 WHERE a.kid_id = ? AND a.role = ? AND p.disabled = 0`, kidID, model.RoleGuardian).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrLastGuardian
	}
	return nil
}

// guardsEveryKidWithOthers is ErrLastGuardian when some kid the parent guards has no other active guardian.
func guardsEveryKidWithOthers(ctx context.Context, tx *sql.Tx, parentID string) error {
	var orphaned int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM kid_access a WHERE a.parent_id = ?1 AND a.role = ?2 AND NOT EXISTS (
		   SELECT 1 FROM kid_access g JOIN parents p ON p.id = g.parent_id
		   WHERE g.kid_id = a.kid_id AND g.role = ?2 AND p.disabled = 0)`, parentID, model.RoleGuardian).Scan(&orphaned); err != nil {
		return err
	}
	if orphaned > 0 {
		return ErrLastGuardian
	}
	return nil
}
