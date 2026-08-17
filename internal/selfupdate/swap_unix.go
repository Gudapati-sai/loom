//go:build !windows

package selfupdate

import (
	"fmt"
	"os/exec"
	"syscall"
)

// scheduleSwap covers the rare case where the direct rename failed on a
// non-Windows platform (e.g. permissions). The helper waits briefly then
// moves the file.
func scheduleSwap(tmp, bin string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf(`sleep 1; mv -f '%s' '%s'`, tmp, bin))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
