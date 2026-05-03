# Announcement Drafts

## Show HN

Title: Show HN: Verse-Driven Development - scripture framing for coding agents

I built Verse-Driven Development, a local MCP/CLI extension for Claude Code and Codex that lets you explicitly frame one coding turn with a passage from wisdom literature, without making that text permanent agent context.

The design has two physically separate modes:

- Manual mode: you type a marker like `/dao 11`, `$dao.11`, or `/bible John 3:16`; the hook resolves the passage locally and injects it for that turn only.
- Recap mode: after a task finishes, a wrapper or Stop hook prints a terminal-only recap that never enters the next model call.

The repo includes lifecycle tests that check the injected context disappears from later turns, plus a small coding-quality benchmark across baseline, preview-only, inject-once, and recap-only modes.

It is a single Go binary, local stdio MCP, no server, and works offline with bundled KJV, Dao De Jing, and Heart Sutra packs.

Repo: https://github.com/MiaoDX/verse-driven

## r/ClaudeAI

I made a small Claude Code extension called Verse-Driven Development.

The goal is not to make a religious chatbot. It is a coding-agent-safe way to optionally frame one task with a real source passage, while keeping Claude's coding context clean.

Claude support:

- slash-command style markers such as `/dao 11` and `/bible John 3:16`
- one-turn injection only
- terminal-only recap through a Stop hook
- no scripture text in persistent agent instructions
- lifecycle tests that check later turns cannot recover the injected text

It also has a Codex adapter, a local stdio MCP server, and bundled offline packs.

Repo: https://github.com/MiaoDX/verse-driven

## r/programming

I released Verse-Driven Development, a small Go project that explores a narrower version of "AI agent context hygiene."

It serves scripture/wisdom-literature passages to coding agents through a local MCP server and CLI, but the main point is the injection protocol:

- default silent behavior
- explicit marker required for model-visible context
- one-turn-only injected context
- terminal-only post-task recap
- tests for prompt-history leakage
- benchmarks for coding-quality regression

The implementation is a single Go binary with embedded data packs, Claude Code and Codex adapters, and no remote service dependency.

Repo: https://github.com/MiaoDX/verse-driven
