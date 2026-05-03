package cli

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode"

	"github.com/MiaoDX/verse-driven/internal/injector"
	"github.com/MiaoDX/verse-driven/internal/packs"
	"github.com/MiaoDX/verse-driven/internal/schema"
)

// runRecap implements `scripture-mcp recap [--tradition=<t>] [--terminal]
// [--first-letter] [--seed=<n>]`.
//
// Picks one verse from the requested tradition (default: any bundled
// tradition) and prints it to stdout. Output goes to the user's terminal
// only — by design, recap is invoked in contexts where its stdout never
// flows back into a model_call input (Claude Stop hook, Codex shell
// wrapper). See plan.md §2.1.
func runRecap(args []string, s Streams) int {
	fs := flag.NewFlagSet("recap", flag.ContinueOnError)
	fs.SetOutput(s.Err)
	tradition := fs.String("tradition", "", "limit to one tradition: bible|dao|sutra|quran")
	terminal := fs.Bool("terminal", false, "print pretty terminal output (default behavior)")
	firstLetter := fs.Bool("first-letter", false, "mask all-but-first character of each word/character (memory mode)")
	learning := fs.Bool("learning", false, "select and schedule verses from ~/.config/scripture-mcp/learning.json")
	seed := fs.Int64("seed", 0, "deterministic seed; 0 = time-based (default)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	_ = terminal // accepted for compatibility; the CLI prints to terminal regardless
	useLearning := *learning
	if !useLearning {
		enabled, err := learningEnabledFromConfig(s)
		if err != nil {
			fmt.Fprintln(s.Err, "error:", err)
			return 1
		}
		useLearning = enabled
	}

	var (
		v   schema.Verse
		ok  bool
		err error
	)
	if useLearning {
		v, ok, err = pickLearningRecapVerse(*tradition, *seed, s)
		if err != nil {
			fmt.Fprintln(s.Err, "error:", err)
			return 1
		}
	} else {
		v, ok = pickRecapVerse(*tradition, *seed)
	}
	if !ok {
		fmt.Fprintln(s.Err, "error: no bundled verses available for tradition", *tradition)
		return 1
	}
	body := v.Text
	if *firstLetter {
		body = firstLetterMask(body)
	}
	fmt.Fprintln(s.Out, "📖", injector.DisplayRef(v))
	if v.Source.Attribution != "" {
		fmt.Fprintf(s.Out, "(%s)\n", v.Source.Attribution)
	}
	fmt.Fprintln(s.Out)
	fmt.Fprintln(s.Out, body)
	return 0
}

// pickRecapVerse selects one bundled verse, optionally filtered to a
// single tradition. Returns (_, false) when no bundled verses match.
//
// Selection strategy: uniform random across all eligible verses. seed=0
// uses time-based randomness; non-zero seed makes the call deterministic
// (used by tests and by future spaced-repetition selectors).
func pickRecapVerse(tradition string, seed int64) (schema.Verse, bool) {
	var pool []schema.Verse
	r := packs.All()
	for _, name := range r.Names() {
		p := r.Pack(name)
		if p.Meta.InclusionMode != "" && p.Meta.InclusionMode != "bundled" {
			continue
		}
		if tradition != "" && p.Meta.Tradition != tradition {
			continue
		}
		pool = append(pool, p.Verses()...)
	}
	if len(pool) == 0 {
		return schema.Verse{}, false
	}
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))
	return pool[rng.Intn(len(pool))], true
}

// firstLetterMask returns a memory-pattern view of s. For each word made
// of ASCII letters we keep the first letter and replace the remaining
// letters with `_`. Whitespace and punctuation pass through. CJK runs are
// rendered character-by-character with each kept and `_` separators
// between them, so "三十辐" → "三 _ _" (helps with rote recall).
func firstLetterMask(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		b.WriteString(maskLine(line))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func maskLine(line string) string {
	var b strings.Builder
	inWord := false
	wordHasFirst := false
	for _, r := range line {
		switch {
		case unicode.Is(unicode.Han, r):
			if inWord {
				inWord = false
				wordHasFirst = false
			}
			b.WriteRune(r)
			b.WriteString(" _")
		case isWordRune(r):
			if !inWord {
				inWord = true
				wordHasFirst = false
			}
			if !wordHasFirst {
				b.WriteRune(r)
				wordHasFirst = true
			} else {
				b.WriteRune('_')
			}
		default:
			inWord = false
			wordHasFirst = false
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isWordRune(r rune) bool {
	if r > unicode.MaxASCII {
		return unicode.IsLetter(r)
	}
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
