package clock

import (
	"sort"
	"sync"
	"time"
)

// waiter is a pending After/Sleep call released once the fake time reaches deadline.
type waiter struct {
	deadline time.Time
	ch       chan time.Time
}

// FakeClock is a manually controlled Clock.
// Time only moves when Advance or Set is called; Sleep and After block until then.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []waiter
}

// NewFakeClock creates a fake clock that starts at the given time.
func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start}
}

// Now implements Clock.Now.
func (f *FakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Since implements Clock.Since.
func (f *FakeClock) Since(t time.Time) time.Duration {
	return f.Now().Sub(t)
}

// After implements Clock.After. A non-positive d fires immediately.
func (f *FakeClock) After(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()

	ch := make(chan time.Time, 1)
	if d <= 0 {
		ch <- f.now
		return ch
	}
	f.waiters = append(f.waiters, waiter{deadline: f.now.Add(d), ch: ch})
	return ch
}

// Sleep implements Clock.Sleep. It returns once the fake time advances by d.
func (f *FakeClock) Sleep(d time.Duration) {
	<-f.After(d)
}

// Advance moves the fake time forward by d and releases every due waiter.
func (f *FakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setLocked(f.now.Add(d))
}

// Set moves the fake time to t and releases every due waiter.
func (f *FakeClock) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setLocked(t)
}

// Pending returns the number of After/Sleep calls that are still waiting.
// Tests use it to wait until a goroutine has actually blocked on the clock.
func (f *FakeClock) Pending() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.waiters)
}

// setLocked updates the time and fires due waiters in deadline order.
// The caller must hold f.mu.
func (f *FakeClock) setLocked(t time.Time) {
	f.now = t

	sort.SliceStable(f.waiters, func(i, j int) bool {
		return f.waiters[i].deadline.Before(f.waiters[j].deadline)
	})

	remaining := f.waiters[:0]
	for _, w := range f.waiters {
		if w.deadline.After(f.now) {
			remaining = append(remaining, w)
			continue
		}
		w.ch <- f.now
	}
	f.waiters = remaining
}
