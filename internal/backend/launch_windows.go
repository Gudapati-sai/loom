//go:build windows

package backend

import (
	"os/exec"
	"syscall"
)

// DETACHED_PROCESS (0x00000008) isn't exported by the syscall package on
// Windows, though CREATE_NEW_PROCESS_GROUP is; define it locally.
const detachedProcess = 0x00000008

// detach starts the command in its own process group with no console, so
// it keeps running after loom exits and doesn't pop up a window.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess,
	}
}
