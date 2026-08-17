//go:build windows

package selfupdate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const detachedProcess = 0x00000008

// scheduleSwap spawns a short-lived detached process that waits for the
// parent (the currently running loom) to exit, then swaps tmp over bin.
// Windows won't let a process replace its own exe while it's running.
//
// The script uses relative filenames with cmd.Dir set, never quoted
// absolute paths: `cmd /c` mangles quoted arguments (a known parsing
// quirk — even `move /y "a" "b"` fails with "syntax is incorrect"),
// while unquoted relative names work reliably regardless of spaces in
// the directory's absolute path. ping is a headless sleep because
// `timeout` needs a console and would fail detached. The helper's stdio
// goes to NUL so it never inherits (and holds open) loom's own stdout.
func scheduleSwap(tmp, bin string) error {
	script := fmt.Sprintf(
		`ping 127.0.0.1 -n 3 >nul & move /y %s %s`,
		filepath.Base(tmp), filepath.Base(bin),
	)
	cmd := exec.Command("cmd", "/c", script)
	cmd.Dir = filepath.Dir(tmp)
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer devnull.Close()
	cmd.Stdout = devnull
	cmd.Stderr = devnull
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess,
	}
	return cmd.Start()
}
