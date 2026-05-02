package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// TestBenchFixturesPresent confirms every fixture referenced by
// docs/benchmarks/tasks.json is actually vendored under
// docs/benchmarks/fixtures/<id>/ with at least the three required
// files: a README, a source file, and a pytest acceptance file.
//
// This is the regression net for the issue #8 acceptance criterion
// "task pack of 10 representative coding tasks". Adding a task to
// tasks.json without vendoring the fixture (or vice versa) fails CI.
func TestBenchFixturesPresent(t *testing.T) {
	path := repoFile(t, "docs/benchmarks/tasks.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var pack taskPack
	if err := json.Unmarshal(body, &pack); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	for _, task := range pack.Tasks {
		dir := repoFile(t, filepath.Join("docs/benchmarks", task.FixtureDir))
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("task %s: fixture dir missing: %v", task.ID, err)
			continue
		}
		readme := filepath.Join(dir, "README.md")
		if _, err := os.Stat(readme); err != nil {
			t.Errorf("task %s: README.md missing", task.ID)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Errorf("task %s: read dir: %v", task.ID, err)
			continue
		}
		var hasSource, hasTest bool
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".py") {
				continue
			}
			if strings.HasPrefix(name, "test_") {
				hasTest = true
			} else {
				hasSource = true
			}
		}
		if !hasSource {
			t.Errorf("task %s: no non-test .py source file in fixture", task.ID)
		}
		if !hasTest {
			t.Errorf("task %s: no test_*.py acceptance file in fixture", task.ID)
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
