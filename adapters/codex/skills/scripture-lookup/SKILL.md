---
name: scripture-lookup
description: Manually preview and one-turn-inject a scripture passage as a reflective frame. Invoke explicitly via the [[tradition:ref]] or $tradition:ref marker, or by name; never auto-trigger.
allow_implicit_invocation: false
allowed-tools:
  - mcp__scripture__lookup
  - mcp__scripture__list_traditions
---

# scripture-lookup

A manual fallback for the inline marker flow used by the Codex
`UserPromptSubmit` hook. The canonical form is `[[bible:John 3:16]]`; short
aliases such as `$dao:11` are also supported. Use this skill when the user
wants to preview a verse and explicitly choose whether to inject it as a
one-turn reflective frame.

This skill is **manually invoked only**. The `agents/openai.yaml` companion
file sets `allow_implicit_invocation: false` so the model never auto-triggers
it during ordinary coding turns.

## Inputs

The user supplies a free-form reference, e.g. `John 3:16`, `约翰福音 3:16`,
`道德经 11`, `Quran 2:255`, or just `心经`. If no reference is supplied,
ask which tradition and reference they want.

## Steps

1. Call `mcp__scripture__lookup` with the user's reference.
2. Render a preview to the user containing exactly three pieces of
   metadata returned by the MCP tool: `display_ref`, `source.attribution`,
   and `checksum_sha256`. Quote the verse `text` verbatim **once**, in
   the preview only. Do not paraphrase, summarize, or extend the metaphor.
3. Ask the user explicitly: "Inject this as a one-turn reflective frame
   for your next coding request? (yes / no)".
4. If the user declines, stop. Do not retain the verse in your reply
   beyond the preview.
5. If the user confirms, instruct them to send their next coding
   request with an inline marker (e.g. `[[bible:John 3:16]] Refactor X.`
   or `$dao:11 Refactor X.`) so the `UserPromptSubmit` hook performs the
   one-turn injection. Do not try to inject the envelope yourself: the hook
   is the only path that produces the temporary, turn-bounded
   developer-context block this project guarantees.

## Invariants

- This skill body contains no scripture text. All verse content comes
  from the MCP `lookup` tool at runtime.
- Preview the verse at most once per invocation.
- Never re-quote the verse in subsequent turns of the same session.
- Never alter coding priorities, testing rigor, or safety behavior in
  response to the verse content.

## Errors

If `lookup` returns "not bundled" or "ambiguous", surface the error
verbatim and ask the user to disambiguate. Do not guess.
