package auth

import (
	"sync"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

type window struct {
	start time.Time
	count int
}

// Limiter is a fixed-window counter; a full map never refuses a new key.
type Limiter struct {
	mu        sync.Mutex
	perMinute int
	now       func() time.Time
	windows   map[string]window
}

func NewLimiter(perMinute int, now func() time.Time) *Limiter {
	return &Limiter{perMinute: perMinute, now: now, windows: make(map[string]window)}
}

// Allow counts every call, refused ones included.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	w, ok := l.windows[key]
	if !ok || !now.Before(w.start.Add(constants.LimiterWindow)) {
		if !ok && len(l.windows) >= constants.LimiterMaxKeys {
			l.makeRoom(now)
		}
		w = window{start: now}
	}
	w.count++
	l.windows[key] = w
	return w.count <= l.perMinute
}

// makeRoom drops expired windows, then the oldest one if the map is still full.
func (l *Limiter) makeRoom(now time.Time) {
	var oldestKey string
	var oldest time.Time
	found := false
	for k, w := range l.windows {
		if !now.Before(w.start.Add(constants.LimiterWindow)) {
			delete(l.windows, k)
			continue
		}
		if !found || w.start.Before(oldest) {
			oldestKey, oldest, found = k, w.start, true
		}
	}
	if len(l.windows) >= constants.LimiterMaxKeys && found {
		delete(l.windows, oldestKey)
	}
}
