# CLAUDE.md

Guidance for Claude (and other coding agents) working in this repo.

## What this project is

`verse-driven` ships a single Go binary (`scripture-mcp`) that serves canonical
scripture passages — KJV Bible, 道德经, 心经, Quran — to coding agents via
local stdio MCP, CLI, and hooks. Read `README.md` and `plan.md` first for the
full picture; `docs/issues-backlog.md` is the work plan.

## Working with scripture text — read this before touching packs

Sacred-text bodies trip Anthropic's output content filter when echoed at
volume in model output. A `400 invalid_request_error: Output blocked by
content filtering policy` will kill the turn mid-tool-call.

**Rule of thumb: scripture text should flow through scripts and files, never
through the model's text output.**

Practical guidance:

- **Don't paste verse text into your responses.** Don't quote passages in
  commit messages, PR bodies, code comments, or chat replies. Reference by
  citation only (`John 3:16`, `道德经第十一章`).
- **Don't echo verse text via Bash tool output.** Avoid `cat verses.jsonl`,
  `head -100 some_pack.txt`, `grep '...' kjv.txt` printed inline. If you need
  to inspect data, redirect to a file (`> /tmp/sample.txt`) and then read
  only structural metadata (line counts, checksums, first-token, field
  names) — not the text itself.
- **Pack builders are write-only.** A build script downloads upstream text
  and writes JSONL/JSON directly to `internal/packs/<name>/`. The model never
  sees the body. The model only writes the script, runs it, and verifies
  byte/verse counts and SHA-256 checksums.
- **Tests assert by checksum, not by content.** `internal/packs/*_test.go`
  should look up a known reference (e.g. `bible.kjv.john.3.16`) and compare
  its `checksum_sha256` to a hard-coded expected hex digest, not its `Text`.
  This keeps test files free of scripture text.
- **Verifier scripts read & hash, never print.** `scripts/verify_quotes.py`
  recomputes SHA-256 over each verse's text and compares to the stored
  checksum. On mismatch it prints the verse `id` and the two hashes — not
  the text.
- **If you must spot-check a verse manually,** do it locally: build the
  binary, run `scripture-mcp lookup "<ref>"`, eyeball the terminal. Don't
  copy the output back into the conversation.

If you hit the content filter mid-task, stop and refactor: move whatever was
about to be quoted into a file write or a script, then continue.

## Repo layout

```
cmd/scripture-mcp/      main package; CLI entrypoint
internal/schema/        Verse struct + JSON Schema (the contract every pack obeys)
internal/resolver/      free-form reference parser ("John 3:16", "道德经 11", ...)
internal/packs/         embed.FS-backed pack data + registry
  bible-kjv/
  dao-de-jing/
  heart-sutra/
internal/mcp/           stdio MCP server (issue #4)
internal/cli/           CLI subcommands (issue #4)
internal/injector/      inject-once envelope helpers (issues #5/#6)
scripts/                pack builders + verifiers (Python or Go, run at build time)
adapters/               per-agent wiring (claude-code, codex) — issues #5/#6
docs/                   issues-backlog.md, benchmarks/
```

## Building & testing

- `make build` → `bin/scripture-mcp`
- `make test` → `go test ./...`
- `make lint` → `go vet` + `staticcheck`
- CI runs all three on push and PR; staticcheck failures block.

## Conventions

- Module path: `github.com/MiaoDX/verse-driven`. Go 1.24.
- Verse IDs are dotted lowercase: `<tradition>.<work>.<book>.<chapter>.<verse>`,
  e.g. `bible.kjv.john.3.16`, `dao.daodejing.11.1`, `sutra.heart-sutra.1`.
- Every verse carries a SHA-256 over its `Text` bytes in `checksum_sha256`,
  computed at pack-build time. Hashes are the integrity boundary between
  upstream sources and the bundled binary.
- Pack metadata (`metadata.json`) lives next to `verses.jsonl` and records
  provider, license, attribution, source URL, and build date.
- Branches: feature work goes on `claude-issue-<number>`; one PR per issue.
