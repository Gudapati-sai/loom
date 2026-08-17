// Package backend manages the local LLM server loom talks to — by default
// Unsloth Studio's OpenAI-compatible server. It attaches to the server if
// it's already running, and otherwise launches it (detached, so it keeps
// running after loom exits and the next run simply attaches) and waits
// until it answers /v1/models. Every failure mode here is recoverable:
// callers treat a non-nil error from Ensure as "fall back to the built-in
// question set", never as a fatal failure.
package backend

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Gudapati-sai/loom/internal/llm"
)

// Launcher holds everything needed to attach to or start the backend.
type Launcher struct {
	URL       string        // base URL of the OpenAI-compatible server, e.g. http://localhost:8888
	Cmd       []string      // launch command and args; empty means attach-only (never spawn)
	TargetDir string        // working directory for the spawned process; its log lives here too
	Timeout   time.Duration // how long to wait for the backend to answer after launching
	Poll      time.Duration // interval between readiness probes
}

// StepFunc receives human-readable progress lines; the wizard wires this
// to its tracer so launch progress shows up in the same step output.
type StepFunc func(format string, args ...any)

// NewLauncher builds a Launcher from flag/env values. An empty cmd string
// yields an attach-only launcher that never spawns a process.
func NewLauncher(url, cmd, targetDir string, timeout time.Duration) *Launcher {
	return &Launcher{
		URL:       strings.TrimRight(url, "/"),
		Cmd:       strings.Fields(cmd),
		TargetDir: targetDir,
		Timeout:   timeout,
		Poll:      750 * time.Millisecond,
	}
}

// Running probes /v1/models, the standard OpenAI-compatible health
// endpoint. The reason string is non-empty on failure so callers can show
// *why* the server didn't answer, not just that it didn't.
func (l *Launcher) Running(ctx context.Context) (bool, string) {
	// The model name is irrelevant for a /v1/models probe.
	return llm.NewClient(l.URL, "").Available(ctx)
}

// Launch starts the backend detached, with the target project as its
// working directory and its output appended to .loom-unsloth.log there.
// Returns the started process and the log path.
func (l *Launcher) Launch() (*os.Process, string, error) {
	path, err := exec.LookPath(l.Cmd[0])
	if err != nil {
		return nil, "", fmt.Errorf("launch command %q not found on PATH", l.Cmd[0])
	}
	if err := os.MkdirAll(l.TargetDir, 0o755); err != nil {
		return nil, "", err
	}
	logPath := filepath.Join(l.TargetDir, ".loom-unsloth.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, "", err
	}
	cmd := exec.Command(path, l.Cmd[1:]...)
	cmd.Dir = l.TargetDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	detach(cmd)
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, "", err
	}
	// The child inherited the log file handle; the parent's copy can close.
	logFile.Close()
	return cmd.Process, logPath, nil
}

// WaitReady polls /v1/models until it answers or ctx expires. On timeout
// it returns the last failure reason so the caller can say something more
// useful than "timed out".
func (l *Launcher) WaitReady(ctx context.Context) (bool, string) {
	ticker := time.NewTicker(l.Poll)
	defer ticker.Stop()
	lastReason := ""
	for {
		ok, reason := l.Running(ctx)
		if ok {
			return true, ""
		}
		lastReason = reason
		select {
		case <-ctx.Done():
			return false, fmt.Sprintf("last probe: %s", lastReason)
		case <-ticker.C:
		}
	}
}

// Ensure makes a backend available: attaches if one is already answering
// at URL; otherwise, if a launch command is configured, spawns it detached
// and waits up to Timeout for it to answer. It returns a status string on
// success and a non-nil error when no backend could be made available. A
// backend that starts but never becomes ready is left running (it may
// still be loading a model) and the log path is included in the error so
// the user can follow along.
func (l *Launcher) Ensure(ctx context.Context, step StepFunc) (string, error) {
	ok, reason := l.Running(ctx)
	if ok {
		return fmt.Sprintf("attached to running backend at %s", l.URL), nil
	}
	if len(l.Cmd) == 0 {
		return "", fmt.Errorf("no backend running at %s (%s) and launching is disabled — using the built-in question set", l.URL, reason)
	}
	if step != nil {
		step("launching backend: %s", strings.Join(l.Cmd, " "))
	}
	_, logPath, err := l.Launch()
	if err != nil {
		return "", fmt.Errorf("could not launch backend %q: %v — using the built-in question set", strings.Join(l.Cmd, " "), err)
	}
	if step != nil {
		step("waiting for backend at %s (up to %s)", l.URL, l.Timeout)
	}
	waitCtx, cancel := context.WithTimeout(ctx, l.Timeout)
	defer cancel()
	ready, lastReason := l.WaitReady(waitCtx)
	if ready {
		return fmt.Sprintf("launched backend %q and attached at %s", strings.Join(l.Cmd, " "), l.URL), nil
	}
	return "", fmt.Errorf("backend %q started but never became ready at %s (%s) — output in %s; using the built-in question set",
		strings.Join(l.Cmd, " "), l.URL, lastReason, logPath)
}
