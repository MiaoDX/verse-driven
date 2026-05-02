package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/MiaoDX/verse-driven/internal/injector"
	"github.com/MiaoDX/verse-driven/internal/resolver"
	"github.com/MiaoDX/verse-driven/internal/schema"
)

// runLookup implements `scripture-mcp lookup "<ref>" [--format=json|text]`.
//
// On success: prints the resolved verse (JSON by default, terminal-pretty
// with --format=text) and exits 0. On failure: prints "error: ..." to
// stderr and exits 1; ambiguous input exits with code 3.
func runLookup(args []string, s Streams) int {
	fs := flag.NewFlagSet("lookup", flag.ContinueOnError)
	fs.SetOutput(s.Err)
	format := fs.String("format", "json", "output format: json|text")
	if err := fs.Parse(reorderFlagsFirst(args)); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(s.Err, "error: usage: scripture-mcp lookup \"<reference>\" [--format=json|text]")
		return 2
	}
	ref := strings.Join(rest, " ")
	v, err := resolveAndLookup(ref)
	if err != nil {
		var amb *resolver.AmbiguousError
		if errors.As(err, &amb) {
			fmt.Fprintf(s.Err, "error: ambiguous reference %q; specify a tradition\n", amb.Input)
			for _, c := range amb.Candidates {
				fmt.Fprintf(s.Err, "  - %s/%s %d:%d\n", c.Tradition, c.Work, c.Chapter, c.VerseStart)
			}
			return 3
		}
		fmt.Fprintln(s.Err, "error:", err)
		return 1
	}
	switch *format {
	case "json":
		return writeJSON(s.Out, v)
	case "text":
		return writeTerminal(s.Out, v)
	default:
		fmt.Fprintf(s.Err, "error: unknown format %q (want json|text)\n", *format)
		return 2
	}
}

func writeJSON(w io.Writer, v schema.Verse) int {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return 1
	}
	return 0
}

func writeTerminal(w io.Writer, v schema.Verse) int {
	fmt.Fprintln(w, injector.DisplayRef(v))
	if v.Source.Attribution != "" {
		fmt.Fprintf(w, "(%s)\n", v.Source.Attribution)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, v.Text)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "checksum: %s\n", v.ChecksumSHA256)
	return 0
}

// runLookupFromPrompt implements `scripture-mcp lookup-from-prompt`. It
// reads the user prompt from stdin, scans for a marker, and emits a JSON
// envelope on stdout suitable for both Claude Code's UserPromptExpansion
// and Codex's UserPromptSubmit hooks.
//
// Recognized markers (case-insensitive on the keyword, single match wins;
// the leftmost marker is used):
//
//   - Slash form (Claude Code): leading `/bible John 3:16` ...
//   - Inline form (Codex):      ... `[[bible:John 3:16]]` ...
//
// If no marker is present, exits 0 silently with no output. The
// integration intent is that each agent's hook pipes the user's prompt
// to this command and merges the emitted `hookSpecificOutput.additionalContext`
// field into the model's input for the current turn only.
func runLookupFromPrompt(args []string, s Streams) int {
	fs := flag.NewFlagSet("lookup-from-prompt", flag.ContinueOnError)
	fs.SetOutput(s.Err)
	hookEvent := fs.String("hook-event", "", "hook event name: UserPromptExpansion|UserPromptSubmit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	body, err := io.ReadAll(s.In)
	if err != nil {
		fmt.Fprintln(s.Err, "error:", err)
		return 1
	}
	prompt, hookEventName := normalizePromptInput(body)
	if *hookEvent != "" {
		hookEventName = *hookEvent
	}
	switch hookEventName {
	case "", "UserPromptExpansion", "UserPromptSubmit":
	default:
		fmt.Fprintf(s.Err, "error: unsupported --hook-event %q\n", hookEventName)
		return 2
	}
	tradition, ref, ok := scanMarker(prompt)
	if !ok {
		// No marker → silent exit so the hook adds nothing.
		return 0
	}
	v, err := resolveTrailing(tradition, ref)
	if err != nil {
		// Soft-fail: hooks should never break the user's prompt. Log to
		// stderr (visible to the agent's debug logs) and exit 0 with no
		// additionalContext.
		fmt.Fprintf(s.Err, "verse-driven: lookup failed for %q %q: %v\n", tradition, ref, err)
		return 0
	}
	additionalContext := injector.Envelope(v)
	if hookEventName == "" {
		// Direct CLI tests and older configs default to the Claude hook name.
		// Installed Codex config passes --hook-event=UserPromptSubmit.
		hookEventName = "UserPromptExpansion"
	}
	out := struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}{}
	out.HookSpecificOutput.HookEventName = hookEventName
	out.HookSpecificOutput.AdditionalContext = additionalContext
	enc := json.NewEncoder(s.Out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return 1
	}
	return 0
}

