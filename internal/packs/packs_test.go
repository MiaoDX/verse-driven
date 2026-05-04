package packs

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/MiaoDX/verse-driven/internal/schema"
)

// TestRegistryLoaded ensures all bilingual packs were registered at init.
func TestRegistryLoaded(t *testing.T) {
	r := All()
	if r == nil {
		t.Fatal("registry nil")
	}
	got := r.Names()
	want := []PackName{
		PackBibleCUVS,
		PackBibleKJV,
		PackDaoDeJing,
		PackDaoLegge,
		PackHeartSutra,
		PackHeartSutraEn,
		PackQuranMajian,
		PackQuranPickthall,
	}
	if len(got) != len(want) {
		t.Fatalf("Names: got %d packs, want %d", len(got), len(want))
	}
	for i, n := range want {
		if got[i] != n {
			t.Errorf("Names[%d]: got %q, want %q", i, got[i], n)
		}
	}
}

func TestKJVCounts(t *testing.T) {
	pack := All().Pack(PackBibleKJV)
	if pack == nil {
		t.Fatal("PackBibleKJV missing")
	}
	if pack.Meta.Tradition != "bible" || pack.Meta.Work != "KJV" {
		t.Errorf("metadata: got tradition=%q work=%q", pack.Meta.Tradition, pack.Meta.Work)
	}
	const want = 31102 // canonical KJV verse count
	if got := len(pack.Verses()); got != want {
		t.Errorf("KJV verse count: got %d, want %d", got, want)
	}
	if got := len(pack.Meta.Books); got != 66 {
		t.Errorf("KJV book count in metadata: got %d, want 66", got)
	}
}

func TestCUVSCounts(t *testing.T) {
	pack := All().Pack(PackBibleCUVS)
	if pack == nil {
		t.Fatal("PackBibleCUVS missing")
	}
	if pack.Meta.Tradition != "bible" || pack.Meta.Work != "CUV-S" || pack.Meta.Lang != "zh-Hans" {
		t.Errorf("metadata: got tradition=%q work=%q lang=%q", pack.Meta.Tradition, pack.Meta.Work, pack.Meta.Lang)
	}
	const want = 31100
	if got := len(pack.Verses()); got != want {
		t.Errorf("CUV-S verse count: got %d, want %d", got, want)
	}
	if got := len(pack.Meta.Books); got != 66 {
		t.Errorf("CUV-S book count in metadata: got %d, want 66", got)
	}
}

func TestDaoCounts(t *testing.T) {
	pack := All().Pack(PackDaoDeJing)
	if pack == nil {
		t.Fatal("PackDaoDeJing missing")
	}
	if got := len(pack.Verses()); got != 81 {
		t.Errorf("Dao chapter count: got %d, want 81", got)
	}
}

func TestDaoLeggeCounts(t *testing.T) {
	pack := All().Pack(PackDaoLegge)
	if pack == nil {
		t.Fatal("PackDaoLegge missing")
	}
	if got := len(pack.Verses()); got != 81 {
		t.Errorf("Dao Legge chapter count: got %d, want 81", got)
	}
	if pack.Meta.Lang != "en" {
		t.Errorf("Dao Legge lang: got %q, want en", pack.Meta.Lang)
	}
}

func TestHeartSutraBundled(t *testing.T) {
	pack := All().Pack(PackHeartSutra)
	if pack == nil {
		t.Fatal("PackHeartSutra missing")
	}
	if got := len(pack.Verses()); got != 1 {
		t.Errorf("HeartSutra verse count: got %d, want 1", got)
	}
	if pack.Meta.InclusionMode != "bundled" {
		t.Errorf("HeartSutra inclusion_mode: got %q, want %q", pack.Meta.InclusionMode, "bundled")
	}
}

func TestHeartSutraEnBundled(t *testing.T) {
	pack := All().Pack(PackHeartSutraEn)
	if pack == nil {
		t.Fatal("PackHeartSutraEn missing")
	}
	if got := len(pack.Verses()); got != 1 {
		t.Errorf("HeartSutraEn verse count: got %d, want 1", got)
	}
	if pack.Meta.Lang != "en" {
		t.Errorf("HeartSutraEn lang: got %q, want en", pack.Meta.Lang)
	}
}

func TestQuranTranslationCounts(t *testing.T) {
	cases := []struct {
		name PackName
		lang string
	}{
		{PackQuranPickthall, "en"},
		{PackQuranMajian, "zh-Hans"},
	}
	for _, c := range cases {
		pack := All().Pack(c.name)
		if pack == nil {
			t.Fatalf("%s missing", c.name)
		}
		if got := len(pack.Verses()); got != 6236 {
			t.Errorf("%s verse count: got %d, want 6236", c.name, got)
		}
		if pack.Meta.Lang != c.lang {
			t.Errorf("%s lang: got %q, want %q", c.name, pack.Meta.Lang, c.lang)
		}
	}
}

