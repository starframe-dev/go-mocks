package logger

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Compile-time check that MemoryHandler satisfies slog.Handler.
var _ slog.Handler = (*MemoryHandler)(nil)

// Entry is a captured log record. Attribute keys of groups are joined with dots ("group.key").
type Entry struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Attrs   map[string]any
}

// recorder is the entry store shared by a handler and every handler derived from it.
type recorder struct {
	mu      sync.RWMutex
	entries []Entry
}

// MemoryHandler is a slog.Handler that keeps records in memory for assertions.
// Use it as slog.New(handler); handlers returned by WithAttrs and WithGroup share the same store.
type MemoryHandler struct {
	rec    *recorder
	level  slog.Leveler
	prefix string
	attrs  map[string]any
}

// NewMemoryHandler creates a handler that records every entry at or above level.
// A nil level records everything (slog.LevelDebug and above).
func NewMemoryHandler(level slog.Leveler) *MemoryHandler {
	if level == nil {
		level = slog.LevelDebug
	}
	return &MemoryHandler{rec: &recorder{}, level: level}
}

// Enabled implements slog.Handler.
func (h *MemoryHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle implements slog.Handler.
func (h *MemoryHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := make(map[string]any, len(h.attrs)+record.NumAttrs())
	for k, v := range h.attrs {
		attrs[k] = v
	}
	record.Attrs(func(a slog.Attr) bool {
		flatten(attrs, h.prefix, a)
		return true
	})

	h.rec.mu.Lock()
	defer h.rec.mu.Unlock()
	h.rec.entries = append(h.rec.entries, Entry{
		Time:    record.Time,
		Level:   record.Level,
		Message: record.Message,
		Attrs:   attrs,
	})
	return nil
}

// WithAttrs implements slog.Handler.
func (h *MemoryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	merged := make(map[string]any, len(h.attrs)+len(attrs))
	for k, v := range h.attrs {
		merged[k] = v
	}
	for _, a := range attrs {
		flatten(merged, h.prefix, a)
	}
	derived := *h
	derived.attrs = merged
	return &derived
}

// WithGroup implements slog.Handler.
func (h *MemoryHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	derived := *h
	derived.prefix = h.prefix + name + "."
	return &derived
}

// Entries returns a copy of all captured entries.
func (h *MemoryHandler) Entries() []Entry {
	h.rec.mu.RLock()
	defer h.rec.mu.RUnlock()
	result := make([]Entry, len(h.rec.entries))
	for i, e := range h.rec.entries {
		result[i] = e.clone()
	}
	return result
}

// Filter returns the captured entries of exactly the given level.
func (h *MemoryHandler) Filter(level slog.Level) []Entry {
	var result []Entry
	for _, e := range h.Entries() {
		if e.Level == level {
			result = append(result, e)
		}
	}
	return result
}

// Reset clears all captured entries.
func (h *MemoryHandler) Reset() {
	h.rec.mu.Lock()
	defer h.rec.mu.Unlock()
	h.rec.entries = nil
}

func (e Entry) clone() Entry {
	attrs := make(map[string]any, len(e.Attrs))
	for k, v := range e.Attrs {
		attrs[k] = v
	}
	e.Attrs = attrs
	return e
}

// flatten stores a under prefix in dst, expanding groups into dotted keys.
func flatten(dst map[string]any, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() != slog.KindGroup {
		dst[prefix+a.Key] = a.Value.Any()
		return
	}

	groupPrefix := prefix
	if a.Key != "" {
		groupPrefix += a.Key + "."
	}
	for _, inner := range a.Value.Group() {
		flatten(dst, groupPrefix, inner)
	}
}
