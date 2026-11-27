package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

// OIDCLogin issues the session of the parent or admin holding (iss, sub); a parent session keeps deviceLabel.
// A disabled linked parent's sessions, push devices and ntfy topic are purged and committed before ErrNotLinked.
func (s *Store) OIDCLogin(ctx context.Context, iss, sub string, tokenHash []byte, now, parentExpires, adminExpires time.Time, deviceLabel string) (kind string, expires time.Time, err error) {
	tx, err := s.w.BeginTx(ctx, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	defer tx.Rollback()

	var (
		principalID string
		disabled    bool
	)
	err = tx.QueryRowContext(ctx,
		`SELECT p.id, p.disabled FROM identities i JOIN parents p ON p.id = i.parent_id
		 WHERE i.provider = ? AND i.provider <> ? AND i.subject = ?`, iss, providerLocal, sub).Scan(&principalID, &disabled)
	switch {
	case err == nil && disabled:
		if err := deleteChannels(ctx, tx, principalID); err != nil {
			return "", time.Time{}, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE parent_id = ?`, principalID); err != nil {
			return "", time.Time{}, err
		}
		if err := tx.Commit(); err != nil {
			return "", time.Time{}, err
		}
		return "", time.Time{}, ErrNotLinked
	case err == nil:
		kind, expires = "parent", parentExpires
	case errors.Is(err, sql.ErrNoRows):
		err = tx.QueryRowContext(ctx,
			`SELECT admin_id FROM admin_identities WHERE provider = ? AND provider <> ? AND subject = ?`,
			iss, providerLocal, sub).Scan(&principalID)
		if errors.Is(err, sql.ErrNoRows) {
			return "", time.Time{}, ErrNotLinked
		}
		if err != nil {
			return "", time.Time{}, err
		}
		kind, expires, deviceLabel = "admin", adminExpires, ""
	default:
		return "", time.Time{}, err
	}
	if _, err := insertSession(ctx, tx, kind, principalID, tokenHash, now, expires, deviceLabel); err != nil {
		return "", time.Time{}, err
	}
	if err := tx.Commit(); err != nil {
		return "", time.Time{}, err
	}
	return kind, expires, nil
}

// RedeemInviteOIDC spends an invite, links (iss, sub) to its parent, ends their sessions and stores the new one.
// Any refusal rolls back, so the token stays usable.
func (s *Store) RedeemInviteOIDC(ctx context.Context, tokenHash []byte, iss, sub string, sessionHash []byte, now, expires time.Time, deviceLabel string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE one_time_tokens SET used_at = ?
			 WHERE token_hash = ? AND kind = ? AND used_at IS NULL AND expires_at > ?
			   AND parent_id IN (SELECT id FROM parents WHERE disabled = 0)`,
			now.Unix(), tokenHash, model.TokenInvite, now.Unix())
		if err := oneRow(res, err); err != nil {
			return err
		}
		var parentID string
		if err := tx.QueryRowContext(ctx, `SELECT parent_id FROM one_time_tokens WHERE token_hash = ?`, tokenHash).Scan(&parentID); err != nil {
			return err
		}
		taken, err := pairTaken(ctx, tx, iss, sub)
		if err != nil {
			return err
		}
		var hasIssuer bool
		if err := tx.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM identities WHERE parent_id = ? AND provider = ? AND provider <> ?)`,
			parentID, iss, providerLocal).Scan(&hasIssuer); err != nil {
			return err
		}
		if taken || hasIssuer {
			return ErrAlreadyLinked
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO identities(parent_id, provider, subject) VALUES (?, ?, ?)`, parentID, iss, sub); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE parent_id = ?`, parentID); err != nil {
			return err
		}
		_, err = insertSession(ctx, tx, "parent", parentID, sessionHash, now, expires, deviceLabel)
		return err
	})
}

// pairTaken reports whether a non-local (iss, sub) belongs to any parent or admin.
func pairTaken(ctx context.Context, tx *sql.Tx, iss, sub string) (bool, error) {
	var taken bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM identities WHERE provider = ? AND provider <> ? AND subject = ?)
		     OR EXISTS (SELECT 1 FROM admin_identities WHERE provider = ? AND provider <> ? AND subject = ?)`,
		iss, providerLocal, sub, iss, providerLocal, sub).Scan(&taken)
	return taken, err
}
