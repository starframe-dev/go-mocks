package env

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// MapEnv is an in-memory, thread-safe Env.
type MapEnv struct {
	mu   sync.RWMutex
	vars map[string]string
}

// NewMapEnv creates an environment pre-filled with a copy of initial (may be nil).
func NewMapEnv(initial map[string]string) *MapEnv {
	vars := make(map[string]string, len(initial))
	for k, v := range initial {
		vars[k] = v
	}
	return &MapEnv{vars: vars}
}

// Get implements Env.Get.
func (m *MapEnv) Get(key string) string {
	value, _ := m.Lookup(key)
	return value
}

// Lookup implements Env.Lookup.
func (m *MapEnv) Lookup(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.vars[key]
	return value, ok
}

// Set implements Env.Set.
func (m *MapEnv) Set(key, value string) error {
	if !validKey(key) {
		return fmt.Errorf("%w: %q", ErrInvalidKey, key)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vars[key] = value
	return nil
}

// Unset implements Env.Unset. Unsetting a missing key is not an error.
func (m *MapEnv) Unset(key string) error {
	if !validKey(key) {
		return fmt.Errorf("%w: %q", ErrInvalidKey, key)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.vars, key)
	return nil
}

// Environ implements Env.Environ.
func (m *MapEnv) Environ() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pairs := make([]string, 0, len(m.vars))
	for k, v := range m.vars {
		pairs = append(pairs, k+"="+v)
	}
	return sortedCopy(pairs)
}

// validKey mirrors the OS rules: a key must be non-empty and contain neither '=' nor NUL.
func validKey(key string) bool {
	return key != "" && !strings.ContainsAny(key, "=\x00")
}

func sortedCopy(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}
