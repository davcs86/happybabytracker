package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// NtfyTarget is one topic a reminder may go to; the topic never leaves the publisher.
type NtfyTarget struct {
	ParentID, Topic string
}

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// newNtfyTopic is prefix-<NtfyTopicRandomLen base62>; bytes at or above the last multiple of 62 are redrawn,
// so every character is equally likely.
func newNtfyTopic(prefix string) string {
	const limit = 256 - 256%len(base62)
	out := make([]byte, 0, constants.NtfyTopicRandomLen)
	buf := make([]byte, constants.NtfyTopicRandomLen)
	for len(out) < constants.NtfyTopicRandomLen {
		rand.Read(buf) // never fails since Go 1.24
		for _, b := range buf {
			if int(b) < limit && len(out) < constants.NtfyTopicRandomLen {
				out = append(out, base62[int(b)%len(base62)])
			}
		}
	}
	return prefix + "-" + string(out)
}

// SetNtfyTopic gives parentID a new topic, or with regenerate replaces the topic and keeps the opt-ins.
// Without regenerate an existing topic is kept (created false); regenerate without one, or a disabled parent, is ErrNotFound.
func (s *Store) SetNtfyTopic(ctx context.Context, parentID, prefix string, regenerate bool, now time.Time) (t model.NtfyTopic, created bool, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		var enabled int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM parents WHERE id = ? AND disabled = 0`, parentID).Scan(&enabled); err != nil {
			return err
		}
		if enabled == 0 {
			return ErrNotFound
		}
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ntfy_topics WHERE parent_id = ?`, parentID).Scan(&n); err != nil {
			return err
		}
		switch {
		case n == 0 && regenerate:
			return ErrNotFound
		case regenerate:
			if _, err := tx.ExecContext(ctx, `UPDATE ntfy_topics SET topic = ?, created_at = ? WHERE parent_id = ?`,
				newNtfyTopic(prefix), now.Unix(), parentID); err != nil {
				return err
			}
		case n == 0:
			if _, err := tx.ExecContext(ctx, `INSERT INTO ntfy_topics(parent_id, topic, created_at) VALUES (?, ?, ?)`,
				parentID, newNtfyTopic(prefix), now.Unix()); err != nil {
				return err
			}
			created = true
		}
		t, err = readNtfyTopic(ctx, tx, parentID)
		return err
	})
	if err != nil {
		return model.NtfyTopic{}, false, err
	}
	return t, created, nil
}

// NtfyTopic is ErrNotFound when parentID has no topic.
func (s *Store) NtfyTopic(ctx context.Context, parentID string) (t model.NtfyTopic, err error) {
	err = s.inReadTx(ctx, func(q querier) error {
		t, err = readNtfyTopic(ctx, q, parentID)
		return err
	})
	return t, err
}

// DeleteNtfyTopic is ErrNotFound when parentID has no topic; its opt-ins cascade.
func (s *Store) DeleteNtfyTopic(ctx context.Context, parentID string) error {
	res, err := s.w.ExecContext(ctx, `DELETE FROM ntfy_topics WHERE parent_id = ?`, parentID)
	return oneRow(res, err)
}

// SetNtfyOptin opts actorID's topic in for kidID; ErrNotFound without an active grant or without a topic.
func (s *Store) SetNtfyOptin(ctx context.Context, kidID, actorID string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := ownTopicOnKid(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO ntfy_kid_optins(parent_id, kid_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, actorID, kidID)
		return err
	})
}

// ClearNtfyOptin is SetNtfyOptin's inverse; clearing an absent opt-in succeeds when the topic exists.
func (s *Store) ClearNtfyOptin(ctx context.Context, kidID, actorID string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := ownTopicOnKid(ctx, tx, kidID, actorID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM ntfy_kid_optins WHERE parent_id = ? AND kid_id = ?`, actorID, kidID)
		return err
	})
}

func ownTopicOnKid(ctx context.Context, tx *sql.Tx, kidID, actorID string) error {
	if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
		return err
	}
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ntfy_topics WHERE parent_id = ?`, actorID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// readNtfyTopic lists the kids filtered to current grants, by id; Kids is never nil.
func readNtfyTopic(ctx context.Context, q querier, parentID string) (model.NtfyTopic, error) {
	t := model.NtfyTopic{Kids: []string{}}
	err := q.QueryRowContext(ctx, `SELECT topic FROM ntfy_topics WHERE parent_id = ?`, parentID).Scan(&t.Topic)
	if errors.Is(err, sql.ErrNoRows) {
		return model.NtfyTopic{}, ErrNotFound
	}
	if err != nil {
		return model.NtfyTopic{}, err
	}
	err = eachRow(ctx, q, `SELECT o.kid_id FROM ntfy_kid_optins o
		JOIN kid_access a ON a.kid_id = o.kid_id AND a.parent_id = o.parent_id
		WHERE o.parent_id = ? ORDER BY o.kid_id`, []any{parentID}, func(row rowScanner) error {
		var kid string
		if err := row.Scan(&kid); err != nil {
			return err
		}
		t.Kids = append(t.Kids, kid)
		return nil
	})
	return t, err
}

// NtfyTargets are the topics opted in for kidID whose parent is enabled and holds a grant on it, by parent id.
func NtfyTargets(ctx context.Context, q querier, kidID string) ([]NtfyTarget, error) {
	out := []NtfyTarget{}
	err := eachRow(ctx, q, `SELECT t.parent_id, t.topic FROM ntfy_kid_optins o
		JOIN ntfy_topics t ON t.parent_id = o.parent_id
		JOIN parents p ON p.id = o.parent_id AND p.disabled = 0
		JOIN kid_access a ON a.kid_id = o.kid_id AND a.parent_id = o.parent_id
		WHERE o.kid_id = ? ORDER BY t.parent_id`, []any{kidID}, func(row rowScanner) error {
		var tg NtfyTarget
		if err := row.Scan(&tg.ParentID, &tg.Topic); err != nil {
			return err
		}
		out = append(out, tg)
		return nil
	})
	return out, err
}

// deleteNtfyOptin removes parentID's opt-in for kidID, never the topic; run it in the transaction that ends the grant.
func deleteNtfyOptin(ctx context.Context, tx *sql.Tx, kidID, parentID string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM ntfy_kid_optins WHERE kid_id = ? AND parent_id = ?`, kidID, parentID)
	return err
}
