package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// The channel column's values.
const (
	ChannelPush = "push"
	ChannelNtfy = "ntfy"
)

// Channels are the delivery channels the server has configured.
type Channels struct{ Push, Ntfy bool }

// Delivery is one claimed send. Reminder is as read at the claim; DeliveryTarget re-reads it before the send.
type Delivery struct {
	ID, Channel, TargetID, ParentID string
	Attempts                        int
	Reminder                        model.Reminder
}

// Outcome is how a claimed send ended.
type Outcome string

const (
	OutcomeSent      Outcome = "sent"
	OutcomeGone      Outcome = "gone"
	OutcomeFailed    Outcome = "failed"
	OutcomeRetry     Outcome = "retry"
	OutcomeCancelled Outcome = "cancelled"
)

// Settle is a send's result; RetryAfter, for OutcomeRetry only, is the earliest next attempt the server asked for.
type Settle struct {
	Outcome    Outcome
	RetryAfter time.Time
}

// Target is what a send needs: Push for ChannelPush, Ntfy for ChannelNtfy. It holds secrets: never log it.
type Target struct {
	KidID, KidName string
	Push           PushTarget
	Ntfy           NtfyTarget
}

var errOutcome = errors.New("store: unknown delivery outcome")

// medicationOff reads only stored rows: medication is on by default in every band (DefaultEnabled).
const medicationOff = `EXISTS (SELECT 1 FROM kid_modules km WHERE km.kid_id = k.id AND km.module = 'medication' AND km.enabled = 0)`

// Materialise keeps Open value 14's rows for every active scheduled course of a kid with the medication module on.
func (s *Store) Materialise(ctx context.Context, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		type course struct{ id, zone string }
		var courses []course
		if err := eachRow(ctx, tx, `SELECT m.id, k.time_zone FROM medications m JOIN kids k ON k.id = m.kid_id
			WHERE m.active = 1 AND m.schedule_kind <> 'none' AND NOT `+medicationOff+` ORDER BY m.id`, nil, func(row rowScanner) error {
			var c course
			if err := row.Scan(&c.id, &c.zone); err != nil {
				return err
			}
			courses = append(courses, c)
			return nil
		}); err != nil {
			return err
		}
		for _, c := range courses {
			loc, err := loadZone(c.zone)
			if err != nil {
				return err
			}
			if err := materialiseCourse(ctx, tx, c.id, loc, now); err != nil {
				return err
			}
		}
		return nil
	})
}

// reminderCheck is a reminder with what decides at send time whether it may still go out.
type reminderCheck struct {
	model.Reminder
	KidName string
	Open    bool
}

// checkReminder: Open is false when the kid's medication module is off or the course is inactive or does not cover
// the due time; found is false for a reminder that is gone.
func checkReminder(ctx context.Context, q querier, id string) (c reminderCheck, found bool, err error) {
	var (
		due, start  int64
		zone        string
		endOn       sql.NullString
		active, off bool
	)
	err = q.QueryRowContext(ctx, `SELECT r.id, r.kid_id, r.medication_id, r.due_at, r.state, k.name, k.time_zone, m.active, m.start_at, m.end_on, `+
		medicationOff+` FROM reminders r JOIN kids k ON k.id = r.kid_id JOIN medications m ON m.id = r.medication_id WHERE r.id = ?`, id).
		Scan(&c.ID, &c.KidID, &c.MedicationID, &due, &c.State, &c.KidName, &zone, &active, &start, &endOn, &off)
	if errors.Is(err, sql.ErrNoRows) {
		return reminderCheck{}, false, nil
	}
	if err != nil {
		return reminderCheck{}, false, err
	}
	loc, err := loadZone(zone)
	if err != nil {
		return reminderCheck{}, false, err
	}
	c.DueAt = unixTime(due)
	c.Open = active && !off && due >= start && (!endOn.Valid || c.DueAt.In(loc).Format(time.DateOnly) <= endOn.String)
	return c, true, nil
}

