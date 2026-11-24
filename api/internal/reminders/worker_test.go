package reminders

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/ntfy"
	"github.com/davcs86/happybabytracker/api/internal/push"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/testlog"
)

var ctx = context.Background()

const origin = "https://hbt.example"

func at(h, m int) time.Time { return time.Date(2026, 11, 23, h, m, 0, 0, time.UTC) }

// testKeys is one subscription key pair per test binary.
var testKeys = sync.OnceValues(func() (string, string) {
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(k.PublicKey().Bytes()), base64.RawURLEncoding.EncodeToString(auth)
})

// fakePush records through a push.Recorder (which keeps its key and topic checks) unless hook answers first.
type fakePush struct {
	rec  push.Recorder
	hook func(ctx context.Context, s push.Subscription, m push.Message) (push.Result, bool)
}

func (f *fakePush) Send(ctx context.Context, s push.Subscription, m push.Message) push.Result {
	if f.hook != nil {
		if r, ok := f.hook(ctx, s, m); ok {
			return r
		}
	}
	return f.rec.Send(ctx, s, m)
}

type published struct{ Topic, Title, Message, Click string }

type fakeNtfy struct {
	mu   sync.Mutex
	got  []published
	hook func(ctx context.Context, topic string) ntfy.Result
}

func (f *fakeNtfy) Publish(ctx context.Context, topic, title, message, click string) ntfy.Result {
	f.mu.Lock()
	f.got = append(f.got, published{topic, title, message, click})
	hook := f.hook
	f.mu.Unlock()
	if hook != nil {
		return hook(ctx, topic)
	}
	return ntfy.Result{Outcome: push.Sent, Status: 200}
}

func (f *fakeNtfy) published() []published {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.got)
}

// env is a UTC kid Mia with guardian ana and caregiver bo, and a 14:00 course created at 10:00, so the 14:00 row
// exists. ana's device and topic are opted in for Mia; bo has a device that is not.
type env struct {
	st             *store.Store
	db             *sql.DB
	clock          atomic.Int64
	log            *bytes.Buffer
	kid            model.Kid
	ana, bo        model.Parent
	anaDev, boDev  model.PushDevice
	anaTopic       string
	course         model.Medication
	push           *fakePush
	ntfy           *fakeNtfy
	banned         []string
	pushTimeout    time.Duration
	ntfyTimeoutSet time.Duration
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "happybabytracker.db")+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	e := &env{st: st, db: db, log: &bytes.Buffer{}, push: &fakePush{}, ntfy: &fakeNtfy{}}
	e.set(at(10, 0))
	e.ana = e.parent(t, "ana")
	e.bo = e.parent(t, "bo")
	e.kid = e.newKid(t, "Mia")
	if err := st.SetAccess(ctx, e.kid.ID, e.ana.ID, "bo", model.RoleCaregiver, e.now()); err != nil {
		t.Fatal(err)
	}
	e.anaDev = e.device(t, e.ana.ID, "https://push.example/ana-phone")
	e.boDev = e.device(t, e.bo.ID, "https://push.example/bo-phone")
	if err := st.SetPushOptin(ctx, e.kid.ID, e.ana.ID, e.anaDev.ID); err != nil {
		t.Fatal(err)
	}
	e.anaTopic = e.topic(t, e.ana.ID)
	e.course = e.newCourse(t, e.kid.ID, "14:00")
	p, a := testKeys()
	e.banned = []string{"Mia", "Amoxicillin", "2.5 ml", "push.example", "ana-phone", "bo-phone", e.anaTopic, p, a}
	return e
}

func (e *env) set(t time.Time)      { e.clock.Store(t.UnixNano()) }
func (e *env) now() time.Time       { return time.Unix(0, e.clock.Load()).UTC() }
func (e *env) logger() *slog.Logger { return slog.New(slog.NewJSONHandler(e.log, nil)) }

