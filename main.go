package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"loom/internal/backend"
	"loom/internal/brand"
	"loom/internal/llm"
	"loom/internal/phase"
	"loom/internal/selfupdate"
	"loom/internal/tui"
	"loom/internal/wizard"
)

func main() {
	brand.ConfigureColor()

	if len(os.Args) < 2 {
		runMenu()
		return
	}
	cmd := os.Args[1]

	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		fmt.Print(brand.Banner())
		usage()
		return
	}

	dirArg, llmURL, llmModel, unslothCmd, unslothTimeout := parseArgs(os.Args[2:])

	dir, err := filepath.Abs(dirArg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving path:", err)
		os.Exit(1)
	}
	fmt.Println("target:", dir)
	chdirInto(dir)

	client := llm.NewClient(llmURL, llmModel)
	launcher := backend.NewLauncher(llmURL, unslothCmd, dir, unslothTimeout)

	switch cmd {
	case "build":
		err = selfupdate.Build()
	case "update":
		err = selfupdate.Update()
	case "new", "retrofit", "resume":
		fmt.Print(brand.Banner())
		switch cmd {
		case "new":
			err = wizard.RunNew(dir, client, launcher)
		case "retrofit":
			err = wizard.RunRetrofit(dir, client, launcher)
		case "resume":
			err = runResume(dir, client, launcher)
		}
	case "status":
		runStatus(dir)
		return
	default:
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// runMenu is the bare-`loom` entry point: a branded menu of everything
// loom can do, driven by the same TUI screens as the wizard — the way
// Claude Code / grok present a project tool, rather than a usage dump.
func runMenu() {
	fmt.Print(brand.Banner())
	dir, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving path:", err)
		os.Exit(1)
	}
	chdirInto(dir)
	_, llmURL, llmModel, unslothCmd, unslothTimeout := parseArgs(nil)
	client := llm.NewClient(llmURL, llmModel)
	launcher := backend.NewLauncher(llmURL, unslothCmd, dir, unslothTimeout)

	choice, _, err := tui.RunSelect("What would you like to do?", []tui.Option{
		{Label: "Scaffold a new project", Explanation: "loom new — create a project with the AGENTS.md kit (plan confirmed in the TUI first)."},
		{Label: "Retrofit an existing repo", Explanation: "loom retrofit — add the kit to the project in this directory."},
		{Label: "Resume an in-progress session", Explanation: "loom resume — continue where the last run stopped."},
		{Label: "Show session status", Explanation: "loom status — print the saved session as JSON."},
		{Label: "Build loom from source", Explanation: "loom build — recompile this binary from any directory."},
		{Label: "Update loom from git", Explanation: "loom update — git pull (if possible) then rebuild."},
		{Label: "Help", Explanation: "Show usage for every command and flag."},
	}, false)
	if err != nil {
		return
	}
	switch choice {
	case "Scaffold a new project":
		err = wizard.RunNew(dir, client, launcher)
	case "Retrofit an existing repo":
		err = wizard.RunRetrofit(dir, client, launcher)
	case "Resume an in-progress session":
		err = runResume(dir, client, launcher)
	case "Show session status":
		runStatus(dir)
		return
	case "Build loom from source":
		err = selfupdate.Build()
	case "Update loom from git":
		err = selfupdate.Update()
	default: // Help
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// chdirInto makes the wizard run from inside the target project, so
// anything it spawns (e.g. the LLM backend) inherits the project as its
// working directory. File writes are all joined against the already
// resolved absolute targetDir, so they're unaffected by the chdir. A
// target that doesn't exist yet (`loom new ./fresh`) is created later by
// the wizard — there's nothing to change into yet, so it's skipped.
func chdirInto(dir string) {
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return
	}
	if err := os.Chdir(dir); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not change into", dir+":", err)
	}
}

