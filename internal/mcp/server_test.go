package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/MiaoDX/verse-driven/internal/packs"
)

// roundTrip drives a Server end-to-end: writes the supplied request lines
// to the server's stdin and returns each parsed response in order.
func roundTrip(t *testing.T, requests ...any) []map[string]any {
	t.Helper()
	srv := New(packs.All())
	var in bytes.Buffer
	enc := json.NewEncoder(&in)
	enc.SetEscapeHTML(false)
	for _, r := range requests {
		if err := enc.Encode(r); err != nil {
			t.Fatalf("encode req: %v", err)
		}
	}
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Serve(ctx, &in, &out); err != nil && err != io.EOF {
		t.Fatalf("Serve: %v", err)
	}
	var responses []map[string]any
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("parse response %q: %v", line, err)
		}
		responses = append(responses, m)
	}
	return responses
}

func TestInitializeReturnsServerInfo(t *testing.T) {
	resp := roundTrip(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	})
	if len(resp) != 1 {
		t.Fatalf("got %d responses, want 1", len(resp))
	}
	r := resp[0]
	if r["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v", r["jsonrpc"])
	}
	result, ok := r["result"].(map[string]any)
	if !ok {
		t.Fatalf("result missing or wrong type: %v", r["result"])
	}
	if result["protocolVersion"] == "" || result["protocolVersion"] == nil {
		t.Errorf("protocolVersion missing")
	}
	srvInfo := result["serverInfo"].(map[string]any)
	if srvInfo["name"] != ServerName {
		t.Errorf("serverInfo.name = %v, want %s", srvInfo["name"], ServerName)
	}
}

func TestNotificationsHaveNoResponse(t *testing.T) {
	// notifications/initialized has no `id` → no response.
	resp := roundTrip(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	if len(resp) != 0 {
		t.Errorf("notification produced %d responses, want 0: %v", len(resp), resp)
	}
}

func TestToolsListExposesAllFour(t *testing.T) {
	resp := roundTrip(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	if len(resp) != 1 {
		t.Fatalf("got %d responses, want 1", len(resp))
	}
	result := resp[0]["result"].(map[string]any)
	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools not a list: %v", result["tools"])
	}
	want := map[string]bool{"lookup": false, "search": false, "random": false, "list_traditions": false}
	for _, t0 := range tools {
		tm := t0.(map[string]any)
		want[tm["name"].(string)] = true
	}
	for name, present := range want {
		if !present {
			t.Errorf("tool %q missing", name)
		}
	}
}

func TestToolsCallLookup(t *testing.T) {
	resp := roundTrip(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "lookup",
			"arguments": map[string]any{"ref": "John 3:16"},
		},
	})
	r := resp[0]
	if e, ok := r["error"]; ok && e != nil {
		t.Fatalf("got JSON-RPC error: %v", e)
	}
	result := r["result"].(map[string]any)
	if result["isError"] == true {
		t.Fatalf("tool reported isError=true: %v", result)
	}
	content := result["content"].([]any)
	if len(content) == 0 {
		t.Fatal("content empty")
	}
	text := content[0].(map[string]any)["text"].(string)
	var v map[string]any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		t.Fatalf("verse text not JSON: %v\n%s", err, text)
	}
	if v["id"] != "bible.kjv.john.3.16" {
		t.Errorf("verse id = %v", v["id"])
	}
	if v["checksum_sha256"] == nil {
		t.Error("checksum missing")
	}
	if v["source"] == nil {
		t.Error("source missing")
	}
}

func TestToolsCallLookupErrorIsToolError(t *testing.T) {
	resp := roundTrip(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "lookup",
			"arguments": map[string]any{"ref": "John 99:99"},
		},
	})
	r := resp[0]
	if e, ok := r["error"]; ok && e != nil {
		t.Fatalf("expected tool-level error, got JSON-RPC error: %v", e)
	}
	result := r["result"].(map[string]any)
	if result["isError"] != true {
		t.Errorf("isError not set: %v", result)
	}
}

func TestToolsCallSearch(t *testing.T) {
	resp := roundTrip(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "search",
			"arguments": map[string]any{"query": "loved the world", "tradition": "bible", "limit": 3},
		},
	})
	result := resp[0]["result"].(map[string]any)
	text := result["content"].([]any)[0].(map[string]any)["text"].(string)
	var body map[string]any
	if err := json.Unmarshal([]byte(text), &body); err != nil {
		t.Fatalf("search payload not JSON: %v\n%s", err, text)
	}
	if body["count"].(float64) < 1 {
		t.Errorf("expected ≥1 match, got %v", body["count"])
	}
}

func TestToolsCallRandomDao(t *testing.T) {
	resp := roundTrip(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "random",
			"arguments": map[string]any{"tradition": "dao"},
		},
	})
	result := resp[0]["result"].(map[string]any)
	text := result["content"].([]any)[0].(map[string]any)["text"].(string)
	var v map[string]any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		t.Fatalf("random payload not JSON: %v\n%s", err, text)
	}
	if v["tradition"] != "dao" {
		t.Errorf("tradition filter ignored: %v", v["tradition"])
	}
}

func TestToolsCallListTraditions(t *testing.T) {
	resp := roundTrip(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      7,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_traditions",
			"arguments": map[string]any{},
		},
	})
	result := resp[0]["result"].(map[string]any)
	text := result["content"].([]any)[0].(map[string]any)["text"].(string)
	var body map[string]any
	if err := json.Unmarshal([]byte(text), &body); err != nil {
		t.Fatalf("list_traditions payload not JSON: %v", err)
	}
	traditions := body["traditions"].([]any)
	if len(traditions) < 3 {
		t.Errorf("expected ≥3 traditions, got %d", len(traditions))
	}
}

func TestUnknownMethodReturnsError(t *testing.T) {
	resp := roundTrip(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      8,
		"method":  "no/such/method",
	})
	r := resp[0]
	e, ok := r["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %v", r)
	}
	if e["code"].(float64) != codeMethodNotFound {
		t.Errorf("error.code = %v, want %d", e["code"], codeMethodNotFound)
	}
}

func TestParseErrorIsFlagged(t *testing.T) {
	srv := New(packs.All())
	in := bytes.NewBufferString("not json\n")
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := srv.Serve(ctx, in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var r map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &r); err != nil {
		t.Fatalf("parse output: %v\n%s", err, out.String())
	}
	e := r["error"].(map[string]any)
	if e["code"].(float64) != codeParseError {
		t.Errorf("code = %v, want %d", e["code"], codeParseError)
	}
}

// TestLookupLatency is a light sanity check for the issue's "<5ms p50"
// criterion. The bundled packs are in-memory and the lookup path is
// O(1), so this should pass with margin on any modern CI runner.
func TestLookupLatency(t *testing.T) {
	srv := New(packs.All())
	const trials = 200
	durations := make([]time.Duration, 0, trials)
	for i := 0; i < trials; i++ {
		t0 := time.Now()
		if _, err := srv.lookupByRef("John 3:16"); err != nil {
			t.Fatalf("lookup err: %v", err)
		}
		durations = append(durations, time.Since(t0))
	}
	// crude p50: sort ascending, take middle.
	for i := 1; i < len(durations); i++ {
		for j := i; j > 0 && durations[j-1] > durations[j]; j-- {
			durations[j-1], durations[j] = durations[j], durations[j-1]
		}
	}
	p50 := durations[trials/2]
	if p50 > 5*time.Millisecond {
		t.Errorf("p50 lookup latency %v exceeds 5ms budget", p50)
	}
}
