// Package reminders runs the one worker that materialises reminder rows and delivers them on the configured channels.
package reminders

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/cause"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/ntfy"
	"github.com/davcs86/happybabytracker/api/internal/push"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type Deps struct {
	Store  *store.Store
	Push   push.Sender    // nil when Web Push is off
	Ntfy   ntfy.Publisher // nil when ntfy is off
	Origin string         // the public origin an ntfy click opens
	Now    func() time.Time
	Log    *slog.Logger
}

type Worker struct {
	d                        Deps
	pushTimeout, ntfyTimeout time.Duration
	last                     atomic.Int64
}

func New(d Deps) *Worker {
	return &Worker{d: d, pushTimeout: constants.PushTimeout, ntfyTimeout: constants.NtfyTimeout}
}

// LastTick is the clock reading of the last finished tick; zero before the first.
func (w *Worker) LastTick() time.Time {
	n := w.last.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n).UTC()
}

// Run ticks at once and then every ReminderTick until ctx is cancelled; each tick logs its own failures.
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(constants.ReminderTick)
	defer t.Stop()
	for {
		_ = w.Tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// tick holds what one pass logs at most once.
type tick struct {
	w           *Worker
	ctx         context.Context
	mu          sync.Mutex
	errs        []error
	failed      map[string]bool
	ntfyRefused int
	ntfyStatus  int
}

// fail logs a step's first failure of the tick; a step failing never skips the steps after it.
func (t *tick) fail(step string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.errs = append(t.errs, err)
	if t.failed[step] || t.ctx.Err() != nil {
		return
	}
	t.failed[step] = true
	t.w.d.Log.Warn("reminder step failed", "step", step, "cause", cause.Of(err))
}

// Tick is one pass: materialise, open due reminders, send what is claimable, expire. It returns every failure.
func (w *Worker) Tick(ctx context.Context) error {
	t := &tick{w: w, ctx: ctx, failed: map[string]bool{}}
	now := w.d.Now()
	if err := w.d.Store.Materialise(ctx, now); err != nil {
		t.fail("materialise", err)
	}
	if _, err := w.d.Store.OpenDue(ctx, now, store.Channels{Push: w.d.Push != nil, Ntfy: w.d.Ntfy != nil}); err != nil {
		t.fail("open_due", err)
	}
	w.deliver(t, now)
	if t.ntfyRefused > 0 {
		w.d.Log.Warn("ntfy refused", "status", t.ntfyStatus, "count", t.ntfyRefused)
	}
	if err := w.d.Store.Expire(ctx, w.d.Now()); err != nil {
		t.fail("expire", err)
	}
	w.last.Store(now.UnixNano())
	return errors.Join(t.errs...)
}

// deliver claims one delivery at a time per slot, so a hung target holds one slot and a row is sending only while sent.
func (w *Worker) deliver(t *tick, now time.Time) {
	var wg sync.WaitGroup
	for range constants.DeliveryConcurrency {
		wg.Go(func() {
			for t.ctx.Err() == nil {
				ds, err := w.d.Store.ClaimDeliveries(t.ctx, now, 1)
				if err != nil {
					t.fail("claim", err)
					return
				}
				if len(ds) == 0 {
					return
				}
				w.send(t, ds[0])
			}
		})
	}
	wg.Wait()
}

// payload is Open value 18; it never carries the medication or the dose.
type payload struct {
	V        int    `json:"v"`
	Category string `json:"category"`
	Kid      string `json:"kid"`
	Title    string `json:"title"`
	Path     string `json:"path"`
	Tag      string `json:"tag"`
}

func (w *Worker) send(t *tick, d store.Delivery) {
	started := time.Now()
	tg, ok, err := w.d.Store.DeliveryTarget(t.ctx, d, w.d.Now())
	if err != nil {
		t.fail("target", err)
		w.settle(t, d, store.Settle{Outcome: store.OutcomeRetry}, 0, started)
		return
	}
	if !ok || (d.Channel == store.ChannelPush && w.d.Push == nil) || (d.Channel == store.ChannelNtfy && w.d.Ntfy == nil) {
		w.settle(t, d, store.Settle{Outcome: store.OutcomeCancelled}, 0, started)
		return
	}
	title := "Medication due for " + tg.KidName
	path := "/kids/" + tg.KidID + "/medication"
	var (
		outcome    push.Outcome
		status     int
		retryAfter time.Duration
	)
	switch d.Channel {
	case store.ChannelPush:
		body, err := json.Marshal(payload{1, "medication", tg.KidID, title, path, "reminder-" + d.Reminder.ID})
		if err != nil {
			t.fail("payload", err)
			w.settle(t, d, store.Settle{Outcome: store.OutcomeFailed}, 0, started)
			return
		}
		ttl := d.Reminder.DueAt.Add(constants.ReminderExpiry).Sub(w.d.Now())
		ctx, cancel := context.WithTimeout(t.ctx, w.pushTimeout)
		r := w.d.Push.Send(ctx, push.Subscription{ID: tg.Push.ID, Endpoint: tg.Push.Endpoint, P256dh: tg.Push.P256dh, Auth: tg.Push.Auth},
			push.Message{Payload: body, TTL: min(constants.PushTTLMax, ttl), Topic: strings.ReplaceAll(d.Reminder.ID, "-", "")})
		cancel()
		outcome, status, retryAfter = r.Outcome, r.Status, r.RetryAfter
	case store.ChannelNtfy:
		ctx, cancel := context.WithTimeout(t.ctx, w.ntfyTimeout)
		r := w.d.Ntfy.Publish(ctx, tg.Ntfy.Topic, title, title, w.d.Origin+path)
		cancel()
		outcome, status, retryAfter = r.Outcome, r.Status, r.RetryAfter
		if r.ConfigFault {
			t.mu.Lock()
			t.ntfyRefused++
			if t.ntfyStatus == 0 {
				t.ntfyStatus = r.Status
			}
			t.mu.Unlock()
			w.settle(t, d, store.Settle{Outcome: store.OutcomeFailed}, status, time.Time{})
			return
		}
		if outcome == push.Gone {
			outcome = push.Failed // ntfy has no subscription to delete
		}
	}
	s := store.Settle{Outcome: store.OutcomeFailed}
	switch outcome {
	case push.Sent:
		s.Outcome = store.OutcomeSent
	case push.Gone:
		s.Outcome = store.OutcomeGone
	case push.Retry:
		s.Outcome = store.OutcomeRetry
		if retryAfter > 0 {
			s.RetryAfter = w.d.Now().Add(retryAfter)
		}
	}
	w.settle(t, d, s, status, started)
}

// settle stores the outcome and logs one line for it unless started is zero; a failed settle leaves the row sending,
// which the next start marks unconfirmed. A row gone since the claim (its course or kid deleted) is no failure.
func (w *Worker) settle(t *tick, d store.Delivery, s store.Settle, status int, started time.Time) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.ctx), constants.ReminderSettleTimeout)
	err := w.d.Store.SettleDelivery(ctx, d.ID, s, w.d.Now())
	cancel()
	if errors.Is(err, store.ErrNotFound) {
		return
	}
	if err != nil {
		t.fail("settle", err)
		return
	}
	if started.IsZero() {
		return
	}
	w.d.Log.Info("delivery", "delivery", d.ID, "reminder", d.Reminder.ID, "channel", d.Channel, "outcome", string(s.Outcome),
		"status", status, "attempt", d.Attempts, "duration_ms", time.Since(started).Milliseconds())
}
