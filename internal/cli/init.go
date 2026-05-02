package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runInit implements `scripture-mcp init --target={claude-code,codex}
// [--recap=on|off] [--uninstall]`.
//
// Idempotent: rerunning with the same flags is a no-op once the snippet
// is installed. We never overwrite the user's config — we splice in (or
// strip out) a marker-fenced block.
func runInit(args []string, s Streams) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(s.Err)
	target := fs.String("target", "", "agent target: claude-code|codex")
	recap := fs.String("recap", "on", "enable Mode B recap: on|off")
	uninstall := fs.Bool("uninstall", false, "remove the snippet instead of installing it")
	dryRun := fs.Bool("dry-run", false, "print what would change without writing")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *target == "" {
		fmt.Fprintln(s.Err, "error: --target is required (claude-code|codex)")
		return 2
	}
	switch *recap {
	case "on", "off":
	default:
		fmt.Fprintln(s.Err, "error: --recap must be on|off")
		return 2
	}
	home, err := homeDir(s)
	if err != nil {
		fmt.Fprintln(s.Err, "error:", err)
		return 1
	}

	switch *target {
	case "claude-code":
		path := filepath.Join(home, ".claude", "settings.json")
		return manageSnippet(path, "// >>> verse-driven", "// <<< verse-driven",
			renderClaudeSnippet(*recap == "on"), *uninstall, *dryRun, s)
	case "codex":
		path := filepath.Join(home, ".codex", "config.toml")
		code := manageSnippet(path, "# >>> verse-driven", "# <<< verse-driven",
			renderCodexSnippet(*recap == "on"), *uninstall, *dryRun, s)
		if code == 0 && !*dryRun && !*uninstall && *recap == "on" {
			printCdxAliasHint(s.Out)
		}
		return code
	default:
		fmt.Fprintf(s.Err, "error: unknown target %q (want claude-code|codex)\n", *target)
		return 2
	}
}

// printCdxAliasHint emits the wrapper-setup instructions for Codex Mode B.
// Codex has no Stop-equivalent hook, so the recap fires from a thin shell
// wrapper that wraps the `codex` invocation; the user has to put the
// wrapper on PATH (or alias `cdx` to it) themselves. We never modify the
// user's shell rc files automatically.
func printCdxAliasHint(out io.Writer) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "verse-driven: Codex Mode B recap requires the `cdx` shell wrapper.")
	fmt.Fprintln(out, "  Wrapper source: adapters/codex/wrapper/cdx (POSIX) or cdx.ps1 (PowerShell).")
	fmt.Fprintln(out, "  Install one of:")
	fmt.Fprintln(out, "    cp adapters/codex/wrapper/cdx ~/.local/bin/cdx && chmod +x ~/.local/bin/cdx")
	fmt.Fprintln(out, "    alias cdx='/path/to/adapters/codex/wrapper/cdx'   # add to ~/.bashrc or ~/.zshrc")
	fmt.Fprintln(out, "  Then run `cdx ...` instead of `codex ...` to get a terminal recap on exit.")
	fmt.Fprintln(out, "  Note: launching `codex` directly bypasses the recap.")
}

func homeDir(s Streams) (string, error) {
	if s.HomeFn != nil {
		return s.HomeFn()
	}
	return os.UserHomeDir()
}

// manageSnippet inserts (or removes) snippet between begin/end marker
// lines in the file at path. The snippet itself must already include the
// markers; we identify the existing block by scanning for them and we
// idempotently replace it.
//
// File creation: when installing into a missing file, we create the
// parent directory and write a file containing only the snippet. When
// uninstalling from a missing file, we treat it as a no-op.
func manageSnippet(path, beginMarker, endMarker, snippet string, uninstall, dryRun bool, s Streams) int {
	existing, exists, err := readIfExists(path)
	if err != nil {
		fmt.Fprintln(s.Err, "error:", err)
		return 1
	}
	updated, changed := spliceSnippet(existing, exists, beginMarker, endMarker, snippet, uninstall)
	if !changed {
		fmt.Fprintf(s.Out, "verse-driven: %s already up to date\n", path)
		return 0
	}
	if dryRun {
		fmt.Fprintf(s.Out, "verse-driven: would write %s\n", path)
		fmt.Fprintln(s.Out, "---")
		fmt.Fprint(s.Out, updated)
		fmt.Fprintln(s.Out, "---")
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(s.Err, "error: mkdir:", err)
		return 1
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		fmt.Fprintln(s.Err, "error: write:", err)
		return 1
	}
	if uninstall {
		fmt.Fprintf(s.Out, "verse-driven: removed snippet from %s\n", path)
	} else {
		fmt.Fprintf(s.Out, "verse-driven: installed snippet into %s\n", path)
	}
	return 0
}

