package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// PushTarget is one subscription a reminder may go to; the endpoint and keys never leave the sender.
type PushTarget struct {
	ID, ParentID, Endpoint, P256dh, Auth string
}

// UpsertDevice stores the subscription for parentID, created reporting a new endpoint. An endpoint bound to another
// parent moves to parentID and loses its opt-ins. ErrConflict at PushDevicesPerParent; ErrNotFound for a disabled parent.
func (s *Store) UpsertDevice(ctx context.Context, parentID, endpoint, p256dh, auth, label string, now time.Time) (d model.PushDevice, created bool, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		var enabled int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM parents WHERE id = ? AND disabled = 0`, parentID).Scan(&enabled); err != nil {
			return err
		}
		if enabled == 0 {
			return ErrNotFound
		}
		var id, owner string
		err := tx.QueryRowContext(ctx, `SELECT id, parent_id FROM push_subscriptions WHERE endpoint = ?`, endpoint).Scan(&id, &owner)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if owner == parentID {
			if _, err := tx.ExecContext(ctx, `UPDATE push_subscriptions SET p256dh = ?, auth = ?, label = ?, updated_at = ? WHERE id = ?`,
				p256dh, auth, label, now.Unix(), id); err != nil {
				return err
			}
		} else {
			var n int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE parent_id = ?`, parentID).Scan(&n); err != nil {
				return err
			}
			if n >= constants.PushDevicesPerParent {
				return ErrConflict
			}
			if id == "" {
				id, created = model.NewID(), true
				_, err = tx.ExecContext(ctx,
					`INSERT INTO push_subscriptions(id, parent_id, endpoint, p256dh, auth, label, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
					id, parentID, endpoint, p256dh, auth, label, now.Unix(), now.Unix())
			} else {
				if _, err := tx.ExecContext(ctx, `DELETE FROM push_kid_optins WHERE subscription_id = ?`, id); err != nil {
					return err
				}
				_, err = tx.ExecContext(ctx,
					`UPDATE push_subscriptions SET parent_id = ?, p256dh = ?, auth = ?, label = ?, created_at = ?, updated_at = ? WHERE id = ?`,
					parentID, p256dh, auth, label, now.Unix(), now.Unix(), id)
			}
			if err != nil {
				return err
			}
		}
		d, err = readDevice(ctx, tx, parentID, id)
		return err
	})
	if err != nil {
		return model.PushDevice{}, false, err
	}
	return d, created, nil
}

// RenameDevice is ErrNotFound for a device that is missing or not parentID's.
func (s *Store) RenameDevice(ctx context.Context, parentID, id, label string, now time.Time) (d model.PushDevice, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE push_subscriptions SET label = ?, updated_at = ? WHERE id = ? AND parent_id = ?`,
			label, now.Unix(), id, parentID)
		if err := oneRow(res, err); err != nil {
			return err
		}
		d, err = readDevice(ctx, tx, parentID, id)
		return err
	})
	return d, err
}

// DeleteDevice is ErrNotFound for a device that is missing or not parentID's; its opt-ins cascade.
func (s *Store) DeleteDevice(ctx context.Context, parentID, id string) error {
	res, err := s.w.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE id = ? AND parent_id = ?`, id, parentID)
	return oneRow(res, err)
}

// SetPushOptin opts actorID's device in for kidID; ErrNotFound without an active grant or for a device not theirs.
func (s *Store) SetPushOptin(ctx context.Context, kidID, actorID, deviceID string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := ownDeviceOnKid(ctx, tx, kidID, actorID, deviceID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO push_kid_optins(subscription_id, kid_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, deviceID, kidID)
		return err
	})
}

// ClearPushOptin is SetPushOptin's inverse; clearing an absent opt-in of an own device succeeds.
func (s *Store) ClearPushOptin(ctx context.Context, kidID, actorID, deviceID string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := ownDeviceOnKid(ctx, tx, kidID, actorID, deviceID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM push_kid_optins WHERE subscription_id = ? AND kid_id = ?`, deviceID, kidID)
		return err
	})
}

