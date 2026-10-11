package clock

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var epoch = time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

const (
	waitTimeout = 2 * time.Second
	waitTick    = time.Millisecond
)

func receive(t *testing.T, ch <-chan time.Time) time.Time {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for clock channel")
		return time.Time{}
	}
}

func assertNotFired(t *testing.T, ch <-chan time.Time) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal("channel fired too early")
	default:
	}
}

func TestFakeClock_NowAndSince(t *testing.T) {
	f := NewFakeClock(epoch)

	assert.Equal(t, epoch, f.Now())
	assert.Equal(t, time.Duration(0), f.Since(epoch))

	f.Advance(90 * time.Second)

	assert.Equal(t, epoch.Add(90*time.Second), f.Now())
	assert.Equal(t, 90*time.Second, f.Since(epoch))
}

func TestFakeClock_Set(t *testing.T) {
	f := NewFakeClock(epoch)
	target := epoch.Add(24 * time.Hour)

	f.Set(target)

	assert.Equal(t, target, f.Now())
}

func TestFakeClock_AfterFiresOnlyWhenDue(t *testing.T) {
	f := NewFakeClock(epoch)
	ch := f.After(10 * time.Second)

	f.Advance(9 * time.Second)
	assertNotFired(t, ch)
	assert.Equal(t, 1, f.Pending())

	f.Advance(time.Second)
	assert.Equal(t, epoch.Add(10*time.Second), receive(t, ch))
	assert.Equal(t, 0, f.Pending())
}

func TestFakeClock_AfterNonPositiveFiresImmediately(t *testing.T) {
	f := NewFakeClock(epoch)

	assert.Equal(t, epoch, receive(t, f.After(0)))
	assert.Equal(t, epoch, receive(t, f.After(-time.Second)))
	assert.Equal(t, 0, f.Pending())
}

func TestFakeClock_MultipleWaiters(t *testing.T) {
	f := NewFakeClock(epoch)
	late := f.After(20 * time.Second)
	early := f.After(5 * time.Second)

	f.Advance(5 * time.Second)
	receive(t, early)
	assertNotFired(t, late)
	assert.Equal(t, 1, f.Pending())

	f.Advance(15 * time.Second)
	receive(t, late)
	assert.Equal(t, 0, f.Pending())
}

func TestFakeClock_SetReleasesWaiters(t *testing.T) {
	f := NewFakeClock(epoch)
	ch := f.After(time.Hour)

	f.Set(epoch.Add(2 * time.Hour))

	assert.Equal(t, epoch.Add(2*time.Hour), receive(t, ch))
}

func TestFakeClock_SleepBlocksUntilAdvance(t *testing.T) {
	f := NewFakeClock(epoch)
	done := make(chan struct{})

	go func() {
		f.Sleep(time.Minute)
		close(done)
	}()

	require.Eventually(t, func() bool { return f.Pending() == 1 }, waitTimeout, waitTick)
	select {
	case <-done:
		t.Fatal("Sleep returned before the clock advanced")
	default:
	}

	f.Advance(time.Minute)

	select {
	case <-done:
	case <-time.After(waitTimeout):
		t.Fatal("Sleep did not return after the clock advanced")
	}
}

func TestFakeClock_ConcurrentAccess(t *testing.T) {
	f := NewFakeClock(epoch)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); f.Advance(time.Millisecond) }()
		go func() { defer wg.Done(); _ = f.Now(); _ = f.Pending() }()
		go func() { defer wg.Done(); _ = f.After(time.Millisecond) }()
	}
	wg.Wait()

	f.Advance(time.Second)
	assert.Equal(t, 0, f.Pending())
}

func TestRealClock(t *testing.T) {
	var c Clock = RealClock{}

	start := c.Now()
	c.Sleep(time.Millisecond)

	assert.GreaterOrEqual(t, c.Since(start), time.Millisecond)
	receive(t, c.After(time.Millisecond))
}
