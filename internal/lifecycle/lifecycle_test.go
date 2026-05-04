package lifecycle

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MiaoDX/verse-driven/internal/packs"
	"github.com/MiaoDX/verse-driven/internal/resolver"
)

// fetchVerseText resolves a reference and returns its body. Tests use
// this to derive the leak-fragment string from the bundled pack rather
// than hard-coding scripture text in the test source.
func fetchVerseText(t *testing.T, ref string) string {
	t.Helper()
	r, err := resolver.Resolve(ref)
	if err != nil {
		t.Fatalf("resolve %q: %v", ref, err)
	}
	v, err := packs.LookupReference(r)
	if err != nil {
		t.Fatalf("lookup %q: %v", ref, err)
	}
	if v.Text == "" {
		t.Fatalf("verse %q has empty text", ref)
	}
	return v.Text
}

// followUpPrompt returns a per-turn no-marker prompt. Variation matters:
// we want to be sure the absence of leakage isn't accidentally caused by
// the follow-up prompt happening to overlap a marker pattern.
func followUpPrompt(i int) string {
	templates := []string{
		"keep going on the helper",
		"now run the tests",
		"refactor that function please",
		"add a docstring to the new method",
		"rename the variable to be clearer",
		"explain why this branch is taken",
		"check the imports for unused ones",
		"split the long function into two",
		"add a unit test for the edge case",
		"fix the staticcheck warning",
	}
	return fmt.Sprintf("turn %d: %s", i, templates[i%len(templates)])
}

// TestLifecycleClaude_TurnNVisibleNPlus1Gone is the make-or-break
// invariant from issue #8 for the Claude adapter. Turn 0 injects via
// the slash marker; turn 1 has no marker; we assert the verse body is
// present in turn 0's ModelInput and absent from turn 1's.
func TestLifecycleClaude_TurnNVisibleNPlus1Gone(t *testing.T) {
	verseText := fetchVerseText(t, "John 3:16")
	sim := New(AdapterClaude)

	turn0 := sim.RunTurn("/bible John 3:16 Refactor the cron-string scheduler.", false)
	if !strings.Contains(turn0.ModelInput, verseText) {
		t.Fatalf("turn 0: verse must be in model input but is not\n%s", turn0.ModelInput)
	}
	if !strings.Contains(turn0.ModelInput, "<scripture_card>") {
		t.Fatalf("turn 0: envelope tags missing")
	}

	turn1 := sim.RunTurn(followUpPrompt(1), false)
	if leaks := sim.FindLeaks(1, verseText); len(leaks) > 0 {
		t.Fatalf("turn 1 leaked verse text:\n%s", leaks[0])
	}
	if strings.Contains(turn1.ModelInput, "<scripture_card>") {
		t.Fatalf("turn 1: envelope tags must not appear")
	}
}

// TestLifecycleCodex_TurnNVisibleNPlus1Gone — same invariant for the
// Codex inline-marker syntax.
func TestLifecycleCodex_TurnNVisibleNPlus1Gone(t *testing.T) {
	verseText := fetchVerseText(t, "John 3:16")
	sim := New(AdapterCodex)

	turn0 := sim.RunTurn("Refactor the cron-string scheduler. [[bible:John 3:16]]", false)
	if !strings.Contains(turn0.ModelInput, verseText) {
		t.Fatalf("turn 0: verse must be in model input but is not")
	}

	sim.RunTurn(followUpPrompt(1), false)
	if leaks := sim.FindLeaks(1, verseText); len(leaks) > 0 {
		t.Fatalf("turn 1 leaked verse text:\n%s", leaks[0])
	}
}

// TestLifecycleCompactionResistantClaude exercises the issue's stronger
// claim: even after 30 follow-up turns, the verse from the first turn
// is not recoverable via the model_call input. This is the
// "compaction-resistant" property — it follows architecturally from the
// stateless-per-turn hook design, but we verify it explicitly.
func TestLifecycleCompactionResistantClaude(t *testing.T) {
	verseText := fetchVerseText(t, "John 3:16")
	sim := New(AdapterClaude)

	sim.RunTurn("/bible John 3:16 Refactor X.", false)
	const followUps = 30
	for i := 1; i <= followUps; i++ {
		sim.RunTurn(followUpPrompt(i), false)
	}
	if leaks := sim.FindLeaks(1, verseText); len(leaks) > 0 {
		t.Fatalf("verse leaked across %d follow-up turns:\n%s", followUps, leaks[0])
	}
}

func TestLifecycleCompactionResistantCodex(t *testing.T) {
	verseText := fetchVerseText(t, "道德经 11")
	sim := New(AdapterCodex)

	sim.RunTurn("Refactor X. [[dao:11]]", false)
	const followUps = 30
	for i := 1; i <= followUps; i++ {
		sim.RunTurn(followUpPrompt(i), false)
	}
	if leaks := sim.FindLeaks(1, verseText); len(leaks) > 0 {
		t.Fatalf("verse leaked across %d follow-up turns:\n%s", followUps, leaks[0])
	}
}

