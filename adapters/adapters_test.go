package adapters

import (
	"strings"
	"testing"

	"github.com/MiaoDX/verse-driven/internal/packs"
)

func TestClaudeOutputStyleHasRequiredFrontmatter(t *testing.T) {
	body := readEmbedded(t, ClaudeOutputStylePath)
	for _, want := range []string{
		"name: Scripture-Aware Coding",
		"keep-coding-instructions: true",
		"<scripture_card>",
		"do not preach",
		"behave exactly like default Claude Code",
	} {
		if !strings.Contains(strings.ToLower(body), strings.ToLower(want)) {
			t.Errorf("output style missing %q", want)
		}
	}
	// Style file is style-only — must not name a specific tradition.
	for _, leak := range []string{"bible", "kjv", "dao", "道德", "sutra", "心经", "quran"} {
		if strings.Contains(strings.ToLower(body), leak) {
			t.Errorf("output style names tradition %q (must be tradition-agnostic)", leak)
		}
	}
}

func TestVerseInjectSkillIsManualOnlyAndShortAndScriptureFree(t *testing.T) {
	body := readEmbedded(t, ClaudeVerseInjectSkillPath)

	// < 60 lines (issue #5 acceptance criterion).
	if got := strings.Count(body, "\n"); got >= 60 {
		t.Errorf("SKILL.md has %d lines (want < 60)", got)
	}

	// Required frontmatter fields.
	for _, want := range []string{
		"name: verse-inject",
		"disable-model-invocation: true",
		"mcp__scripture__lookup",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("SKILL.md missing %q", want)
		}
	}

	// Skill body must contain no scripture text. Spot-check against
	// every bundled verse: any verse text appearing as a substring of
	// the skill body would mean we leaked scripture content.
	r := packs.All()
	for _, name := range r.Names() {
		p := r.Pack(name)
		if p.Meta.InclusionMode != "" && p.Meta.InclusionMode != "bundled" {
			continue
		}
		for _, v := range p.Verses() {
			if len(v.Text) < 12 {
				continue // too short to be a meaningful match
			}
			if strings.Contains(body, v.Text) {
				t.Errorf("SKILL.md contains scripture text from %s", v.ID)
				return
			}
		}
	}
}

func readEmbedded(t *testing.T, path string) string {
	t.Helper()
	b, err := ClaudeCodeFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
