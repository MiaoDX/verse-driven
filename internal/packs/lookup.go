package packs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MiaoDX/verse-driven/internal/resolver"
	"github.com/MiaoDX/verse-driven/internal/schema"
)

// ErrNotBundled signals the requested reference resolved to a tradition
// that is shipped api-only in this build. Callers can distinguish this
// from a hard "id not in pack" miss with errors.Is.
var ErrNotBundled = errors.New("packs: verse not bundled in this build")

// LookupReference maps a parsed resolver.Reference to a pack verse id and
// returns the matching verse. Used by both the CLI and the MCP server so
// the two interfaces share one mapping definition.
func LookupReference(r resolver.Reference) (schema.Verse, error) {
	id, err := ReferenceID(r)
	if err != nil {
		return schema.Verse{}, err
	}
	v, ok := All().Lookup(id)
	if !ok {
		return schema.Verse{}, fmt.Errorf("verse not found: %s", id)
	}
	return v, nil
}

// ReferenceID renders a resolver.Reference as the dotted lowercase pack
// id (`<tradition>.<work>.<book?>.<chapter>.<verse>`). For traditions
// where the resolver doesn't carry a verse number (e.g. dao "chapter 11",
// heart sutra), VerseStart defaults to 1.
func ReferenceID(r resolver.Reference) (string, error) {
	verse := r.VerseStart
	switch r.Tradition {
	case resolver.TraditionBible:
		if r.Book == "" {
			return "", fmt.Errorf("bible reference missing book")
		}
		if r.Chapter < 1 || verse < 1 {
			return "", fmt.Errorf("bible reference must include chapter and verse")
		}
		work := "kjv"
		if r.Work == resolver.WorkCUVS || r.Lang == "zh-Hans" {
			work = "cuv-s"
		}
		return fmt.Sprintf("bible.%s.%s.%d.%d", work, bookSlug(r.Book), r.Chapter, verse), nil
	case resolver.TraditionDao:
		if r.Chapter < 1 {
			return "", fmt.Errorf("dao reference missing chapter")
		}
		if verse < 1 {
			verse = 1
		}
		work := "daodejing"
		if r.Work == resolver.WorkDaoLegge || r.Lang == "en" {
			work = "legge"
		}
		return fmt.Sprintf("dao.%s.%d.%d", work, r.Chapter, verse), nil
	case resolver.TraditionSutra:
		if verse < 1 {
			verse = 1
		}
		work := "heart-sutra"
		if r.Work == resolver.WorkHeartSutraEn || r.Lang == "en" {
			work = "heart-sutra-en"
		}
		return fmt.Sprintf("sutra.%s.%d", work, verse), nil
	case resolver.TraditionQuran:
		if r.Chapter < 1 || verse < 1 {
			return "", fmt.Errorf("quran reference must include surah and verse")
		}
		work := "pickthall"
		if r.Work == resolver.WorkQuranMajian || r.Lang == "zh-Hans" {
			work = "majian"
		}
		return fmt.Sprintf("quran.%s.%d.%d", work, r.Chapter, verse), nil
	}
	return "", fmt.Errorf("unsupported tradition: %s", r.Tradition)
}

func bookSlug(book string) string {
	out := strings.ToLower(strings.TrimSpace(book))
	out = strings.Join(strings.Fields(out), "-")
	return out
}
