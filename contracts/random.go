package contracts

import "github.com/starframe-dev/go-mocks/random"

// RandomContract defines the expected interface for random number source implementations.
type RandomContract interface {
	// Int63 returns a non-negative pseudo-random 63-bit integer.
	Int63() int64

	// Intn returns a pseudo-random number in [0, n). It panics if n <= 0.
	Intn(n int) int

	// Float64 returns a pseudo-random number in [0.0, 1.0).
	Float64() float64

	// Read fills p with random bytes.
	Read(p []byte) (int, error)
}

// UUIDContract defines the expected interface for UUID generator implementations.
type UUIDContract interface {
	// NewUUID returns a UUID in the canonical 8-4-4-4-12 form.
	NewUUID() string
}

// AssertRandomContract compiles at compile time that all random implementations satisfy their contracts.
var (
	_ RandomContract = random.RealRandom{}
	_ RandomContract = (*random.SeededRandom)(nil)
	_ UUIDContract   = random.RealUUID{}
	_ UUIDContract   = (*random.SequenceUUID)(nil)
)
