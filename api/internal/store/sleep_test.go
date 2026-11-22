package store

import (
	"database/sql"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

func timep(t time.Time) *time.Time { return &t }

func mustSleep(t *testing.T, s *Store, kidID, actorID string, v model.SleepSession) model.SleepSession {
	t.Helper()
	got, err := s.CreateSleep(ctx, kidID, actorID, v, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// keepAll is an UpdateSleep edit that changes nothing.
func keepAll(v model.SleepSession) (model.SleepSession, error) { return v, nil }

func TestSleepCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := model.SleepSession{StartedAt: t0.Add(-2 * time.Hour), EndedAt: timep(t0.Add(-time.Hour))}
	in.Note = strp("nap")
	created, err := s.CreateSleep(ctx, k.ID, ana.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || !created.CreatedAt.Equal(t0) ||
		!created.StartedAt.Equal(in.StartedAt) || created.EndedAt == nil || !created.EndedAt.Equal(*in.EndedAt) {
		t.Fatalf("created = %+v", created)
	}
	if got, err := s.Sleep(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Sleep = %+v, %v; want %+v", got, err, created)
	}

	later := t0.Add(time.Hour)
	updated, err := s.UpdateSleep(ctx, k.ID, bo.ID, created.ID, later, func(v model.SleepSession) (model.SleepSession, error) {
		v.StartedAt, v.Note = t0.Add(-3*time.Hour), nil
		return v, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := created
	want.StartedAt, want.Note, want.UpdatedBy, want.UpdatedAt = t0.Add(-3*time.Hour), nil, bo.ID, later
	if !reflect.DeepEqual(updated, want) {
		t.Fatalf("updated = %+v, want %+v", updated, want)
	}
	if got, err := s.Sleep(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("after update = %+v, %v", got, err)
	}
	if list, err := s.ListSleep(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("ListSleep = %+v, %v", list, err)
	}
	if err := s.DeleteSleep(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sleep(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if _, err := s.UpdateSleep(ctx, k.ID, bo.ID, created.ID, later, keepAll); !errors.Is(err, ErrNotFound) {
		t.Errorf("update after delete: %v", err)
	}
}

func TestSleepBelongsToItsKidAndItsWriter(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	mustGrant(t, s, mine.ID, ana.ID, "bo", model.RoleCaregiver)
	v := mustSleep(t, s, mine.ID, ana.ID, model.SleepSession{StartedAt: t0})

	if _, err := s.Sleep(ctx, other.ID, v.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	if _, err := s.UpdateSleep(ctx, other.ID, ana.ID, v.ID, t0, keepAll); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if _, err := s.EndSleep(ctx, other.ID, ana.ID, v.ID, t0, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("end through another kid: %v", err)
	}
	if err := s.DeleteSleep(ctx, other.ID, ana.ID, v.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	for name, actor := range map[string]string{"no grant": cy.ID, "disabled caregiver": bo.ID} {
		if _, err := s.CreateSleep(ctx, mine.ID, actor, model.SleepSession{StartedAt: t0, EndedAt: timep(t0)}, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if _, err := s.UpdateSleep(ctx, mine.ID, actor, v.ID, t0, keepAll); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if _, err := s.EndSleep(ctx, mine.ID, actor, v.ID, t0, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s end: %v", name, err)
		}
		if err := s.DeleteSleep(ctx, mine.ID, actor, v.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if got, err := s.Sleep(ctx, mine.ID, v.ID); err != nil || !reflect.DeepEqual(got, v) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestSecondStartIsAlreadyRunning(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	running := mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0.Add(-time.Hour)})
	if _, err := s.CreateSleep(ctx, k.ID, ana.ID, model.SleepSession{StartedAt: t0}, t0); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second sleep start: %v, want ErrAlreadyRunning", err)
	}
	// A finished session may still be recorded while one runs.
	mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0.Add(-5 * time.Hour), EndedAt: timep(t0.Add(-4 * time.Hour))})
	if got, err := s.RunningSleep(ctx, k.ID); err != nil || got == nil || got.ID != running.ID {
		t.Fatalf("RunningSleep = %+v, %v; want %s", got, err, running.ID)
	}

	mustNursing(t, s, k.ID, ana.ID, runningNursing(t0.Add(-time.Hour), "left"))
	if _, err := s.CreateNursing(ctx, k.ID, ana.ID, runningNursing(t0, "right"), t0); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second nursing start: %v, want ErrAlreadyRunning", err)
	}
}

func TestSecondKidMayRunOneToo(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	first, second := mustKid(t, s, ana.ID, "First"), mustKid(t, s, ana.ID, "Second")
	for _, kid := range []string{first.ID, second.ID} {
		mustSleep(t, s, kid, ana.ID, model.SleepSession{StartedAt: t0})
		mustNursing(t, s, kid, ana.ID, runningNursing(t0, "left"))
	}
	for _, kid := range []string{first.ID, second.ID} {
		if v, err := s.RunningSleep(ctx, kid); err != nil || v == nil || v.KidID != kid {
			t.Errorf("RunningSleep(%s) = %+v, %v", kid, v, err)
		}
		if v, err := s.RunningNursing(ctx, kid); err != nil || v == nil || v.KidID != kid {
			t.Errorf("RunningNursing(%s) = %+v, %v", kid, v, err)
		}
	}
}

func TestRunningIsNilWhenNoneRuns(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0.Add(-time.Hour)})
	if _, err := s.EndSleep(ctx, k.ID, ana.ID, v.ID, t0, t0); err != nil {
		t.Fatal(err)
	}
	if got, err := s.RunningSleep(ctx, k.ID); err != nil || got != nil {
		t.Errorf("RunningSleep = %+v, %v; want nil, nil", got, err)
	}
	if got, err := s.RunningNursing(ctx, k.ID); err != nil || got != nil {
		t.Errorf("RunningNursing = %+v, %v; want nil, nil", got, err)
	}
	// Once ended, a new one may start.
	mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0})
}

func TestEndNotRunningIsNotRunning(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0.Add(-time.Hour)})
	ended, err := s.EndSleep(ctx, k.ID, ana.ID, v.ID, t0.Add(-time.Minute), t0)
	if err != nil {
		t.Fatal(err)
	}
	if ended.EndedAt == nil || !ended.EndedAt.Equal(t0.Add(-time.Minute)) || !ended.UpdatedAt.Equal(t0) {
		t.Fatalf("ended = %+v", ended)
	}
	if _, err := s.EndSleep(ctx, k.ID, ana.ID, v.ID, t0, t0); !errors.Is(err, ErrNotRunning) {
		t.Errorf("second end: %v, want ErrNotRunning", err)
	}
	if got, err := s.Sleep(ctx, k.ID, v.ID); err != nil || !reflect.DeepEqual(got, ended) {
		t.Errorf("the refused end changed the row: %+v, %v", got, err)
	}

	n := mustNursing(t, s, k.ID, ana.ID, finishedNursing(t0.Add(-time.Hour), t0, 60, 60))
	if _, err := s.EndNursing(ctx, k.ID, ana.ID, n.ID, t0, t0); !errors.Is(err, ErrNotRunning) {
		t.Errorf("end finished nursing: %v, want ErrNotRunning", err)
	}
}

func TestEndBeforeStartRefused(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	start := t0.Add(-time.Hour)
	v := mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: start})
	if _, err := s.EndSleep(ctx, k.ID, ana.ID, v.ID, start.Add(-time.Second), t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("sleep end before start: %v, want ErrInvalidSession", err)
	}
	if _, err := s.CreateSleep(ctx, k.ID, ana.ID, model.SleepSession{StartedAt: start, EndedAt: timep(start.Add(-time.Second))}, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("finished sleep ending before it starts: %v, want ErrInvalidSession", err)
	}
	if _, err := s.UpdateSleep(ctx, k.ID, ana.ID, v.ID, t0, func(v model.SleepSession) (model.SleepSession, error) {
		v.EndedAt = timep(start.Add(-time.Second))
		return v, nil
	}); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("update to end before start: %v, want ErrInvalidSession", err)
	}
	if got, err := s.RunningSleep(ctx, k.ID); err != nil || got == nil || got.ID != v.ID {
		t.Errorf("the refused writes changed the running row: %+v, %v", got, err)
	}

	n := mustNursing(t, s, k.ID, ana.ID, runningNursing(start, "left"))
	if _, err := s.EndNursing(ctx, k.ID, ana.ID, n.ID, start.Add(-time.Second), t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("nursing end before start: %v, want ErrInvalidSession", err)
	}
	if _, err := s.SwitchNursingSide(ctx, k.ID, ana.ID, n.ID, "right", start.Add(-time.Second), t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("nursing switch before start: %v, want ErrInvalidSession", err)
	}
	if _, err := s.CreateSleep(ctx, k.ID, ana.ID, model.SleepSession{StartedAt: start, EndedAt: timep(start)}, t0); err != nil {
		t.Errorf("a session ending as it starts: %v", err)
	}
}

