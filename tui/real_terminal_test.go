package tui

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealTestTerminal_ExecuteUpdatesView(t *testing.T) {
	term := NewRealTestTerminal("/bin/app")

	view := term.Execute("ls -la")

	assert.Equal(t, View{Lines: []string{"$ ls -la"}}, view)
	assert.Equal(t, view, term.Read())
}

func TestRealTestTerminal_ReadBeforeAnyInput(t *testing.T) {
	term := NewRealTestTerminal("/bin/app")

	assert.Empty(t, term.Read().Lines)
}

func TestRealTestTerminal_SendAppendsInput(t *testing.T) {
	term := NewRealTestTerminal("/bin/app")
	term.Execute("login")

	require.NoError(t, term.Send("anya"))

	assert.Equal(t, []string{"$ login", "anya"}, term.Read().Lines)
}

func TestRealTestTerminal_CleanupWithoutProcess(t *testing.T) {
	term := NewRealTestTerminal("/bin/app")

	assert.NoError(t, term.Cleanup())
}

func TestRealTestTerminal_CleanupKillsProcess(t *testing.T) {
	term := NewRealTestTerminal("/bin/app")
	term.cmd = exec.Command("sleep", "30")
	require.NoError(t, term.cmd.Start())
	t.Cleanup(func() { _ = term.cmd.Wait() })

	require.NoError(t, term.Cleanup())

	assert.Error(t, term.cmd.Wait(), "killed process must report a non-nil wait error")
}
