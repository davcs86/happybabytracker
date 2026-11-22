package store

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

func runningNursing(start time.Time, side string) model.NursingSession {
	return model.NursingSession{StartedAt: start, Side: &side}
}

func finishedNursing(start, end time.Time, left, right int) model.NursingSession {
	return model.NursingSession{StartedAt: start, EndedAt: &end, LeftSeconds: left, RightSeconds: right}
}

func mustNursing(t *testing.T, s *Store, kidID, actorID string, n model.NursingSession) model.NursingSession {
	t.Helper()
	got, err := s.CreateNursing(ctx, kidID, actorID, n, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func keepNursing(n model.NursingSession) (model.NursingSession, error) { return n, nil }

func TestNursingCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := finishedNursing(t0.Add(-time.Hour), t0.Add(-30*time.Minute), 600, 900)
	in.Note = strp("calm")
	created, err := s.CreateNursing(ctx, k.ID, ana.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || created.LeftSeconds != 600 ||
		created.RightSeconds != 900 || created.Side != nil || created.SideSince != nil || created.EndedAt == nil {
		t.Fatalf("created = %+v", created)
	}
	if got, err := s.Nursing(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Nursing = %+v, %v; want %+v", got, err, created)
	}
	later := t0.Add(time.Hour)
	updated, err := s.UpdateNursing(ctx, k.ID, bo.ID, created.ID, later, func(n model.NursingSession) (model.NursingSession, error) {
		n.LeftSeconds, n.RightSeconds, n.Note = 1, 2, nil
		return n, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := created
	want.LeftSeconds, want.RightSeconds, want.Note, want.UpdatedBy, want.UpdatedAt = 1, 2, nil, bo.ID, later
	if !reflect.DeepEqual(updated, want) {
		t.Fatalf("updated = %+v, want %+v", updated, want)
	}
	if got, err := s.Nursing(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("after update = %+v, %v", got, err)
	}
	if list, err := s.ListNursing(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("ListNursing = %+v, %v", list, err)
	}
	if err := s.DeleteNursing(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Nursing(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
}

func TestNursingBelongsToItsKidAndItsWriter(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	mustGrant(t, s, mine.ID, ana.ID, "bo", model.RoleCaregiver)
	n := mustNursing(t, s, mine.ID, ana.ID, runningNursing(t0, "left"))

	if _, err := s.Nursing(ctx, other.ID, n.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	if _, err := s.SwitchNursingSide(ctx, other.ID, ana.ID, n.ID, "right", t0, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("switch through another kid: %v", err)
	}
	if _, err := s.EndNursing(ctx, other.ID, ana.ID, n.ID, t0, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("end through another kid: %v", err)
	}
	if err := s.DeleteNursing(ctx, other.ID, ana.ID, n.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	for name, actor := range map[string]string{"no grant": cy.ID, "disabled caregiver": bo.ID} {
		if _, err := s.CreateNursing(ctx, mine.ID, actor, finishedNursing(t0, t0, 0, 0), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if _, err := s.UpdateNursing(ctx, mine.ID, actor, n.ID, t0, keepNursing); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if _, err := s.SwitchNursingSide(ctx, mine.ID, actor, n.ID, "right", t0, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s switch: %v", name, err)
		}
		if _, err := s.EndNursing(ctx, mine.ID, actor, n.ID, t0, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s end: %v", name, err)
		}
		if err := s.DeleteNursing(ctx, mine.ID, actor, n.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if got, err := s.Nursing(ctx, mine.ID, n.ID); err != nil || !reflect.DeepEqual(got, n) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestNursingStartSetsSideSince(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	start := t0.Add(-time.Minute)
	n := mustNursing(t, s, k.ID, ana.ID, runningNursing(start, "right"))
	if n.EndedAt != nil || n.Side == nil || *n.Side != "right" || n.SideSince == nil || !n.SideSince.Equal(start) ||
		n.LeftSeconds != 0 || n.RightSeconds != 0 {
		t.Fatalf("started = %+v", n)
	}
	if got, err := s.RunningNursing(ctx, k.ID); err != nil || got == nil || !reflect.DeepEqual(*got, n) {
		t.Fatalf("RunningNursing = %+v, %v", got, err)
	}
}

func TestNursingSwitchAccumulatesSeconds(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	start := t0.Add(-time.Hour)
	n := mustNursing(t, s, k.ID, ana.ID, runningNursing(start, "left"))

	at := start.Add(10 * time.Minute)
	n, err := s.SwitchNursingSide(ctx, k.ID, ana.ID, n.ID, "right", at, t0)
	if err != nil || n.LeftSeconds != 600 || n.RightSeconds != 0 || *n.Side != "right" || !n.SideSince.Equal(at) {
		t.Fatalf("after left 0:10 = %+v, %v", n, err)
	}
	at = at.Add(5 * time.Minute)
	if n, err = s.SwitchNursingSide(ctx, k.ID, ana.ID, n.ID, "left", at, t0); err != nil || n.RightSeconds != 300 || *n.Side != "left" {
		t.Fatalf("after right 0:05 = %+v, %v", n, err)
	}
	at = at.Add(7*time.Minute + 3*time.Second)
	n, err = s.EndNursing(ctx, k.ID, ana.ID, n.ID, at, t0)
	if err != nil {
		t.Fatal(err)
	}
	if n.LeftSeconds != 600+423 || n.RightSeconds != 300 || n.EndedAt == nil || !n.EndedAt.Equal(at) || n.Side != nil || n.SideSince != nil {
		t.Fatalf("ended = %+v", n)
	}
	if got, err := s.Nursing(ctx, k.ID, n.ID); err != nil || !reflect.DeepEqual(got, n) {
		t.Fatalf("stored = %+v, %v; want %+v", got, err, n)
	}
	if total := n.LeftSeconds + n.RightSeconds; total != int(n.EndedAt.Sub(n.StartedAt)/time.Second) {
		t.Errorf("left + right = %d s, want the whole session", total)
	}
}

func TestSwitchOnFinishedSession(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	n := mustNursing(t, s, k.ID, ana.ID, finishedNursing(t0.Add(-time.Hour), t0, 60, 60))
	if _, err := s.SwitchNursingSide(ctx, k.ID, ana.ID, n.ID, "left", t0, t0); !errors.Is(err, ErrNotRunning) {
		t.Errorf("switch on a finished session: %v, want ErrNotRunning", err)
	}
	if got, err := s.Nursing(ctx, k.ID, n.ID); err != nil || !reflect.DeepEqual(got, n) {
		t.Errorf("the refused switch changed the row: %+v, %v", got, err)
	}
}

func TestSwitchBeforeSideSinceRefused(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	start := t0.Add(-time.Hour)
	n := mustNursing(t, s, k.ID, ana.ID, runningNursing(start, "left"))
	n, err := s.SwitchNursingSide(ctx, k.ID, ana.ID, n.ID, "right", start.Add(time.Minute), t0)
	if err != nil {
		t.Fatal(err)
	}
	early := start.Add(30 * time.Second)
	if _, err := s.SwitchNursingSide(ctx, k.ID, ana.ID, n.ID, "left", early, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("switch before side_since: %v, want ErrInvalidSession", err)
	}
	if _, err := s.EndNursing(ctx, k.ID, ana.ID, n.ID, early, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("end before side_since: %v, want ErrInvalidSession", err)
	}
	if _, err := s.SwitchNursingSide(ctx, k.ID, ana.ID, n.ID, "middle", t0, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("switch to an unknown side: %v, want ErrInvalidSession", err)
	}
	if got, err := s.Nursing(ctx, k.ID, n.ID); err != nil || !reflect.DeepEqual(got, n) {
		t.Errorf("the refused writes changed the row: %+v, %v", got, err)
	}
}

func TestFinishedNursingSecondsCannotExceedDuration(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	start, end := t0.Add(-time.Hour), t0.Add(-50*time.Minute) // 600 s
	for name, n := range map[string]model.NursingSession{
		"one over":        finishedNursing(start, end, 301, 300),
		"left negative":   finishedNursing(start, end, -1, 0),
		"right negative":  finishedNursing(start, end, 0, -1),
		"sum wraps int64": finishedNursing(start, end, 1<<62, 1<<62),
	} {
		if _, err := s.CreateNursing(ctx, k.ID, ana.ID, n, t0); !errors.Is(err, ErrInvalidSession) {
			t.Errorf("create %s: %v, want ErrInvalidSession", name, err)
		}
	}
	n := mustNursing(t, s, k.ID, ana.ID, finishedNursing(start, end, 300, 300))
	if _, err := s.UpdateNursing(ctx, k.ID, ana.ID, n.ID, t0, func(n model.NursingSession) (model.NursingSession, error) {
		n.RightSeconds = 301
		return n, nil
	}); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("update over: %v, want ErrInvalidSession", err)
	}
	if _, err := s.UpdateNursing(ctx, k.ID, ana.ID, n.ID, t0, func(n model.NursingSession) (model.NursingSession, error) {
		n.StartedAt = n.StartedAt.Add(time.Second) // shortens the session under the stored seconds
		return n, nil
	}); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("update that shortens the session: %v, want ErrInvalidSession", err)
	}
	if _, err := s.UpdateNursing(ctx, k.ID, ana.ID, n.ID, t0, func(n model.NursingSession) (model.NursingSession, error) {
		n.LeftSeconds, n.RightSeconds = 1<<62, 1<<62
		return n, nil
	}); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("update to a wrapping sum: %v, want ErrInvalidSession", err)
	}
	if got, err := s.Nursing(ctx, k.ID, n.ID); err != nil || !reflect.DeepEqual(got, n) {
		t.Errorf("the refused updates changed the row: %+v, %v", got, err)
	}

	running := mustNursing(t, s, k.ID, ana.ID, runningNursing(start, "left"))
	if _, err := s.UpdateNursing(ctx, k.ID, ana.ID, running.ID, t0, func(n model.NursingSession) (model.NursingSession, error) {
		n.SideSince = timep(end)
		n.LeftSeconds, n.RightSeconds = 1<<62, 1<<62
		return n, nil
	}); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("running row with a wrapping sum: %v, want ErrInvalidSession", err)
	}
}

// side and side_since are both set exactly while the session runs; the schema does not couple them.
func TestNursingSideIsSetIffRunning(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	left := "left"

	withSide := finishedNursing(t0.Add(-time.Hour), t0, 0, 0)
	withSide.Side = &left
	if _, err := s.CreateNursing(ctx, k.ID, ana.ID, withSide, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("finished with a side: %v, want ErrInvalidSession", err)
	}
	if _, err := s.CreateNursing(ctx, k.ID, ana.ID, model.NursingSession{StartedAt: t0}, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("running without a side: %v, want ErrInvalidSession", err)
	}
	if _, err := s.CreateNursing(ctx, k.ID, ana.ID, runningNursing(t0, "both"), t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("running on side both: %v, want ErrInvalidSession", err)
	}
	withSeconds := runningNursing(t0, "left")
	withSeconds.LeftSeconds = 1
	if _, err := s.CreateNursing(ctx, k.ID, ana.ID, withSeconds, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("starting with accrued seconds: %v, want ErrInvalidSession", err)
	}

	running := mustNursing(t, s, k.ID, ana.ID, runningNursing(t0.Add(-time.Hour), "left"))
	finished := mustNursing(t, s, k.ID, ana.ID, finishedNursing(t0.Add(-3*time.Hour), t0.Add(-2*time.Hour), 0, 0))
	for name, c := range map[string]struct {
		id   string
		edit func(model.NursingSession) (model.NursingSession, error)
	}{
		"running loses its side": {running.ID, func(n model.NursingSession) (model.NursingSession, error) {
			n.Side = nil
			return n, nil
		}},
		"running loses side_since": {running.ID, func(n model.NursingSession) (model.NursingSession, error) {
			n.SideSince = nil
			return n, nil
		}},
		"side_since before the start": {running.ID, func(n model.NursingSession) (model.NursingSession, error) {
			n.StartedAt = n.SideSince.Add(time.Second)
			return n, nil
		}},
		"finished gains a side": {finished.ID, func(n model.NursingSession) (model.NursingSession, error) {
			n.Side = &left
			return n, nil
		}},
		"finished gains side_since": {finished.ID, func(n model.NursingSession) (model.NursingSession, error) {
			n.SideSince = n.EndedAt
			return n, nil
		}},
		"finished is re-opened": {finished.ID, func(n model.NursingSession) (model.NursingSession, error) {
			n.EndedAt, n.Side, n.SideSince = nil, &left, &n.StartedAt
			return n, nil
		}},
		"ended without clearing the side": {running.ID, func(n model.NursingSession) (model.NursingSession, error) {
			n.EndedAt = timep(t0)
			return n, nil
		}},
	} {
		if _, err := s.UpdateNursing(ctx, k.ID, ana.ID, c.id, t0, c.edit); !errors.Is(err, ErrInvalidSession) {
			t.Errorf("%s: %v, want ErrInvalidSession", name, err)
		}
	}
	for _, want := range []model.NursingSession{running, finished} {
		if got, err := s.Nursing(ctx, k.ID, want.ID); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("a refused update changed %s: %+v, %v", want.ID, got, err)
		}
	}
}

// Restarted moves an unswitched timer's side_since with its start, so ending it credits the whole shown duration.
func TestRestartedUnswitchedNursingCreditsTheShownDuration(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for _, start := range []time.Time{t0.Add(-30 * time.Minute), t0.Add(-10 * time.Minute)} {
		n := mustNursing(t, s, k.ID, ana.ID, runningNursing(t0.Add(-20*time.Minute), "left"))
		moved, err := s.UpdateNursing(ctx, k.ID, ana.ID, n.ID, t0, func(n model.NursingSession) (model.NursingSession, error) {
			return n.Restarted(start), nil
		})
		if err != nil || !moved.SideSince.Equal(start) {
			t.Fatalf("restart at %v: %+v, %v", start, moved, err)
		}
		ended, err := s.EndNursing(ctx, k.ID, ana.ID, n.ID, t0, t0)
		if err != nil || ended.LeftSeconds != int(t0.Sub(start)/time.Second) || ended.RightSeconds != 0 {
			t.Errorf("end after a restart at %v: %+v, %v", start, ended, err)
		}
	}
	n := mustNursing(t, s, k.ID, ana.ID, runningNursing(t0.Add(-20*time.Minute), "left"))
	switched, err := s.SwitchNursingSide(ctx, k.ID, ana.ID, n.ID, "right", t0.Add(-10*time.Minute), t0)
	if err != nil {
		t.Fatal(err)
	}
	if got := switched.Restarted(t0.Add(-30 * time.Minute)); !got.SideSince.Equal(*switched.SideSince) || got.LeftSeconds != 600 {
		t.Errorf("a switched session's side_since moved: %+v", got)
	}
}
