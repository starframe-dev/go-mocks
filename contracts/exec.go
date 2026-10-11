package contracts

import (
	"context"

	"github.com/starframe-dev/go-mocks/exec"
)

// ExecContract defines the expected interface for command runner implementations.
type ExecContract interface {
	// Run executes name with args and waits for it to finish.
	// A non-zero exit code is reported as *exec.ExitError together with a populated Result.
	Run(ctx context.Context, name string, args ...string) (exec.Result, error)
}

// AssertExecContract compiles at compile time that OSRunner and MockRunner implement ExecContract.
var (
	_ ExecContract = exec.OSRunner{}
	_ ExecContract = (*exec.MockRunner)(nil)
)
