package contracts

import "github.com/human-horizon/go-mocks/tui"

// TerminalContract defines the expected interface for terminal implementations.
type TerminalContract interface {
	// Print renders the given view to the terminal.
	Print(view tui.View) error

	// View returns the current terminal state as a View.
	View() tui.View
}

// AssertTerminalContract compiles at compile time that MockTerminal implements TerminalContract.
var _ TerminalContract = (*tui.MockTerminal)(nil)
