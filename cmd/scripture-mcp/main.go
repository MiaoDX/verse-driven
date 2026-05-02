// scripture-mcp is the verse-driven binary. The full CLI surface
// (serve / lookup / lookup-from-prompt / recap / init) is implemented
// in issue #4. This entrypoint exposes just enough now to demonstrate
// that the embedded packs from issue #3 are reachable from main.
//
// Usage:
//
//	scripture-mcp                       # prints version and pack summary
//	scripture-mcp --packs               # prints loaded pack metadata
//	scripture-mcp --lookup-id <id>      # prints the canonical reference and
//	                                    # SHA-256 of one verse (text omitted
//	                                    # to keep terminal output filter-safe)
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MiaoDX/verse-driven/internal/packs"
)

const Version = "v0.0.0"

func main() {
	listPacks := flag.Bool("packs", false, "list loaded packs and exit")
	lookupID := flag.String("lookup-id", "", "look up a verse by id and print metadata only")
	flag.Parse()

	switch {
	case *listPacks:
		printPacks()
	case *lookupID != "":
		if err := printLookup(*lookupID); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("scripture-mcp %s\n", Version)
		fmt.Printf("packs loaded: %d  total verses: %d\n",
			len(packs.All().Names()), packs.All().TotalVerses())
	}
}

func printPacks() {
	r := packs.All()
	for _, name := range r.Names() {
		p := r.Pack(name)
		mode := p.Meta.InclusionMode
		if mode == "" {
			mode = "(unset)"
		}
		fmt.Printf("%-14s tradition=%-6s work=%-12s lang=%-6s verses=%-6d mode=%s\n",
			name, p.Meta.Tradition, p.Meta.Work, p.Meta.Lang, len(p.Verses()), mode)
	}
}

func printLookup(id string) error {
	v, ok := packs.All().Lookup(id)
	if !ok {
		return fmt.Errorf("verse not found: %s", id)
	}
	// Deliberately print only structural fields — never the verse text.
	// Callers who need the body should go through the MCP `lookup` tool
	// (issue #4), which has explicit user-confirm gating.
	fmt.Printf("id:        %s\n", v.ID)
	fmt.Printf("tradition: %s/%s\n", v.Tradition, v.Work)
	fmt.Printf("ref:       %s %d:%d", v.CanonicalRef.Book, v.CanonicalRef.Chapter, v.CanonicalRef.VerseStart)
	if v.CanonicalRef.VerseEnd != 0 {
		fmt.Printf("-%d", v.CanonicalRef.VerseEnd)
	}
	fmt.Println()
	fmt.Printf("lang:      %s\n", v.Lang)
	fmt.Printf("checksum:  %s\n", v.ChecksumSHA256)
	fmt.Printf("text_len:  %d bytes\n", len(v.Text))
	fmt.Printf("source:    %s — %s\n", v.Source.Provider, v.Source.License)
	return nil
}
