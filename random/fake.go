package random

import (
	"encoding/binary"
	"math/rand"
	"sync"
)

// SeededRandom is a deterministic, thread-safe Random: equal seeds produce equal sequences.
type SeededRandom struct {
	mu  sync.Mutex
	rnd *rand.Rand
}

// NewSeededRandom creates a deterministic random source from seed.
func NewSeededRandom(seed int64) *SeededRandom {
	return &SeededRandom{rnd: rand.New(rand.NewSource(seed))}
}

// Int63 implements Random.Int63.
func (s *SeededRandom) Int63() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rnd.Int63()
}

// Intn implements Random.Intn.
func (s *SeededRandom) Intn(n int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rnd.Intn(n)
}

// Float64 implements Random.Float64.
func (s *SeededRandom) Float64() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rnd.Float64()
}

// Read implements Random.Read. It never returns an error.
func (s *SeededRandom) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	const wordSize = 8
	var word [wordSize]byte
	for i := 0; i < len(p); i += wordSize {
		binary.LittleEndian.PutUint64(word[:], s.rnd.Uint64())
		copy(p[i:], word[:])
	}
	return len(p), nil
}

// SequenceUUID is a deterministic, thread-safe UUIDGenerator that returns
// 00000000-0000-4000-8000-000000000001, ...0002, and so on.
type SequenceUUID struct {
	mu      sync.Mutex
	counter uint64
}

// NewSequenceUUID creates a generator whose first UUID ends with 1.
func NewSequenceUUID() *SequenceUUID {
	return &SequenceUUID{}
}

// NewUUID implements UUIDGenerator.NewUUID.
func (s *SequenceUUID) NewUUID() string {
	s.mu.Lock()
	s.counter++
	n := s.counter
	s.mu.Unlock()

	var b [uuidSize]byte
	b[6] = uuidVersion4
	binary.BigEndian.PutUint64(b[8:], n)
	b[8] = b[8]&uuidVariantMask | uuidVariantRFC4122
	return formatUUID(b)
}

// Reset restarts the sequence from the beginning.
func (s *SequenceUUID) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter = 0
}
