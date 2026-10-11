package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockTerminal_Print(t *testing.T) {
	m := NewMockTerminal()

	view := View{Lines: []string{"hello", "world"}}
	err := m.Print(view)

	assert.NoError(t, err)
	assert.Equal(t, view, m.View())
}

func TestMockTerminal_View_Empty(t *testing.T) {
	m := NewMockTerminal()
	assert.Equal(t, View{}, m.View())
}

func TestMockTerminal_PrintFunc(t *testing.T) {
	called := false
	m := NewMockTerminal()
	m.PrintFunc = func(view View) error {
		called = true
		return nil
	}

	m.Print(View{Lines: []string{"test"}})
	assert.True(t, called)
}

func TestMockTerminal_PrintFuncError(t *testing.T) {
	m := NewMockTerminal()
	m.PrintFunc = func(view View) error {
		return assert.AnError
	}

	err := m.Print(View{Lines: []string{"test"}})
	assert.ErrorIs(t, err, assert.AnError)
}

func TestInMemoryTestTerminal_Execute(t *testing.T) {
	term := NewInMemoryTestTerminal()
	term.SetOnExecute(func(cmd string) View {
		return View{Lines: []string{"output for: " + cmd}}
	})

	view := term.Execute("test-cmd")
	assert.Equal(t, "output for: test-cmd", view.Lines[0])
}

func TestInMemoryTestTerminal_ExecuteDefault(t *testing.T) {
	term := NewInMemoryTestTerminal()
	view := term.Execute("ls -la")

	assert.Contains(t, view.Lines[0], "ls -la")
}

func TestInMemoryTestTerminal_Send(t *testing.T) {
	term := NewInMemoryTestTerminal()
	term.SetOnSend(func(input string) View {
		return View{Lines: []string{"received: " + input}}
	})

	err := term.Send("answer")
	assert.NoError(t, err)
	assert.Equal(t, "received: answer", term.Read().Lines[0])
}

func TestInMemoryTestTerminal_History(t *testing.T) {
	term := NewInMemoryTestTerminal()
	term.Execute("cmd1")
	term.Send("input1")
	term.Execute("cmd2")

	history := term.History()
	assert.Equal(t, []string{"cmd1", "input1", "cmd2"}, history)
}

func TestInMemoryTestTerminal_HistoryEmpty(t *testing.T) {
	term := NewInMemoryTestTerminal()
	assert.Empty(t, term.History())
}

func TestMockTestTerminal_ExecuteFunc(t *testing.T) {
	m := NewMockTestTerminal()
	m.ExecuteFunc = func(cmd string) View {
		return View{Lines: []string{"executed: " + cmd}}
	}

	view := m.Execute("test")
	assert.Equal(t, "executed: test", view.Lines[0])
}

func TestMockTestTerminal_ExecuteNoFunc(t *testing.T) {
	m := NewMockTestTerminal()
	view := m.Execute("cmd")

	// Returns empty view when no func set
	assert.Empty(t, view.Lines)
}

func TestMockTestTerminal_ReadFunc(t *testing.T) {
	m := NewMockTestTerminal()
	m.ReadFunc = func() View {
		return View{Lines: []string{"read result"}}
	}

	view := m.Read()
	assert.Equal(t, "read result", view.Lines[0])
}

func TestMockTestTerminal_SendFunc(t *testing.T) {
	m := NewMockTestTerminal()
	m.SendFunc = func(input string) error {
		return nil
	}

	err := m.Send("input")
	assert.NoError(t, err)
	assert.Equal(t, []string{"input"}, m.History())
}

func TestMockTestTerminal_SendFuncError(t *testing.T) {
	m := NewMockTestTerminal()
	m.SendFunc = func(input string) error {
		return assert.AnError
	}

	err := m.Send("input")
	assert.ErrorIs(t, err, assert.AnError)
}

func TestMockTestTerminal_History(t *testing.T) {
	m := NewMockTestTerminal()
	m.Execute("cmd1")
	m.Send("inp1")
	m.Execute("cmd2")

	history := m.History()
	assert.Equal(t, []string{"cmd1", "inp1", "cmd2"}, history)
}

func TestMockTestTerminal_ConcurrentAccess(t *testing.T) {
	m := NewMockTestTerminal()
	m.ExecuteFunc = func(cmd string) View {
		return View{Lines: []string{cmd}}
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.Execute("cmd")
				m.Read()
				m.Send("input")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
	// If we get here without panic, concurrency is safe
}

func TestView(t *testing.T) {
	v := View{Lines: []string{"a", "b", "c"}}
	assert.Equal(t, 3, len(v.Lines))
	assert.Equal(t, "a", v.Lines[0])
}
