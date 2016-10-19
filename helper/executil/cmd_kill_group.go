//+build !windows

// Pkg descr
package executil

import (
	"context"
	"os/exec"
	"syscall"
)

type Cmd struct {
	ctx context.Context
	*exec.Cmd
	waitDone chan struct{}
}

func Command(ctx context.Context, name string, arg ...string) *Cmd {
	if ctx == nil {
		panic("nil Context")
	}
	innerCmd := exec.Command(name, arg...)
	cmd := &Cmd{ctx: ctx, Cmd: innerCmd, waitDone: make(chan struct{})}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd
}

func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Cmd) Start() error {
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	default:
	}

	go func() {
		select {
		case <-c.ctx.Done():
			syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		case <-c.waitDone:
		}
	}()
	return c.Cmd.Start()
}

func (c *Cmd) Wait() error {
	defer close(c.waitDone)
	return c.Cmd.Wait()
}
