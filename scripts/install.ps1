# install.ps1 - one-command installer for loom (PowerShell / Windows).
#
# Builds loom from the GitHub source using the committed go.sum, so it works
# even where `go install @latest` fails on checksum-DB/proxy issues for a
# brand-new module. Installs to $(go env GOPATH)\bin\loom.exe and prints the
# PATH line if the Go bin dir isn't already on PATH.
#
# Usage:
#   git clone https://github.com/Gudapati-sai/loom.git; cd loom
#   powershell -ExecutionPolicy Bypass -File scripts\install.ps1
# (For a private repo, clone first — the raw one-liner needs a public repo.)
# Optional: pass a version tag, e.g. -Version v1.0.0

param([string]$Version = "v1.0.0")

$ErrorActionPreference = "Stop"

if (-not (Get-Command git -ErrorAction SilentlyContinue)) { throw "git is required - install it from https://git-scm.com" }
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw "Go is required - install it from https://go.dev/dl" }

$binDir = Join-Path (go env GOPATH) "bin"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

$tmp = Join-Path $env:TEMP "loom-install-$PID"
git clone -q --depth 1 --branch $Version https://github.com/Gudapati-sai/loom.git $tmp

Push-Location $tmp
try {
  go build -o (Join-Path $binDir "loom.exe") .
} finally {
  Pop-Location
  Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "  installed loom to $binDir\loom.exe" -ForegroundColor Green
$onPath = (($env:PATH -split ';') -contains $binDir)
if (-not $onPath) {
  Write-Host "  The Go bin directory is not on your PATH yet." -ForegroundColor Yellow
  Write-Host "  This session:    `$env:PATH += `";$binDir`""
  Write-Host "  Permanently:     [Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path','User') + ';$binDir', 'User')"
}
Write-Host ""
Write-Host "  Run:  loom help" -ForegroundColor Cyan
