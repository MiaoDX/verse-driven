// Package injector renders the temporary, one-turn scripture envelope
// described in plan.md §6.3. The envelope is the only place a verse body
// is rendered into model-visible context, and it is rendered on a per-turn
// basis only.
package injector

import (
	"fmt"
	"strings"

	"github.com/MiaoDX/verse-driven/internal/schema"
)

// Envelope wraps a verse in the plan §6.3 reflection-context block.
//
// The output is plain text with a <scripture_card>...</scripture_card>
// island. The wrapping language (1) marks the block as one-turn-only and
// (2) instructs the model not to alter engineering rigor or preach. The
// model-side reinforcement lives in the output style (issue #5); the
// envelope's job here is to mark the block clearly.
func Envelope(v schema.Verse) string {
	var b strings.Builder
	b.WriteString("Temporary reflection context for this turn only.\n")
	b.WriteString("Do not alter engineering rigor, verification, testing, or safety behavior.\n")
	b.WriteString("Use the scripture only as an optional reflective frame.\n")
	b.WriteString("Quote verbatim if you mention it. Do not preach.\n\n")
	b.WriteString("<scripture_card>\n")
	b.WriteString(v.Text)
	b.WriteString("\n\n— ")
	b.WriteString(DisplayRef(v))
	if v.Source.Attribution != "" {
		b.WriteString(", ")
		b.WriteString(v.Source.Attribution)
	}
	if v.ChecksumSHA256 != "" {
		// Surface the integrity checksum in the preview so the user can
		// verify the verse came from the bundled pack and was not
		// rewritten in transit. The model is instructed not to act on
		// this beyond reproducing it verbatim if asked.
		b.WriteString("\nchecksum: sha256:")
		b.WriteString(v.ChecksumSHA256)
	}
	b.WriteString("\n</scripture_card>\n")
	return b.String()
}

// DisplayRef formats a verse's canonical reference for human display.
// Prefers the verse language's DisplayRef when set, then English, then
// Simplified Chinese, and finally a tradition-specific rendering.
func DisplayRef(v schema.Verse) string {
	if v.DisplayRef != nil {
		if v.Lang != "" {
			if s, ok := v.DisplayRef[v.Lang]; ok && s != "" {
				return s
			}
		}
		if s, ok := v.DisplayRef["en"]; ok && s != "" {
			return s
		}
		if s, ok := v.DisplayRef["zh-Hans"]; ok && s != "" {
			return s
		}
	}
	switch v.Tradition {
	case "bible":
		if v.CanonicalRef.VerseEnd != 0 && v.CanonicalRef.VerseEnd != v.CanonicalRef.VerseStart {
			return fmt.Sprintf("%s %d:%d-%d", v.CanonicalRef.Book, v.CanonicalRef.Chapter, v.CanonicalRef.VerseStart, v.CanonicalRef.VerseEnd)
		}
		return fmt.Sprintf("%s %d:%d", v.CanonicalRef.Book, v.CanonicalRef.Chapter, v.CanonicalRef.VerseStart)
	case "dao":
		return fmt.Sprintf("道德经 第%d章", v.CanonicalRef.Chapter)
	case "sutra":
		return "心经"
	case "quran":
		return fmt.Sprintf("Quran %d:%d", v.CanonicalRef.Chapter, v.CanonicalRef.VerseStart)
	}
	return v.ID
}
