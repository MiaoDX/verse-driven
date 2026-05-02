// Package lifecycle simulates the per-turn prompt-passing behavior of a
// coding agent (Claude Code or Codex) wired to scripture-mcp's hooks, so
// that the v0.1 injection-lifecycle invariant is testable in CI without a
// live model.
//
// The invariant under test (issue #8):
//
//  1. Turn N: a marker in the user prompt → the verse appears in turn N's
//     model_call input via the hook envelope.
//  2. Turn N+1: no marker → the verse is GONE. It is not in turn N+1's
//     model_call input.
//  3. Compaction-resistant: this remains true across an arbitrarily long
//     run of follow-up turns (the harness exercises 30+).
//  4. Mode B recap stdout flows to a separate "terminal" channel that is
//     never fed back as input to subsequent turns.
//
// The harness drives the same scripture-mcp surface the real agents drive
// (lookup-from-prompt and recap), in-process via internal/cli, so the
// invariant follows from the binary's behavior rather than from a mock.
//
// What we do NOT simulate: the model itself. We assert on what enters the
// model_call input — the leftmost surface where the lifecycle invariant
// must hold. Whether the model can quote a verse it was given, and
// whether it cannot quote one it was not given, are LLM-side properties
// that the optional online check in TestLifecycleEnd2EndOnline covers
// when ANTHROPIC_API_KEY is set.
package lifecycle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MiaoDX/verse-driven/internal/cli"
)

// Adapter selects which marker syntax a simulated turn uses in its user
// prompt. Both adapters share the same lookup-from-prompt binary, so the
// only difference is how the user expresses the marker.
type Adapter int

const (
	// Claude: leading slash form, e.g. "/bible John 3:16 do the thing".
	AdapterClaude Adapter = iota
	// Codex: inline brackets form, e.g. "do the thing [[bible:John 3:16]]".
	AdapterCodex
)

func (a Adapter) String() string {
	switch a {
	case AdapterClaude:
		return "claude"
	case AdapterCodex:
		return "codex"
	}
	return "unknown"
}

// Turn captures everything that happens on one simulated user turn:
// the user-typed prompt, the model_call input the agent would assemble
// after running its hooks, and any text the Stop-hook recap printed to
// the user's terminal channel after the turn ended.
//
// ModelInput is the channel the lifecycle invariant cares about — it is
// what the model sees on this turn. RecapTerminal is the channel that
// must NEVER appear in any subsequent turn's ModelInput.
type Turn struct {
	UserPrompt    string
	ModelInput    string
	RecapTerminal string
}

// Simulator threads a sequence of Turns. Stateless across turns by
// design: each turn's ModelInput is computed solely from that turn's
// UserPrompt plus any envelope the hook produces — no carry-over from
// prior turns. This is the architectural property that makes the
// "verse is gone next turn" invariant true; the simulator makes it
// observable.
type Simulator struct {
	Adapter    Adapter
	transcript []Turn
	// recapSeed makes the optional Mode B recap deterministic for tests.
	// Zero leaves the recap CLI to its time-based default.
	recapSeed int64
	// recapTradition optionally constrains which tradition recap pulls
	// from — useful for tests that want a known-content recap.
	recapTradition string
}

// New returns a fresh Simulator wired to the given adapter.
func New(adapter Adapter) *Simulator {
	return &Simulator{Adapter: adapter}
}

// SetRecapSeed makes recap deterministic for tests; pair with
// SetRecapTradition to fully pin the recap output.
func (s *Simulator) SetRecapSeed(seed int64)    { s.recapSeed = seed }
func (s *Simulator) SetRecapTradition(t string) { s.recapTradition = t }

// Transcript returns the full slice of recorded turns.
func (s *Simulator) Transcript() []Turn { return s.transcript }

// Last returns the most recent recorded turn.
func (s *Simulator) Last() Turn {
	if len(s.transcript) == 0 {
		return Turn{}
	}
	return s.transcript[len(s.transcript)-1]
}

// RunTurn drives one turn through the hook surface. It runs the same
// `lookup-from-prompt` command the real adapter would invoke, captures
// any envelope the hook emits, builds the model_call input as
// "prompt + envelope" (the field semantics of UserPromptExpansion /
// UserPromptSubmit's additionalContext), and appends a Turn to the
// transcript.
//
// withRecap=true also fires the Stop-hook recap after the turn ends and
// records its output in RecapTerminal — the user-terminal-only channel
// the invariant says must never enter a model_call input.
func (s *Simulator) RunTurn(prompt string, withRecap bool) Turn {
	envelope := s.invokeLookupFromPrompt(prompt)
	model := prompt
	if envelope != "" {
		// additionalContext is concatenated with the user prompt; the
		// exact glue character doesn't matter for the invariant — what
		// matters is whether the verse text is present.
		model = prompt + "\n\n" + envelope
	}
	turn := Turn{UserPrompt: prompt, ModelInput: model}
	if withRecap {
		turn.RecapTerminal = s.invokeRecap()
	}
	s.transcript = append(s.transcript, turn)
	return turn
}