// TestSpotChecksums asserts known stable verses by their SHA-256, never by
// text content. The checksums here were computed by scripts/build_packs.py
// from canonical Project Gutenberg sources; if upstream PG #10 or PG #7337
// ever change, regenerate via `python3 scripts/build_packs.py`, run
// `python3 scripts/verify_quotes.py`, and update the values below.
func TestSpotChecksums(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		// KJV anchors at the start of OT, the most-cited NT verse, and
		// the very last verse of the canon.
		{"bible.kjv.genesis.1.1", "6f785a86b2716dcc5a48caa0de944396ba871d5c7f3bf776993648335fcb2bb2"},
		{"bible.kjv.john.3.16", "8473c0b1c7664945528317faf77351258eb79f8b11ba821ef76d7e916cde711a"},
		{"bible.cuv-s.john.3.16", "eb9377734c9de55a5fb47ff21405fbd9fc7aac478ac88339491348fa117f9509"},
		{"bible.kjv.revelation.22.21", "76128832e1fddeeda339fb4424682d629e372e7965425ba19efbf31038b54ab2"},
		// Dao chapter 11 is the README example ("三十辐共一毂...").
		{"dao.daodejing.11.1", "81ba9b4c9a51241154bf5f1c7a8b37d16234717b4f29c9522b58d04ad73d95b3"},
		{"dao.legge.11.1", "8ef8185a229525015081a5d3bcb5150bdcc0cd13d9a58d70efffb9141f2546f9"},
		{"sutra.heart-sutra.1", "08cd20f4996c4b7f44b5978fbc65f6d82e738a3f3a01b2715303d7d94852fff2"},
		{"sutra.heart-sutra-en.1", "b3f7511dba60c53a7a8f536d8b749bceea7cb1123218f37af8145bbf1fef2e95"},
		{"quran.pickthall.2.255", "f174338173480fd74890bc2dcc0d605c6f418a9923abb17d4e76bee129a5cd64"},
		{"quran.majian.2.255", "afe17706c0849792de579c9382c285c073fb09e4102a5791414b4d08456e5475"},
	}
	r := All()
	for _, c := range cases {
		v, ok := r.Lookup(c.id)
		if !ok {
			t.Errorf("Lookup(%q): not found", c.id)
			continue
		}
		actual := hashOf(v.Text)
		if actual != v.ChecksumSHA256 {
			t.Errorf("%s: stored checksum %q != recomputed %q", c.id, v.ChecksumSHA256, actual)
		}
		// We compare to the test's expected only when it isn't the
		// placeholder zeros. The build emits authoritative values; this
		// table acts as a sanity tripwire and is updated alongside the
		// pack regen.
		if !isPlaceholder(c.want) && actual != c.want {
			t.Errorf("%s: checksum drift: got %q, want %q (regenerate test fixtures)", c.id, actual, c.want)
		}
	}
}

// TestEveryVerseChecksumSelfConsistent ensures every loaded verse's stored
// checksum_sha256 matches the SHA-256 of its Text — i.e. the JSONL did not
// drift from the text. This is the guarantee verify_quotes.py also enforces
// at build time.
func TestEveryVerseChecksumSelfConsistent(t *testing.T) {
	r := All()
	for _, name := range r.Names() {
		pack := r.Pack(name)
		for _, v := range pack.Verses() {
			if v.ChecksumSHA256 != hashOf(v.Text) {
				t.Errorf("%s: checksum drift", v.ID)
				break // one failure per pack is enough
			}
		}
	}
}

func TestEveryVerseValidatesAgainstSchema(t *testing.T) {
	r := All()
	for _, name := range r.Names() {
		pack := r.Pack(name)
		for _, v := range pack.Verses() {
			if err := schema.Validate(v); err != nil {
				t.Errorf("%s: schema invalid: %v", v.ID, err)
				break
			}
		}
	}
}

// TestKJVBookCoverage ensures every one of the 66 books has at least one
// verse — catches regressions like the Haggai parse bug.
func TestKJVBookCoverage(t *testing.T) {
	pack := All().Pack(PackBibleKJV)
	seen := make(map[string]int, 66)
	for _, v := range pack.Verses() {
		// id format: bible.kjv.<slug>.<chapter>.<verse>
		parts := strings.Split(v.ID, ".")
		if len(parts) < 5 {
			t.Errorf("malformed id: %q", v.ID)
			continue
		}
		seen[parts[2]]++
	}
	if len(seen) != 66 {
		t.Errorf("expected 66 KJV books, got %d", len(seen))
	}
	for slug := range pack.Meta.Books {
		if seen[slug] == 0 {
			t.Errorf("KJV book %q has no verses", slug)
		}
	}
}

func hashOf(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// isPlaceholder distinguishes "not yet captured" sentinels (all zeros) from
// real expected hashes.
func isPlaceholder(s string) bool {
	for _, c := range s {
		if c != '0' {
			return false
		}
	}
	return true
}
