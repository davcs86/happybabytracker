package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

// CreateSession stores a session by token hash; parentID is empty for an admin session.
func (s *Store) CreateSession(ctx context.Context, kind, parentID string, tokenHash []byte, now, expires time.Time) (string, error) {
	var parent any
	if parentID != "" {
		parent = parentID
	}
	id := model.NewID()
	_, err := s.w.ExecContext(ctx,
		`INSERT INTO sessions(id, token_hash, kind, parent_id, created_at, last_seen_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, tokenHash, kind, parent, now.UnixMilli(), now.UnixMilli(), expires.UnixMilli())
	if err != nil {
		return "", err
	}
	return id, nil
}

// SessionByToken returns ErrNotFound for an unknown or expired token and for a disabled parent's session.
func (s *Store) SessionByToken(ctx context.Context, tokenHash []byte, now time.Time) (model.Principal, error) {
	var (
		p       model.Principal
		parent  sql.NullString
		expires int64
	)
	err := s.r.QueryRowContext(ctx,
		`SELECT s.id, s.kind, s.parent_id, s.expires_at
		 FROM sessions s LEFT JOIN parents p ON p.id = s.parent_id
		 WHERE s.token_hash = ? AND s.expires_at > ? AND (p.id IS NULL OR p.disabled = 0)`,
		tokenHash, now.UnixMilli()).Scan(&p.SessionID, &p.Kind, &parent, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Principal{}, ErrNotFound
	}
	if err != nil {
		return model.Principal{}, err
	}
	p.ParentID = parent.String
	p.ExpiresAt = time.UnixMilli(expires).UTC()
	return p, nil
}

// TouchSession records the use and moves the expiry; a vanished session is not an error.
func (s *Store) TouchSession(ctx context.Context, id string, now, expires time.Time) error {
	_, err := s.w.ExecContext(ctx,
		`UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE id = ?`, now.UnixMilli(), expires.UnixMilli(), id)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.w.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteAdminSessions(ctx context.Context) error {
	_, err := s.w.ExecContext(ctx, `DELETE FROM sessions WHERE kind = 'admin'`)
	return err
}
