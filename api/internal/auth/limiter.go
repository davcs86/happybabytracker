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

// makeRoom drops expired windows, then the oldest unblocked one; a blocked key goes only when all are blocked.
func (l *Limiter) makeRoom(now time.Time) {
	var openKey, blockedKey string
	var openStart, blockedStart time.Time
	haveOpen, haveBlocked := false, false
	for k, w := range l.windows {
		if !now.Before(w.start.Add(constants.LimiterWindow)) {
			delete(l.windows, k)
			continue
		}
		if w.count >= l.perMinute {
			if !haveBlocked || w.start.Before(blockedStart) {
				blockedKey, blockedStart, haveBlocked = k, w.start, true
			}
		} else if !haveOpen || w.start.Before(openStart) {
			openKey, openStart, haveOpen = k, w.start, true
		}
	}
	if len(l.windows) < constants.LimiterMaxKeys {
		return
	}
	if haveOpen {
		delete(l.windows, openKey)
	} else if haveBlocked {
		delete(l.windows, blockedKey)
	}
}
