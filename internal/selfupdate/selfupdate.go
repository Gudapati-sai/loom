// Package selfupdate implements `loom build` and `loom update` — loom
// rebuilding itself from source, so it works on any machine and from any
// pwd without machine-specific launcher scripts. The source tree is found
// via the LOOM_SRC env var, or next to the running binary if that
// directory has a go.mod. On Windows a running exe cannot be replaced, so
// the final swap is handed to a short detached helper that runs once this
// process has exited.
package selfupdate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const binName = "loom"

// SourceDir resolves the loom source tree. LOOM_SRC wins; otherwise the
// directory of the running binary is used when it contains go.mod.
func SourceDir() (string, error) {
	if v := os.Getenv("LOOM_SRC"); v != "" {
		dir, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
			return "", fmt.Errorf("LOOM_SRC=%s does not point at the loom source (no go.mod)", v)
		}
		return dir, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return dir, nil
	}
	return "", fmt.Errorf("loom source not found — run loom from its source directory or set LOOM_SRC to it")
}

// Build compiles the source and swaps the fresh binary in.
func Build() error {
	src, err := SourceDir()
	if err != nil {
		return err
	}
	bin := filepath.Join(src, binName)
	self := sameFile(bin)

	fmt.Println("● building loom from", src)
	tmp := filepath.Join(src, binName+".tmp")
	if err := buildTo(src, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, bin); err == nil {
		fmt.Println("✓ rebuilt", bin)
		return nil
	}
	// Direct swap failed — on Windows that means we're rebuilding the exe
	// we're running from, which is locked until we exit. Hand the swap to
	// a detached helper; it takes effect on the next run.
	if self {
		if err := scheduleSwap(tmp, bin); err != nil {
			return fmt.Errorf("built %s but could not replace %s (%v) — copy it over manually", tmp, bin, err)
		}
		fmt.Println("✓ rebuilt", bin, "— the running copy is swapped on exit, next run uses the new binary")
		return nil
	}
	return fmt.Errorf("built %s but could not replace %s — copy it over manually", tmp, bin)
}

// Update pulls the latest source when the source is a git repo with a
// remote, then rebuilds. A failed pull is a warning, never fatal — an
// offline or non-git source still rebuilds from local files.
func Update() error {
	src, err := SourceDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(src, ".git")); err == nil {
		cmd := exec.Command("git", "pull")
		cmd.Dir = src
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("⚠ git pull failed — continuing with the local source:", err)
		}
	} else {
		fmt.Println("no git repo at", src, "— skipping pull")
	}
	return Build()
}

// buildTo compiles the module in src into out, streaming compiler output.
func buildTo(src, out string) error {
	_ = os.Remove(out)
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = src
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}
	return nil
}

// sameFile reports whether path refers to the binary this process is
// running from (with symlinks resolved, so it works when launched via a
// symlink or the ~/bin wrapper).
func sameFile(path string) bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	a, err1 := filepath.EvalSymlinks(exe)
	b, err2 := filepath.EvalSymlinks(path)
	return err1 == nil && err2 == nil && a == b
}