func TestMaxSessionDuration(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	start := t0.Add(-30 * time.Hour)
	limit, over := start.Add(constants.MaxSessionDuration), start.Add(constants.MaxSessionDuration+time.Second)

	if _, err := s.CreateSleep(ctx, k.ID, ana.ID, model.SleepSession{StartedAt: start, EndedAt: &over}, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("finished sleep over the limit: %v, want ErrInvalidSession", err)
	}
	if _, err := s.CreateNursing(ctx, k.ID, ana.ID, finishedNursing(start, over, 0, 0), t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("finished nursing over the limit: %v, want ErrInvalidSession", err)
	}
	mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: start, EndedAt: &limit})
	mustNursing(t, s, k.ID, ana.ID, finishedNursing(start, limit, 0, 0))

	v := mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: start})
	if _, err := s.EndSleep(ctx, k.ID, ana.ID, v.ID, over, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("sleep end over the limit: %v, want ErrInvalidSession", err)
	}
	if _, err := s.EndSleep(ctx, k.ID, ana.ID, v.ID, limit, t0); err != nil {
		t.Errorf("sleep end at the limit: %v", err)
	}
	n := mustNursing(t, s, k.ID, ana.ID, runningNursing(start, "left"))
	if _, err := s.EndNursing(ctx, k.ID, ana.ID, n.ID, over, t0); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("nursing end over the limit: %v, want ErrInvalidSession", err)
	}
	if got, err := s.EndNursing(ctx, k.ID, ana.ID, n.ID, limit, t0); err != nil || got.LeftSeconds != int(constants.MaxSessionDuration/time.Second) {
		t.Errorf("nursing end at the limit: %+v, %v", got, err)
	}
}

