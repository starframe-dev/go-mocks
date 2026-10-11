package logger

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryHandler_RecordsEntries(t *testing.T) {
	h := NewMemoryHandler(nil)
	log := slog.New(h)

	log.Info("hello", "user", "anya", "count", 3)

	entries := h.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, slog.LevelInfo, entries[0].Level)
	assert.Equal(t, "hello", entries[0].Message)
	assert.Equal(t, "anya", entries[0].Attrs["user"])
	assert.EqualValues(t, 3, entries[0].Attrs["count"])
	assert.False(t, entries[0].Time.IsZero())
}

func TestMemoryHandler_LevelFiltering(t *testing.T) {
	h := NewMemoryHandler(slog.LevelWarn)
	log := slog.New(h)

	log.Debug("d")
	log.Info("i")
	log.Warn("w")
	log.Error("e")

	entries := h.Entries()
	require.Len(t, entries, 2)
	assert.Equal(t, "w", entries[0].Message)
	assert.Equal(t, "e", entries[1].Message)
	assert.True(t, h.Enabled(context.Background(), slog.LevelError))
	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
}

func TestMemoryHandler_Filter(t *testing.T) {
	h := NewMemoryHandler(nil)
	log := slog.New(h)

	log.Info("a")
	log.Error("b")
	log.Info("c")

	infos := h.Filter(slog.LevelInfo)
	require.Len(t, infos, 2)
	assert.Equal(t, "a", infos[0].Message)
	assert.Equal(t, "c", infos[1].Message)
	assert.Empty(t, h.Filter(slog.LevelWarn))
}

func TestMemoryHandler_WithAttrsSharesStore(t *testing.T) {
	h := NewMemoryHandler(nil)
	child := slog.New(h).With("request_id", "r1")

	child.Info("handled", "status", 200)

	entries := h.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, "r1", entries[0].Attrs["request_id"])
	assert.EqualValues(t, 200, entries[0].Attrs["status"])
}

func TestMemoryHandler_WithAttrsDoesNotLeakIntoParent(t *testing.T) {
	h := NewMemoryHandler(nil)
	parent := slog.New(h)
	_ = parent.With("scoped", true)

	parent.Info("plain")

	assert.NotContains(t, h.Entries()[0].Attrs, "scoped")
}

func TestMemoryHandler_Groups(t *testing.T) {
	h := NewMemoryHandler(nil)
	log := slog.New(h).WithGroup("http").With("method", "GET")

	log.Info("req", slog.Group("client", slog.String("ip", "10.0.0.1")), "path", "/")

	attrs := h.Entries()[0].Attrs
	assert.Equal(t, "GET", attrs["http.method"])
	assert.Equal(t, "10.0.0.1", attrs["http.client.ip"])
	assert.Equal(t, "/", attrs["http.path"])
}

func TestMemoryHandler_EmptyGroupAndEmptyAttrAreIgnored(t *testing.T) {
	h := NewMemoryHandler(nil)
	log := slog.New(h).WithGroup("")

	log.Info("x", slog.Attr{}, slog.Group("", slog.Int("inline", 1)))

	attrs := h.Entries()[0].Attrs
	assert.Len(t, attrs, 1)
	assert.EqualValues(t, 1, attrs["inline"])
}

func TestMemoryHandler_ResolvesLogValuer(t *testing.T) {
	h := NewMemoryHandler(nil)

	slog.New(h).Info("x", "secret", maskedValue{})

	assert.Equal(t, "***", h.Entries()[0].Attrs["secret"])
}

func TestMemoryHandler_EntriesAreCopies(t *testing.T) {
	h := NewMemoryHandler(nil)
	slog.New(h).Info("x", "k", "v")

	got := h.Entries()
	got[0].Attrs["k"] = "mutated"
	got[0].Message = "mutated"

	assert.Equal(t, "v", h.Entries()[0].Attrs["k"])
	assert.Equal(t, "x", h.Entries()[0].Message)
}

func TestMemoryHandler_Reset(t *testing.T) {
	h := NewMemoryHandler(nil)
	log := slog.New(h)
	log.Info("a")

	h.Reset()

	assert.Empty(t, h.Entries())
	log.Info("b")
	assert.Len(t, h.Entries(), 1)
}

func TestMemoryHandler_ConcurrentAccess(t *testing.T) {
	h := NewMemoryHandler(nil)
	log := slog.New(h)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); log.With("a", 1).Info("x") }()
		go func() { defer wg.Done(); _ = h.Entries() }()
	}
	wg.Wait()

	assert.Len(t, h.Entries(), 50)
}

// maskedValue is a slog.LogValuer used to verify values are resolved before capture.
type maskedValue struct{}

func (maskedValue) LogValue() slog.Value { return slog.StringValue("***") }
