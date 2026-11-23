package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

func getRec(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealth(t *testing.T) {
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	now := t0
	h := Health(s, func() time.Time { return now }, func() time.Time { return now })

	rec := getRec(h, "/healthz")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("open store: %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}

	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := getRec(h, "/healthz"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed store with a fresh tick: %d, want 503", rec.Code)
	}
}

func TestHealthWorkerStaleness(t *testing.T) {
	stale := constants.WorkerStaleAfter
	for _, tc := range []struct {
		name     string
		lastTick time.Time // zero: the worker has not finished a tick
		now      time.Time
		want     int
	}{
		{"ticked just now", t0, t0, http.StatusOK},
		{"ticked a full window ago", t0, t0.Add(stale), http.StatusOK},
		{"ticked just over a window ago", t0, t0.Add(stale + time.Nanosecond), http.StatusServiceUnavailable},
		{"no tick yet, a fresh start", time.Time{}, t0.Add(stale), http.StatusOK},
		{"no tick yet, past the first window", time.Time{}, t0.Add(stale + time.Nanosecond), http.StatusServiceUnavailable},
		{"a tick older than the start counts from the start", t0.Add(-time.Hour), t0.Add(stale), http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := store.OpenMemory()
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			now := t0 // the process starts at t0
			h := Health(s, func() time.Time { return now }, func() time.Time { return tc.lastTick })
			now = tc.now
			rec := getRec(h, "/healthz")
			if rec.Code != tc.want {
				t.Fatalf("%d %q, want %d", rec.Code, rec.Body.String(), tc.want)
			}
		})
	}
}

func TestRoutesRefuseANilLastTick(t *testing.T) {
	d := newKit(t).deps()
	d.LastTick = nil
	defer func() {
		if recover() == nil {
			t.Fatal("Routes accepted a nil LastTick")
		}
	}()
	Routes(d)
}

// dueDelivery makes one pending ntfy delivery for a reminder due at due.
func dueDelivery(t *testing.T, k *kit, due time.Time) {
	t.Helper()
	ctx := context.Background()
	g := k.addParent("gwen")
	kid, err := k.Store.CreateKid(ctx, model.Kid{Name: "Kid K", BirthDate: "2026-03-04", Sex: "female", TimeZone: "UTC", GrowthSource: "who",
		ReminderOverdueMinutes: constants.DefaultReminderOverdueMinutes}, g.ID, due)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.Store.CreateMedication(ctx, kid.ID, g.ID, model.Medication{Name: "Amoxicillin", Dose: "2.5 ml", Active: true, StartAt: due,
		Schedule: schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 1}}, due); err != nil {
		t.Fatal(err)
	}
	if _, _, err := k.Store.SetNtfyTopic(ctx, g.ID, k.Cfg.Ntfy.TopicPrefix, false, due); err != nil {
		t.Fatal(err)
	}
	if err := k.Store.SetNtfyOptin(ctx, kid.ID, g.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := k.Store.OpenDue(ctx, due, store.Channels{Ntfy: true}); err != nil || n != 1 {
		t.Fatalf("OpenDue = %d, %v", n, err)
	}
}

func TestRemindersHealth(t *testing.T) {
	stale := constants.ReminderHealthStale
	ctx := context.Background()
	due := t0
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, k *kit) // runs after the delivery exists; may settle it
		now   time.Time
		want  int
	}{
		{"nothing stuck", func(*testing.T, *kit) {}, due.Add(stale - time.Second), http.StatusOK},
		{"pending past the window", func(*testing.T, *kit) {}, due.Add(stale + time.Second), http.StatusServiceUnavailable},
		{"sending past the window", func(t *testing.T, k *kit) {
			if got, err := k.Store.ClaimDeliveries(ctx, due, 8); err != nil || len(got) != 1 {
				t.Fatalf("claim: %d, %v", len(got), err)
			}
		}, due.Add(stale + time.Second), http.StatusServiceUnavailable},
		{"failed just now", func(t *testing.T, k *kit) {
			settle(t, k, due, store.OutcomeFailed)
		}, due.Add(stale - time.Second), http.StatusServiceUnavailable},
		{"failed longer ago than the window", func(t *testing.T, k *kit) {
			settle(t, k, due, store.OutcomeFailed)
		}, due.Add(stale + time.Second), http.StatusOK},
		{"sent", func(t *testing.T, k *kit) {
			settle(t, k, due, store.OutcomeSent)
		}, due.Add(stale + time.Second), http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k := newKit(t)
			dueDelivery(t, k, due)
			tc.setup(t, k)
			k.now = tc.now
			rec := k.serve(k.newReq(http.MethodGet, "/healthz/reminders", nil))
			wantBody := `{"status":"ok"}`
			if tc.want != http.StatusOK {
				wantBody = `{"status":"unavailable"}`
			}
			if rec.Code != tc.want || strings.TrimSpace(rec.Body.String()) != wantBody {
				t.Fatalf("%d %q, want %d %s", rec.Code, rec.Body.String(), tc.want, wantBody)
			}
		})
	}
}

func settle(t *testing.T, k *kit, at time.Time, o store.Outcome) {
	t.Helper()
	got, err := k.Store.ClaimDeliveries(context.Background(), at, 8)
	if err != nil || len(got) != 1 {
		t.Fatalf("claim: %d, %v", len(got), err)
	}
	if err := k.Store.SettleDelivery(context.Background(), got[0].ID, store.Settle{Outcome: o}, at); err != nil {
		t.Fatal(err)
	}
}

func TestRemindersHealthWithNoChannelConfigured(t *testing.T) {
	k := newKitNoPush(t)
	k.Cfg.NtfyEnabled = false
	k.Handler = NewHandler(k.deps())
	rec := k.serve(k.newReq(http.MethodGet, "/healthz/reminders", nil))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("%d %q", rec.Code, rec.Body.String())
	}
}

func TestRemindersHealthStoreDown(t *testing.T) {
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	h := RemindersHealth(s, func() time.Time { return t0 })
	if rec := getRec(h, "/healthz/reminders"); rec.Code != http.StatusOK {
		t.Fatalf("open store: %d", rec.Code)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	rec := getRec(h, "/healthz/reminders")
	if rec.Code != http.StatusServiceUnavailable || strings.TrimSpace(rec.Body.String()) != `{"status":"unavailable"}` {
		t.Fatalf("closed store: %d %q", rec.Code, rec.Body.String())
	}
}
