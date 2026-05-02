# Verse-Driven Development

> *"三十辐共一毂，当其无，有车之用。" — 道德经 第十一章*
> *"Look at the birds of the air, they neither sow nor reap." — Matthew 6:26*

A coding-agent extension that lets you frame coding tasks with passages from
wisdom literature — currently KJV Bible and 道德经, with Heart Sutra/Quran
adapters stubbed for later data work — **without polluting the agent's
coding context**.

Works with [Claude Code](https://claude.com/code) and
[Codex](https://github.com/openai/codex). Local stdio MCP, zero remote dependency,
zero impact on coding quality when not in use.

> **Status:** v0.1 candidate, source-install ready. Core binary, CLI, MCP
> server, Claude Code adapter, Codex adapter, lifecycle tests, and benchmark
> fixtures are implemented. GitHub release artifacts are not published yet;
> issue #8's lifecycle + coding-quality benchmark gate is published in
> [`docs/benchmarks/v0.1.md`](./docs/benchmarks/v0.1.md). The remaining
> release gate is issue #9 launch polish.

---

## Why this exists

Writing code is a focused, technical act. But every now and then you want to
pause and reframe — *"is this refactor preserving what users actually depend
on, or am I just enjoying the cleanup?"* — and a well-chosen line from a
classic text frames the moment differently than yet another stand-up checklist.

Existing options didn't quite fit:

- **Generic "zen-master" personas** ([example](https://github.com/hesreallyhim/awesome-claude-code-output-styles-that-i-really-like))
  write fake-koan parodies, not real passages.
- **Scripture MCP servers**
  ([sacred-scriptures-mcp](https://github.com/Traves-Theberge/sacred-scriptures-mcp),
  [quran-mcp](https://github.com/quran/quran-mcp), etc.) are built for
  *studying* scripture, not for *using* scripture inside a coding session.
- **Naive approaches** (put it in `CLAUDE.md`, `AGENTS.md`, an output style)
  leak into the agent's permanent context and quietly degrade coding quality.

VDD is the missing piece: a clean, agent-safe protocol for **optional,
one-turn scripture framing** that defaults to silent.

This continues a tradition that runs from
[The Tao of Programming](https://www.mit.edu/~xela/tao.html) (1987) through
the [Unix Koans of Master Foo](https://github.com/lumenwrites/fictionhub/blob/master/stories/The%20Unix%20Koans%20of%20Master%20Foo.md)
to the [Zen of Python](https://peps.python.org/pep-0020/) — only this time
with real source texts and a verifiable injection protocol.

---

## Two modes, physically separated

| | **Mode A: Manual** | **Mode B: Recap** |
|---|---|---|
| **Trigger** | User explicitly invokes a marker | After a coding task completes |
| **Visible to model?** | ✅ Current turn only — gone next turn | ❌ Never. Printed to your terminal only |
| **Use case** | Frame this task with a verse | Reflection / memory training |
| **Default** | Silent. Nothing happens unless you ask | Off. Opt-in via init flag |

The key invariant: **a verse never enters a model's input unless the user
explicitly invoked it for that turn.** Mode B writes to your terminal, not
to the agent's transcript.

### Mode A example

```
> /bible John 3:13

John 3:13 (KJV)
[visible verse preview]
sha256:... · Project Gutenberg eBook #10 · Public domain (US)

> Refactor this scheduler. Preserve the cron-string API.

[Claude Code does the refactor; the verse was available for the slash-command turn only]
```

Codex uses inline markers instead of slash commands:

```
> $dao.11 Refactor this scheduler. Preserve the cron-string API.
> $bible.John.3.13 Refactor this parser.
> [[bible:John 3:13]] Refactor this parser.
```

### Mode B example

```
[Claude Code finishes a long PR]

— terminal printout, not seen by the model —
📖 道德经 第十一章
"三十辐共一毂，当其无，有车之用。"
你写得最优雅的那部分接口，是你"没写"的部分。
```

---

## Architecture

```
                  ┌─ packs/        (bundled KJV + 道德经; Heart Sutra api-only stub)
   shared core  ──┼─ resolver/     (reference parsing + checksum)
                  └─ mcp-server/   (local stdio MCP)
                            │
              ┌─────────────┴─────────────┐
              │                           │
      Claude adapter                Codex adapter
      ├─ slash commands            ├─ inline markers
      ├─ output-style              ├─ skill (scripture-lookup)
      ├─ skill (verse-inject)      ├─ UserPromptSubmit hook
      └─ hooks (UserPromptExpansion + Stop)
                                   └─ shell wrapper for recap
```

One Go binary, three invocation modes:

```bash
scripture-mcp serve                              # stdio MCP for cc/codex
scripture-mcp lookup "John 3:13" --format=json   # CLI for hooks
scripture-mcp recap --tradition=dao --terminal   # Mode B recap
```

Supported reference forms:

```text
# Claude Code slash commands
/bible John 3:13
/bible john.3.13
/dao 11

# Codex inline markers
[[bible:John 3:13]]
$bible:John 3:13
$bible.John.3.13
$dao:11
$dao.11

# Direct CLI/MCP lookup
John 3:13
dao 11
dao.11
quran.2.255   # parses, but Quran text is not bundled yet
```

**~80% of the code is shared between Claude Code and Codex.** Differences are
mostly in config file format (`settings.json` vs `config.toml`) and Codex's
recap path needs a shell wrapper (it has no `Stop` hook equivalent).

This single-binary / multi-interface design follows the pattern from
[Gentleman-Programming/engram](https://github.com/Gentleman-Programming/engram).

---

## Why local stdio MCP

- **Fast**: `stdio` is an in-process pipe — verse lookup completes in <5ms,
  ~10× faster than HTTP/remote MCP
- **Offline**: packs are embedded in the binary via `embed.FS`. Works on a
  plane.
- **No runtime dependencies**: single Go binary. No Node, no Python, no Docker.
- **Standard protocol**: same MCP server works with Claude Code, Codex,
  Gemini CLI, Cursor, OpenCode, Aider — anything MCP-compatible.

---

## Install From Source

There is no published GitHub release yet. For the current checkout, build and
install the local binary:

```bash
go test ./...
make build
mkdir -p ~/.local/bin
cp bin/scripture-mcp ~/.local/bin/scripture-mcp
chmod +x ~/.local/bin/scripture-mcp
```

Make sure `~/.local/bin` is on `PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then wire the agents:

```bash
scripture-mcp init --target=codex

# Claude Code's current MCP registry is managed by the claude CLI:
claude mcp add --scope user scripture -- scripture-mcp serve
scripture-mcp init --target=claude-code
```

Install the static adapter assets shown below. They are intentionally separate
from `init` today so users can inspect the slash commands, output style, and
skills before copying or symlinking them.

### Release Installer Status

`install.sh` exists and is tested, but it expects release tarballs that have
not been published yet. Once v0.1.0 is released, the intended one-line install
path is:

```bash
curl -fsSL https://raw.githubusercontent.com/MiaoDX/verse-driven/main/install.sh | bash
```

Release installer behavior:

1. Detects your OS (macOS / Linux) and arch (arm64 / x86_64), downloads
   the matching `scripture-mcp` release binary, and installs it to
   `~/.local/bin/scripture-mcp` (override with `--prefix`).
2. Detects which coding agents are on `$PATH` (`claude`, `codex`) and
   wires each by calling `scripture-mcp init --target=<agent>`. The
   prompt reads from `/dev/tty`, so it still works under `curl | bash`;
   pass `--yes` to skip confirmation.
3. Re-running upgrades the binary in place; `init` is idempotent on
   configs (it splices a marker-fenced block, never overwrites your
   settings).

Useful flags:

```bash
install.sh --version v0.1.0          # pin a specific release
install.sh --prefix /usr/local/bin   # install elsewhere
install.sh --no-wire                 # binary only, skip agent wiring
install.sh --uninstall               # strip wiring + remove binary
```

Or, equivalently, manage just the wiring with `init` itself:

```bash
scripture-mcp init --target=claude-code
scripture-mcp init --target=codex
scripture-mcp init --uninstall --target=claude-code   # remove
```

Homebrew / `apt` packages are not shipped yet.

### Claude Code adapter assets

In addition to the `settings.json` snippet that `init --target=claude-code`
installs, the Claude adapter ships static assets you symlink (or copy) into
your Claude config:

```bash
# Slash commands — preview visibly and inject for this turn.
mkdir -p ~/.claude/commands
ln -s "$PWD/adapters/claude-code/commands/bible.md" ~/.claude/commands/bible.md
ln -s "$PWD/adapters/claude-code/commands/dao.md" ~/.claude/commands/dao.md
ln -s "$PWD/adapters/claude-code/commands/sutra.md" ~/.claude/commands/sutra.md
ln -s "$PWD/adapters/claude-code/commands/quran.md" ~/.claude/commands/quran.md

# Output style — turns on the <scripture_card> reading mode.
mkdir -p ~/.claude/output-styles
ln -s "$PWD/adapters/claude-code/output-styles/scripture-recap.md" \
      ~/.claude/output-styles/scripture-recap.md

# Manual fallback skill — preview a verse without typing the slash marker.
mkdir -p ~/.claude/skills/verse-inject
ln -s "$PWD/adapters/claude-code/skills/verse-inject/SKILL.md" \
      ~/.claude/skills/verse-inject/SKILL.md
```

These files are deliberately tiny and contain **no scripture text** — verses
are fetched from the bundled MCP server at lookup time. The skill sets
`disable-model-invocation: true` so the model never auto-triggers it.

### Codex adapter assets

Codex Mode A is wired entirely through the `config.toml` snippet
`init --target=codex` writes (the `UserPromptSubmit` hook calls
`scripture-mcp lookup-from-prompt`, which recognizes the canonical inline
marker syntax `[[bible:John 3:16]]` plus ergonomic aliases like `$dao:11`,
`$dao.11`, and `$bible.John.3.16`).

In addition, the adapter ships:

```bash
# Manual fallback skill — preview before sending a marker.
mkdir -p ~/.codex/skills/scripture-lookup/agents
ln -s "$PWD/adapters/codex/skills/scripture-lookup/SKILL.md" \
      ~/.codex/skills/scripture-lookup/SKILL.md
ln -s "$PWD/adapters/codex/skills/scripture-lookup/agents/openai.yaml" \
      ~/.codex/skills/scripture-lookup/agents/openai.yaml

# Mode B recap wrapper — Codex has no Stop hook, so recap fires from
# a thin shell wrapper that runs *outside* the Codex transcript.
cp adapters/codex/wrapper/cdx ~/.local/bin/cdx
chmod +x ~/.local/bin/cdx
# Then run `cdx ...` instead of `codex ...`.
# A PowerShell counterpart lives at adapters/codex/wrapper/cdx.ps1.
```

The skill's `agents/openai.yaml` pins `allow_implicit_invocation: false`
so the OpenAI/Codex model cannot auto-invoke `scripture-lookup` during a
coding turn — it must be called by name. The skill body, like its Claude
counterpart, contains **no scripture text**: verses are fetched from the
bundled MCP server at lookup time.

**Codex skill path note:** official Codex docs and OSS repos disagree on
whether per-user skills live under `~/.codex/skills`, `$CODEX_HOME/skills`,
or `~/.agents/skills`. The snippets above use `~/.codex/skills`, matching
the same `~/.codex/config.toml` location `init --target=codex` writes to.
If your Codex build expects a different path, symlink the skill there
instead.

**Honest limitation:** the recap fires because the wrapping shell stays
alive after `codex` exits and runs `scripture-mcp recap --terminal`.
Launching `codex` directly (without `cdx`) bypasses the recap entirely.
This is a deliberate physical-isolation trade-off — see [`plan.md`](./plan.md)
§5.2 — and means there is no way for the recap text to leak into a future
Codex `model_call` input.

---

## Roadmap

See [`plan.md`](./plan.md) for the full design and
[`docs/issues-backlog.md`](./docs/issues-backlog.md) for the 9-issue work
plan. Current implementation status:

- ✅ `scripture-mcp serve` runs and exposes MCP tools: `lookup`, `search`,
  `random`, and `list_traditions`.
- ✅ CLI commands exist for lookup, prompt-hook lookup, recap, and config init.
- ✅ Claude Code Mode A works through slash commands such as `/bible John 3:13`
  and `/dao 11`; the command shows a visible preview and injects the card for
  that turn.
- ✅ Codex Mode A works through `[[bible:John 3:13]]`, `$dao:11`, `$dao.11`,
  and `$bible.John.3.13`.
- ✅ Mode B recap is isolated: Claude uses a `Stop` hook, Codex uses the `cdx`
  wrapper.
- ✅ Local lifecycle simulations verify injected scripture disappears from later
  model inputs, including 30-turn follow-up simulations.
- ✅ Coding-quality benchmark scaffolding exists: 10 fixtures × 4 modes
  (`baseline`, `preview-only`, `inject-once`, `recap-only`).
- ✅ Issue #8 release gate passed: external lifecycle probe passed for Claude
  and Codex hook events, and the 80-row live Claude/Codex coding-quality
  benchmark had 0 regressions vs baseline. See
  [`docs/benchmarks/v0.1.md`](./docs/benchmarks/v0.1.md).

Remaining v0.1 release gate:

- Issue #9: launch polish — localized aliases, learning-mode polish, GIFs,
  changelog, release binaries, and announcement draft.

---

## Pack Status

| Pack | Source | License | Current state |
|---|---|---|---|
| Bible KJV (en) | [Project Gutenberg](https://www.gutenberg.org/) eBook #10 | Public domain (US) | ✅ Bundled, 31,102 verses |
| 道德经 (zh-Hans) | [Project Gutenberg](https://www.gutenberg.org/) eBook #7337 | Public domain | ✅ Bundled, 81 chapters |
| 心经 (zh-Hans) | [CBETA](https://cbeta.org/) | Pending license audit | ⚠️ API-only stub, 0 bundled verses |
| Quran | planned | pending | ❌ Resolver support only; no bundled text |
| 中文圣经 | planned | complex licensing | ❌ Phase 2 |

---

## Acknowledgments

This project stands on the shoulders of:

- **[hesreallyhim/awesome-claude-code-output-styles-that-i-really-like](https://github.com/hesreallyhim/awesome-claude-code-output-styles-that-i-really-like)** —
  the conceptual ancestor. Its `zen-master` and `existential-poet` output
  styles proved that "switch to a reflective register while coding" is a real
  workflow people want.
- **[Traves-Theberge/sacred-scriptures-mcp](https://github.com/Traves-Theberge/sacred-scriptures-mcp)** —
  the closest data-layer precedent. Demonstrated that a single MCP can serve
  KJV / Quran / Tao Te Ching / Dhammapada in one place.
- **[quran/quran-mcp](https://github.com/quran/quran-mcp)** — official Quran
  Foundation MCP, reference for handling sacred-text fidelity.
- **[Gentleman-Programming/engram](https://github.com/Gentleman-Programming/engram)** —
  single-Go-binary / CLI + HTTP + MCP architecture template.
- **[The Tao of Programming](https://www.mit.edu/~xela/tao.html)** &
  **[Unix Koans of Master Foo](https://github.com/lumenwrites/fictionhub/blob/master/stories/The%20Unix%20Koans%20of%20Master%20Foo.md)** —
  the spiritual lineage.

Full reference list (including official Claude Code / Codex docs) lives in
[`plan.md`](./plan.md#10-参考资料).

---

## License

Code: MIT.
Bundled data packs keep their own upstream attribution metadata. The current
binary embeds KJV from Project Gutenberg eBook #10 and 道德经 from Project
Gutenberg eBook #7337, both public-domain sources. Heart Sutra and Quran
content are not bundled yet; those packs need license review and attribution
work before release.

---

## Contributing

The core proof of concept is implemented and the issue #8 benchmark gate has
passed. The best places to help now are issue #9 launch polish and additional
data packs with clear source provenance.

Tradition contributions especially welcome — if you can verify a pack against
its canonical source and add the right attribution, please open a PR.
