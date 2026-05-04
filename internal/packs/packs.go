// Package packs holds embedded verse data (Bible, Dao, Quran, 心经, ...).
//
// On import, init() decompresses each pack's verses.jsonl.gz, materializes
// schema.Verse values from compact JSONL rows + metadata.json, and indexes
// them by id. Lookups are O(1).
//
// Compact JSONL row format (one per line in verses.jsonl.gz):
//
//	{"id":"bible.kjv.john.3.16","c":3,"v":16,"t":"...","s":"<hex64>"}
//
// Optional fields: "ve" (verse_end), "b" (book display name; defaults to
// metadata.books[<slug>] for multi-book traditions), and "d" (per-language
// display_ref strings). Pack-shared fields
// (tradition, work, lang, source.*, inclusion_mode, sensitivity) live in
// metadata.json so the JSONL stays small enough to fit the 6 MB budget.
package packs

import (
	"bufio"
	"compress/gzip"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/MiaoDX/verse-driven/internal/schema"
)

//go:embed all:bible-kjv all:bible-cuv-s all:dao-de-jing all:dao-legge all:heart-sutra all:heart-sutra-en all:quran-pickthall all:quran-majian
var fs embed.FS

// PackName identifies an embedded pack on disk.
type PackName string

const (
	PackBibleKJV       PackName = "bible-kjv"
	PackBibleCUVS      PackName = "bible-cuv-s"
	PackDaoDeJing      PackName = "dao-de-jing"
	PackDaoLegge       PackName = "dao-legge"
	PackHeartSutra     PackName = "heart-sutra"
	PackHeartSutraEn   PackName = "heart-sutra-en"
	PackQuranPickthall PackName = "quran-pickthall"
	PackQuranMajian    PackName = "quran-majian"
)

// Metadata is the parsed contents of a pack's metadata.json.
type Metadata struct {
	Tradition     string            `json:"tradition"`
	Work          string            `json:"work"`
	Lang          string            `json:"lang"`
	Provider      string            `json:"provider"`
	License       string            `json:"license"`
	Attribution   string            `json:"attribution"`
	SourceURL     string            `json:"source_url,omitempty"`
	EditionID     string            `json:"edition_id,omitempty"`
	InclusionMode string            `json:"inclusion_mode,omitempty"`
	Sensitivity   string            `json:"sensitivity,omitempty"`
	Transform     string            `json:"transform,omitempty"`
	Note          string            `json:"note,omitempty"`
	Books         map[string]string `json:"books,omitempty"`
	VerseCount    int               `json:"verse_count"`
	BuildDate     string            `json:"build_date,omitempty"`
}

// Pack is one loaded pack: metadata + indexed verses.
type Pack struct {
	Name   PackName
	Meta   Metadata
	verses []schema.Verse
	byID   map[string]int // id -> index in verses
}

// Verses returns all verses in pack order.
func (p *Pack) Verses() []schema.Verse { return p.verses }

// Lookup returns the verse with the given id and whether it exists.
func (p *Pack) Lookup(id string) (schema.Verse, bool) {
	i, ok := p.byID[id]
	if !ok {
		return schema.Verse{}, false
	}
	return p.verses[i], true
}

// Registry is the union of all loaded packs, keyed by PackName.
type Registry struct {
	packs map[PackName]*Pack
}

// All returns the singleton registry.
func All() *Registry { return registry }

// Pack returns the pack by name, or nil if unknown.
func (r *Registry) Pack(name PackName) *Pack {
	if r == nil {
		return nil
	}
	return r.packs[name]
}

