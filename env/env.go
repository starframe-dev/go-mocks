package env

import (
	"errors"
	"os"
)

// ErrInvalidKey is returned by Set and Unset for keys the OS would reject (empty or containing '=' or NUL).
var ErrInvalidKey = errors.New("env: invalid key")

// Env abstracts process environment variables.
// MapEnv implements this interface for testing.
type Env interface {
	// Get returns the value of key, or an empty string if it is not set.
	Get(key string) string

	// Lookup returns the value of key and whether it is set.
	Lookup(key string) (string, bool)

	// Set sets key to value.
	Set(key, value string) error

	// Unset removes key.
	Unset(key string) error

	// Environ returns all variables as a sorted "key=value" list.
	Environ() []string
}

// Compile-time checks that both implementations satisfy Env.
var (
	_ Env = OSEnv{}
	_ Env = (*MapEnv)(nil)
)

// OSEnv is the real process environment (used in production).
type OSEnv struct{}

// Get implements Env.Get.
func (OSEnv) Get(key string) string { return os.Getenv(key) }

// Lookup implements Env.Lookup.
func (OSEnv) Lookup(key string) (string, bool) { return os.LookupEnv(key) }

// Set implements Env.Set.
func (OSEnv) Set(key, value string) error { return os.Setenv(key, value) }

// Unset implements Env.Unset.
func (OSEnv) Unset(key string) error { return os.Unsetenv(key) }

// Environ implements Env.Environ.
func (OSEnv) Environ() []string { return sortedCopy(os.Environ()) }