// invokeLookupFromPrompt calls the real CLI subcommand in-process. The
// hook's contract is to emit hookSpecificOutput.additionalContext on stdout
// when a marker is present, and exit 0 with no output otherwise.
func (s *Simulator) invokeLookupFromPrompt(prompt string) string {
	var stdout, stderr bytes.Buffer
	streams := cli.Streams{
		In:  strings.NewReader(prompt),
		Out: &stdout,
		Err: &stderr,
	}
	args := []string{"--hook-event=UserPromptExpansion"}
	if s.Adapter == AdapterCodex {
		args = []string{"--hook-event=UserPromptSubmit"}
	}
	code := cli.Run("lookup-from-prompt", args, streams)
	if code != 0 {
		// A failing hook still must not break the turn; the binary
		// soft-fails by spec. Treat as no envelope.
		return ""
	}
	if stdout.Len() == 0 {
		return ""
	}
	var resp struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return ""
	}
	return resp.HookSpecificOutput.AdditionalContext
}

// invokeRecap calls `recap` and captures its terminal-only output.
func (s *Simulator) invokeRecap() string {
	var stdout, stderr bytes.Buffer
	args := []string{"--terminal"}
	if s.recapTradition != "" {
		args = append(args, "--tradition="+s.recapTradition)
	}
	if s.recapSeed != 0 {
		args = append(args, fmt.Sprintf("--seed=%d", s.recapSeed))
	}
	streams := cli.Streams{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
	}
	if cli.Run("recap", args, streams) != 0 {
		return ""
	}
	return stdout.String()
}

// Leak describes a residual fragment found in a turn's ModelInput where
// the lifecycle invariant says it should not be. Returned by the assert
// helpers so test failures can name exactly which turn leaked and what
// content was found.
type Leak struct {
	TurnIndex int
	Fragment  string
	Found     string // a window around the leak for human inspection
}

func (l Leak) String() string {
	return fmt.Sprintf("turn %d leaked %q\n  context: %s",
		l.TurnIndex, l.Fragment, l.Found)
}

// FindLeaks scans every turn at index >= startIdx and reports every
// occurrence of fragment in that turn's ModelInput. Used to assert that
// a verse injected on turn N does not survive past turn N.
func (s *Simulator) FindLeaks(startIdx int, fragment string) []Leak {
	if fragment == "" {
		return nil
	}
	var out []Leak
	for i := startIdx; i < len(s.transcript); i++ {
		mi := s.transcript[i].ModelInput
		idx := strings.Index(mi, fragment)
		if idx < 0 {
			continue
		}
		out = append(out, Leak{
			TurnIndex: i,
			Fragment:  fragment,
			Found:     contextWindow(mi, idx, len(fragment)),
		})
	}
	return out
}

// FindRecapLeaks scans every turn's ModelInput for any non-empty recap
// text recorded in any prior turn. The architectural guarantee that
// recap stdout never enters a future model_call input is the core of
// Mode B's safety.
func (s *Simulator) FindRecapLeaks() []Leak {
	var leaks []Leak
	for i, t := range s.transcript {
		recap := strings.TrimSpace(t.RecapTerminal)
		if recap == "" {
			continue
		}
		// Use the verse-body line of the recap (line 3 onward, after
		// the header and blank line). This avoids false positives on
		// the "📖" emoji or attribution that could legitimately appear
		// in unrelated context.
		body := recapVerseBody(recap)
		if body == "" {
			continue
		}
		for j := i + 1; j < len(s.transcript); j++ {
			mi := s.transcript[j].ModelInput
			idx := strings.Index(mi, body)
			if idx < 0 {
				continue
			}
			leaks = append(leaks, Leak{
				TurnIndex: j,
				Fragment:  body,
				Found:     contextWindow(mi, idx, len(body)),
			})
		}
	}
	return leaks
}

// recapVerseBody pulls just the verse-body of a recap printout: skip
// the header line ("📖 ..."), the optional attribution line, and the
// blank line, and return the rest. This is the substring whose
// presence in a future model_call input would constitute a leak.
func recapVerseBody(recap string) string {
	lines := strings.Split(recap, "\n")
	// Drop leading non-empty header lines until we hit a blank.
	var i int
	for i = 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			i++
			break
		}
	}
	body := strings.TrimSpace(strings.Join(lines[i:], "\n"))
	return body
}

// contextWindow returns a few chars on each side of the leak position
// for human-readable failure messages.
func contextWindow(s string, start, length int) string {
	const pad = 32
	from := start - pad
	if from < 0 {
		from = 0
	}
	to := start + length + pad
	if to > len(s) {
		to = len(s)
	}
	w := s[from:to]
	w = strings.ReplaceAll(w, "\n", "⏎")
	return "..." + w + "..."
}
