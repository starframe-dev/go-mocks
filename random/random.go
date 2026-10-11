package random

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
)

// Random abstracts a source of random numbers and bytes.
// SeededRandom implements this interface for deterministic tests.
type Random interface {
	// Int63 returns a non-negative pseudo-random 63-bit integer.
	Int63() int64

	// Intn returns a pseudo-random number in [0, n). It panics if n <= 0.
	Intn(n int) int

	// Float64 returns a pseudo-random number in [0.0, 1.0).
	Float64() float64

	// Read fills p with random bytes.
	Read(p []byte) (int, error)
}

// UUIDGenerator abstracts creation of UUID strings.
type UUIDGenerator interface {
	// NewUUID returns a UUID in the canonical 8-4-4-4-12 form.
	NewUUID() string
}

// Compile-time checks that all implementations satisfy their interfaces.
var (
	_ Random        = RealRandom{}
	_ Random        = (*SeededRandom)(nil)
	_ UUIDGenerator = RealUUID{}
	_ UUIDGenerator = (*SequenceUUID)(nil)
)

// uuidVersion4 and uuidVariantRFC4122 are the bit patterns required by RFC 4122 for random UUIDs.
const (
	uuidVersion4       = 0x40
	uuidVersionMask    = 0x0f
	uuidVariantRFC4122 = 0x80
	uuidVariantMask    = 0x3f
	uuidSize           = 16
)

// RealRandom uses the process-wide math/rand source and crypto/rand for bytes (used in production).
type RealRandom struct{}

// Int63 implements Random.Int63.
func (RealRandom) Int63() int64 { return rand.Int63() }

// Intn implements Random.Intn.
func (RealRandom) Intn(n int) int { return rand.Intn(n) }

// Float64 implements Random.Float64.
func (RealRandom) Float64() float64 { return rand.Float64() }

// Read implements Random.Read using crypto/rand.
func (RealRandom) Read(p []byte) (int, error) { return crand.Read(p) }

// RealUUID generates random (version 4) UUIDs from crypto/rand (used in production).
type RealUUID struct{}

// NewUUID implements UUIDGenerator.NewUUID. It panics if the system entropy source fails.
func (RealUUID) NewUUID() string {
	var b [uuidSize]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(fmt.Errorf("random: read entropy: %w", err))
	}
	b[6] = b[6]&uuidVersionMask | uuidVersion4
	b[8] = b[8]&uuidVariantMask | uuidVariantRFC4122
	return formatUUID(b)
}

func formatUUID(b [uuidSize]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
