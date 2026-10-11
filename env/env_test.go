package env

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapEnv_SetGetLookup(t *testing.T) {
	e := NewMapEnv(nil)

	require.NoError(t, e.Set("HOME", "/home/test"))

	assert.Equal(t, "/home/test", e.Get("HOME"))
	value, ok := e.Lookup("HOME")
	assert.True(t, ok)
	assert.Equal(t, "/home/test", value)
}

func TestMapEnv_MissingKey(t *testing.T) {
	e := NewMapEnv(nil)

	assert.Equal(t, "", e.Get("MISSING"))
	_, ok := e.Lookup("MISSING")
	assert.False(t, ok)
}

func TestMapEnv_EmptyValueIsSet(t *testing.T) {
	e := NewMapEnv(map[string]string{"EMPTY": ""})

	value, ok := e.Lookup("EMPTY")

	assert.True(t, ok)
	assert.Equal(t, "", value)
}

func TestMapEnv_InitialIsCopied(t *testing.T) {
	initial := map[string]string{"A": "1"}
	e := NewMapEnv(initial)

	initial["A"] = "changed"
	initial["B"] = "2"

	assert.Equal(t, "1", e.Get("A"))
	_, ok := e.Lookup("B")
	assert.False(t, ok)
}

func TestMapEnv_Unset(t *testing.T) {
	e := NewMapEnv(map[string]string{"A": "1"})

	require.NoError(t, e.Unset("A"))
	require.NoError(t, e.Unset("NEVER_SET"))

	_, ok := e.Lookup("A")
	assert.False(t, ok)
}

func TestMapEnv_InvalidKeys(t *testing.T) {
	e := NewMapEnv(nil)

	for _, key := range []string{"", "A=B", "A\x00B"} {
		t.Run(fmt.Sprintf("%q", key), func(t *testing.T) {
			assert.ErrorIs(t, e.Set(key, "v"), ErrInvalidKey)
			assert.ErrorIs(t, e.Unset(key), ErrInvalidKey)
		})
	}
	assert.Empty(t, e.Environ())
}

func TestMapEnv_EnvironSortedAndIsolated(t *testing.T) {
	e := NewMapEnv(map[string]string{"B": "2", "A": "1"})

	got := e.Environ()
	got[0] = "MUTATED=x"

	assert.Equal(t, []string{"A=1", "B=2"}, e.Environ())
}

func TestMapEnv_ConcurrentAccess(t *testing.T) {
	e := NewMapEnv(nil)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("K%d", i)
			_ = e.Set(key, "v")
			_ = e.Get(key)
			_ = e.Environ()
			_ = e.Unset(key)
		}(i)
	}
	wg.Wait()

	assert.Empty(t, e.Environ())
}

func TestOSEnv(t *testing.T) {
	const key = "GO_MOCKS_ENV_TEST_KEY"
	var e Env = OSEnv{}
	t.Cleanup(func() { _ = e.Unset(key) })

	_, ok := e.Lookup(key)
	require.False(t, ok)

	require.NoError(t, e.Set(key, "value"))
	assert.Equal(t, "value", e.Get(key))
	assert.Contains(t, e.Environ(), key+"=value")

	require.NoError(t, e.Unset(key))
	_, ok = e.Lookup(key)
	assert.False(t, ok)
}
