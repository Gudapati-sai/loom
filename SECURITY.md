# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.0.x   | ✓         |

## Reporting a Vulnerability

**Do not file a public issue for security vulnerabilities.**

Instead, please report them privately by emailing **security@gudapati-sai.dev** (or use GitHub's "Report a vulnerability" advisory tab).

We will acknowledge receipt within 48 hours and aim to provide a fix within 7 days for critical issues. You will receive credit in the advisory unless you prefer anonymity.

## Out of Scope

The following are **not** considered vulnerabilities:
- Issues requiring local machine access (e.g., local file reads)
- Denial of service via malformed input that crashes the TUI (the app is a local developer tool)
- Missing features or UX preferences

## Hardening Practices (What This Project Does)

- **No network egress by default** — the only HTTP calls are to a local LLM endpoint (`http://localhost:8888/v1`); nothing phones home.
- **No shell injection** — all subprocesses are spawned with explicit argv arrays (`exec.Command(path, args...)`), never through a shell.
- **Least-privilege file writes** — output is scoped to the target project directory (`os.MkdirAll(targetDir)` only); no absolute-path escapes.
- **Static linking** — single binary, no runtime dependency resolution.
- **Pinned dependencies** — Go module checksums (`go.sum`) enforced; Dependabot alerts on vulnerable deps.
- **Secret scanning** — Gitleaks runs on every PR/push to catch API keys, tokens, passwords.
- **Vulnerability scanning** — `govulncheck` runs weekly on the dependency graph.