// parseArgs pulls flags out of args regardless of where they appear
// relative to the positional directory argument — `loom new -llm-model x
// myproject` and `loom new myproject -llm-model x` both work. Go's
// standard flag package only supports the first form (it stops parsing at
// the first non-flag token), which was silently swallowing flags placed
// after the directory.
func parseArgs(args []string) (dir, llmURL, llmModel, unslothCmd string, unslothTimeout time.Duration) {
	dir = "."
	llmURL = "http://localhost:8888" // Unsloth Studio's OpenAI-compatible server; override for Ollama (11434), llama.cpp (8080), LM Studio (1234)
	llmModel = "loom-wizard"
	unslothCmd = "unsloth studio"
	unslothTimeout = 45 * time.Second
	if v := os.Getenv("UNSLOTH_CMD"); v != "" {
		unslothCmd = v
	}
	if v := os.Getenv("UNSLOTH_URL"); v != "" {
		llmURL = v
	}

	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-llm-url" || a == "--llm-url":
			if i+1 < len(args) {
				llmURL = args[i+1]
				i++
			}
		case a == "-llm-model" || a == "--llm-model":
			if i+1 < len(args) {
				llmModel = args[i+1]
				i++
			}
		case a == "-unsloth-cmd" || a == "--unsloth-cmd":
			if i+1 < len(args) {
				unslothCmd = args[i+1]
				i++
			}
		case a == "-unsloth-timeout" || a == "--unsloth-timeout":
			if i+1 < len(args) {
				if secs, err := strconv.Atoi(args[i+1]); err == nil && secs > 0 {
					unslothTimeout = time.Duration(secs) * time.Second
				}
				i++
			}
		case a == "-plan" || a == "--plan":
			// Planning is always shown in the TUI now; the flag is accepted
			// for backwards compatibility and does nothing.
		case strings.HasPrefix(a, "-llm-url="):
			llmURL = strings.TrimPrefix(a, "-llm-url=")
		case strings.HasPrefix(a, "-llm-model="):
			llmModel = strings.TrimPrefix(a, "-llm-model=")
		case strings.HasPrefix(a, "-unsloth-cmd="):
			unslothCmd = strings.TrimPrefix(a, "-unsloth-cmd=")
		case strings.HasPrefix(a, "-unsloth-timeout="):
			if secs, err := strconv.Atoi(strings.TrimPrefix(a, "-unsloth-timeout=")); err == nil && secs > 0 {
				unslothTimeout = time.Duration(secs) * time.Second
			}
		default:
			positional = append(positional, a)
		}
	}
	if len(positional) > 0 {
		dir = positional[0]
	}
	return
}

func usage() {
	fmt.Println(`loom — TUI project-setup wizard

Usage:
  loom [menu]                  open the interactive menu (bare "loom")
  loom new [dir] [flags]       scaffold a new project (default dir: current directory)
  loom retrofit [dir] [flags]  adopt the kit into an existing repo (default: current directory)
  loom resume [dir] [flags]    continue an in-progress session (default: current directory)
  loom status [dir]            show current phase/progress (default: current directory)
  loom build                   rebuild the loom binary from source (LOOM_SRC, or the binary's own dir)
  loom update                  pull latest source (if a git repo) and rebuild
  loom help                    this help

Flags (work in any position):
  -llm-url string       local OpenAI-compatible model server — Unsloth Studio (8888),
                        Ollama (11434), llama.cpp server (8080), LM Studio (1234)
                        (default "http://localhost:8888"; env UNSLOTH_URL)
  -llm-model string     model name to request if the server is available (default "loom-wizard")
  -unsloth-cmd string   launch command for the LLM backend when it isn't running;
                        empty disables launching, attach-only (default "unsloth studio"; env UNSLOTH_CMD)
  -unsloth-timeout int  seconds to wait for a launched backend to become ready (default 45)

Every run prints the resolved absolute "target:" path first, and the
wizard then runs from inside that directory — anything it launches (like
the LLM backend) inherits the project as its working directory. The file
plan is always shown inside the TUI and confirmed before anything is
written.

The LLM backend is attach-or-launch: if Unsloth Studio is already running
it attaches; if not, it launches it via -unsloth-cmd and waits until it
answers. With no local model at all it falls back to a built-in question
set automatically, and says so.

"loom build" and "loom update" are self-updating and machine-portable: the
source tree is found via the LOOM_SRC env var, or next to the binary if
that directory has a go.mod.`)
}

func runResume(dir string, client *llm.Client, launcher *backend.Launcher) error {
	st, err := phase.Load(dir)
	if err != nil {
		return fmt.Errorf("no in-progress session found in %s", dir)
	}
	if st.Done {
		fmt.Println("this session is already complete —", st.ProjectName)
		return nil
	}
	if st.Mode == "retrofit" {
		return wizard.RunRetrofit(dir, client, launcher)
	}
	return wizard.RunNew(dir, client, launcher)
}

func runStatus(dir string) {
	st, err := phase.Load(dir)
	if err != nil {
		fmt.Println("no loom session found in", dir)
		return
	}
	b, _ := json.MarshalIndent(st, "", "  ")
	fmt.Println(string(b))
}
