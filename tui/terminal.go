package tui

// Terminal defines the interface for terminal rendering operations.
type Terminal interface {
	// Print renders the given view to the terminal.
	Print(view View) error

	// View returns the current terminal state as a View.
	View() View
}

// View represents the terminal's current rendered output.
type View struct {
	Lines []string
}

// TestTerminal is a real terminal for AI to run commands and inspect results.
// Used for integration tests where AI controls the terminal directly.
type TestTerminal interface {
	// Execute runs a command and returns the resulting view.
	Execute(cmd string) View

	// Read returns the current terminal state without executing anything.
	Read() View

	// Send sends input to the terminal (for interactive prompts, answers, etc.).
	Send(input string) error
}
