package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// IssueToken stores the hash of a one-time token, replacing the parent's unused tokens of that kind.
func (s *Store) IssueToken(ctx context.Context, kind, parentID string, hash []byte, expires time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM one_time_tokens WHERE parent_id = ? AND kind = ? AND used_at IS NULL`, parentID, kind); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO one_time_tokens(token_hash, kind, parent_id, expires_at) VALUES (?, ?, ?, ?)`,
			hash, kind, parentID, expires.Unix())
		return err
	})
}

// PeekToken returns ErrNotFound for an unknown, used or expired token and for a disabled parent.
func (s *Store) PeekToken(ctx context.Context, hash []byte, now time.Time) (kind, username string, err error) {
	err = s.r.QueryRowContext(ctx,
		`SELECT t.kind, p.username FROM one_time_tokens t JOIN parents p ON p.id = t.parent_id
		 WHERE t.token_hash = ? AND t.used_at IS NULL AND t.expires_at > ? AND p.disabled = 0`,
		hash, now.Unix()).Scan(&kind, &username)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return kind, username, err
}

// RedeemToken spends the token, sets the parent's password and ends all their sessions in one transaction.
// A refusal (ErrNotFound) leaves the token unspent.
func (s *Store) RedeemToken(ctx context.Context, hash []byte, secretHash string, now time.Time) (parentID string, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE one_time_tokens SET used_at = ?
			 WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?
			   AND parent_id IN (SELECT id FROM parents WHERE disabled = 0)`,
			now.Unix(), hash, now.Unix())
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil {
			return err
		} else if n == 0 {
			return ErrNotFound
		}
		if err := tx.QueryRowContext(ctx, `SELECT parent_id FROM one_time_tokens WHERE token_hash = ?`, hash).Scan(&parentID); err != nil {
			return err
		}
		return setPassword(ctx, tx, parentID, secretHash, "")
	})
	if err != nil {
		return "", err
	}
	return parentID, nil
}
