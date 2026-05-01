# Initial Issues Backlog

This file is the source of truth for the initial issues created from
[`plan.md`](../plan.md). Once issues are filed on GitHub, this file remains
as a historical record of the original intent. Issues may evolve on GitHub;
this file does not.

The work is split into 9 issues by **functional grouping and dependency**.
There's no time budget — each issue is the smallest unit that delivers
something demonstrable. Sequencing comes from the dependency graph at the
bottom of this file.

**Title convention:** `<type>(<scope>): <description>` so that PRs naturally
chain.

**Recommended labels to create up front:**
`area:foundation`, `area:core`, `area:claude`, `area:codex`, `area:test`,
`area:polish`.

---

## #1 — chore(foundation): Go skeleton, CI, and verse schema

**Area:** foundation **Depends on:** none

### What
Set up the Go module, package layout matching `plan.md` §4, minimal CI, and
the verse data schema that every pack must satisfy.

### Acceptance criteria
- [ ] `go.mod` initialized with module path `github.com/MiaoDX/verse-driven`
- [ ] Directory skeleton: `cmd/scripture-mcp/`, `internal/{packs,schema,resolver,mcp,cli,injector}/`
- [ ] `cmd/scripture-mcp/main.go` builds and prints `scripture-mcp v0.0.0`
- [ ] GitHub Actions workflow: `go vet` + `staticcheck` + `go test ./...` on push and PR
- [ ] `Makefile` with `build`, `test`, `lint` targets
- [ ] `internal/schema/verse.schema.json` matches `plan.md` §6.1
- [ ] `internal/schema/verse.go` defines the Go struct + `Validate(v Verse) error`
- [ ] Required fields enforced: `id`, `tradition`, `lang`, `text`, `canonical_ref`, `source`, `checksum_sha256`
- [ ] `inclusion_mode` ∈ `{bundled, api_only}`; `sensitivity` ∈ `{sacred_exact_quote}` (extensible)
- [ ] Schema tests cover happy path + each required-field omission + bad enum values

### Notes
The schema is the contract every pack builder must satisfy — lock it before
writing any pack builder. Keep the binary buildable from day one.

---

## #2 — feat(foundation): reference resolver with EN + ZH support

**Area:** foundation **Depends on:** #1

### What
Parse a free-form scripture reference into a canonical
`(tradition, work, ref)` tuple. Must handle the variants from `plan.md` §8.1.
Includes the full test suite — not split out — because the resolver is only
trustworthy if its tests cover the matrix.

### Acceptance criteria
- [ ] Parses Bible: `John 3:16`, `Jn 3:16`, `1 John 3:16`, `约翰福音 3:16`, `约翰一书 3:16`
- [ ] Parses Tao Te Ching: `dao 11`, `daodejing chapter 11`, `道德经 11`, `道德经第十一章`
- [ ] Parses Heart Sutra: `sutra heart`, `心经`, `般若波罗蜜多心经`
- [ ] Parses Quran: `Quran 2:255`, `2:255`, `Surah Al-Baqarah 255`
- [ ] Returns canonical `Reference{Tradition, Work, Book?, Chapter, VerseStart, VerseEnd?}`
- [ ] Ambiguous input (e.g. bare `3:16` with no tradition context) returns a typed error listing candidates
- [ ] Table-driven tests with ≥ 50 inputs across all 4 traditions, both languages, plus negatives

### Notes
This is the hardest "small" module in the project. Don't over-engineer with
NLP — a per-tradition lookup table of aliases is enough.

---

## #3 — feat(packs): bundled KJV, 道德经, and 心经

**Area:** foundation **Depends on:** #1

### What
Build the three v0.1 bundled packs. KJV first establishes the pattern; 道德经
and 心经 reuse it. All three embedded via `embed.FS`.

### Acceptance criteria
- [ ] `scripts/build_pack.py` (or Go) downloads KJV from Project Gutenberg, normalizes, writes `internal/packs/bible-kjv/verses.jsonl` with all ~31,102 verses
- [ ] 道德经 pack at `internal/packs/dao-de-jing/verses.jsonl` (81 chapters from Chinese Text Project)
- [ ] 心经 pack at `internal/packs/heart-sutra/verses.jsonl` (sentence-segmented, ~15 entries from CBETA)
- [ ] Every pack has `metadata.json` with provider / license / attribution per `plan.md` §6.2
- [ ] Every verse has SHA-256 `checksum_sha256` computed at build time
- [ ] `scripts/verify_quotes.py` recomputes checksums and fails CI on mismatch
- [ ] All three packs embed via `embed.FS` and are queryable from main
- [ ] Spot checks: `John 3:16` returns canonical KJV; 道德经 11 returns "三十辐共一毂..."
- [ ] Total bundled pack size < 6 MB