// TestRecapNeverEntersModelInput is the Mode B half of issue #8. After
// every turn we fire the Stop hook (recap), and we assert that the
// recap text never appears in any subsequent turn's ModelInput. The
// architectural reason it doesn't: recap output goes to stdout (= the
// user's terminal), not to additionalContext. The harness models that
// channel separation explicitly via Turn.RecapTerminal.
func TestRecapNeverEntersModelInputClaude(t *testing.T) {
	sim := New(AdapterClaude)
	sim.SetRecapSeed(42)
	sim.SetRecapTradition("dao") // small pool, deterministic

	const turns = 30
	for i := 0; i < turns; i++ {
		sim.RunTurn(followUpPrompt(i), true /*withRecap*/)
	}
	leaks := sim.FindRecapLeaks()
	if len(leaks) > 0 {
		t.Fatalf("recap leaked into a future model input: %s", leaks[0])
	}
	// Sanity: at least one turn must actually have produced a recap, or
	// we'd be passing the test trivially.
	any := false
	for _, tn := range sim.Transcript() {
		if strings.TrimSpace(tn.RecapTerminal) != "" {
			any = true
			break
		}
	}
	if !any {
		t.Fatalf("no recap output recorded across %d turns; cannot assert isolation", turns)
	}
}

func TestRecapNeverEntersModelInputCodex(t *testing.T) {
	sim := New(AdapterCodex)
	sim.SetRecapSeed(7)
	sim.SetRecapTradition("dao")

	const turns = 30
	for i := 0; i < turns; i++ {
		sim.RunTurn(followUpPrompt(i), true)
	}
	if leaks := sim.FindRecapLeaks(); len(leaks) > 0 {
		t.Fatalf("recap leaked into a future model input: %s", leaks[0])
	}
}

// TestLeakFailureMessageNamesTurnAndContent verifies that when a leak
// IS found, the harness's failure message identifies the turn index and
// shows a context window — required by issue #8 ("On failure, prints
// which turn leaked and what residual content was found").
func TestLeakFailureMessageNamesTurnAndContent(t *testing.T) {
	sim := New(AdapterClaude)
	sim.RunTurn("/bible John 3:16 do X", false)
	// Manually corrupt turn 1's model input to simulate a regression.
	sim.transcript = append(sim.transcript, Turn{
		UserPrompt: "follow up",
		ModelInput: "follow up\nstray verse fragment: For God so loved",
	})
	leaks := sim.FindLeaks(1, "For God so loved")
	if len(leaks) != 1 {
		t.Fatalf("expected 1 leak, got %d", len(leaks))
	}
	msg := leaks[0].String()
	if !strings.Contains(msg, "turn 1") {
		t.Errorf("failure message must name the turn index; got: %s", msg)
	}
	if !strings.Contains(msg, "context:") {
		t.Errorf("failure message must include a content window; got: %s", msg)
	}
}

// TestEnvelopeIsolatedToInjectingTurn confirms that re-running
// lookup-from-prompt on a turn-N+1 prompt that does not contain the
// marker yields no envelope at all — the lifecycle invariant is a
// straightforward consequence.
func TestEnvelopeIsolatedToInjectingTurn(t *testing.T) {
	sim := New(AdapterClaude)
	sim.RunTurn("/bible John 3:16 do X", false)
	turn1 := sim.RunTurn("just a regular follow-up", false)
	if strings.Contains(turn1.ModelInput, "<scripture_card>") {
		t.Errorf("turn 1 must have no envelope; got:\n%s", turn1.ModelInput)
	}
	if turn1.ModelInput != turn1.UserPrompt {
		t.Errorf("turn 1 model input must equal the user prompt; got %q vs %q",
			turn1.ModelInput, turn1.UserPrompt)
	}
}

// TestSlashMarkerTraditions covers all four traditions for the Claude
// slash-marker form. The lifecycle invariant must hold regardless of
// which tradition is injected.
func TestSlashMarkerTraditionsLifecycle(t *testing.T) {
	cases := []struct {
		marker string
		ref    string
	}{
		{"/bible John 3:16", "John 3:16"},
		{"/bible 约翰福音 3:16", "约翰福音 3:16"},
		{"/dao 11", "dao 11"},
		{"/dao 道德经第十一章", "道德经 11"},
		{"/sutra Heart Sutra", "Heart Sutra"},
		{"/sutra 心经", "心经"},
		{"/quran 2:255", "Quran 2:255"},
		{"/quran 古兰经 2:255", "古兰经 2:255"},
	}
	for _, tc := range cases {
		t.Run(tc.marker, func(t *testing.T) {
			verseText := fetchVerseText(t, tc.ref)
			sim := New(AdapterClaude)
			turn0 := sim.RunTurn(tc.marker+" do the thing", false)
			if !strings.Contains(turn0.ModelInput, verseText) {
				t.Fatalf("turn 0: verse missing for %s", tc.marker)
			}
			sim.RunTurn("follow up no marker", false)
			if leaks := sim.FindLeaks(1, verseText); len(leaks) > 0 {
				t.Fatalf("%s leaked next turn:\n%s", tc.marker, leaks[0])
			}
		})
	}
}

// TestRecapBodyExtractor verifies that recapVerseBody isolates the
// verse text from the recap header so FindRecapLeaks doesn't generate
// false positives on header glyphs.
func TestRecapBodyExtractor(t *testing.T) {
	sample := "📖 Some Reference\n(some attribution)\n\nthe actual body line"
	body := recapVerseBody(sample)
	if body != "the actual body line" {
		t.Errorf("recapVerseBody = %q, want %q", body, "the actual body line")
	}
}

// TestContextWindowTrimsToBounds is a small sanity check — the harness
// must report leaks even when they sit at the start or end of a string
// without panicking.
func TestContextWindowTrimsToBounds(t *testing.T) {
	w := contextWindow("hello world", 0, 5)
	if !strings.Contains(w, "hello") {
		t.Errorf("contextWindow at start dropped content: %q", w)
	}
	w = contextWindow("hello world", 6, 5)
	if !strings.Contains(w, "world") {
		t.Errorf("contextWindow at end dropped content: %q", w)
	}
}
