package contracts

import "github.com/starframe-dev/go-mocks/env"

// EnvContract defines the expected interface for environment variable implementations.
type EnvContract interface {
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

// AssertEnvContract compiles at compile time that OSEnv and MapEnv implement EnvContract.
var (
	_ EnvContract = env.OSEnv{}
	_ EnvContract = (*env.MapEnv)(nil)
)