### Notes
For 心经, double-check CBETA's redistribution terms before vendoring the text;
fall back to api-only mode for that pack if uncertain.

---

## #4 — feat(core): stdio MCP server + CLI subcommands

**Area:** core **Depends on:** #2, #3

### What
The full surface of the binary. Same binary exposes:
- `serve` — stdio MCP for cc/codex
- `lookup`, `lookup-from-prompt` — for hooks
- `recap` — for Mode B
- `init` — for adapter wiring

### Acceptance criteria
- [ ] `scripture-mcp serve` launches a stdio MCP server
- [ ] MCP tools: `lookup`, `search`, `random`, `list_traditions` — all return verses with `checksum_sha256` and `source`
- [ ] Connects successfully to Claude Code via the snippet in `plan.md` §5.1
- [ ] Connects successfully to Codex via the snippet in §5.2
- [ ] `scripture-mcp lookup "<ref>" --format=json` → JSON Verse on stdout
- [ ] `scripture-mcp lookup-from-prompt` reads stdin, emits `additionalContext` JSON suitable for both `UserPromptExpansion` (Claude) and `UserPromptSubmit` (Codex)
- [ ] `scripture-mcp recap --tradition=<t> --terminal` prints pretty terminal output, exit 0
- [ ] `scripture-mcp recap --first-letter` prints first-letter pattern
- [ ] `scripture-mcp init --target={claude-code,codex}` merges config snippet into the user's existing config without overwriting; idempotent
- [ ] p50 latency for MCP `lookup` < 5 ms

### Notes
`lookup-from-prompt` is the integration point for both platforms — keep its
I/O format identical so the Claude and Codex hooks are nearly the same script.

---

## #5 — feat(claude): Claude Code adapter (Mode A + Mode B + skill)

**Area:** claude **Depends on:** #4

### What
The full Claude Code adapter. Mode A (manual one-turn injection), Mode B
(terminal-only recap), plus the manually-invokable skill — must be validated
together because they share preview/inject infrastructure.

### Acceptance criteria

**Mode A — UserPromptExpansion hook:**
- [ ] `/bible John 3:16` shows a preview card with verse text + source + checksum
- [ ] User must explicitly confirm before the model sees the verse
- [ ] On confirm, the next coding request is wrapped with the temporary envelope from `plan.md` §6.3
- [ ] `keep-coding-instructions: true` set on the active output style
- [ ] Works for all 4 marker traditions: `bible`, `dao`, `sutra`, `quran`
- [ ] Hook exits silently if the prompt has no marker

**Mode A — `verse-inject` skill:**
- [ ] `adapters/claude-code/skills/verse-inject/SKILL.md` exists and is < 60 lines
- [ ] `disable-model-invocation: true` in frontmatter
- [ ] Skill body contains **no scripture text** — calls `scripture-mcp lookup` via MCP
- [ ] Manually invoking the skill triggers the same preview/confirm/inject flow

**Mode B — output style + Stop hook:**
- [ ] `adapters/claude-code/output-styles/scripture-recap.md` defines style only — no scripture text or specific traditions
- [ ] Stop hook calls `scripture-mcp recap --terminal`; recap reaches the user's terminal
- [ ] **Critical:** the recap text does NOT appear in the next prompt's input (verified manually with a follow-up test prompt)
- [ ] Mode B toggleable via `scripture-mcp init --target=claude-code --recap=on|off`

### Notes
The "recap text never enters next prompt" invariant is what separates VDD
from naive output-style approaches. Hook output goes to terminal stdout,
NOT to model input. Issue #8 verifies this rigorously; this issue verifies
it manually as a smoke test.

---

## #6 — feat(codex): Codex adapter (Mode A + Mode B)

**Area:** codex **Depends on:** #4

### What
The full Codex adapter. Mode A via `UserPromptSubmit` hook with marker
syntax `[[bible:John 3:16]]`. Mode B via shell wrapper `cdx` since Codex has
no `Stop` hook equivalent.

### Acceptance criteria

**Mode A:**
- [ ] `adapters/codex/skills/scripture-lookup/SKILL.md` exists
- [ ] `agents/openai.yaml` sets `allow_implicit_invocation: false`
- [ ] `[features] codex_hooks = true` in config template
- [ ] `UserPromptSubmit` hook calls `scripture-mcp lookup-from-prompt`
- [ ] Submitting `[[bible:John 3:16]] Refactor X.` → model sees verse only this turn
- [ ] Without a marker, hook exits with no `additionalContext`

