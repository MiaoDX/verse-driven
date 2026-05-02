// Package cli implements the scripture-mcp subcommand surface:
// lookup, lookup-from-prompt, recap, and init.
//
// Subcommands are dispatched from cmd/scripture-mcp/main.go via Run.
// Tests for each subcommand live alongside the file that implements it.
package cli

import (
	"fmt"
	"io"

	"github.com/MiaoDX/verse-driven/internal/packs"
	"github.com/MiaoDX/verse-driven/internal/resolver"
	"github.com/MiaoDX/verse-driven/internal/schema"
)

// Streams bundles the stdio handles a subcommand reads/writes. Tests inject
// in-memory buffers; main wires os.Stdin/Stdout/Stderr.
type Streams struct {
	In     io.Reader
	Out    io.Writer
	Err    io.Writer
	HomeFn func() (string, error) // override for init's config-file location
}

// Run dispatches to the named subcommand. args excludes the subcommand
// itself (i.e. caller passes os.Args[2:]).
func Run(name string, args []string, s Streams) int {
	switch name {
	case "lookup":
		return runLookup(args, s)
	case "lookup-from-prompt":
		return runLookupFromPrompt(args, s)
	case "recap":
		return runRecap(args, s)
	case "init":
		return runInit(args, s)
	default:
		fmt.Fprintf(s.Err, "unknown subcommand: %q\n", name)
		fmt.Fprintln(s.Err, "usage: scripture-mcp {serve|lookup|lookup-from-prompt|recap|init} [flags]")
		return 2
	}
}

// reorderFlagsFirst moves all `--flag` / `-flag` (with optional `=value`)
// args ahead of positional args, so that `lookup "John 3:16" --format=json`
// parses identically to `lookup --format=json "John 3:16"`. The standard
// flag package stops at the first positional arg by design; the issue
// spec mixes flag and positional order, so we normalize before parsing.
//
// We deliberately do not handle `--flag value` (two-token form): the
// CLI uses `=value` exclusively to keep the rule trivially correct.
func reorderFlagsFirst(args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			// Standard convention: everything after `--` is positional.
			positional = append(positional, args[i+1:]...)
			break
		}
		if len(a) >= 2 && a[0] == '-' {
			flags = append(flags, a)
			continue
		}
		positional = append(positional, a)
	}
	return append(flags, positional...)
}

// resolveAndLookup parses a free-form reference and resolves it to a
// bundled verse. Returns packs.ErrNotBundled when the reference resolved
// to a tradition shipped api-only in this build (heart-sutra, quran).
func resolveAndLookup(ref string) (schema.Verse, error) {
	r, err := resolver.Resolve(ref)
	if err != nil {
		return schema.Verse{}, err
	}
	return packs.LookupReference(r)
}
