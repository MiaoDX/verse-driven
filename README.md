# Verse-Driven Development

> *"三十辐共一毂，当其无，有车之用。" — 道德经 第十一章*
> *"Look at the birds of the air, they neither sow nor reap." — Matthew 6:26*

A coding-agent extension that lets you frame coding tasks with passages from
the world's wisdom literature — the Bible, Quran, Tao Te Ching, Heart Sutra —
**without polluting the agent's coding context**.

Works with [Claude Code](https://claude.com/code) and
[Codex](https://github.com/openai/codex). Local stdio MCP, zero remote dependency,
zero impact on coding quality when not in use.

> **Status:** 🚧 pre-alpha. Repo just bootstrapped. See [`plan.md`](./plan.md)
> for the two-week PoC roadmap.

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
> /bible Matthew 6:26

📖 Matthew 6:26 (KJV)
"Behold the fowls of the air: for they sow not, neither do they reap..."

Source: Project Gutenberg / KJV
[ Inject for next turn only ]  [ Cancel ]

> inject once
> Refactor this scheduler. Preserve the cron-string API.

[Claude Code does the refactor, with the verse framing this single turn]
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
                  ┌─ packs/        (KJV / 道德经 / 心经 / Quran ...)
   shared core  ──┼─ resolver/     (reference parsing + checksum)
                  └─ mcp-server/   (local stdio MCP)
                            │
              ┌─────────────┴─────────────┐
              │                           │
      Claude adapter                Codex adapter
      ├─ output-style              ├─ skill (scripture-lookup)
      ├─ skill (verse-inject)      ├─ hook (UserPromptSubmit)
      └─ hooks (UserPromptExpansion└─ shell wrapper for recap
          + Stop)
```

One Go binary, three invocation modes:

```bash
scripture-mcp serve                              # stdio MCP for cc/codex
scripture-mcp lookup "John 3:16" --format=json   # CLI for hooks
scripture-mcp recap --tradition=dao --terminal   # Mode B recap
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

## Install (planned)

```bash
brew install verse-driven
# or
curl -fsSL https://raw.githubusercontent.com/MiaoDX/verse-driven/main/install.sh | bash

# wire it into your coding agents
scripture-mcp init --target=claude-code
scripture-mcp init --target=codex
```

`init` merges config snippets into your existing `settings.json` /
`config.toml` — it does not overwrite your config.

### Claude Code adapter assets

In addition to the `settings.json` snippet that `init --target=claude-code`
installs, the Claude adapter ships two static assets you symlink (or copy)
into your Claude config:

```bash
# Output style — turns on the <scripture_card> reading mode.
ln -s "$PWD/adapters/claude-code/output-styles/scripture-recap.md" \
      ~/.claude/output-styles/scripture-recap.md

# Manual fallback skill — preview a verse without typing the slash marker.
mkdir -p ~/.claude/skills/verse-inject
ln -s "$PWD/adapters/claude-code/skills/verse-inject/SKILL.md" \
      ~/.claude/skills/verse-inject/SKILL.md
```

Both files are deliberately tiny and contain **no scripture text** —
verses are fetched from the bundled MCP server at lookup time. The skill
sets `disable-model-invocation: true` so the model never auto-triggers it.

---

## Roadmap

See [`plan.md`](./plan.md) for the full design and
[`docs/issues-backlog.md`](./docs/issues-backlog.md) for the 9-issue work
plan. v0.1 acceptance criteria:

1. ✅ `scripture-mcp serve` runs on macOS and Linux
2. ✅ `/bible John 3:16` in Claude Code shows a preview card
3. ✅ "Inject once" makes the verse visible to the model in the next turn
4. ✅ The turn after that, the model can no longer see the verse *(lifecycle)*
5. ✅ Mode B recap reaches the terminal but never enters a `model_call` input

---

## Initial pack list

| Pack | Source | License | Bundled? |
|---|---|---|---|
| Bible KJV (en) | [Project Gutenberg](https://www.gutenberg.org/) | Public domain (US) | ✅ |
| 道德经 (zh) | [Chinese Text Project](https://ctext.org/) | Open access | ✅ |
| 心经 (zh) | [CBETA](https://cbeta.org/) | See pack release notes | ✅ |
| Quran (en) | [Tanzil](https://tanzil.net/) | CC BY 3.0 | ✅ |
| Quran (ar + 译本) | [quran.com API](https://api.quran.com/) | API-only | ⏳ |
| 中文圣经 | (multiple, complex licensing) | — | ❌ Phase 2 |

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
Data packs: each pack ships with its own `LICENSE` reflecting the upstream
source. KJV and Tao Te Ching are public domain; Tanzil-based packs require
the CC BY 3.0 attribution notice be preserved.

---

## Contributing

This is a brand-new project. The first thing that needs to exist is the Go
binary and the KJV pack builder. If you want to help, the best place to start
is reading [`plan.md`](./plan.md) and opening an issue with what part of the
PoC you'd like to tackle.

Tradition contributions especially welcome — if you can verify a pack against
its canonical source and add the right attribution, please open a PR.
