//go:build windows
// +build windows

package executil

import (
	"context"
	"os/exec"

	"github.com/ystia/yorc/v4/log"
)

// Cmd represents an external command being prepared or run.
//
// It's an  extension of exec.Cmd that kills the whole process tree instead of just the parent process
type Cmd struct {
	*exec.Cmd
}

// Command returns the Cmd struct to execute the named program with
// the given arguments.
//
// The provided context is used to kill the process if the context becomes done before the command
// completes on its own.
func Command(ctx context.Context, name string, arg ...string) *Cmd {
	log.Debugf("The standard command '%s %q' will be executed...", name, arg)
	innerCmd := exec.CommandContext(ctx, name, arg...)
	return &Cmd{Cmd: innerCmd}
}
