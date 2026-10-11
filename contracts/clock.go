package contracts

import (
	"time"

	"github.com/starframe-dev/go-mocks/clock"
)

// ClockContract defines the expected interface for clock implementations.
type ClockContract interface {
	// Now returns the current time.
	Now() time.Time

	// Since returns the time elapsed since t.
	Since(t time.Time) time.Duration

	// Sleep pauses the caller for at least d.
	Sleep(d time.Duration)

	// After returns a channel that receives the current time once d has elapsed.
	After(d time.Duration) <-chan time.Time
}

// AssertClockContract compiles at compile time that RealClock and FakeClock implement ClockContract.
var (
	_ ClockContract = clock.RealClock{}
	_ ClockContract = (*clock.FakeClock)(nil)
)