func readIfExists(path string) (string, bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	return string(b), true, nil
}

// spliceSnippet returns the updated file contents and whether anything
// would change. snippet is wrapped in a marker-fenced block; if the file
// already contains markers, the block between them is replaced. On
// install into a missing file, a fresh file with only the snippet block
// is produced. On uninstall, the block is stripped (and trailing blank
// lines collapsed).
func spliceSnippet(existing string, exists bool, beginMarker, endMarker, snippet string, uninstall bool) (string, bool) {
	block := beginMarker + "\n" + snippet
	if !strings.HasSuffix(block, "\n") {
		block += "\n"
	}
	block += endMarker + "\n"

	beginIdx := strings.Index(existing, beginMarker)
	endIdx := strings.Index(existing, endMarker)
	hasBlock := beginIdx >= 0 && endIdx > beginIdx

	if uninstall {
		if !exists || !hasBlock {
			return existing, false
		}
		afterEnd := endIdx + len(endMarker)
		// Eat the newline immediately following endMarker (and only one).
		if afterEnd < len(existing) && existing[afterEnd] == '\n' {
			afterEnd++
		}
		// Eat one preceding newline before beginIdx if present so we don't
		// leave a blank line where the block was.
		startCut := beginIdx
		if startCut > 0 && existing[startCut-1] == '\n' {
			startCut--
		}
		out := existing[:startCut] + existing[afterEnd:]
		return out, out != existing
	}

	if !exists {
		return block, true
	}
	if hasBlock {
		afterEnd := endIdx + len(endMarker)
		if afterEnd < len(existing) && existing[afterEnd] == '\n' {
			afterEnd++
		}
		out := existing[:beginIdx] + block + existing[afterEnd:]
		return out, out != existing
	}
	prefix := existing
	if !strings.HasSuffix(prefix, "\n") {
		prefix += "\n"
	}
	if prefix != "" && !strings.HasSuffix(prefix, "\n\n") {
		prefix += "\n"
	}
	out := prefix + block
	return out, out != existing
}

// renderClaudeSnippet returns the JSON-with-comments snippet inserted
// into ~/.claude/settings.json. We deliberately use shell-style comments
// as fence markers (rather than valid JSON) because settings.json
// itself is parsed as JSON5/JSON-with-comments by Claude Code.
//
// The snippet matches plan.md §5.1.
func renderClaudeSnippet(recapOn bool) string {
	stopBlock := ""
	if recapOn {
		stopBlock = `,
    "Stop": [
      {
        "hooks": [
          { "type": "command", "command": "scripture-mcp recap --terminal" }
        ]
      }
    ]`
	}
	return `// Generated by scripture-mcp init --target=claude-code.
// This block is managed automatically; edit between the markers will be
// overwritten on the next ` + "`" + `scripture-mcp init` + "`" + `.
{
  "mcpServers": {
    "scripture": {
      "command": "scripture-mcp",
      "args": ["serve"]
    }
  },
  "hooks": {
    "UserPromptExpansion": [
      {
        "matcher": "^(bible|sutra|dao|quran)$",
        "hooks": [
          { "type": "command", "command": "scripture-mcp lookup-from-prompt --hook-event=UserPromptExpansion" }
        ]
      }
    ]` + stopBlock + `
  }
}
`
}

// renderCodexSnippet returns the TOML snippet for ~/.codex/config.toml,
// matching plan.md §5.2. Codex has no Stop-equivalent hook, so the recap
// is wired via the `cdx` shell wrapper from issue #6 — recap=on here is
// informational; the actual wrapper install path isn't part of init's
// JSON/TOML splice.
func renderCodexSnippet(recapOn bool) string {
	recapNote := "# recap: launch via the cdx shell wrapper (see adapters/codex/wrapper/cdx).\n"
	if !recapOn {
		recapNote = "# recap: disabled (cdx wrapper omitted).\n"
	}
	return `# Generated by scripture-mcp init --target=codex.
# This block is managed automatically; edits between the markers will be
# overwritten on the next ` + "`" + `scripture-mcp init` + "`" + `.
[mcp_servers.scripture]
command = "scripture-mcp"
args = ["serve"]

[features]
codex_hooks = true

[[hooks.UserPromptSubmit]]
[[hooks.UserPromptSubmit.hooks]]
type = "command"
command = "scripture-mcp lookup-from-prompt --hook-event=UserPromptSubmit"
timeout = 15
` + recapNote
}
