//go:build !windows

package backend

import (
	"os/exec"
	"syscall"
)

// detach starts the command in its own session so it keeps running after
// loom exits instead of being tied to loom's process group.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
