package exec

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockRunner_RegisteredResult(t *testing.T) {
	m := NewMockRunner()
	m.Register("git", []string{"status"}, Result{Stdout: []byte("clean")}, nil)

	res, err := m.Run(context.Background(), "git", "status")

	require.NoError(t, err)
	assert.Equal(t, "clean", string(res.Stdout))
	assert.Equal(t, 0, res.ExitCode)
}

func TestMockRunner_NotRegistered(t *testing.T) {
	m := NewMockRunner()

	_, err := m.Run(context.Background(), "git", "push")

	assert.ErrorIs(t, err, ErrNotRegistered)
	assert.ErrorContains(t, err, "git push")
}

func TestMockRunner_ArgsMustMatchExactly(t *testing.T) {
	m := NewMockRunner()
	m.Register("echo", []string{"a", "b"}, Result{}, nil)

	_, err := m.Run(context.Background(), "echo", "a b")

	assert.ErrorIs(t, err, ErrNotRegistered)
}

func TestMockRunner_NonZeroExitBecomesExitError(t *testing.T) {
	m := NewMockRunner()
	m.Register("false", nil, Result{Stderr: []byte("boom"), ExitCode: 3}, nil)

	res, err := m.Run(context.Background(), "false")

	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 3, exitErr.Code)
	assert.Equal(t, "boom", string(res.Stderr))
	assert.Equal(t, 3, res.ExitCode)
}

func TestMockRunner_RegisteredError(t *testing.T) {
	m := NewMockRunner()
	m.Register("missing", nil, Result{}, assert.AnError)

	_, err := m.Run(context.Background(), "missing")

	assert.ErrorIs(t, err, assert.AnError)
}

func TestMockRunner_RunFuncTakesPriority(t *testing.T) {
	m := NewMockRunner()
	m.Register("echo", []string{"x"}, Result{Stdout: []byte("registered")}, nil)
	m.RunFunc = func(_ context.Context, name string, args ...string) (Result, error) {
		return Result{Stdout: []byte("callback:" + name)}, nil
	}

	res, err := m.Run(context.Background(), "echo", "x")

	require.NoError(t, err)
	assert.Equal(t, "callback:echo", string(res.Stdout))
}

func TestMockRunner_RunFuncReceivesContext(t *testing.T) {
	type ctxKey struct{}
	m := NewMockRunner()
	var got any
	m.RunFunc = func(ctx context.Context, _ string, _ ...string) (Result, error) {
		got = ctx.Value(ctxKey{})
		return Result{}, nil
	}

	_, err := m.Run(context.WithValue(context.Background(), ctxKey{}, "v"), "any")

	require.NoError(t, err)
	assert.Equal(t, "v", got)
}

func TestMockRunner_HistoryAndReset(t *testing.T) {
	m := NewMockRunner()
	m.Register("ls", []string{"-la"}, Result{}, nil)

	_, _ = m.Run(context.Background(), "ls", "-la")
	_, _ = m.Run(context.Background(), "pwd")

	history := m.History()
	require.Len(t, history, 2)
	assert.Equal(t, Call{Name: "ls", Args: []string{"-la"}}, history[0])
	assert.Equal(t, "ls -la", history[0].String())
	assert.Equal(t, "pwd", history[1].String())

	history[0].Args[0] = "mutated"
	assert.Equal(t, []string{"-la"}, m.History()[0].Args)

	m.Reset()
	assert.Empty(t, m.History())
	_, err := m.Run(context.Background(), "ls", "-la")
	assert.NoError(t, err, "registered results survive Reset")
}

func TestMockRunner_ConcurrentAccess(t *testing.T) {
	m := NewMockRunner()
	m.Register("echo", []string{"x"}, Result{}, nil)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = m.Run(context.Background(), "echo", "x") }()
		go func() { defer wg.Done(); _ = m.History() }()
	}
	wg.Wait()

	assert.Len(t, m.History(), 50)
}

func TestOSRunner_Success(t *testing.T) {
	res, err := OSRunner{}.Run(context.Background(), "go", "version")

	require.NoError(t, err)
	assert.Contains(t, string(res.Stdout), "go version")
}

func TestOSRunner_NonZeroExit(t *testing.T) {
	res, err := OSRunner{}.Run(context.Background(), "go", "definitely-not-a-subcommand")

	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.NotZero(t, exitErr.Code)
	assert.Equal(t, exitErr.Code, res.ExitCode)
	assert.NotEmpty(t, res.Stderr)
}

func TestOSRunner_CommandNotFound(t *testing.T) {
	_, err := OSRunner{}.Run(context.Background(), "definitely-not-a-real-binary-xyz")

	require.Error(t, err)
	var exitErr *ExitError
	assert.False(t, errors.As(err, &exitErr))
}