// normalizePromptInput accepts either a raw prompt string or a JSON object
// with a "prompt" or "user_prompt" field (the form Claude Code / Codex
// hooks send on stdin).
func normalizePromptInput(b []byte) (prompt string, hookEventName string) {
	trimmed := strings.TrimSpace(string(b))
	if trimmed == "" {
		return "", ""
	}
	if trimmed[0] == '{' {
		var obj map[string]any
		if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
			if s, ok := obj["hook_event_name"].(string); ok {
				hookEventName = s
			}
			if s, ok := obj["hookEventName"].(string); ok {
				hookEventName = s
			}
			for _, k := range []string{"prompt", "user_prompt", "input", "text"} {
				if s, ok := obj[k].(string); ok && s != "" {
					return s, hookEventName
				}
			}
		}
	}
	return trimmed, hookEventName
}

var (
	// /bible John 3:16 ...   /dao 11   /sutra   /quran 2:255
	// The trailing capture is everything to the end of the line; the
	// caller (resolveTrailing) trims tokens from the right until the
	// resolver accepts what remains.
	reSlashMarker = regexp.MustCompile(`(?i)(?:^|\s)/(bible|dao|sutra|quran)(?:[ \t]+([^\n]*))?`)
	// [[bible:John 3:16]]   [[dao:11]]   [[sutra]]   [[quran:2:255]]
	reInlineMarker = regexp.MustCompile(`(?i)\[\[(bible|dao|sutra|quran)(?::([^\]\n]*))?\]\]`)
)

// resolveTrailing attempts to resolve a reference, progressively trimming
// whitespace-separated tokens from the right end of `rest` until the
// resolver accepts what remains. This lets a marker like
// `/bible John 3:16 Refactor X.` extract just the reference and leave the
// rest of the prompt untouched, without the marker scanner having to
// know what a reference looks like (the resolver is the source of truth).
//
// The candidate string passed to the resolver depends on tradition: bible
// references are self-describing (the book name carries the tradition);
// dao and quran require a tradition keyword prefix; sutra takes no
// chapter/verse and ignores trailing input.
func resolveTrailing(tradition, rest string) (schema.Verse, error) {
	if tradition == "sutra" {
		return resolveAndLookup("sutra")
	}
	cur := strings.TrimSpace(rest)
	candidate := func() string {
		switch tradition {
		case "bible":
			return cur
		default: // dao, quran
			if cur == "" {
				return tradition
			}
			return tradition + " " + cur
		}
	}
	var lastErr error
	for {
		c := candidate()
		if c == "" {
			break
		}
		v, err := resolveAndLookup(c)
		if err == nil {
			return v, nil
		}
		lastErr = err
		idx := strings.LastIndexAny(cur, " \t")
		if idx < 0 {
			break
		}
		cur = strings.TrimSpace(cur[:idx])
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("could not resolve %s reference", tradition)
	}
	return schema.Verse{}, lastErr
}

// scanMarker extracts (tradition, ref) from the leftmost marker in prompt.
// Returns (_, _, false) when no marker is present.
func scanMarker(prompt string) (tradition, ref string, ok bool) {
	type hit struct {
		idx       int
		tradition string
		ref       string
	}
	var best *hit
	if m := reSlashMarker.FindStringSubmatchIndex(prompt); m != nil {
		// FindStringSubmatchIndex returns indices of the entire match and submatches.
		full := strings.TrimSpace(prompt[m[0]:m[1]])
		sub := reSlashMarker.FindStringSubmatch(full)
		best = &hit{idx: m[0], tradition: strings.ToLower(sub[1]), ref: strings.TrimSpace(sub[2])}
	}
	if m := reInlineMarker.FindStringSubmatchIndex(prompt); m != nil {
		sub := reInlineMarker.FindStringSubmatch(prompt[m[0]:m[1]])
		if best == nil || m[0] < best.idx {
			best = &hit{idx: m[0], tradition: strings.ToLower(sub[1]), ref: strings.TrimSpace(sub[2])}
		}
	}
	if best == nil {
		return "", "", false
	}
	// For "sutra" with no ref, that's still valid (whole text).
	if best.ref == "" && best.tradition != "sutra" {
		// e.g. "/bible" alone — invalid. Treat as no marker.
		return "", "", false
	}
	return best.tradition, best.ref, true
}
