//+build windows

// Pkg descr
package executil

import (
	"context"
	"os/exec"
)

type Cmd struct {
	*exec.Cmd
}

func Command(ctx context.Context, name string, arg ...string) *Cmd {
	innerCmd := exec.CommandContext(ctx, name, arg...)
	return &Cmd{Cmd: innerCmd}
}
