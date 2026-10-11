package exec

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrNotRegistered is returned by MockRunner for commands without a registered result or RunFunc.
var ErrNotRegistered = errors.New("exec: command not registered")

// argSeparator joins a command and its arguments into a lookup key.
// NUL cannot appear in real arguments, so distinct commands never collide.
const argSeparator = "\x00"

// Call records a single command invocation.
type Call struct {
	Name string
	Args []string
}

// String renders the call as a shell-like command line.
func (c Call) String() string {
	return strings.Join(append([]string{c.Name}, c.Args...), " ")
}

// registered is a canned outcome for one exact command.
type registered struct {
	result Result
	err    error
}

// MockRunner implements Runner for unit testing.
// Resolution order: RunFunc, then registered results, then ErrNotRegistered.
type MockRunner struct {
	mu        sync.RWMutex
	history   []Call
	responses map[string]registered

	RunFunc func(ctx context.Context, name string, args ...string) (Result, error)
}

// NewMockRunner creates a new mock runner.
func NewMockRunner() *MockRunner {
	return &MockRunner{responses: make(map[string]registered)}
}

// Register sets the outcome for the exact command name + args.
// A non-zero result.ExitCode with a nil err is reported as *ExitError, like a real process.
func (m *MockRunner) Register(name string, args []string, result Result, err error) {
	if err == nil && result.ExitCode != 0 {
		err = &ExitError{Code: result.ExitCode}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[commandKey(name, args)] = registered{result: result, err: err}
}

// Run implements Runner.Run.
func (m *MockRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	m.mu.Lock()
	m.history = append(m.history, Call{Name: name, Args: append([]string(nil), args...)})
	fn := m.RunFunc
	response, ok := m.responses[commandKey(name, args)]
	m.mu.Unlock()

	if fn != nil {
		return fn(ctx, name, args...)
	}
	if !ok {
		return Result{}, fmt.Errorf("%w: %s", ErrNotRegistered, Call{Name: name, Args: args})
	}
	return response.result, response.err
}

// History returns a copy of all recorded calls.
func (m *MockRunner) History() []Call {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Call, len(m.history))
	for i, c := range m.history {
		result[i] = Call{Name: c.Name, Args: append([]string(nil), c.Args...)}
	}
	return result
}

// Reset clears the call history. Registered results are kept.
func (m *MockRunner) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = nil
}

func commandKey(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), argSeparator)
}
