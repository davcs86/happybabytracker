package store

import (
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

var allChannels = Channels{Push: true, Ntfy: true}

// remKit is a UTC kid "Mia" with guardian ana and caregiver bo, each with a device and a topic opted in for her,
// and a 14:00 course created at 10:00, so the 14:00 row exists.
type remKit struct {
	s             *Store
	kid           model.Kid
	ana, bo       model.Parent
	anaDev, boDev model.PushDevice
	course        model.Medication
}

func newRemKit(t *testing.T) remKit {
	t.Helper()
	k := remKit{s: openMem(t)}
	k.ana, k.bo = mustParent(t, k.s, "ana"), mustParent(t, k.s, "bo")
	k.kid = mustKidIn(t, k.s, k.ana.ID, "Mia", "UTC")
	mustGrant(t, k.s, k.kid.ID, k.ana.ID, "bo", model.RoleCaregiver)
	k.anaDev = mustDevice(t, k.s, k.ana.ID, "https://push.example/ana")
	k.boDev = mustDevice(t, k.s, k.bo.ID, "https://push.example/bo")
	for _, p := range []struct {
		parent string
		dev    string
	}{{k.ana.ID, k.anaDev.ID}, {k.bo.ID, k.boDev.ID}} {
		if err := k.s.SetPushOptin(ctx, k.kid.ID, p.parent, p.dev); err != nil {
			t.Fatal(err)
		}
		mustTopic(t, k.s, p.parent)
		if err := k.s.SetNtfyOptin(ctx, k.kid.ID, p.parent); err != nil {
			t.Fatal(err)
		}
	}
	k.course = mustCourse(t, k.s, k.kid.ID, k.ana.ID, timesCourse(clock(0, 0), "14:00"), clock(10, 0))
	return k
}

func reminderAt(t *testing.T, s *Store, medicationID string, due time.Time) string {
	t.Helper()
	var id string
	if err := s.w.QueryRow(`SELECT id FROM reminders WHERE medication_id = ? AND due_at = ?`, medicationID, due.Unix()).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (k remKit) reminder(t *testing.T) string {
	t.Helper()
	return reminderAt(t, k.s, k.course.ID, clock(14, 0))
}

func reminderState(t *testing.T, s *Store, id string) string {
	t.Helper()
	var state string
	if err := s.w.QueryRow(`SELECT state FROM reminders WHERE id = ?`, id).Scan(&state); err != nil {
		t.Fatal(err)
	}
	return state
}

type deliveryRow struct {
	Channel, TargetID, ParentID, State string
}

func deliveryRows(t *testing.T, s *Store, reminderID string) []deliveryRow {
	t.Helper()
	rows, err := s.w.Query(`SELECT channel, target_id, parent_id, state FROM reminder_deliveries WHERE reminder_id = ?
		ORDER BY channel, parent_id`, reminderID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := []deliveryRow{}
	for rows.Next() {
		var d deliveryRow
		if err := rows.Scan(&d.Channel, &d.TargetID, &d.ParentID, &d.State); err != nil {
			t.Fatal(err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func seedDelivery(t *testing.T, s *Store, reminderID, channel, targetID, parentID, state string, next time.Time, settled *int64) string {
	t.Helper()
	id := model.NewID()
	execOK(t, s, `INSERT INTO reminder_deliveries(id, reminder_id, channel, target_id, parent_id, state, attempts, next_attempt_at, settled_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)`, id, reminderID, channel, targetID, parentID, state, next.Unix(), settled)
	return id
}

func deliveryState(t *testing.T, s *Store, id string) (state string, next int64, settled *int64) {
	t.Helper()
	if err := s.w.QueryRow(`SELECT state, next_attempt_at, settled_at FROM reminder_deliveries WHERE id = ?`, id).Scan(&state, &next, &settled); err != nil {
		t.Fatal(err)
	}
	return state, next, settled
}

func mustOpenDue(t *testing.T, s *Store, now time.Time, ch Channels) int {
	t.Helper()
	n, err := s.OpenDue(ctx, now, ch)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func mustClaim(t *testing.T, s *Store, now time.Time, limit int) []Delivery {
	t.Helper()
	got, err := s.ClaimDeliveries(ctx, now, limit)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func mustSettle(t *testing.T, s *Store, id string, o Settle, now time.Time) {
	t.Helper()
	if err := s.SettleDelivery(ctx, id, o, now); err != nil {
		t.Fatal(err)
	}
}

func byTarget(ds []Delivery, channel, parentID string) Delivery {
	for _, d := range ds {
		if d.Channel == channel && d.ParentID == parentID {
			return d
		}
	}
	return Delivery{}
}

// inactiveCourse stores a course without rows, then switches it on behind the store's back, so only Materialise
// inserts its rows.
func inactiveCourse(t *testing.T, s *Store, kidID, actorID string, times ...string) model.Medication {
	t.Helper()
	v := timesCourse(clock(0, 0), times...)
	v.Active = false
	c := mustCourse(t, s, kidID, actorID, v, clock(10, 0))
	execOK(t, s, `UPDATE medications SET active = 1 WHERE id = ?`, c.ID)
	return c
}

func mustMaterialise(t *testing.T, s *Store, now time.Time) {
	t.Helper()
	if err := s.Materialise(ctx, now); err != nil {
		t.Fatal(err)
	}
}

func TestMedicationIsOnInEveryBand(t *testing.T) {
	// The store reads the medication module as on unless a stored row turns it off.
	for _, b := range []kidtime.Band{kidtime.Band0to11, kidtime.Band12to23, kidtime.Band24Plus} {
		if !DefaultEnabled("medication", b) {
			t.Fatalf("medication off by default in band %v", b)
		}
	}
}

func TestMaterialiseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := mustParent(t, s, "ana")
	kid := mustKidIn(t, s, p.ID, "Mia", "UTC")
	c := inactiveCourse(t, s, kid.ID, p.ID, "13:40", "14:00")
	if rows := reminderRows(t, s, c.ID); len(rows) != 0 {
		t.Fatalf("rows before Materialise = %v", rows)
	}
	now := clock(14, 5)
	mustMaterialise(t, s, now)
	mustMaterialise(t, s, now)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	mustMaterialise(t, s, now)
	want := []reminderRow{pending(clock(13, 40)), pending(clock(14, 0)), pending(clock(13, 40).AddDate(0, 0, 1))}
	if got := reminderRows(t, s, c.ID); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
}

func TestMaterialiseNeverCreatesAnOccurrenceMoreThan30MinutesOld(t *testing.T) {
	for _, tc := range []struct {
		now  time.Time
		want []reminderRow
	}{
		{clock(14, 30), []reminderRow{pending(clock(14, 0)), pending(clock(14, 0).AddDate(0, 0, 1))}},
		{clock(14, 31), []reminderRow{pending(clock(14, 0).AddDate(0, 0, 1))}},
	} {
		s := openMem(t)
		p := mustParent(t, s, "ana")
		kid := mustKidIn(t, s, p.ID, "Mia", "UTC")
		c := inactiveCourse(t, s, kid.ID, p.ID, "14:00")
		mustMaterialise(t, s, tc.now)
		if got := reminderRows(t, s, c.ID); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("at %v rows = %v, want %v", tc.now, got, tc.want)
		}
	}
}

func TestMaterialiseSkipsAKidWithTheModuleOff(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	off := mustKidIn(t, s, p.ID, "Mia", "UTC")
	on := mustKidIn(t, s, p.ID, "Leo", "UTC")
	mustSetModule(t, s, off.ID, p.ID, "medication", false)
	cOff := inactiveCourse(t, s, off.ID, p.ID, "14:00")
	cOn := inactiveCourse(t, s, on.ID, p.ID, "14:00")
	mustMaterialise(t, s, clock(14, 0))
	if got := reminderRows(t, s, cOff.ID); len(got) != 0 {
		t.Fatalf("module off rows = %v", got)
	}
	if got := reminderRows(t, s, cOn.ID); len(got) != 2 {
		t.Fatalf("module on rows = %v", got)
	}
}

func TestOpenDueCreatesOneDeliveryPerAllowedTarget(t *testing.T) {
	k := newRemKit(t)
	id := k.reminder(t)
	if n := mustOpenDue(t, k.s, clock(13, 59), allChannels); n != 0 || len(deliveryRows(t, k.s, id)) != 0 {
		t.Fatalf("before due: n = %d, rows = %v", n, deliveryRows(t, k.s, id))
	}
	if n := mustOpenDue(t, k.s, clock(14, 0), allChannels); n != 4 {
		t.Fatalf("n = %d, want 4", n)
	}
	want := []deliveryRow{
		{"ntfy", k.ana.ID, k.ana.ID, "pending"}, {"ntfy", k.bo.ID, k.bo.ID, "pending"},
		{"push", k.anaDev.ID, k.ana.ID, "pending"}, {"push", k.boDev.ID, k.bo.ID, "pending"},
	}
	want = sortedRows(want)
	if got := sortedRows(deliveryRows(t, k.s, id)); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	if n := mustOpenDue(t, k.s, clock(14, 1), allChannels); n != 0 || len(deliveryRows(t, k.s, id)) != 4 {
		t.Fatalf("second OpenDue: n = %d, rows = %v", n, deliveryRows(t, k.s, id))
	}
	if st := reminderState(t, k.s, id); st != "pending" {
		t.Fatalf("state = %s", st)
	}
}

func sortedRows(rows []deliveryRow) []deliveryRow {
	return slices.SortedFunc(slices.Values(rows), func(a, b deliveryRow) int {
		if a.Channel != b.Channel {
			if a.Channel < b.Channel {
				return -1
			}
			return 1
		}
		switch {
		case a.ParentID < b.ParentID:
			return -1
		case a.ParentID > b.ParentID:
			return 1
		}
		return 0
	})
}

func TestOpenDueGates(t *testing.T) {
	cancelling := map[string]func(t *testing.T, k remKit){
		"module off": func(t *testing.T, k remKit) { mustSetModule(t, k.s, k.kid.ID, k.ana.ID, "medication", false) },
		"inactive": func(t *testing.T, k remKit) {
			execOK(t, k.s, `UPDATE medications SET active = 0 WHERE id = ?`, k.course.ID)
		},
		"end_on passed": func(t *testing.T, k remKit) {
			execOK(t, k.s, `UPDATE medications SET end_on = '2026-11-19' WHERE id = ?`, k.course.ID)
		},
		"before start_at": func(t *testing.T, k remKit) {
			execOK(t, k.s, `UPDATE medications SET start_at = ? WHERE id = ?`, clock(15, 0).Unix(), k.course.ID)
		},
	}
	for name, gate := range cancelling {
		t.Run(name, func(t *testing.T) {
			k := newRemKit(t)
			id := k.reminder(t)
			gate(t, k)
			if n := mustOpenDue(t, k.s, clock(14, 0), allChannels); n != 0 {
				t.Fatalf("n = %d", n)
			}
			if st := reminderState(t, k.s, id); st != "cancelled" {
				t.Fatalf("state = %s, want cancelled", st)
			}
			if rows := deliveryRows(t, k.s, id); len(rows) != 0 {
				t.Fatalf("rows = %v", rows)
			}
		})
	}
	t.Run("end_on today stays open", func(t *testing.T) {
		k := newRemKit(t)
		execOK(t, k.s, `UPDATE medications SET end_on = '2026-11-20' WHERE id = ?`, k.course.ID)
		if n := mustOpenDue(t, k.s, clock(14, 0), allChannels); n != 4 || reminderState(t, k.s, k.reminder(t)) != "pending" {
			t.Fatalf("n = %d", n)
		}
	})
	gone := map[string]func(t *testing.T, k remKit){
		"course deleted": func(t *testing.T, k remKit) {
			if err := k.s.DeleteMedication(ctx, k.kid.ID, k.ana.ID, k.course.ID); err != nil {
				t.Fatal(err)
			}
		},
		"kid deleted": func(t *testing.T, k remKit) {
			if err := k.s.DeleteKid(ctx, k.kid.ID, k.ana.ID); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, del := range gone {
		t.Run(name, func(t *testing.T) {
			k := newRemKit(t)
			del(t, k)
			if n := mustOpenDue(t, k.s, clock(14, 0), allChannels); n != 0 || countOf(t, k.s, "reminders") != 0 ||
				countOf(t, k.s, "reminder_deliveries") != 0 {
				t.Fatalf("n = %d, reminders = %d", n, countOf(t, k.s, "reminders"))
			}
		})
	}
	skipBo := map[string]func(t *testing.T, k remKit){
		"revoked": func(t *testing.T, k remKit) {
			if err := k.s.RemoveAccess(ctx, k.kid.ID, k.ana.ID, k.bo.ID); err != nil {
				t.Fatal(err)
			}
		},
		"grant row gone": func(t *testing.T, k remKit) {
			execOK(t, k.s, `DELETE FROM kid_access WHERE kid_id = ? AND parent_id = ?`, k.kid.ID, k.bo.ID)
		},
		"disabled": func(t *testing.T, k remKit) {
			execOK(t, k.s, `UPDATE parents SET disabled = 1 WHERE id = ?`, k.bo.ID)
		},
		"not opted in": func(t *testing.T, k remKit) {
			if err := errors.Join(k.s.ClearPushOptin(ctx, k.kid.ID, k.bo.ID, k.boDev.ID), k.s.ClearNtfyOptin(ctx, k.kid.ID, k.bo.ID)); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, gate := range skipBo {
		t.Run(name, func(t *testing.T) {
			k := newRemKit(t)
			id := k.reminder(t)
			gate(t, k)
			if n := mustOpenDue(t, k.s, clock(14, 0), allChannels); n != 2 {
				t.Fatalf("n = %d, want 2", n)
			}
			want := []deliveryRow{{"ntfy", k.ana.ID, k.ana.ID, "pending"}, {"push", k.anaDev.ID, k.ana.ID, "pending"}}
			if got := deliveryRows(t, k.s, id); !reflect.DeepEqual(got, want) {
				t.Fatalf("rows = %v, want %v", got, want)
			}
			if st := reminderState(t, k.s, id); st != "pending" {
				t.Fatalf("state = %s, want pending", st)
			}
		})
	}
	t.Run("no allowed target stays pending", func(t *testing.T) {
		k := newRemKit(t)
		id := k.reminder(t)
		execOK(t, k.s, `DELETE FROM push_kid_optins`)
		execOK(t, k.s, `DELETE FROM ntfy_kid_optins`)
		if n := mustOpenDue(t, k.s, clock(14, 0), allChannels); n != 0 || reminderState(t, k.s, id) != "pending" {
			t.Fatalf("n = %d, state = %s", n, reminderState(t, k.s, id))
		}
	})
	t.Run("only pending reminders get deliveries", func(t *testing.T) {
		k := newRemKit(t)
		id := k.reminder(t)
		mustDose(t, k.s, k.kid.ID, k.ana.ID, doseOf(k.course.ID, clock(13, 50)), clock(13, 50))
		if n := mustOpenDue(t, k.s, clock(14, 0), allChannels); n != 0 || reminderState(t, k.s, id) != "done" {
			t.Fatalf("n = %d, state = %s", n, reminderState(t, k.s, id))
		}
	})
}

func TestNoChannelNoDelivery(t *testing.T) {
	k := newRemKit(t)
	id := k.reminder(t)
	if n := mustOpenDue(t, k.s, clock(14, 0), Channels{}); n != 0 || len(deliveryRows(t, k.s, id)) != 0 || reminderState(t, k.s, id) != "pending" {
		t.Fatalf("Channels{}: n = %d", n)
	}
	if n := mustOpenDue(t, k.s, clock(14, 1), Channels{Ntfy: true}); n != 2 {
		t.Fatalf("ntfy only: n = %d", n)
	}
	want := []deliveryRow{{"ntfy", k.ana.ID, k.ana.ID, "pending"}, {"ntfy", k.bo.ID, k.bo.ID, "pending"}}
	if got := sortedRows(deliveryRows(t, k.s, id)); !reflect.DeepEqual(got, sortedRows(want)) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	if n := mustOpenDue(t, k.s, clock(14, 2), Channels{Push: true}); n != 2 || len(deliveryRows(t, k.s, id)) != 4 {
		t.Fatalf("push on later: n = %d", n)
	}
}

func TestClaimDeliveriesReturnsWhatTheSenderNeeds(t *testing.T) {
	k := newRemKit(t)
	id := k.reminder(t)
	mustOpenDue(t, k.s, clock(14, 0), allChannels)
	got := mustClaim(t, k.s, clock(14, 0), 10)
	if len(got) != 4 {
		t.Fatalf("claimed %d, want 4", len(got))
	}
	d := byTarget(got, ChannelPush, k.bo.ID)
	want := model.Reminder{ID: id, KidID: k.kid.ID, MedicationID: k.course.ID, DueAt: clock(14, 0), State: "pending"}
	if d.ID == "" || d.TargetID != k.boDev.ID || d.Attempts != 1 || d.Reminder != want {
		t.Fatalf("delivery = %+v, want reminder %+v", d, want)
	}
	if st, _, _ := deliveryState(t, k.s, d.ID); st != "sending" {
		t.Fatalf("state = %s", st)
	}
	if again := mustClaim(t, k.s, clock(14, 0), 10); len(again) != 0 {
		t.Fatalf("claimed again %v", again)
	}
}

func TestClaimDeliveriesNeverReturnsARowTwice(t *testing.T) {
	k := newRemKit(t)
	id := k.reminder(t)
	const total = 40
	for i := range total {
		seedDelivery(t, k.s, id, ChannelPush, model.NewID()+string(rune('a'+i%26)), k.ana.ID, "pending", clock(14, 0), nil)
	}
	var (
		mu   sync.Mutex
		seen = map[string]int{}
		wg   sync.WaitGroup
		errs = make(chan error, 2)
	)
	for range 2 {
		wg.Go(func() {
			for {
				got, err := k.s.ClaimDeliveries(ctx, clock(14, 0), 3)
				if err != nil {
					errs <- err
					return
				}
				if len(got) == 0 {
					return
				}
				mu.Lock()
				for _, d := range got {
					seen[d.ID]++
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if len(seen) != total {
		t.Fatalf("claimed %d distinct rows, want %d", len(seen), total)
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("row %s claimed %d times", id, n)
		}
	}
	if n := scalar(t, k.s, `SELECT COUNT(*) FROM reminder_deliveries WHERE state = 'sending' AND attempts = 1`); n != total {
		t.Fatalf("sending with one attempt = %d", n)
	}
}

func TestRetrySchedule(t *testing.T) {
	k := newRemKit(t)
	id := k.reminder(t)
	d := seedDelivery(t, k.s, id, ChannelNtfy, k.ana.ID, k.ana.ID, "pending", clock(14, 0), nil)
	at := clock(14, 0)
	step := func(retryAfter time.Time, wantNext time.Time) {
		t.Helper()
		got := mustClaim(t, k.s, at, 5)
		if len(got) != 1 || got[0].ID != d {
			t.Fatalf("claim at %v = %v", at, got)
		}
		mustSettle(t, k.s, d, Settle{Outcome: OutcomeRetry, RetryAfter: retryAfter}, at)
		st, next, settled := deliveryState(t, k.s, d)
		if st != "pending" || next != wantNext.Unix() || settled != nil {
			t.Fatalf("after attempt %d: state %s next %v settled %v; want next %v", got[0].Attempts, st, unixTime(next), settled, wantNext)
		}
		if early := mustClaim(t, k.s, wantNext.Add(-time.Second), 5); len(early) != 0 {
			t.Fatalf("claimed before next attempt: %v", early)
		}
		at = wantNext
	}
	step(time.Time{}, clock(14, 0).Add(30*time.Second))
	step(clock(14, 1), clock(14, 0).Add(30*time.Second+2*time.Minute))
	step(clock(14, 20), clock(14, 20))
	got := mustClaim(t, k.s, at, 5)
	if len(got) != 1 || got[0].Attempts != 4 {
		t.Fatalf("fourth claim = %v", got)
	}
	mustSettle(t, k.s, d, Settle{Outcome: OutcomeRetry}, at)
	if st, _, settled := deliveryState(t, k.s, d); st != "failed" || settled == nil || *settled != at.Unix() {
		t.Fatalf("after the last delay: state %s settled %v", st, settled)
	}

	for _, tc := range []struct {
		name       string
		retryAfter time.Time
		now        time.Time
		want       string
	}{
		{"retry-after past the expiry", clock(14, 31), clock(14, 0), "expired"},
		{"delay past the expiry", time.Time{}, clock(14, 29).Add(31 * time.Second), "expired"},
		{"delay ends at the expiry", time.Time{}, clock(14, 29).Add(30 * time.Second), "pending"},
	} {
		d := seedDelivery(t, k.s, id, ChannelPush, model.NewID(), k.ana.ID, "pending", tc.now, nil)
		if got := mustClaim(t, k.s, tc.now, 5); len(got) != 1 {
			t.Fatalf("%s: claim = %v", tc.name, got)
		}
		mustSettle(t, k.s, d, Settle{Outcome: OutcomeRetry, RetryAfter: tc.retryAfter}, tc.now)
		if st, _, _ := deliveryState(t, k.s, d); st != tc.want {
			t.Fatalf("%s: state %s, want %s", tc.name, st, tc.want)
		}
		execOK(t, k.s, `DELETE FROM reminder_deliveries WHERE id = ?`, d)
	}
}

func TestClaimStopsAtTheExpiry(t *testing.T) {
	k := newRemKit(t)
	id := k.reminder(t)
	seedDelivery(t, k.s, id, ChannelNtfy, k.ana.ID, k.ana.ID, "pending", clock(14, 29), nil)
	if got := mustClaim(t, k.s, clock(14, 30).Add(time.Second), 5); len(got) != 0 {
		t.Fatalf("claimed past due + 30 min: %v", got)
	}
	if got := mustClaim(t, k.s, clock(14, 30), 5); len(got) != 1 {
		t.Fatalf("claim at due + 30 min = %v", got)
	}
}

func TestSettleOutcomes(t *testing.T) {
	k := newRemKit(t)
	mustOpenDue(t, k.s, clock(14, 0), allChannels)
	now := clock(14, 0).Add(5 * time.Second)
	got := mustClaim(t, k.s, clock(14, 0), 10)
	for o, d := range map[Outcome]Delivery{
		OutcomeSent:      byTarget(got, ChannelNtfy, k.ana.ID),
		OutcomeFailed:    byTarget(got, ChannelNtfy, k.bo.ID),
		OutcomeCancelled: byTarget(got, ChannelPush, k.bo.ID),
	} {
		mustSettle(t, k.s, d.ID, Settle{Outcome: o}, now)
		if st, _, settled := deliveryState(t, k.s, d.ID); st != string(o) || settled == nil || *settled != now.Unix() {
			t.Fatalf("%s: state %s settled %v", o, st, settled)
		}
	}
	sent := byTarget(got, ChannelNtfy, k.ana.ID)
	if err := k.s.SettleDelivery(ctx, sent.ID, Settle{Outcome: OutcomeFailed}, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("settling a settled row: %v", err)
	}
	if err := k.s.SettleDelivery(ctx, byTarget(got, ChannelPush, k.ana.ID).ID, Settle{Outcome: "bogus"}, now); err == nil {
		t.Fatal("unknown outcome accepted")
	}
}

func TestSettleGoneDeletesExactlyThatSubscription(t *testing.T) {
	k := newRemKit(t)
	mustOpenDue(t, k.s, clock(14, 0), allChannels)
	got := mustClaim(t, k.s, clock(14, 0), 10)
	d := byTarget(got, ChannelPush, k.ana.ID)
	mustSettle(t, k.s, d.ID, Settle{Outcome: OutcomeGone}, clock(14, 0))
	if st, _, settled := deliveryState(t, k.s, d.ID); st != "gone" || settled == nil {
		t.Fatalf("state %s settled %v", st, settled)
	}
	if n := scalar(t, k.s, `SELECT COUNT(*) FROM push_subscriptions WHERE id = ?`, k.anaDev.ID); n != 0 {
		t.Fatal("gone kept the subscription")
	}
	if n := scalar(t, k.s, `SELECT COUNT(*) FROM push_subscriptions WHERE id = ?`, k.boDev.ID); n != 1 {
		t.Fatal("gone deleted another subscription")
	}
	if n := countOf(t, k.s, "ntfy_topics"); n != 2 {
		t.Fatalf("topics = %d", n)
	}
}

func TestMarkUnconfirmed(t *testing.T) {
	k := newRemKit(t)
	id := k.reminder(t)
	mustOpenDue(t, k.s, clock(14, 0), allChannels)
	claimed := mustClaim(t, k.s, clock(14, 0), 2)
	if err := k.s.MarkUnconfirmed(ctx); err != nil {
		t.Fatal(err)
	}
	for _, d := range claimed {
		if st, _, _ := deliveryState(t, k.s, d.ID); st != "unconfirmed" {
			t.Fatalf("state = %s", st)
		}
	}
	rest := mustClaim(t, k.s, clock(14, 1), 10)
	if len(rest) != 2 {
		t.Fatalf("rest = %v", rest)
	}
	for _, d := range rest {
		if slices.ContainsFunc(claimed, func(c Delivery) bool { return c.ID == d.ID }) {
			t.Fatalf("unconfirmed row %s claimed again", d.ID)
		}
	}
	if n := mustOpenDue(t, k.s, clock(14, 2), allChannels); n != 0 || len(deliveryRows(t, k.s, id)) != 4 {
		t.Fatalf("OpenDue after restart: n = %d", n)
	}
}

func TestExpire(t *testing.T) {
	k := newRemKit(t)
	id := k.reminder(t)
	mustMaterialise(t, k.s, clock(14, 0))
	mustOpenDue(t, k.s, clock(14, 0), allChannels)
	sent := mustClaim(t, k.s, clock(14, 0), 1)[0]
	mustSettle(t, k.s, sent.ID, Settle{Outcome: OutcomeSent}, clock(14, 0))
	if err := k.s.Expire(ctx, clock(14, 30)); err != nil {
		t.Fatal(err)
	}
	if st := reminderState(t, k.s, id); st != "pending" {
		t.Fatalf("at due + 30 min state = %s", st)
	}
	later := clock(14, 30).Add(time.Second)
	if err := k.s.Expire(ctx, later); err != nil {
		t.Fatal(err)
	}
	if st := reminderState(t, k.s, id); st != "expired" {
		t.Fatalf("state = %s", st)
	}
	states := map[string]int{}
	for _, d := range deliveryRows(t, k.s, id) {
		states[d.State]++
	}
	if !reflect.DeepEqual(states, map[string]int{"sent": 1, "expired": 3}) {
		t.Fatalf("delivery states = %v", states)
	}
	if n := scalar(t, k.s, `SELECT COUNT(*) FROM reminder_deliveries WHERE state = 'expired' AND settled_at = ?`, later.Unix()); n != 3 {
		t.Fatalf("expired with settled_at = %d", n)
	}
	if n := scalar(t, k.s, `SELECT COUNT(*) FROM reminders WHERE due_at > ? AND state = 'pending'`, clock(14, 0).Unix()); n != 1 {
		t.Fatalf("the next row was touched: %d pending", n)
	}
}

func TestExpireNeverTouchesDoneOrCancelled(t *testing.T) {
	k := newRemKit(t)
	done := k.reminder(t)
	other := mustCourse(t, k.s, k.kid.ID, k.ana.ID, timesCourse(clock(0, 0), "14:00"), clock(10, 0))
	cancelled := reminderAt(t, k.s, other.ID, clock(14, 0))
	mustDose(t, k.s, k.kid.ID, k.ana.ID, doseOf(k.course.ID, clock(14, 0)), clock(14, 0))
	execOK(t, k.s, `UPDATE reminders SET state = 'cancelled' WHERE id = ?`, cancelled)
	if err := k.s.Expire(ctx, clock(16, 0)); err != nil {
		t.Fatal(err)
	}
	if st := reminderState(t, k.s, done); st != "done" {
		t.Fatalf("done became %s", st)
	}
	if st := reminderState(t, k.s, cancelled); st != "cancelled" {
		t.Fatalf("cancelled became %s", st)
	}
}

func TestDeliveryTargetGivesWhatTheSenderNeeds(t *testing.T) {
	k := newRemKit(t)
	mustOpenDue(t, k.s, clock(14, 0), allChannels)
	got := mustClaim(t, k.s, clock(14, 0), 10)
	boTopic, err := k.s.NtfyTopic(ctx, k.bo.ID)
	if err != nil {
		t.Fatal(err)
	}
	tg, ok, err := k.s.DeliveryTarget(ctx, byTarget(got, ChannelPush, k.bo.ID), clock(14, 0))
	if err != nil || !ok {
		t.Fatalf("push: ok %v err %v", ok, err)
	}
	wantPush := PushTarget{ID: k.boDev.ID, ParentID: k.bo.ID, Endpoint: "https://push.example/bo", P256dh: testP256dh, Auth: testAuth}
	if tg.KidID != k.kid.ID || tg.KidName != "Mia" || tg.Push != wantPush || tg.Ntfy != (NtfyTarget{}) {
		t.Fatalf("push target = %+v", tg)
	}
	tg, ok, err = k.s.DeliveryTarget(ctx, byTarget(got, ChannelNtfy, k.bo.ID), clock(14, 0))
	if err != nil || !ok || tg.KidName != "Mia" || tg.Ntfy != (NtfyTarget{ParentID: k.bo.ID, Topic: boTopic.Topic}) || tg.Push != (PushTarget{}) {
		t.Fatalf("ntfy target = %+v, ok %v, err %v", tg, ok, err)
	}
	regen, _, err := k.s.SetNtfyTopic(ctx, k.bo.ID, testPrefix, true, clock(14, 0))
	if err != nil {
		t.Fatal(err)
	}
	tg, ok, err = k.s.DeliveryTarget(ctx, byTarget(got, ChannelNtfy, k.bo.ID), clock(14, 0))
	if err != nil || !ok || tg.Ntfy.Topic != regen.Topic || regen.Topic == boTopic.Topic {
		t.Fatalf("after regenerate: ok %v err %v, topic re-read %v", ok, err, tg.Ntfy.Topic == regen.Topic)
	}
}

// refusesBo claims every delivery at 14:00, runs change, then wants bo's (or every) delivery refused and the rest allowed.
func refusesBo(t *testing.T, k remKit, all bool, change func()) {
	t.Helper()
	mustOpenDue(t, k.s, clock(14, 0), allChannels)
	got := mustClaim(t, k.s, clock(14, 0), 10)
	if len(got) != 4 {
		t.Fatalf("claimed %d", len(got))
	}
	change()
	for _, d := range got {
		_, ok, err := k.s.DeliveryTarget(ctx, d, clock(14, 0).Add(10*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if want := !all && d.ParentID != k.bo.ID; ok != want {
			t.Fatalf("%s for parent %s: ok %v, want %v", d.Channel, d.ParentID, ok, want)
		}
	}
}

func TestDeliveryTargetRefusesADoneReminder(t *testing.T) {
	t.Run("dose between delivery creation and claim", func(t *testing.T) {
		k := newRemKit(t)
		mustOpenDue(t, k.s, clock(14, 0), allChannels)
		mustDose(t, k.s, k.kid.ID, k.ana.ID, doseOf(k.course.ID, clock(14, 0)), clock(14, 0))
		got := mustClaim(t, k.s, clock(14, 0), 10)
		if len(got) != 4 {
			t.Fatalf("claimed %d", len(got))
		}
		for _, d := range got {
			if _, ok, err := k.s.DeliveryTarget(ctx, d, clock(14, 0)); ok || err != nil {
				t.Fatalf("done reminder: ok %v err %v", ok, err)
			}
		}
	})
	t.Run("dose between claim and send", func(t *testing.T) {
		k := newRemKit(t)
		refusesBo(t, k, true, func() {
			mustDose(t, k.s, k.kid.ID, k.ana.ID, doseOf(k.course.ID, clock(14, 0)), clock(14, 0))
		})
	})
	t.Run("cancelled", func(t *testing.T) {
		k := newRemKit(t)
		refusesBo(t, k, true, func() { execOK(t, k.s, `UPDATE reminders SET state = 'cancelled'`) })
	})
}

func TestDeliveryTargetRefusesADeletedDevice(t *testing.T) {
	k := newRemKit(t)
	mustOpenDue(t, k.s, clock(14, 0), Channels{Push: true})
	got := mustClaim(t, k.s, clock(14, 0), 10)
	if err := k.s.DeleteDevice(ctx, k.bo.ID, k.boDev.ID); err != nil {
		t.Fatal(err)
	}
	for _, d := range got {
		_, ok, err := k.s.DeliveryTarget(ctx, d, clock(14, 0))
		if err != nil || ok != (d.ParentID == k.ana.ID) {
			t.Fatalf("parent %s: ok %v err %v", d.ParentID, ok, err)
		}
	}
}

func TestDeliveryTargetRefusesAMovedDevice(t *testing.T) {
	t.Run("re-registered by another parent", func(t *testing.T) {
		k := newRemKit(t)
		mustOpenDue(t, k.s, clock(14, 0), Channels{Push: true})
		got := mustClaim(t, k.s, clock(14, 0), 10)
		moved := mustDevice(t, k.s, k.ana.ID, "https://push.example/bo")
		if err := k.s.SetPushOptin(ctx, k.kid.ID, k.ana.ID, moved.ID); err != nil {
			t.Fatal(err)
		}
		d := byTarget(got, ChannelPush, k.bo.ID)
		if _, ok, err := k.s.DeliveryTarget(ctx, d, clock(14, 0)); ok || err != nil {
			t.Fatalf("moved device: ok %v err %v", ok, err)
		}
	})
	t.Run("same id, another parent", func(t *testing.T) {
		k := newRemKit(t)
		mustOpenDue(t, k.s, clock(14, 0), Channels{Push: true})
		got := mustClaim(t, k.s, clock(14, 0), 10)
		execOK(t, k.s, `UPDATE push_subscriptions SET parent_id = ? WHERE id = ?`, k.ana.ID, k.boDev.ID)
		d := byTarget(got, ChannelPush, k.bo.ID)
		if _, ok, err := k.s.DeliveryTarget(ctx, d, clock(14, 0)); ok || err != nil {
			t.Fatalf("re-owned device: ok %v err %v", ok, err)
		}
	})
}

func TestDeliveryTargetRefusesARevokedOptin(t *testing.T) {
	k := newRemKit(t)
	refusesBo(t, k, false, func() {
		if err := errors.Join(k.s.ClearPushOptin(ctx, k.kid.ID, k.bo.ID, k.boDev.ID), k.s.ClearNtfyOptin(ctx, k.kid.ID, k.bo.ID)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDeliveryTargetRefusesAfterARevokeBetweenClaimAndSend(t *testing.T) {
	t.Run("RemoveAccess", func(t *testing.T) {
		k := newRemKit(t)
		refusesBo(t, k, false, func() {
			if err := k.s.RemoveAccess(ctx, k.kid.ID, k.ana.ID, k.bo.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("grant row only", func(t *testing.T) {
		k := newRemKit(t)
		refusesBo(t, k, false, func() {
			execOK(t, k.s, `DELETE FROM kid_access WHERE kid_id = ? AND parent_id = ?`, k.kid.ID, k.bo.ID)
		})
	})
	t.Run("disabled", func(t *testing.T) {
		k := newRemKit(t)
		refusesBo(t, k, false, func() { execOK(t, k.s, `UPDATE parents SET disabled = 1 WHERE id = ?`, k.bo.ID) })
	})
}

func TestDeliveryTargetRechecksTheReminderGates(t *testing.T) {
	for name, change := range map[string]func(t *testing.T, k remKit){
		"module off": func(t *testing.T, k remKit) { mustSetModule(t, k.s, k.kid.ID, k.ana.ID, "medication", false) },
		"inactive":   func(t *testing.T, k remKit) { execOK(t, k.s, `UPDATE medications SET active = 0`) },
		"end_on":     func(t *testing.T, k remKit) { execOK(t, k.s, `UPDATE medications SET end_on = '2026-11-19'`) },
		"course deleted": func(t *testing.T, k remKit) {
			if err := k.s.DeleteMedication(ctx, k.kid.ID, k.ana.ID, k.course.ID); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			k := newRemKit(t)
			refusesBo(t, k, true, func() { change(t, k) })
		})
	}
	t.Run("past due + 30 min", func(t *testing.T) {
		k := newRemKit(t)
		mustOpenDue(t, k.s, clock(14, 0), allChannels)
		d := mustClaim(t, k.s, clock(14, 0), 1)[0]
		if _, ok, err := k.s.DeliveryTarget(ctx, d, clock(14, 30)); !ok || err != nil {
			t.Fatalf("at due + 30 min: ok %v err %v", ok, err)
		}
		if _, ok, err := k.s.DeliveryTarget(ctx, d, clock(14, 30).Add(time.Second)); ok || err != nil {
			t.Fatalf("past due + 30 min: ok %v err %v", ok, err)
		}
	})
}

func TestRemindersHealthy(t *testing.T) {
	due := clock(14, 0)
	stale := constants.ReminderHealthStale
	justIn, justOut := stale-time.Second, stale+time.Second
	settledAt := func(d time.Duration) *int64 { v := due.Add(-d).Unix(); return &v }
	for _, tc := range []struct {
		name    string
		state   string
		settled *int64
		now     time.Time
		want    bool
	}{
		{"no deliveries", "", nil, due.Add(time.Hour), true},
		{"pending 14:59 after due", "pending", nil, due.Add(justIn), true},
		{"pending 15:01 after due", "pending", nil, due.Add(justOut), false},
		{"sending 14:59 after due", "sending", nil, due.Add(justIn), true},
		{"sending 15:01 after due", "sending", nil, due.Add(justOut), false},
		{"failed 14:59 ago", "failed", settledAt(justIn), due, false},
		{"failed 15:01 ago", "failed", settledAt(justOut), due, true},
		{"expired 14:59 ago", "expired", settledAt(justIn), due, false},
		{"expired 15:01 ago", "expired", settledAt(justOut), due, true},
		{"sent long ago", "sent", settledAt(0), due.Add(time.Hour), true},
		{"unconfirmed", "unconfirmed", nil, due.Add(time.Hour), true},
		{"cancelled", "cancelled", settledAt(0), due.Add(time.Minute), true},
		{"gone", "gone", settledAt(0), due.Add(time.Minute), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k := newRemKit(t)
			if tc.state != "" {
				seedDelivery(t, k.s, k.reminder(t), ChannelPush, k.anaDev.ID, k.ana.ID, tc.state, due, tc.settled)
			}
			got, err := k.s.RemindersHealthy(ctx, tc.now)
			if err != nil || got != tc.want {
				t.Fatalf("healthy = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}
