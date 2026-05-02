package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// taskPack mirrors docs/benchmarks/tasks.json. Only the fields the
// regression test checks are decoded.
type taskPack struct {
	SchemaVersion int `json:"schema_version"`
	Modes         []struct {
		ID          string `json:"id"`
		Description string `json:"description"`
	} `json:"modes"`
	Tasks []struct {
		ID          string `json:"id"`
		Type        string `json:"type"`
		Name        string `json:"name"`
		Description string `json:"description"`
		FixtureDir  string `json:"fixture_dir"`
		Acceptance  string `json:"acceptance"`
		LinesOfCode int    `json:"lines_of_code"`
	} `json:"tasks"`
}

// TestBenchPackStructure locks in the contract docs/benchmarks/tasks.json
// must satisfy. It is the regression net for the issue #8 acceptance
// criterion "Task pack of 10 representative coding tasks (mix of
// refactor / bug fix / new feature)" and "Four modes: baseline,
// preview-only, inject-once, recap-only".
//
// The test is deliberately structural — it does NOT try to dispatch
// the agent or grade tasks (that requires a live API key and lives in
// scripts/bench_runner.py). What it locks in is: the pack exists, has
// the right shape, the right counts, the modes the runner expects,
// and one fixture vendored as a working template.
func TestBenchPackStructure(t *testing.T) {
	path := repoFile(t, "docs/benchmarks/tasks.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var pack taskPack
	if err := json.Unmarshal(body, &pack); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if pack.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", pack.SchemaVersion)
	}
	if got := len(pack.Tasks); got != 10 {
		t.Errorf("want 10 tasks, got %d", got)
	}
	wantModes := map[string]bool{
		"baseline": false, "preview-only": false,
		"inject-once": false, "recap-only": false,
	}
	for _, m := range pack.Modes {
		if _, ok := wantModes[m.ID]; ok {
			wantModes[m.ID] = true
		}
		if m.Description == "" {
			t.Errorf("mode %s missing description", m.ID)
		}
	}
	for id, seen := range wantModes {
		if !seen {
			t.Errorf("missing required mode: %s", id)
		}
	}

	wantTypes := map[string]int{"refactor": 0, "bugfix": 0, "feature": 0}
	seenIDs := map[string]bool{}
	for _, task := range pack.Tasks {
		if task.ID == "" {
			t.Errorf("task missing id: %+v", task)
			continue
		}
		if seenIDs[task.ID] {
			t.Errorf("duplicate task id: %s", task.ID)
		}
		seenIDs[task.ID] = true
		if _, ok := wantTypes[task.Type]; !ok {
			t.Errorf("task %s: type %q not in {refactor,bugfix,feature}", task.ID, task.Type)
		}
		wantTypes[task.Type]++
		if task.Description == "" {
			t.Errorf("task %s: empty description", task.ID)
		}
		if task.FixtureDir == "" {
			t.Errorf("task %s: empty fixture_dir", task.ID)
		}
		if task.Acceptance == "" {
			t.Errorf("task %s: empty acceptance command", task.ID)
		}
	}
	for typ, n := range wantTypes {
		if n == 0 {
			t.Errorf("task pack must contain at least one %s task", typ)
		}
	}
}

// TestBenchTemplateFixtureExists checks that the one vendored fixture
// referenced by the pack is actually present and has the expected
// shape (source file + test file). This locks in the convention so
// future fixture additions follow it.
func TestBenchTemplateFixtureExists(t *testing.T) {
	dir := repoFile(t, "docs/benchmarks/fixtures/bugfix-off-by-one")
	for _, f := range []string{"windows.py", "test_windows.py", "README.md"} {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("template fixture missing %s: %v", f, err)
		}
	}
}

// TestBenchReportTemplateExists confirms the publication template
// exists so the issue #8 release artifact has a known location.
func TestBenchReportTemplateExists(t *testing.T) {
	path := repoFile(t, "docs/benchmarks/v0.1.md")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("report template missing: %v", err)
	}
}

// repoFile resolves a repo-root-relative path. Tests run with cwd =
// the package directory, so we walk up to find the go.mod.
func repoFile(t *testing.T, rel string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, rel)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not find go.mod walking up from %s", file)
	return ""
}
