# cdx.ps1 — Windows PowerShell counterpart of the POSIX `cdx` wrapper.
# Fires a Mode B scripture recap after the wrapped Codex session ends.
# Codex has no equivalent of Claude Code's `Stop` hook, so the recap runs
# in the wrapping PowerShell process, physically outside any Codex
# transcript or model_call input.
#
# Install: place this file on $env:PATH (or alias `cdx` to it) and run
# `cdx` instead of `codex`. The wrapper preserves the exit code of the
# wrapped `codex` invocation; recap fires on success and on non-zero exit.
#
# Limitation: launching `codex` directly bypasses this wrapper, so the
# recap is opt-in by use of `cdx`.

$ErrorActionPreference = 'Continue'

$codexBin = if ($env:CODEX_BIN) { $env:CODEX_BIN } else { 'codex' }
$scriptureBin = if ($env:SCRIPTURE_MCP_BIN) { $env:SCRIPTURE_MCP_BIN } else { 'scripture-mcp' }

if (-not (Get-Command $codexBin -ErrorAction SilentlyContinue)) {
    Write-Error "cdx: $codexBin not found on PATH"
    exit 127
}

& $codexBin @args
$exitCode = $LASTEXITCODE

if (Get-Command $scriptureBin -ErrorAction SilentlyContinue) {
    & $scriptureBin recap --terminal
} else {
    Write-Error "cdx: $scriptureBin not found on PATH; skipping recap"
}

exit $exitCode
