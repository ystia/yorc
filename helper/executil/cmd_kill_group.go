//+build !windows

package executil

import (
	"context"
	"os/exec"
	"syscall"

	"novaforge.bull.com/starlings-janus/janus/log"
)

// Cmd represents an external command being prepared or run.
//
// It's an  extension of exec.Cmd that kills the whole process tree instead of just the parent process
type Cmd struct {
	ctx context.Context
	*exec.Cmd
	waitDone chan struct{}
}

// Command returns the Cmd struct to execute the named program with
// the given arguments.
//
// The provided context is used to kill the process tree (by calling
// syscall.Kill(-c.Process.Pid, syscall.SIGKILL)) if the context becomes done before the command
// completes on its own.
func Command(ctx context.Context, name string, arg ...string) *Cmd {
	log.Debugf("The 'kill group' command '%s %q' will be executed...", name, arg)
	if ctx == nil {
		panic("nil Context")
	}
	innerCmd := exec.Command(name, arg...)
	cmd := &Cmd{ctx: ctx, Cmd: innerCmd, waitDone: make(chan struct{})}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd
}

// Run starts the specified command and waits for it to complete.
//
// The returned error is nil if the command runs, has no problems
// copying stdin, stdout, and stderr, and exits with a zero exit
// status.
//
// If the command fails to run or doesn't complete successfully, the
// error is of type *ExitError. Other error types may be
// returned for I/O problems.
func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// Start starts the specified command but does not wait for it to complete.
//
// The Wait method will return the exit code and release associated resources
// once the command exits.
func (c *Cmd) Start() error {
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	default:
	}

	go func() {
		select {
		case <-c.ctx.Done():
			if c.Process != nil {
				err := syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
				if err != nil {
					log.Fatal(err)
				}
			}
		case <-c.waitDone:
		}
	}()
	return c.Cmd.Start()
}

// Wait waits for the command to exit.
// It must have been started by Start.
//
// The returned error is nil if the command runs, has no problems
// copying stdin, stdout, and stderr, and exits with a zero exit
// status.
//
// If the command fails to run or doesn't complete successfully, the
// error is of type *ExitError. Other error types may be
// returned for I/O problems.
//
// If c.Stdin is not an *os.File, Wait also waits for the I/O loop
// copying from c.Stdin into the process's standard input
// to complete.
//
// Wait releases any resources associated with the Cmd.
func (c *Cmd) Wait() error {
	defer close(c.waitDone)
	return c.Cmd.Wait()
}
