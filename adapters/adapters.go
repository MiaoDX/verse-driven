// Package adapters bundles the per-agent assets shipped alongside the
// scripture-mcp binary — output styles, skills, and any wrapper scripts
// the agents need to be wired into. The files themselves remain on disk
// (so users can read and copy them) and are also embedded here so unit
// tests can assert structural properties without duplicating content.
package adapters

import "embed"

// ClaudeCodeFS exposes adapters/claude-code/** for tests and for any
// future installer that wants to materialize the assets into a user's
// agent config directory.
//
//go:embed claude-code/output-styles/scripture-recap.md
//go:embed claude-code/skills/verse-inject/SKILL.md
var ClaudeCodeFS embed.FS

// ClaudeOutputStylePath is the in-FS path of the output style.
const ClaudeOutputStylePath = "claude-code/output-styles/scripture-recap.md"

// ClaudeVerseInjectSkillPath is the in-FS path of the verse-inject skill.
const ClaudeVerseInjectSkillPath = "claude-code/skills/verse-inject/SKILL.md"

// CodexFS exposes adapters/codex/** for tests and the future installer.
//
//go:embed codex/skills/scripture-lookup/SKILL.md
//go:embed codex/skills/scripture-lookup/agents/openai.yaml
//go:embed codex/wrapper/cdx
//go:embed codex/wrapper/cdx.ps1
var CodexFS embed.FS

// CodexScriptureLookupSkillPath is the in-FS path of the Codex skill.
const CodexScriptureLookupSkillPath = "codex/skills/scripture-lookup/SKILL.md"

// CodexOpenAIAgentPath is the in-FS path of the per-agent skill manifest
// that pins allow_implicit_invocation: false for the OpenAI/Codex model.
const CodexOpenAIAgentPath = "codex/skills/scripture-lookup/agents/openai.yaml"

// CodexWrapperPOSIXPath is the in-FS path of the POSIX `cdx` wrapper.
const CodexWrapperPOSIXPath = "codex/wrapper/cdx"

// CodexWrapperPowerShellPath is the in-FS path of the PowerShell wrapper.
const CodexWrapperPowerShellPath = "codex/wrapper/cdx.ps1"
