package random

import (
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSeed = 42

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestSeededRandom_Deterministic(t *testing.T) {
	a := NewSeededRandom(testSeed)
	b := NewSeededRandom(testSeed)

	for i := 0; i < 10; i++ {
		assert.Equal(t, a.Int63(), b.Int63())
		assert.Equal(t, a.Intn(100), b.Intn(100))
		assert.Equal(t, a.Float64(), b.Float64())
	}
}

func TestSeededRandom_DifferentSeedsDiffer(t *testing.T) {
	assert.NotEqual(t, NewSeededRandom(1).Int63(), NewSeededRandom(2).Int63())
}

func TestSeededRandom_Ranges(t *testing.T) {
	r := NewSeededRandom(testSeed)

	for i := 0; i < 1000; i++ {
		assert.Less(t, r.Intn(10), 10)
		assert.GreaterOrEqual(t, r.Intn(10), 0)
		f := r.Float64()
		assert.GreaterOrEqual(t, f, 0.0)
		assert.Less(t, f, 1.0)
		assert.GreaterOrEqual(t, r.Int63(), int64(0))
	}
}

func TestSeededRandom_Read(t *testing.T) {
	for _, size := range []int{0, 1, 7, 8, 9, 33} {
		a := make([]byte, size)
		b := make([]byte, size)

		na, errA := NewSeededRandom(testSeed).Read(a)
		nb, errB := NewSeededRandom(testSeed).Read(b)

		require.NoError(t, errA)
		require.NoError(t, errB)
		assert.Equal(t, size, na)
		assert.Equal(t, size, nb)
		assert.Equal(t, a, b)
	}
}

func TestSeededRandom_ConcurrentAccess(t *testing.T) {
	r := NewSeededRandom(testSeed)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Int63()
			_ = r.Intn(10)
			_ = r.Float64()
			_, _ = r.Read(make([]byte, 16))
		}()
	}
	wg.Wait()
}

func TestSequenceUUID_Sequence(t *testing.T) {
	g := NewSequenceUUID()

	assert.Equal(t, "00000000-0000-4000-8000-000000000001", g.NewUUID())
	assert.Equal(t, "00000000-0000-4000-8000-000000000002", g.NewUUID())
}

func TestSequenceUUID_ValidFormat(t *testing.T) {
	g := NewSequenceUUID()

	for i := 0; i < 300; i++ {
		assert.Regexp(t, uuidV4Pattern, g.NewUUID())
	}
}

func TestSequenceUUID_Reset(t *testing.T) {
	g := NewSequenceUUID()
	first := g.NewUUID()
	g.NewUUID()

	g.Reset()

	assert.Equal(t, first, g.NewUUID())
}

func TestSequenceUUID_ConcurrentUnique(t *testing.T) {
	g := NewSequenceUUID()
	const workers = 50

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		seen = make(map[string]struct{}, workers)
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := g.NewUUID()
			mu.Lock()
			seen[id] = struct{}{}
			mu.Unlock()
		}()
	}
	wg.Wait()

	assert.Len(t, seen, workers)
}

func TestRealUUID_ValidAndUnique(t *testing.T) {
	g := RealUUID{}

	a, b := g.NewUUID(), g.NewUUID()

	assert.Regexp(t, uuidV4Pattern, a)
	assert.Regexp(t, uuidV4Pattern, b)
	assert.NotEqual(t, a, b)
}

func TestRealRandom(t *testing.T) {
	r := RealRandom{}

	assert.GreaterOrEqual(t, r.Int63(), int64(0))
	assert.Less(t, r.Intn(5), 5)
	assert.Less(t, r.Float64(), 1.0)

	buf := make([]byte, 16)
	n, err := r.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, len(buf), n)
}
