# loom

TUI wizard that scaffolds a new project with the AGENTS.md kit (fixed files) already in place, or retrofits it into an existing repo — asking questions with explanations along the way, falling back to a built-in question set if no local model is running.

The LLM backend is **attach-or-launch**: if Unsloth Studio is already running, loom attaches to it; if not, it launches it through loom's own launch command (`unsloth studio` by default) and waits until it answers — so the backend is there when you need it without you starting it by hand. (Pass `-unsloth-cmd ""` to disable launching and go back to attach-only.)

## Build
```
go mod tidy    # first time only — fetches bubbletea/lipgloss
go build -o loom .
```
(`go.mod` has `replace` lines pointing `golang.org/x/*` at their `github.com/golang/*` mirrors — a workaround for this sandbox's restricted network, harmless to keep, or delete them and re-run `go mod tidy` on a normal machine.)

## Run
```
./loom new ./my-project        # scaffold a new project
./loom retrofit ./some-repo    # adopt the kit into an existing repo
./loom resume ./my-project     # continue an interrupted session
./loom status ./my-project     # show progress
./loom build                   # rebuild loom from source (works from any directory)
./loom update                  # pull latest source (if a git repo) then rebuild
./loom help
```

Every run prints the resolved absolute `target:` path first — no ambiguity about which directory it's operating on, regardless of where you're standing when you run it. The wizard then **runs from inside that directory** (it chdirs into the target), so anything it spawns — like the LLM backend — inherits the project as its working directory.

Flags work in any position now (`loom new myproj -llm-url x` and `loom new -llm-url x myproj` both work):
- The **file plan is always shown inside the TUI and confirmed before anything is written** — both `new` and `retrofit` print exactly which files would be created/kept and ask "Proceed?" first. (The old `-plan` flag is accepted but does nothing; planning is always on.)
- `-llm-url` / `-llm-model` — point at a local **OpenAI-compatible** server (Unsloth Studio, Ollama, llama.cpp's own `llama-server`, LM Studio, vLLM — whatever you're using to serve a model you trained with Unsloth). Default is `http://localhost:8888` (Unsloth Studio's OpenAI-compatible API; env `UNSLOTH_URL`); `/v1/...` is appended automatically. If nothing answers, it says so and falls back to the built-in question set — it never hangs waiting.
- `-unsloth-cmd` — launch command for the backend when it isn't running; `unsloth studio` by default (env `UNSLOTH_CMD`). The command is spawned **detached** with the target project as its working directory, its output appended to `.loom-unsloth.log` in the project, and loom polls `/v1/models` until it answers (up to `-unsloth-timeout` seconds, default 45). It keeps running after loom exits, so the next run simply attaches. If the command isn't on PATH or the backend never comes up, loom warns and falls back to the built-in question set — launching is an enhancement, never a blocker. Empty value (`-unsloth-cmd ""`) disables launching.

## Build & update are machine-portable

`loom build` and `loom update` work from any directory and any machine — nothing is hardcoded to a path. The source tree is found via the `LOOM_SRC` env var, or next to the running binary if that directory has a `go.mod`. `loom build` compiles and swaps the fresh binary in place; `loom update` also does `git pull` first when the source is a git repo (a failed pull is a warning, never fatal — offline it just rebuilds). On Windows a running exe can't be replaced, so the swap is handed to a short detached helper that finishes once loom exits — the next run uses the new binary. The `~/bin/loom` launcher follows the same `LOOM_SRC` rule.

## Traces
Every run prints Claude-Code-style step lines (`●` in progress, `✓` done, `⚠` warning, `✗` error) and appends the same entries as structured JSON to `.loom-trace.jsonl` in the target project — one object per line, local file only, never sent anywhere. `.loom-log.md` stays the human-readable answer transcript; `.loom-trace.jsonl` is the machine-readable execution record alongside it.

## Verified working (this build)
- `go build` / `go vet` clean.
- Full `loom new` run end-to-end through a real pty: welcome → stack choice → plan screen (always shown) → confirm → 13 fixed files written → PRD drafted → quality gate passed → state + provenance log + trace written.
- Full `loom retrofit` run: detects an existing repo, shows the plan first, writes missing fixed files untouched, asks before replacing a conflicting `AGENTS.md`, leaves `README.md`/`go.mod` alone.
- Plan screen confirmed in both `new` and `retrofit` — the TUI always shows what would be written and asks before touching disk.
- `status` / `resume` (including "already complete" and "no session found") all confirmed.
- `loom build` / `loom update` verified from a different working directory: rebuild in place, Windows self-replacement via the detached helper (tmp cleaned, next run uses the fresh binary), non-git source skips the pull gracefully.

## Two bugs fixed since the first build
1. **LLM backend wasn't reaching Unsloth-served models.** The client only spoke Ollama's proprietary `/api/generate`. Switched to the standard OpenAI-compatible `/v1/chat/completions` + `/v1/models`, which llama.cpp server, Ollama, LM Studio, and vLLM all support — so whatever you're actually using to serve an Unsloth-exported model should now be reachable. It also now tells you *why* it couldn't connect, instead of failing silently.
2. **Flags after the directory argument were being dropped.** `loom new myproject -llm-model x` silently ignored `-llm-model` because Go's `flag` package stops parsing at the first non-flag token. Replaced with order-independent parsing, and every run now prints its resolved absolute target path up front so there's no question which directory it's touching.

## Two more bugs fixed: why it felt "fixed" even with a model connected
3. **The model's own options were being thrown away.** `feature_name`/`feature_problem`/`feature_criteria` always called the plain-text screen no matter what the model returned — a working model could send back real, reasoned options and the UI would still show a generic text box. Added an `ask()` helper that actually checks the response and shows a select screen when options are present.
4. **Every prompt sent to the model was context-blind.** None of the questions told the model the project name, stack, or feature already chosen, so even a connected model couldn't tailor anything. Added a running context summary passed into every call — confirmed by logging what a fake local model actually received: by the last question it read *"project name: X; stack: Python; feature being built: Y; problem it solves: Z"* before being asked for acceptance criteria, and its real multi-option response (with real explanations) rendered and got selected correctly, verified against a fake OpenAI-compatible server driven through a real pty.

## Fixed in the attach-or-launch build
5. **Backend was check-only, never started.** The wizard probed the model URL once and gave up on failure. Now it's attach-or-launch: probe → if down, spawn `-unsloth-cmd` detached → poll `/v1/models` until ready → attach, with clear tracer lines at each step (`internal/backend`). Verified end-to-end against a mock server: launched, attached, process survives loom's exit, second run attaches without re-launching.
6. **Commands/paths only ever resolved from the process's current directory.** The wizard now chdirs into the resolved target directory before running, so spawned processes (the backend) run with the project as their cwd — and the backend's output lands in `.loom-unsloth.log` in the project, not in the shell's directory.
7. **HTTP status ignored on chat calls.** `AskQuestion` decodes the body without checking status, so a 400/500 that returns valid JSON was misreported as "model returned no choices". Non-2xx now yields a clear "server responded N" error.
8. **Retry loop had no backoff and a missing-key hole.** The two model attempts ran back-to-back (a freshly launched backend could still be settling), and an unknown phase key returned a zero-value `Question` (blank screen). Added a short pause before the retry and a safe free-text fallback for unknown keys.
9. **Dead code and cosmetic path bugs.** Removed the unreachable `absOrDir` helper; `slug()` now trims stray dashes and falls back to `feature` for all-non-ASCII names so a PRD filename can never be `2026-08-17-.md`; trace-flush errors are reported instead of silently discarded.

## Not built yet (from the PRD's open questions)
- No Unsloth fine-tune exists — `internal/llm` talks to any OpenAI-compatible server, but you'd still need to train and serve `loom-wizard` yourself.
- Retrofit's conflict UI is keep/replace only, no line-level merge.
- No test suite yet — the honest next step, per `AGENTS.md` in the generated projects: write the failing tests first.