**Mode B:**
- [ ] `adapters/codex/wrapper/cdx` (POSIX shell + Windows `.ps1`) wraps `codex` invocation
- [ ] After Codex exits, wrapper calls `scripture-mcp recap --terminal`
- [ ] Recap fires on success and on non-zero exit
- [ ] `scripture-mcp init --target=codex --recap=on` adds `cdx` to PATH and prints alias setup
- [ ] README documents the limitation honestly: launching `codex` directly bypasses the recap

### Notes
README must call out which Codex version this was tested against, and which
skill path (`~/.agents/skills` vs `$CODEX_HOME/skills`) was used — official
docs and OSS repos disagree on this and we need to pick one.

---

## #7 — feat(install): one-line install.sh

**Area:** core **Depends on:** #5, #6

### What
Make installation a single curl-pipe-bash. Detects which agents are
installed, wires each.

### Acceptance criteria
- [ ] `install.sh` detects OS/arch, downloads the right binary release, places it on `PATH`
- [ ] Detects which agents are installed (`claude` / `codex` on `PATH`) and offers to wire each
- [ ] Calls `scripture-mcp init --target=...` per detected agent
- [ ] Re-running upgrades the binary and is idempotent on configs
- [ ] Uninstall path: `scripture-mcp init --uninstall --target=...` removes the snippets

### Notes
Don't ship a Homebrew formula yet — install.sh only.

---

## #8 — test(critical): injection lifecycle + coding-quality regression

**Area:** test **Depends on:** #5, #6

### What
The two tests that gate v0.1. These are intentionally not folded into the
adapter issues because (a) they need both adapters present to be meaningful
and (b) a regression here must block any feature PR.

### Acceptance criteria

**Injection lifecycle (the make-or-break invariant):**
- [ ] Automated end-to-end test driving Claude Code in headless mode (or replaying via SDK)
- [ ] Sequence: turn N injects verse → turn N+1 model can quote it → turn N+2 model cannot
- [ ] Compaction-resistant: after a 30-turn conversation following the inject, turn N+1's verse is still not recoverable
- [ ] Mode B recap: prompt-history inspection confirms recap text never appears in any subsequent `model_call` input
- [ ] Same test runs against both Claude and Codex adapters
- [ ] On failure, prints which turn leaked and what residual content was found

**Coding-quality regression:**
- [ ] Task pack of 10 representative coding tasks (mix of refactor / bug fix / new feature)
- [ ] Four modes: `baseline`, `preview-only`, `inject-once`, `recap-only`
- [ ] Metrics per task: success rate (tests pass), input tokens, output tokens, p50 latency
- [ ] Report posted to `docs/benchmarks/<date>.md` per release
- [ ] Acceptance: no mode regresses success rate by >5pp vs baseline

### Notes
This issue gates the v0.1 release. If we can't keep the lifecycle invariant,
the project doesn't ship.

---

## #9 — polish & launch: aliases, learning mode, GIFs, release

**Area:** polish **Depends on:** #7, #8

### What
Everything that turns "it works" into "it's worth posting." Quality-of-life
features for Chinese users + memory training, plus the release artifacts.

### Acceptance criteria
- [ ] `/dao 第十一章`, `/sutra 心经`, `/bible 约翰福音 3:16` all resolve
- [ ] `scripture-mcp recap --first-letter` prints first-letter pattern (e.g. later words masked)
- [ ] Optional SM-2 spaced repetition selecting next verse based on `~/.config/scripture-mcp/learning.json`
- [ ] Toggleable via `init --learning=on`
- [ ] README has 1 GIF for Mode A and 1 GIF for Mode B
- [ ] `docs/benchmarks/v0.1.md` published with #8 results
- [ ] `CHANGELOG.md` with v0.1.0 entry
- [ ] GitHub release v0.1.0 with prebuilt binaries (macOS arm64 + x86_64, Linux x86_64)
- [ ] Announcement draft for HN Show HN / r/ClaudeAI / r/programming

### Notes
Don't post until #8's lifecycle test has been green for sustained daily use.

---

## Dependency graph

```
                     #1
                    /  \
                   v    v
                  #2   #3
                    \  /
                     v
                     #4
                    /  \
                   v    v
                  #5    #6
                    \  /
                  ┌──┴──┐
                  v     v
                  #7    #8
                    \  /
                     v
                     #9
```

#1 unblocks everything. #2 and #3 can run in parallel after #1. #5 and #6
can run in parallel after #4. #7 and #8 can run in parallel after both
adapters are in. #9 is the final polish + launch.
