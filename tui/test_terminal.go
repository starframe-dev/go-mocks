package tui

import (
	"fmt"
	"os/exec"
	"sync"
)

// RealTestTerminal implements TestTerminal for AI-driven integration tests.
// It spawns the actual application and communicates with it via stdin/stdout.
type RealTestTerminal struct {
	mu     sync.RWMutex
	view   View
	appBin string
	cmd    *exec.Cmd
}

var _ TestTerminal = (*RealTestTerminal)(nil)

// NewRealTestTerminal creates a new real test terminal for the given app binary.
func NewRealTestTerminal(appBin string) *RealTestTerminal {
	return &RealTestTerminal{
		appBin: appBin,
	}
}

// Execute runs a command and returns the resulting view.
func (t *RealTestTerminal) Execute(cmd string) View {
	t.mu.Lock()
	defer t.mu.Unlock()

	// TODO: implement actual command execution
	// For now, just update the view for testing
	t.view = View{Lines: []string{fmt.Sprintf("$ %s", cmd)}}
	return t.view
}

// Read returns the current terminal state.
func (t *RealTestTerminal) Read() View {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.view
}

// Send sends input to the terminal.
func (t *RealTestTerminal) Send(input string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// TODO: implement actual input sending
	t.view = View{Lines: append(t.view.Lines, input)}
	return nil
}

// Cleanup stops the terminal process.
func (t *RealTestTerminal) Cleanup() error {
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}

// InMemoryTestTerminal is a simpler in-memory implementation for testing.
// Useful when you don't need a real process but want to test command flow.
type InMemoryTestTerminal struct {
	mu       sync.RWMutex
	view     View
	history  []string
	onSend   func(input string) View
	onExecute func(cmd string) View
}

var _ TestTerminal = (*InMemoryTestTerminal)(nil)

// NewInMemoryTestTerminal creates a new in-memory test terminal.
func NewInMemoryTestTerminal() *InMemoryTestTerminal {
	return &InMemoryTestTerminal{
		view:    View{Lines: []string{}},
		history: []string{},
	}
}

// SetOnSend sets a callback for Send operations.
func (t *InMemoryTestTerminal) SetOnSend(fn func(input string) View) {
	t.onSend = fn
}

// SetOnExecute sets a callback for Execute operations.
func (t *InMemoryTestTerminal) SetOnExecute(fn func(cmd string) View) {
	t.onExecute = fn
}

func (t *InMemoryTestTerminal) Execute(cmd string) View {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.history = append(t.history, cmd)
	if t.onExecute != nil {
		t.view = t.onExecute(cmd)
	} else {
		t.view = View{Lines: []string{fmt.Sprintf("$ %s", cmd)}}
	}
	return t.view
}

func (t *InMemoryTestTerminal) Read() View {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.view
}

func (t *InMemoryTestTerminal) Send(input string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.history = append(t.history, input)
	if t.onSend != nil {
		t.view = t.onSend(input)
	} else {
		t.view = View{Lines: append(t.view.Lines, input)}
	}
	return nil
}

// History returns all executed commands and inputs.
func (t *InMemoryTestTerminal) History() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]string{}, t.history...)
}

// MockTestTerminal implements TestTerminal for testing purposes.
type MockTestTerminal struct {
	mu       sync.RWMutex
	view     View
	history  []string
	ExecuteFunc func(cmd string) View
	ReadFunc   func() View
	SendFunc   func(input string) error
}

var _ TestTerminal = (*MockTestTerminal)(nil)

// NewMockTestTerminal creates a new mock test terminal.
func NewMockTestTerminal() *MockTestTerminal {
	return &MockTestTerminal{}
}

func (m *MockTestTerminal) Execute(cmd string) View {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = append(m.history, cmd)
	if m.ExecuteFunc != nil {
		m.view = m.ExecuteFunc(cmd)
	}
	return m.view
}

func (m *MockTestTerminal) Read() View {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ReadFunc != nil {
		return m.ReadFunc()
	}
	return m.view
}

func (m *MockTestTerminal) Send(input string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = append(m.history, input)
	if m.SendFunc != nil {
		return m.SendFunc(input)
	}
	return nil
}

func (m *MockTestTerminal) History() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.history...)
}