func ownDeviceOnKid(ctx context.Context, tx *sql.Tx, kidID, actorID, deviceID string) error {
	if _, err := actorRole(ctx, tx, kidID, actorID); err != nil {
		return err
	}
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE id = ? AND parent_id = ?`, deviceID, actorID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Devices returns the parent's devices, oldest first; never nil.
func (s *Store) Devices(ctx context.Context, parentID string) (out []model.PushDevice, err error) {
	err = s.inReadTx(ctx, func(q querier) error {
		out, err = readDevices(ctx, q, parentID, "")
		return err
	})
	return out, err
}

func readDevice(ctx context.Context, q querier, parentID, id string) (model.PushDevice, error) {
	list, err := readDevices(ctx, q, parentID, id)
	if err != nil {
		return model.PushDevice{}, err
	}
	if len(list) == 0 {
		return model.PushDevice{}, ErrNotFound
	}
	return list[0], nil
}

// readDevices lists parentID's devices (only id when not empty) with their kids, filtered to current grants.
func readDevices(ctx context.Context, q querier, parentID, id string) ([]model.PushDevice, error) {
	out := []model.PushDevice{}
	index := map[string]int{}
	err := eachRow(ctx, q, `SELECT id, label, created_at FROM push_subscriptions WHERE parent_id = ?1 AND (?2 = '' OR id = ?2)
		ORDER BY created_at, id`, []any{parentID, id}, func(row rowScanner) error {
		var (
			d       model.PushDevice
			created int64
		)
		if err := row.Scan(&d.ID, &d.Label, &created); err != nil {
			return err
		}
		d.CreatedAt, d.Kids = unixTime(created), []string{}
		index[d.ID] = len(out)
		out = append(out, d)
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = eachRow(ctx, q, `SELECT o.subscription_id, o.kid_id FROM push_kid_optins o
		JOIN push_subscriptions s ON s.id = o.subscription_id
		JOIN kid_access a ON a.kid_id = o.kid_id AND a.parent_id = s.parent_id
		WHERE s.parent_id = ?1 AND (?2 = '' OR s.id = ?2) ORDER BY o.kid_id`, []any{parentID, id}, func(row rowScanner) error {
		var sub, kid string
		if err := row.Scan(&sub, &kid); err != nil {
			return err
		}
		out[index[sub]].Kids = append(out[index[sub]].Kids, kid)
		return nil
	})
	return out, err
}

// PushTargets are the devices opted in for kidID whose parent is enabled and holds a grant on it, by id.
func PushTargets(ctx context.Context, q querier, kidID string) ([]PushTarget, error) {
	out := []PushTarget{}
	err := eachRow(ctx, q, `SELECT s.id, s.parent_id, s.endpoint, s.p256dh, s.auth FROM push_kid_optins o
		JOIN push_subscriptions s ON s.id = o.subscription_id
		JOIN parents p ON p.id = s.parent_id AND p.disabled = 0
		JOIN kid_access a ON a.kid_id = o.kid_id AND a.parent_id = s.parent_id
		WHERE o.kid_id = ? ORDER BY s.id`, []any{kidID}, func(row rowScanner) error {
		var t PushTarget
		if err := row.Scan(&t.ID, &t.ParentID, &t.Endpoint, &t.P256dh, &t.Auth); err != nil {
			return err
		}
		out = append(out, t)
		return nil
	})
	return out, err
}

// deletePushOptins removes parentID's opt-ins for kidID; run it in the transaction that ends the grant.
func deletePushOptins(ctx context.Context, tx *sql.Tx, kidID, parentID string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM push_kid_optins WHERE kid_id = ?
		AND subscription_id IN (SELECT id FROM push_subscriptions WHERE parent_id = ?)`, kidID, parentID)
	return err
}

// deleteChannels removes the parent's devices and ntfy topic; their opt-ins cascade.
func deleteChannels(ctx context.Context, tx *sql.Tx, parentID string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE parent_id = ?`, parentID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM ntfy_topics WHERE parent_id = ?`, parentID)
	return err
}
