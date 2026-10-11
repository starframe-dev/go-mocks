package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	osexec "os/exec"
)

// Result holds the captured output of a finished command.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// ExitError is returned when a command finishes with a non-zero exit code.
type ExitError struct {
	Code int
}

// Error implements error.
func (e *ExitError) Error() string {
	return fmt.Sprintf("exec: command exited with code %d", e.Code)
}

// Runner abstracts running external commands.
// MockRunner implements this interface for testing.
type Runner interface {
	// Run executes name with args and waits for it to finish.
	// A non-zero exit code is reported as *ExitError together with a populated Result.
	Run(ctx context.Context, name string, args ...string) (Result, error)
}

// Compile-time checks that both implementations satisfy Runner.
var (
	_ Runner = OSRunner{}
	_ Runner = (*MockRunner)(nil)
)

// OSRunner runs real processes via os/exec (used in production).
type OSRunner struct{}

// Run implements Runner.Run.
func (OSRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	var stdout, stderr bytes.Buffer
	cmd := osexec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}

	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, &ExitError{Code: result.ExitCode}
	}
	return result, err
}
