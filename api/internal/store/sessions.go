package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

// CreateSession stores a session by token hash; principalID is the admin id when kind is "admin", else the parent id.
func (s *Store) CreateSession(ctx context.Context, kind, principalID string, tokenHash []byte, now, expires time.Time, deviceLabel string) (string, error) {
	var parent, admin any
	if principalID != "" && kind == "admin" {
		admin = principalID
	} else if principalID != "" {
		parent = principalID
	}
	var label any
	if deviceLabel != "" {
		label = deviceLabel
	}
	id := model.NewID()
	_, err := s.w.ExecContext(ctx,
		`INSERT INTO sessions(id, token_hash, kind, parent_id, admin_id, created_at, last_seen_at, expires_at, device_label) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tokenHash, kind, parent, admin, now.Unix(), now.Unix(), expires.Unix(), label)
	if err != nil {
		return "", err
	}
	return id, nil
}

// SessionByToken returns ErrNotFound for an unknown or expired token and for a disabled parent's session.
func (s *Store) SessionByToken(ctx context.Context, tokenHash []byte, now time.Time) (model.Principal, error) {
	var (
		p             model.Principal
		parent, admin sql.NullString
		expires       int64
	)
	err := s.r.QueryRowContext(ctx,
		`SELECT s.id, s.kind, s.parent_id, s.admin_id, s.expires_at
		 FROM sessions s LEFT JOIN parents p ON p.id = s.parent_id
		 WHERE s.token_hash = ? AND s.expires_at > ? AND (p.id IS NULL OR p.disabled = 0)`,
		tokenHash, now.Unix()).Scan(&p.SessionID, &p.Kind, &parent, &admin, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Principal{}, ErrNotFound
	}
	if err != nil {
		return model.Principal{}, err
	}
	p.ParentID, p.AdminID = parent.String, admin.String
	p.ExpiresAt = time.Unix(expires, 0).UTC()
	return p, nil
}

// TouchSession records the use and moves the expiry; a vanished session is not an error.
func (s *Store) TouchSession(ctx context.Context, id string, now, expires time.Time) error {
	_, err := s.w.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE id = ?`, now.Unix(), expires.Unix(), id)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.w.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// ParentActivity maps each parent that has a session to the newest last_seen_at among them.
func (s *Store) ParentActivity(ctx context.Context) (map[string]time.Time, error) {
	rows, err := s.r.QueryContext(ctx, `SELECT parent_id, MAX(last_seen_at) FROM sessions WHERE parent_id IS NOT NULL GROUP BY parent_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]time.Time{}
	for rows.Next() {
		var (
			id   string
			seen int64
		)
		if err := rows.Scan(&id, &seen); err != nil {
			return nil, err
		}
		out[id] = time.Unix(seen, 0).UTC()
	}
	return out, rows.Err()
}

// ListSessions returns the parent's unexpired sessions, newest first; never nil.
func (s *Store) ListSessions(ctx context.Context, parentID string, now time.Time) ([]model.SessionInfo, error) {
	rows, err := s.r.QueryContext(ctx,
		`SELECT id, created_at, last_seen_at, device_label FROM sessions WHERE parent_id = ? AND expires_at > ? ORDER BY created_at DESC, id DESC`,
		parentID, now.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SessionInfo{}
	for rows.Next() {
		var (
			si            model.SessionInfo
			created, seen int64
			label         sql.NullString
		)
		if err := rows.Scan(&si.ID, &created, &seen, &label); err != nil {
			return nil, err
		}
		si.DeviceLabel = label.String
		si.CreatedAt, si.LastSeenAt = time.Unix(created, 0).UTC(), time.Unix(seen, 0).UTC()
		out = append(out, si)
	}
	return out, rows.Err()
}

// DeleteParentSession ends one of the parent's own sessions; ErrNotFound when it is not theirs.
func (s *Store) DeleteParentSession(ctx context.Context, parentID, sessionID string) error {
	res, err := s.w.ExecContext(ctx, `DELETE FROM sessions WHERE id = ? AND parent_id = ?`, sessionID, parentID)
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