// Names returns the loaded pack names in deterministic order.
func (r *Registry) Names() []PackName {
	if r == nil {
		return nil
	}
	out := make([]PackName, 0, len(r.packs))
	for n := range r.packs {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Lookup searches all packs for the given verse id.
func (r *Registry) Lookup(id string) (schema.Verse, bool) {
	if r == nil {
		return schema.Verse{}, false
	}
	for _, n := range r.Names() {
		if v, ok := r.packs[n].Lookup(id); ok {
			return v, true
		}
	}
	return schema.Verse{}, false
}

// TotalVerses sums verse_count across loaded packs.
func (r *Registry) TotalVerses() int {
	total := 0
	for _, p := range r.packs {
		total += len(p.verses)
	}
	return total
}

// ErrPackEmpty is returned for packs whose verses.jsonl.gz contains no rows
// (e.g. heart-sutra is shipped as inclusion_mode=api_only).
var ErrPackEmpty = errors.New("packs: pack contains no bundled verses")

var registry *Registry

func init() {
	r, err := loadAll()
	if err != nil {
		// Fail loudly: a broken pack at startup means a build-side problem
		// the user has to fix; silent fallback would hide regressions.
		panic(fmt.Errorf("packs: init failed: %w", err))
	}
	registry = r
}

func loadAll() (*Registry, error) {
	r := &Registry{packs: make(map[PackName]*Pack)}
	for _, name := range []PackName{
		PackBibleKJV,
		PackBibleCUVS,
		PackDaoDeJing,
		PackDaoLegge,
		PackHeartSutra,
		PackHeartSutraEn,
		PackQuranPickthall,
		PackQuranMajian,
	} {
		p, err := loadPack(name)
		if err != nil {
			return nil, fmt.Errorf("pack %s: %w", name, err)
		}
		r.packs[name] = p
	}
	return r, nil
}

func loadPack(name PackName) (*Pack, error) {
	metaBytes, err := fs.ReadFile(string(name) + "/metadata.json")
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	var meta Metadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	gzData, err := fs.ReadFile(string(name) + "/verses.jsonl.gz")
	if err != nil {
		return nil, fmt.Errorf("read verses.jsonl.gz: %w", err)
	}
	gr, err := gzip.NewReader(strings.NewReader(string(gzData)))
	if err != nil {
		return nil, fmt.Errorf("gzip open: %w", err)
	}
	defer gr.Close()

	verses, byID, err := parseRows(gr, meta)
	if err != nil {
		return nil, err
	}
	return &Pack{
		Name:   name,
		Meta:   meta,
		verses: verses,
		byID:   byID,
	}, nil
}

type compactRow struct {
	ID         string            `json:"id"`
	Chapter    int               `json:"c"`
	Verse      int               `json:"v"`
	VerseEnd   int               `json:"ve,omitempty"`
	Book       string            `json:"b,omitempty"`
	DisplayRef map[string]string `json:"d,omitempty"`
	Text       string            `json:"t"`
	Checksum   string            `json:"s"`
}

func parseRows(r io.Reader, meta Metadata) ([]schema.Verse, map[string]int, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<16), 1<<20)
	var out []schema.Verse
	byID := make(map[string]int)
	source := schema.Source{
		Provider:    meta.Provider,
		License:     meta.License,
		Attribution: meta.Attribution,
	}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var row compactRow
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		book := row.Book
		if book == "" && meta.Books != nil {
			// Slug is the third dotted segment: tradition.work.<slug>.chapter.verse
			parts := strings.Split(row.ID, ".")
			if len(parts) >= 5 {
				if disp, ok := meta.Books[parts[2]]; ok {
					book = disp
				}
			}
		}
		v := schema.Verse{
			ID:        row.ID,
			Tradition: meta.Tradition,
			Lang:      meta.Lang,
			Work:      meta.Work,
			CanonicalRef: schema.CanonicalRef{
				Book:       book,
				Chapter:    row.Chapter,
				VerseStart: row.Verse,
				VerseEnd:   row.VerseEnd,
			},
			DisplayRef:     row.DisplayRef,
			Text:           row.Text,
			Source:         source,
			ChecksumSHA256: row.Checksum,
			InclusionMode:  meta.InclusionMode,
			Sensitivity:    meta.Sensitivity,
		}
		if err := schema.Validate(v); err != nil {
			return nil, nil, fmt.Errorf("line %d (%s): %w", lineNo, row.ID, err)
		}
		byID[row.ID] = len(out)
		out = append(out, v)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan: %w", err)
	}
	return out, byID, nil
}