// A finished session is never re-opened (Open value 10).
func TestFinishedSleepIsNeverReopened(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0.Add(-time.Hour), EndedAt: timep(t0)})
	if _, err := s.UpdateSleep(ctx, k.ID, ana.ID, v.ID, t0, func(v model.SleepSession) (model.SleepSession, error) {
		v.EndedAt = nil
		return v, nil
	}); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("re-open: %v, want ErrInvalidSession", err)
	}
	if got, err := s.RunningSleep(ctx, k.ID); err != nil || got != nil {
		t.Errorf("a row runs after the refused re-open: %+v, %v", got, err)
	}
}

// The edit runs inside the write transaction and sees the row as stored; its error aborts the write.
func TestUpdateSleepEditSeesStoredRowAndCanRefuse(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0.Add(-time.Hour)})
	refused := errors.New("refused")
	_, err := s.UpdateSleep(ctx, k.ID, ana.ID, v.ID, t0, func(got model.SleepSession) (model.SleepSession, error) {
		if !reflect.DeepEqual(got, v) {
			t.Errorf("edit saw %+v, want %+v", got, v)
		}
		got.Note = strp("changed")
		return got, refused
	})
	if !errors.Is(err, refused) {
		t.Fatalf("err = %v, want the edit's error", err)
	}
	if got, err := s.Sleep(ctx, k.ID, v.ID); err != nil || !reflect.DeepEqual(got, v) {
		t.Errorf("a refused edit was written: %+v, %v", got, err)
	}
}

func TestConcurrentStarts(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for name, start := range map[string]func() error{
		"sleep": func() error {
			_, err := s.CreateSleep(ctx, k.ID, ana.ID, model.SleepSession{StartedAt: t0}, t0)
			return err
		},
		"nursing": func() error {
			_, err := s.CreateNursing(ctx, k.ID, ana.ID, runningNursing(t0, "left"), t0)
			return err
		},
	} {
		var (
			wg   sync.WaitGroup
			errs [2]error
		)
		for i := range errs {
			wg.Go(func() { errs[i] = start() })
		}
		wg.Wait()
		ok, running := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				ok++
			case errors.Is(err, ErrAlreadyRunning):
				running++
			default:
				t.Errorf("%s: unexpected error %v", name, err)
			}
		}
		if ok != 1 || running != 1 {
			t.Errorf("%s: %d started and %d got ErrAlreadyRunning, want 1 and 1", name, ok, running)
		}
	}
}

// The partial unique index is the backstop when the in-transaction check is bypassed.
func TestRunningIndexMapsToAlreadyRunning(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0})
	mustNursing(t, s, k.ID, ana.ID, runningNursing(t0, "left"))
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		v := model.SleepSession{StartedAt: t0}
		v.ID, v.KidID, v.CreatedBy, v.UpdatedBy, v.CreatedAt, v.UpdatedAt = model.NewID(), k.ID, ana.ID, ana.ID, t0, t0
		return insertSleep(ctx, tx, v)
	})
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("sleep insert past the check: %v, want ErrAlreadyRunning", err)
	}
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		n := runningNursing(t0, "left")
		n.SideSince = timep(t0)
		n.ID, n.KidID, n.CreatedBy, n.UpdatedBy, n.CreatedAt, n.UpdatedAt = model.NewID(), k.ID, ana.ID, ana.ID, t0, t0
		return insertNursing(ctx, tx, n)
	})
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("nursing insert past the check: %v, want ErrAlreadyRunning", err)
	}
}

func TestSleepListByStart(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	older := mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0.Add(-3 * time.Hour), EndedAt: timep(t0)})
	newer := mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: t0.Add(-time.Hour)})
	got, err := s.ListSleep(ctx, k.ID, ListQuery{From: t0.Add(-2 * time.Hour), To: t0, Limit: 10})
	if err != nil || len(got) != 1 || got[0].ID != newer.ID {
		t.Fatalf("[-2h, now) = %+v, %v; want only the session that starts inside", got, err)
	}
	got, err = s.ListSleep(ctx, k.ID, wideQuery())
	if err != nil || len(got) != 2 || got[0].ID != newer.ID || got[1].ID != older.ID {
		t.Fatalf("wide = %+v, %v; want newest start first", got, err)
	}
}
