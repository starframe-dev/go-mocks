package clock

import "time"

// Clock abstracts access to the current time and to time-based waiting.
// FakeClock implements this interface for testing.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// Since returns the time elapsed since t.
	Since(t time.Time) time.Duration

	// Sleep pauses the caller for at least d.
	Sleep(d time.Duration)

	// After returns a channel that receives the current time once d has elapsed.
	After(d time.Duration) <-chan time.Time
}

// Compile-time checks that both implementations satisfy Clock.
var (
	_ Clock = RealClock{}
	_ Clock = (*FakeClock)(nil)
)

// RealClock is the wall clock (used in production).
type RealClock struct{}

// Now implements Clock.Now.
func (RealClock) Now() time.Time { return time.Now() }

// Since implements Clock.Since.
func (RealClock) Since(t time.Time) time.Duration { return time.Since(t) }

// Sleep implements Clock.Sleep.
func (RealClock) Sleep(d time.Duration) { time.Sleep(d) }

// After implements Clock.After.
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