func (e *env) parent(t *testing.T, username string) model.Parent {
	t.Helper()
	p, err := e.st.CreateParent(ctx, username, "Name of "+username, e.now())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (e *env) newKid(t *testing.T, name string) model.Kid {
	t.Helper()
	k, err := e.st.CreateKid(ctx, model.Kid{Name: name, BirthDate: "2026-01-02", Sex: "female", TimeZone: "UTC", GrowthSource: "who",
		ReminderOverdueMinutes: constants.DefaultReminderOverdueMinutes}, e.ana.ID, e.now())
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func (e *env) newCourse(t *testing.T, kidID string, times ...string) model.Medication {
	t.Helper()
	m, err := e.st.CreateMedication(ctx, kidID, e.ana.ID, model.Medication{Name: "Amoxicillin", Dose: "2.5 ml", Active: true,
		StartAt: at(0, 0), Schedule: schedule.Schedule{Kind: schedule.KindTimes, Times: times}}, e.now())
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func (e *env) device(t *testing.T, parentID, endpoint string) model.PushDevice {
	t.Helper()
	p, a := testKeys()
	d, _, err := e.st.UpsertDevice(ctx, parentID, endpoint, p, a, "Phone", e.now())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func (e *env) topic(t *testing.T, parentID string) string {
	t.Helper()
	tp, _, err := e.st.SetNtfyTopic(ctx, parentID, "hbt-test", false, e.now())
	if err != nil {
		t.Fatal(err)
	}
	if err := e.st.SetNtfyOptin(ctx, e.kid.ID, parentID); err != nil {
		t.Fatal(err)
	}
	return tp.Topic
}

// worker builds a worker on e's store; a channel argument false leaves that channel off (a nil interface).
func (e *env) worker(withPush, withNtfy bool) *Worker {
	d := Deps{Store: e.st, Origin: origin, Now: e.now, Log: e.logger()}
	if withPush {
		d.Push = e.push
	}
	if withNtfy {
		d.Ntfy = e.ntfy
	}
	w := New(d)
	if e.pushTimeout != 0 {
		w.pushTimeout = e.pushTimeout
	}
	if e.ntfyTimeoutSet != 0 {
		w.ntfyTimeout = e.ntfyTimeoutSet
	}
	return w
}

func (e *env) tick(t *testing.T, w *Worker, now time.Time) {
	t.Helper()
	e.set(now)
	if err := w.Tick(ctx); err != nil {
		t.Fatal(err)
	}
}

func (e *env) reminderID(t *testing.T, due time.Time) string {
	t.Helper()
	var id string
	if err := e.db.QueryRow(`SELECT id FROM reminders WHERE medication_id = ? AND due_at = ?`, e.course.ID, due.Unix()).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (e *env) reminderStates(t *testing.T, medicationID string) map[time.Time]string {
	t.Helper()
	rows, err := e.db.Query(`SELECT due_at, state FROM reminders WHERE medication_id = ?`, medicationID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[time.Time]string{}
	for rows.Next() {
		var (
			due   int64
			state string
		)
		if err := rows.Scan(&due, &state); err != nil {
			t.Fatal(err)
		}
		out[time.Unix(due, 0).UTC()] = state
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// deliveries maps "channel target" to state.
func (e *env) deliveries(t *testing.T) map[string]string {
	t.Helper()
	rows, err := e.db.Query(`SELECT channel, target_id, state FROM reminder_deliveries`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var ch, target, state string
		if err := rows.Scan(&ch, &target, &state); err != nil {
			t.Fatal(err)
		}
		out[ch+" "+target] = state
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

var workerLogFields = []string{"time", "level", "msg", "step", "cause", "delivery", "reminder", "channel", "outcome", "status",
	"attempt", "count", "duration_ms"}

// logLines checks every worker line against workerLogFields and e.banned, and returns them decoded.
func (e *env) logLines(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(e.log.String()), "\n") {
		if line == "" {
			continue
		}
		for _, leak := range testlog.Leaks(line, workerLogFields, e.banned...) {
			t.Errorf("%s: %s", leak, line)
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func countMsg(lines []map[string]any, msg string) int {
	n := 0
	for _, l := range lines {
		if l["msg"] == msg {
			n++
		}
	}
	return n
}

func TestOneSendPerOptedInTarget(t *testing.T) {
	e := newEnv(t)
	w := e.worker(true, true)
	e.tick(t, w, at(14, 10))
	rid := e.reminderID(t, at(14, 0))

	sends := e.push.rec.Sends()
	if len(sends) != 1 || sends[0].Device != e.anaDev.ID {
		t.Fatalf("push sends %+v, want one to ana's device %s", sends, e.anaDev.ID)
	}
	s := sends[0]
	if want := int((20 * time.Minute) / time.Second); s.TTLSeconds != want {
		t.Errorf("TTL %d, want %d (seconds to due + ReminderExpiry)", s.TTLSeconds, want)
	}
	if s.Urgency != "high" {
		t.Errorf("urgency %q", s.Urgency)
	}
	if want := strings.ReplaceAll(rid, "-", ""); s.Topic != want || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(s.Topic) {
		t.Errorf("topic %q, want %q (32 hex)", s.Topic, want)
	}
	var payload map[string]any
	if err := json.Unmarshal(s.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	path := "/kids/" + e.kid.ID + "/medication"
	want := map[string]any{"v": float64(1), "category": "medication", "kid": e.kid.ID, "title": "Medication due for Mia",
		"path": path, "tag": "reminder-" + rid}
	if len(payload) != len(want) {
		t.Errorf("payload keys %v, want exactly %v", payload, want)
	}
	for k, v := range want {
		if payload[k] != v {
			t.Errorf("payload %s = %v, want %v", k, payload[k], v)
		}
	}
	for _, leak := range []string{"Amoxicillin", "2.5 ml", "2.5"} {
		if bytes.Contains(s.Payload, []byte(leak)) {
			t.Errorf("payload names %q: %s", leak, s.Payload)
		}
	}

	pubs := e.ntfy.published()
	wantPub := published{Topic: e.anaTopic, Title: "Medication due for Mia", Message: "Medication due for Mia", Click: origin + path}
	if len(pubs) != 1 || pubs[0] != wantPub {
		t.Fatalf("ntfy %+v, want [%+v]", pubs, wantPub)
	}

	e.tick(t, w, at(14, 11))
	if n, m := len(e.push.rec.Sends()), len(e.ntfy.published()); n != 1 || m != 1 {
		t.Fatalf("a second tick sent again: push %d, ntfy %d", n, m)
	}
	got := e.deliveries(t)
	wantDel := map[string]string{"push " + e.anaDev.ID: "sent", "ntfy " + e.ana.ID: "sent"}
	if len(got) != len(wantDel) {
		t.Fatalf("deliveries %v, want %v", got, wantDel)
	}
	for k, v := range wantDel {
		if got[k] != v {
			t.Errorf("delivery %s = %q, want %q", k, got[k], v)
		}
	}
	if lines := e.logLines(t); countMsg(lines, "delivery") != 2 {
		t.Errorf("%d delivery lines, want 2: %s", countMsg(lines, "delivery"), e.log)
	}
}

func TestNoChannelsStillMaterialises(t *testing.T) {
	e := newEnv(t)
	w := e.worker(false, false)
	if !w.LastTick().IsZero() {
		t.Fatal("LastTick before any tick")
	}
	e.tick(t, w, at(14, 0))
	if !w.LastTick().Equal(at(14, 0)) {
		t.Fatalf("LastTick %v, want %v", w.LastTick(), at(14, 0))
	}
	tomorrow := at(14, 0).AddDate(0, 0, 1)
	states := e.reminderStates(t, e.course.ID)
	if states[at(14, 0)] != "pending" || states[tomorrow] != "pending" {
		t.Fatalf("after the 14:00 tick: %v, want today's and tomorrow's rows pending", states)
	}
	e.tick(t, w, at(14, 31))
	if s := e.reminderStates(t, e.course.ID)[at(14, 0)]; s != "expired" {
		t.Fatalf("14:00 row at 14:31 is %q, want expired", s)
	}
	if err := e.st.SetModule(ctx, e.kid.ID, e.ana.ID, "medication", false, e.now()); err != nil {
		t.Fatal(err)
	}
	e.tick(t, w, tomorrow)
	if s := e.reminderStates(t, e.course.ID)[tomorrow]; s != "cancelled" {
		t.Fatalf("tomorrow's row with the module off is %q, want cancelled", s)
	}
	if got := e.deliveries(t); len(got) != 0 {
		t.Fatalf("deliveries with no channel: %v", got)
	}
	if lines := e.logLines(t); len(lines) != 0 {
		t.Fatalf("log lines with nothing to report: %s", e.log)
	}
}

func TestEachTargetOnItsOwn(t *testing.T) {
	e := newEnv(t)
	second := e.device(t, e.ana.ID, "https://push.example/ana-tablet")
	if err := e.st.SetPushOptin(ctx, e.kid.ID, e.ana.ID, second.ID); err != nil {
		t.Fatal(err)
	}
	e.banned = append(e.banned, "ana-tablet")
	const timeout = time.Second
	e.pushTimeout, e.ntfyTimeoutSet = timeout, timeout
	secondDone := make(chan struct{})
	var hungSawSecond, ntfySawSecond atomic.Bool
	var hungDeadline atomic.Int64
	// The hung target waits out its own deadline; the other device must be sent meanwhile.
	e.push.hook = func(ctx context.Context, s push.Subscription, m push.Message) (push.Result, bool) {
		if s.ID != e.anaDev.ID {
			r := e.push.rec.Send(ctx, s, m)
			close(secondDone)
			return r, true
		}
		if dl, ok := ctx.Deadline(); ok {
			hungDeadline.Store(int64(time.Until(dl)))
		}
		select {
		case <-secondDone:
			hungSawSecond.Store(true)
		case <-ctx.Done():
		}
		<-ctx.Done()
		return push.Result{Outcome: push.Retry}, true
	}
	e.ntfy.hook = func(ctx context.Context, _ string) ntfy.Result {
		select {
		case <-secondDone:
			ntfySawSecond.Store(true)
		case <-ctx.Done():
		}
		<-ctx.Done()
		return ntfy.Result{Outcome: push.Retry}
	}
	w := e.worker(true, true)
	start := time.Now()
	e.tick(t, w, at(14, 0))
	elapsed := time.Since(start)

	if !hungSawSecond.Load() || !ntfySawSecond.Load() {
		t.Fatalf("the other device was not sent while a target hung: push %v, ntfy %v", hungSawSecond.Load(), ntfySawSecond.Load())
	}
	if d := time.Duration(hungDeadline.Load()); d <= 0 || d > timeout {
		t.Fatalf("hung send's deadline %v, want within its own %v", d, timeout)
	}
	if elapsed >= 2*timeout {
		t.Fatalf("tick took %v: the hung targets ran one after the other", elapsed)
	}
	got := e.deliveries(t)
	for k, want := range map[string]string{"push " + e.anaDev.ID: "pending", "push " + second.ID: "sent", "ntfy " + e.ana.ID: "pending"} {
		if got[k] != want {
			t.Errorf("delivery %s = %q, want %q (all: %v)", k, got[k], want, got)
		}
	}
	if sends := e.push.rec.Sends(); len(sends) != 1 || sends[0].Device != second.ID {
		t.Fatalf("recorded sends %+v", sends)
	}
	e.logLines(t)
}

func TestNoDuplicateAfterRestartMidSend(t *testing.T) {
	t.Run("a row left sending is never sent again", func(t *testing.T) {
		e := newEnv(t)
		e.set(at(14, 0))
		// A process killed mid-send leaves its claimed row sending.
		if _, err := e.st.OpenDue(ctx, e.now(), store.Channels{Push: true}); err != nil {
			t.Fatal(err)
		}
		if ds, err := e.st.ClaimDeliveries(ctx, e.now(), 10); err != nil || len(ds) != 1 {
			t.Fatalf("claimed %d, %v", len(ds), err)
		}
		if err := e.st.MarkUnconfirmed(ctx); err != nil {
			t.Fatal(err)
		}
		w := e.worker(true, false)
		e.tick(t, w, at(14, 1))
		e.tick(t, w, at(14, 5))
		if n := len(e.push.rec.Sends()); n != 0 {
			t.Fatalf("resent after a restart: %d", n)
		}
		if got := e.deliveries(t)["push "+e.anaDev.ID]; got != "unconfirmed" {
			t.Fatalf("delivery %q, want unconfirmed", got)
		}
		e.logLines(t)
	})
	for _, tc := range []struct {
		name    string
		outcome push.Outcome
		status  int
		after   string // the row's state once the cancelled tick returns
		resends int
		final   string
	}{
		{"a send finished during shutdown keeps its outcome", push.Sent, 201, "sent", 0, "sent"},
		{"a retry the push service answered during shutdown is kept", push.Retry, 503, "pending", 1, "sent"},
		// No answer: the service may have the message, so it is never sent again.
		{"a send cut off by shutdown is never sent again", push.Retry, 0, "sending", 0, "unconfirmed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			tctx, cancel := context.WithCancel(ctx)
			defer cancel()
			var calls atomic.Int32
			e.push.hook = func(context.Context, push.Subscription, push.Message) (push.Result, bool) {
				calls.Add(1)
				cancel()
				return push.Result{Outcome: tc.outcome, Status: tc.status}, true
			}
			e.set(at(14, 0))
			_ = e.worker(true, false).Tick(tctx)
			if got := e.deliveries(t)["push "+e.anaDev.ID]; got != tc.after || calls.Load() != 1 {
				t.Fatalf("after the cancelled tick: %q, %d calls; want %q", got, calls.Load(), tc.after)
			}

			if err := e.st.MarkUnconfirmed(ctx); err != nil {
				t.Fatal(err)
			}
			e.push.hook = nil
			w := e.worker(true, false)
			e.tick(t, w, at(14, 1))
			e.tick(t, w, at(14, 5))
			if n := len(e.push.rec.Sends()); n != tc.resends || calls.Load() != 1 {
				t.Fatalf("after a restart: %d recorded, %d hooked; want %d recorded", n, calls.Load(), tc.resends)
			}
			if got := e.deliveries(t)["push "+e.anaDev.ID]; got != tc.final {
				t.Fatalf("delivery %q, want %q", got, tc.final)
			}
			if n := countMsg(e.logLines(t), "reminder step failed"); n != 0 {
				t.Fatalf("%d step failure lines: %s", n, e.log)
			}
		})
	}
	t.Run("two workers on one store send each target once", func(t *testing.T) {
		e := newEnv(t)
		a, b := e.worker(true, true), e.worker(true, true)
		e.set(at(14, 0))
		var wg sync.WaitGroup
		for _, w := range []*Worker{a, b} {
			wg.Go(func() {
				if err := w.Tick(ctx); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
		e.tick(t, a, at(14, 1))
		e.tick(t, b, at(14, 1))
		if n, m := len(e.push.rec.Sends()), len(e.ntfy.published()); n != 1 || m != 1 {
			t.Fatalf("push %d, ntfy %d, want one each", n, m)
		}
		e.logLines(t)
	})
}

func TestDoseBeforePushCancels(t *testing.T) {
	e := newEnv(t)
	e.set(at(14, 0))
	if err := e.st.Materialise(ctx, e.now()); err != nil {
		t.Fatal(err)
	}
	if n, err := e.st.OpenDue(ctx, e.now(), store.Channels{Push: true, Ntfy: true}); err != nil || n != 2 {
		t.Fatalf("OpenDue: %d %v, want 2 deliveries", n, err)
	}
	if _, err := e.st.CreateDose(ctx, e.kid.ID, e.ana.ID, model.MedicationDose{MedicationID: e.course.ID, GivenAt: at(13, 59)}, e.now()); err != nil {
		t.Fatal(err)
	}
	e.tick(t, e.worker(true, true), at(14, 0))
	if n, m := len(e.push.rec.Sends()), len(e.ntfy.published()); n != 0 || m != 0 {
		t.Fatalf("sent after the dose: push %d, ntfy %d", n, m)
	}
	got := e.deliveries(t)
	if len(got) != 2 || got["push "+e.anaDev.ID] != "cancelled" || got["ntfy "+e.ana.ID] != "cancelled" {
		t.Fatalf("deliveries %v, want both cancelled", got)
	}
	if s := e.reminderStates(t, e.course.ID)[at(14, 0)]; s != "done" {
		t.Fatalf("reminder %q, want done", s)
	}
	e.logLines(t)
}

func TestNtfyRefusalLogsOncePerTickAndIsNotRetried(t *testing.T) {
	e := newEnv(t)
	boTopic := e.topic(t, e.bo.ID)
	e.banned = append(e.banned, boTopic, "hbt-test")
	e.ntfy.hook = func(context.Context, string) ntfy.Result {
		return ntfy.Result{Outcome: push.Failed, Status: 401, ConfigFault: true}
	}
	w := e.worker(false, true)
	e.tick(t, w, at(14, 0))
	if n := len(e.ntfy.published()); n != 2 {
		t.Fatalf("%d publishes, want 2", n)
	}
	lines := e.logLines(t)
	if n := countMsg(lines, "ntfy refused"); n != 1 {
		t.Fatalf("%d ntfy refused lines, want 1: %s", n, e.log)
	}
	for _, l := range lines {
		if l["msg"] == "ntfy refused" && (l["status"] != float64(401) || l["count"] != float64(2)) {
			t.Fatalf("refusal line %v", l)
		}
	}
	e.tick(t, w, at(14, 1))
	e.tick(t, w, at(14, 5))
	if n := len(e.ntfy.published()); n != 2 {
		t.Fatalf("retried: %d publishes", n)
	}
	if n := countMsg(e.logLines(t), "ntfy refused"); n != 1 {
		t.Fatalf("%d refusal lines after quiet ticks, want 1", n)
	}
	for k, v := range e.deliveries(t) {
		if v != "failed" {
			t.Errorf("delivery %s = %q, want failed", k, v)
		}
	}
}

func TestAFailingStepRunsTheRest(t *testing.T) {
	e := newEnv(t)
	leo := e.newKid(t, "Leo")
	e.newCourse(t, leo.ID, "18:00")
	if _, err := e.db.Exec(`UPDATE kids SET time_zone = 'Nowhere/Atlantis' WHERE id = ?`, leo.ID); err != nil {
		t.Fatal(err)
	}
	e.banned = append(e.banned, "Leo", "Atlantis")
	w := e.worker(true, false)
	e.set(at(14, 0))
	if err := w.Tick(ctx); err == nil {
		t.Fatal("Tick hid the materialise failure")
	}
	if sends := e.push.rec.Sends(); len(sends) != 1 {
		t.Fatalf("a materialise failure stopped the send: %d sends", len(sends))
	}
	e.set(at(14, 31))
	_ = w.Tick(ctx)
	if s := e.reminderStates(t, e.course.ID)[at(14, 0)]; s != "expired" {
		t.Fatalf("Expire skipped after a materialise failure: %q", s)
	}
	lines := e.logLines(t)
	if n := countMsg(lines, "reminder step failed"); n != 2 {
		t.Fatalf("%d step failure lines over two ticks, want 2: %s", n, e.log)
	}
	for _, l := range lines {
		if l["msg"] == "reminder step failed" && (l["step"] != "materialise" || l["cause"] == "" || l["cause"] == nil) {
			t.Fatalf("failure line %v", l)
		}
	}
}

func TestADeliveryDeletedMidSendIsNotAFailure(t *testing.T) {
	e := newEnv(t)
	// The course is deleted between the claim and the settle, so its delivery row cascades away.
	e.push.hook = func(context.Context, push.Subscription, push.Message) (push.Result, bool) {
		if err := e.st.DeleteMedication(ctx, e.kid.ID, e.ana.ID, e.course.ID); err != nil {
			t.Error(err)
		}
		return push.Result{Outcome: push.Sent, Status: 201}, true
	}
	e.tick(t, e.worker(true, false), at(14, 0))
	lines := e.logLines(t)
	if n := countMsg(lines, "reminder step failed"); n != 0 {
		t.Fatalf("%d step failure lines: %s", n, e.log)
	}
	// The push went out: its ids-only line stays as the trace.
	if n := countMsg(lines, "delivery"); n != 1 {
		t.Fatalf("%d delivery lines, want 1: %s", n, e.log)
	}
	if got := e.deliveries(t); len(got) != 0 {
		t.Fatalf("deliveries %v, want none", got)
	}
}

func TestABadZoneDoesNotStopOtherKids(t *testing.T) {
	e := newEnv(t)
	leo := e.newKid(t, "Leo")
	e.newCourse(t, leo.ID, "13:50")
	if _, err := e.db.Exec(`UPDATE kids SET time_zone = 'Nowhere/Atlantis' WHERE id = ?`, leo.ID); err != nil {
		t.Fatal(err)
	}
	e.banned = append(e.banned, "Leo", "Atlantis")
	w := e.worker(true, false)
	e.set(at(14, 0))
	if err := w.Tick(ctx); err == nil {
		t.Fatal("Tick hid the bad zone")
	}
	if sends := e.push.rec.Sends(); len(sends) != 1 || sends[0].Device != e.anaDev.ID {
		t.Fatalf("Mia's reminder was not sent: %+v", sends)
	}
	if s := e.reminderStates(t, e.course.ID)[at(14, 0).Add(24*time.Hour)]; s != "pending" {
		t.Fatalf("Mia's next row %q, want pending", s)
	}
	steps := map[any]int{}
	for _, l := range e.logLines(t) {
		if l["msg"] == "reminder step failed" {
			steps[l["step"]]++
		}
	}
	if steps["materialise"] != 1 || steps["open_due"] != 1 || len(steps) != 2 {
		t.Fatalf("step failures %v, want materialise and open_due once each", steps)
	}
}

func TestRunReturnsOnCancel(t *testing.T) {
	e := newEnv(t)
	e.set(at(14, 0))
	w := e.worker(true, true)
	rctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { w.Run(rctx); close(done) }()
	deadline := time.Now().Add(5 * time.Second)
	for w.LastTick().IsZero() {
		if time.Now().After(deadline) {
			t.Fatal("Run never ticked")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return on cancel")
	}
	if n := len(e.push.rec.Sends()); n != 1 {
		t.Fatalf("first tick sent %d, want 1", n)
	}
	e.logLines(t)
}
