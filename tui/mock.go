package tui

import "sync"

// MockTerminal implements Terminal for unit testing.
type MockTerminal struct {
	mu   sync.RWMutex
	view View

	PrintFunc func(view View) error
}

// NewMockTerminal creates a new mock terminal.
func NewMockTerminal() *MockTerminal {
	return &MockTerminal{
		PrintFunc: func(view View) error {
			return nil
		},
	}
}

func (m *MockTerminal) Print(view View) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.view = view
	return m.PrintFunc(view)
}

func (m *MockTerminal) View() View {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.view
}
