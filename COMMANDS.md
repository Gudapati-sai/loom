# loom — Command Reference

`loom` is a TUI project-setup wizard: it scaffolds the AGENTS.md kit into a
new or existing project, asking questions with explanations along the way and
falling back to a built-in question set when no local LLM is available.

Everything works from **any working directory** — the wizard resolves the
target itself, prints the absolute `target:` path, and runs from inside it.

---

## Global behavior

- **Target directory** — every project command takes an optional `[dir]`
  (default: the current directory). Relative paths resolve against the
  directory you run loom from.
- **Plan always confirmed** — `new` and `retrofit` show exactly which files
  would be written/kept inside the TUI and ask *Proceed?* before touching
  anything. The `-plan` flag is accepted for compatibility and does nothing.
- **LLM backend (attach-or-launch)** — loom probes the OpenAI-compatible
  endpoint and attaches if it's running; otherwise it launches it detached
  via `-unsloth-cmd` and waits until it answers. Any failure falls back to
  the built-in question set — never a blocker.
- **Colors** — Claude-inspired palette (terracotta `#D97757`) in a terminal;
  automatic clean black & white when piped, redirected, `NO_COLOR=1`, or
  `TERM=dumb`.

---

## Commands

### `loom` — interactive menu
Bare `loom` (no arguments) shows the logo banner and an interactive menu:
Scaffold a new project · Retrofit an existing repo · Resume a session ·
Show status · Build loom · Update loom · Help. Pick with `↑/↓` + `Enter`.

### `loom new [dir] [flags]` — scaffold a new project
Asks the welcome/stack/feature questions, then writes the 13-file AGENTS.md
kit plus a drafted PRD, a provenance log, and a machine-readable trace.

```bash
loom new ./my-project
loom new            # scaffold in the current directory
```

### `loom retrofit [dir] [flags]` — add the kit to an existing repo
Scans the repo (git, README, stack markers), shows the plan, writes any
missing fixed files, and asks before replacing existing ones (keep/replace
per file).

```bash
loom retrofit ./some-repo
```

### `loom resume [dir] [flags]` — continue an interrupted session
Loads `.loom-state.json` and only asks the questions that were never
answered; never re-asks.

```bash
loom resume ./my-project
```

### `loom status [dir]` — show session progress
Prints the saved session state as JSON (phase, answers, timestamps).

```bash
loom status ./my-project
```

### `loom build` — rebuild loom from source
Compiles the current source and swaps the fresh binary in place. Works from
any directory and any machine — the source tree is found via `LOOM_SRC`, or
next to the running binary if that directory has a `go.mod`. On Windows the
running exe can't be replaced, so the swap is finished by a short detached
helper once loom exits; the next run uses the new binary.

```bash
loom build
```

### `loom update` — pull latest source, then rebuild
`git pull` when the source is a git repo with a remote (a failed pull is a
warning, never fatal — offline it just rebuilds), then `loom build`.

```bash
loom update
```

### `loom help` — usage
Prints the banner and full usage.

```bash
loom help
```

---

## Flags

Flags work in any position (`loom new myproj -llm-url x` and
`loom new -llm-url x myproj` both work).

| Flag | Default | Description |
|---|---|---|
| `-llm-url` | `http://localhost:8888` | OpenAI-compatible server URL. Unsloth Studio (8888), Ollama (11434), llama.cpp (8080), LM Studio (1234). Env: `UNSLOTH_URL`. |
| `-llm-model` | `loom-wizard` | Model name requested from the server. |
| `-unsloth-cmd` | `unsloth studio` | Launch command for the backend when it isn't running. Empty value disables launching (attach-only). Env: `UNSLOTH_CMD`. |
| `-unsloth-timeout` | `45` | Seconds to wait for a launched backend to become ready. |
| `-plan` | — | Accepted for backwards compatibility; planning is always shown in the TUI. |

Example:

```bash
loom new ./demo -llm-url http://localhost:11434 -llm-model llama3.1 -unsloth-cmd ""
```

---

## Environment variables

| Variable | Purpose |
|---|---|
| `UNSLOTH_URL` | Default for `-llm-url` |
| `UNSLOTH_CMD` | Default for `-unsloth-cmd` |
| `LOOM_SRC` | Source tree used by `loom build` / `loom update` when the binary isn't next to `go.mod` |
| `NO_COLOR` | Disable colors (black & white) |

---

## Runtime artifacts

Written into the target project:

| File | Purpose |
|---|---|
| `AGENTS.md`, `.agent/`, `.ci/`, `docs/` | The kit (13 fixed files) |
| `README.md` | Rendered per project |
| `docs/prd/<date>-<feature>.md` | Drafted feature PRD |
| `.loom-state.json` | Resume state |
| `.loom-log.md` | Human-readable answer provenance |
| `.loom-trace.jsonl` | Machine-readable execution trace |
| `.loom-unsloth.log` | Backend process output (when launched) |

---

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success (or gracefully declined in the plan screen) |
| `1` | Error — usage error, build failure, or wizard failure |

---

## Examples

### Install (see README) then run

```bash
# any directory
loom                      # menu
loom new ./demo           # scaffold with plan confirmation
loom status ./demo        # inspect the session
loom retrofit ./repo      # kit an existing project
loom build                # rebuild loom itself
loom update               # git pull + rebuild
```

### PowerShell (5.1 has no `&&` — use `;` or separate lines)

```powershell
git clone https://github.com/Gudapati-sai/loom.git; cd loom; go build -o loom.exe .; .\loom.exe
# or, after installing:
loom new .\demo
```