// OpenDue cancels each due pending reminder a reminder-level gate closes and otherwise adds a pending delivery for each
// allowed target of each configured channel; it returns how many deliveries it added.
func (s *Store) OpenDue(ctx context.Context, now time.Time, ch Channels) (int, error) {
	var added int
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		added = 0
		var ids []string
		if err := eachRow(ctx, tx, `SELECT id FROM reminders WHERE state = 'pending' AND due_at <= ? AND due_at >= ? ORDER BY due_at, id`,
			[]any{now.Unix(), now.Add(-constants.ReminderExpiry).Unix()}, func(row rowScanner) error {
				var id string
				if err := row.Scan(&id); err != nil {
					return err
				}
				ids = append(ids, id)
				return nil
			}); err != nil {
			return err
		}
		insert := func(reminderID, channel, targetID, parentID string) error {
			res, err := tx.ExecContext(ctx, `INSERT INTO reminder_deliveries(id, reminder_id, channel, target_id, parent_id, state, attempts, next_attempt_at)
				VALUES (?, ?, ?, ?, ?, 'pending', 0, ?) ON CONFLICT(reminder_id, channel, target_id) DO NOTHING`,
				model.NewID(), reminderID, channel, targetID, parentID, now.Unix())
			if err != nil {
				return err
			}
			n, err := res.RowsAffected()
			added += int(n)
			return err
		}
		for _, id := range ids {
			c, found, err := checkReminder(ctx, tx, id)
			if err != nil {
				return err
			}
			if !found {
				continue
			}
			if !c.Open {
				if _, err := tx.ExecContext(ctx, `UPDATE reminders SET state = 'cancelled' WHERE id = ?`, id); err != nil {
					return err
				}
				continue
			}
			if ch.Push {
				targets, err := PushTargets(ctx, tx, c.KidID)
				if err != nil {
					return err
				}
				for _, t := range targets {
					if err := insert(id, ChannelPush, t.ID, t.ParentID); err != nil {
						return err
					}
				}
			}
			if ch.Ntfy {
				targets, err := NtfyTargets(ctx, tx, c.KidID)
				if err != nil {
					return err
				}
				for _, t := range targets {
					if err := insert(id, ChannelNtfy, t.ParentID, t.ParentID); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return added, nil
}

// ClaimDeliveries moves up to limit pending deliveries that are due for an attempt, and whose reminder is not past
// due + ReminderExpiry, to sending, counting the attempt; a claimed row is never returned again.
func (s *Store) ClaimDeliveries(ctx context.Context, now time.Time, limit int) ([]Delivery, error) {
	var out []Delivery
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		out = []Delivery{}
		if err := eachRow(ctx, tx, `SELECT d.id, d.channel, d.target_id, d.parent_id, d.attempts, r.id, r.kid_id, r.medication_id, r.due_at, r.state
			FROM reminder_deliveries d JOIN reminders r ON r.id = d.reminder_id
			WHERE d.state = 'pending' AND d.next_attempt_at <= ? AND r.due_at >= ? ORDER BY d.next_attempt_at, d.id LIMIT ?`,
			[]any{now.Unix(), now.Add(-constants.ReminderExpiry).Unix(), limit}, func(row rowScanner) error {
				var (
					d   Delivery
					due int64
				)
				if err := row.Scan(&d.ID, &d.Channel, &d.TargetID, &d.ParentID, &d.Attempts,
					&d.Reminder.ID, &d.Reminder.KidID, &d.Reminder.MedicationID, &due, &d.Reminder.State); err != nil {
					return err
				}
				d.Reminder.DueAt = unixTime(due)
				out = append(out, d)
				return nil
			}); err != nil {
			return err
		}
		for i := range out {
			if _, err := tx.ExecContext(ctx, `UPDATE reminder_deliveries SET state = 'sending', attempts = attempts + 1 WHERE id = ?`, out[i].ID); err != nil {
				return err
			}
			out[i].Attempts++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SettleDelivery settles a sending row (else ErrNotFound); gone deletes that subscription. A retry waits the next
// ReminderRetryDelays (RetryAfter if later): failed once they run out, expired past due + ReminderExpiry.
func (s *Store) SettleDelivery(ctx context.Context, id string, o Settle, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var (
			channel, target, parent string
			attempts                int
			due                     int64
		)
		err := tx.QueryRowContext(ctx, `SELECT d.channel, d.target_id, d.parent_id, d.attempts, r.due_at
			FROM reminder_deliveries d JOIN reminders r ON r.id = d.reminder_id WHERE d.id = ? AND d.state = 'sending'`, id).
			Scan(&channel, &target, &parent, &attempts, &due)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		state := string(o.Outcome)
		switch o.Outcome {
		case OutcomeSent, OutcomeFailed, OutcomeCancelled:
		case OutcomeGone:
			if channel == ChannelPush {
				if _, err := tx.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE id = ? AND parent_id = ?`, target, parent); err != nil {
					return err
				}
			}
		case OutcomeRetry:
			delays := constants.ReminderRetryDelays
			if attempts > len(delays) {
				state = string(OutcomeFailed)
				break
			}
			next := now.Add(delays[max(attempts-1, 0)])
			if o.RetryAfter.After(next) {
				next = o.RetryAfter
			}
			if next.After(unixTime(due).Add(constants.ReminderExpiry)) {
				state = "expired"
				break
			}
			_, err := tx.ExecContext(ctx, `UPDATE reminder_deliveries SET state = 'pending', next_attempt_at = ? WHERE id = ?`, next.Unix(), id)
			return err
		default:
			return errOutcome
		}
		_, err = tx.ExecContext(ctx, `UPDATE reminder_deliveries SET state = ?, settled_at = ? WHERE id = ?`, state, now.Unix(), id)
		return err
	})
}

// Expire turns pending deliveries and pending reminders past due + ReminderExpiry into expired; done and cancelled
// reminders keep their state.
func (s *Store) Expire(ctx context.Context, now time.Time) error {
	cutoff := now.Add(-constants.ReminderExpiry).Unix()
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE reminder_deliveries SET state = 'expired', settled_at = ?
			WHERE state = 'pending' AND reminder_id IN (SELECT id FROM reminders WHERE due_at < ?)`, now.Unix(), cutoff); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE reminders SET state = 'expired' WHERE state = 'pending' AND due_at < ?`, cutoff)
		return err
	})
}

// MarkUnconfirmed runs before the first claim: a row left sending may have been delivered, so it is never resent.
func (s *Store) MarkUnconfirmed(ctx context.Context) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE reminder_deliveries SET state = 'unconfirmed' WHERE state = 'sending'`)
		return err
	})
}

// DeliveryTarget re-reads d at send time: false (settle it cancelled) unless its reminder is pending, open and not
// past due + ReminderExpiry, and its target is still opted in by the same enabled parent with a grant.
func (s *Store) DeliveryTarget(ctx context.Context, d Delivery, now time.Time) (Target, bool, error) {
	var (
		tg Target
		ok bool
	)
	err := s.inReadTx(ctx, func(q querier) error {
		c, found, err := checkReminder(ctx, q, d.Reminder.ID)
		if err != nil || !found || !c.Open || c.State != "pending" || now.After(c.DueAt.Add(constants.ReminderExpiry)) {
			return err
		}
		tg.KidID, tg.KidName = c.KidID, c.KidName
		switch d.Channel {
		case ChannelPush:
			targets, err := PushTargets(ctx, q, c.KidID)
			for _, t := range targets {
				if t.ID == d.TargetID && t.ParentID == d.ParentID {
					tg.Push, ok = t, true
				}
			}
			return err
		case ChannelNtfy:
			targets, err := NtfyTargets(ctx, q, c.KidID)
			for _, t := range targets {
				if t.ParentID == d.TargetID && t.ParentID == d.ParentID {
					tg.Ntfy, ok = t, true
				}
			}
			return err
		}
		return nil
	})
	if err != nil || !ok {
		return Target{}, false, err
	}
	return tg, true, nil
}

// RemindersHealthy is Open value 30: false while a delivery is pending or sending more than ReminderHealthStale after
// its reminder's due time, or was settled failed or expired within the last ReminderHealthStale.
func (s *Store) RemindersHealthy(ctx context.Context, now time.Time) (bool, error) {
	stale := now.Add(-constants.ReminderHealthStale).Unix()
	var bad bool
	err := s.r.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM reminder_deliveries d JOIN reminders r ON r.id = d.reminder_id
		WHERE (d.state IN ('pending', 'sending') AND r.due_at < ?) OR (d.state IN ('failed', 'expired') AND d.settled_at >= ?))`,
		stale, stale).Scan(&bad)
	if err != nil {
		return false, err
	}
	return !bad, nil
}
