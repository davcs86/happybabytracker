package auth

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func newClock() *fakeClock { return &fakeClock{t: time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)} }

func TestLimiterWindow(t *testing.T) {
	c := newClock()
	l := NewLimiter(5, c.now)
	for i := 1; i <= 5; i++ {
		if !l.Allow("a") {
			t.Fatalf("call %d refused", i)
		}
	}
	if l.Allow("a") {
		t.Fatal("6th call in the window allowed")
	}
	if !l.Allow("b") {
		t.Fatal("keys are not independent")
	}
	c.t = c.t.Add(time.Minute - time.Nanosecond)
	if l.Allow("a") {
		t.Fatal("allowed before the window elapsed")
	}
	c.t = c.t.Add(time.Nanosecond)
	if !l.Allow("a") {
		t.Fatal("refused after the window elapsed")
	}
}

func TestLimiterCountsRefusedCalls(t *testing.T) {
	c := newClock()
	l := NewLimiter(2, c.now)
	l.Allow("a")
	l.Allow("a")
	for i := 0; i < 50; i++ {
		if l.Allow("a") {
			t.Fatal("allowed past the limit")
		}
	}
}

func TestLimiterMapStaysBounded(t *testing.T) {
	c := newClock()
	l := NewLimiter(5, c.now)
	for i := 0; i < constants.LimiterMaxKeys+500; i++ {
		c.t = c.t.Add(time.Microsecond)
		if !l.Allow(fmt.Sprintf("k%d", i)) {
			t.Fatalf("new key %d refused", i)
		}
		if n := len(l.windows); n > constants.LimiterMaxKeys {
			t.Fatalf("map holds %d keys, bound is %d", n, constants.LimiterMaxKeys)
		}
	}
	if len(l.windows) != constants.LimiterMaxKeys {
		t.Fatalf("map holds %d keys, want it full at %d", len(l.windows), constants.LimiterMaxKeys)
	}
}

func TestLimiterEvictsOldestWindow(t *testing.T) {
	c := newClock()
	l := NewLimiter(5, c.now)
	for i := 0; i < constants.LimiterMaxKeys; i++ {
		c.t = c.t.Add(time.Microsecond)
		l.Allow(fmt.Sprintf("k%d", i))
	}
	c.t = c.t.Add(time.Microsecond)
	l.Allow("newcomer")
	if _, ok := l.windows["k0"]; ok {
		t.Fatal("oldest key survived eviction")
	}
	if _, ok := l.windows["k1"]; !ok {
		t.Fatal("second-oldest key was evicted")
	}
	if _, ok := l.windows["newcomer"]; !ok {
		t.Fatal("new key was not stored")
	}
}

func TestLimiterSweepsExpiredWindows(t *testing.T) {
	c := newClock()
	l := NewLimiter(5, c.now)
	for i := 0; i < constants.LimiterMaxKeys; i++ {
		l.Allow(fmt.Sprintf("k%d", i))
	}
	c.t = c.t.Add(time.Minute)
	l.Allow("fresh")
	if n := len(l.windows); n != 1 {
		t.Fatalf("map holds %d keys after the sweep, want 1", n)
	}
}

func TestLimiterConcurrent(t *testing.T) {
	const limit, workers, calls = 50, 8, 100
	l := NewLimiter(limit, func() time.Time { return time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC) })
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < calls; i++ {
				if l.Allow("shared") {
					allowed.Add(1)
				}
				l.Allow(fmt.Sprintf("own-%d-%d", w, i))
			}
		}(w)
	}
	wg.Wait()
	if got := allowed.Load(); got != limit {
		t.Fatalf("%d calls allowed on the shared key, want exactly %d", got, limit)
	}
}

func TestLimiterBlockedKeySurvivesFlood(t *testing.T) {
	c := newClock()
	l := NewLimiter(5, c.now)
	for i := 0; i < 6; i++ {
		l.Allow("victim")
	}
	for i := 0; i < constants.LimiterMaxKeys+500; i++ {
		c.t = c.t.Add(time.Microsecond)
		if !l.Allow(fmt.Sprintf("flood%d", i)) {
			t.Fatalf("new key %d refused", i)
		}
		if n := len(l.windows); n > constants.LimiterMaxKeys {
			t.Fatalf("map holds %d keys, bound is %d", n, constants.LimiterMaxKeys)
		}
	}
	if _, ok := l.windows["victim"]; !ok {
		t.Fatal("blocked key was evicted by a flood of fresh keys")
	}
	if l.Allow("victim") {
		t.Fatal("blocked key got a fresh window")
	}
}

func TestLimiterAllBlockedEvictsOldest(t *testing.T) {
	c := newClock()
	l := NewLimiter(5, c.now)
	for i := 0; i < constants.LimiterMaxKeys; i++ {
		c.t = c.t.Add(time.Microsecond)
		for j := 0; j < 5; j++ {
			l.Allow(fmt.Sprintf("k%d", i))
		}
	}
	c.t = c.t.Add(time.Microsecond)
	if !l.Allow("newcomer") {
		t.Fatal("new key refused while the map was full of blocked keys")
	}
	if n := len(l.windows); n != constants.LimiterMaxKeys {
		t.Fatalf("map holds %d keys, want %d", n, constants.LimiterMaxKeys)
	}
	if _, ok := l.windows["k0"]; ok {
		t.Fatal("oldest blocked key survived")
	}
	if _, ok := l.windows["k1"]; !ok {
		t.Fatal("second-oldest blocked key was evicted")
	}
	if _, ok := l.windows["newcomer"]; !ok {
		t.Fatal("new key was not stored")
	}
}
