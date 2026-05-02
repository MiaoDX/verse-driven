// Package mcp implements a minimal stdio MCP server for the
// scripture-mcp binary. It exposes four tools: lookup, search, random,
// and list_traditions. The wire format is newline-delimited JSON-RPC 2.0
// over stdio (the MCP stdio transport).
//
// We deliberately do not depend on a third-party MCP SDK: the surface we
// need is small (3 methods, 4 tools, no resources/prompts), and pulling
// in an SDK doubles the dependency footprint of a project that today has
// zero non-stdlib deps.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/MiaoDX/verse-driven/internal/packs"
	"github.com/MiaoDX/verse-driven/internal/resolver"
	"github.com/MiaoDX/verse-driven/internal/schema"
)

const (
	// ProtocolVersion is the MCP protocol revision this server speaks back
	// to clients during the initialize handshake. Clients that report a
	// different version still get a response — they are expected to
	// negotiate based on what we return rather than refuse outright.
	ProtocolVersion = "2024-11-05"

	ServerName    = "scripture-mcp"
	ServerVersion = "0.1.0"
)

// Lookuper is the slice of the verse registry the server depends on. The
// concrete *packs.Registry satisfies this; a fake satisfies it in tests.
type Lookuper interface {
	Lookup(id string) (schema.Verse, bool)
	Pack(name packs.PackName) *packs.Pack
	Names() []packs.PackName
}

// Server is one stdio MCP session. Construct via New, then call Serve.
type Server struct {
	registry Lookuper
	rng      *rand.Rand
}

// New builds a Server backed by the given verse registry. Pass packs.All()
// for production; pass a stub in tests.
func New(reg Lookuper) *Server {
	return &Server{
		registry: reg,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Serve reads NDJSON from r, dispatches each request, and writes the
// reply to w. It returns when r returns io.EOF or ctx is canceled.
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	br := bufio.NewReaderSize(r, 1<<16)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			if resp, ok := s.handleLine(line); ok {
				if _, werr := w.Write(append(resp, '\n')); werr != nil {
					return werr
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSON-RPC error codes. -32600..-32603 are reserved by the spec; we use
// the standard ones plus a method-not-found for unknown tools.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// handleLine parses one NDJSON line and returns (response, true) when a
// reply must be written, or (_, false) for notifications and blank lines.
func (s *Server) handleLine(line []byte) ([]byte, bool) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return nil, false
	}
	var req rpcRequest
	if err := json.Unmarshal([]byte(trimmed), &req); err != nil {
		return marshalResp(rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: codeParseError, Message: "parse error: " + err.Error()},
		}), true
	}
	// Notifications have no id; we never reply to them.
	isNotification := len(req.ID) == 0

	resp := s.dispatch(req)
	if isNotification {
		return nil, false
	}
	resp.JSONRPC = "2.0"
	resp.ID = req.ID
	return marshalResp(resp), true
}

func marshalResp(r rpcResponse) []byte {
	b, err := json.Marshal(r)
	if err != nil {
		// Marshalling our own struct should never fail; if it does, emit a
		// hard-coded fallback so the client at least sees a JSON-RPC error.
		return []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"marshal failed"}}`)
	}
	return b
}

func (s *Server) dispatch(req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return rpcResponse{Result: s.handleInitialize()}
	case "initialized", "notifications/initialized":
		return rpcResponse{} // notification — caller drops it
	case "ping":
		return rpcResponse{Result: map[string]any{}}
	case "tools/list":
		return rpcResponse{Result: s.handleToolsList()}
	case "tools/call":
		return s.handleToolsCall(req.Params)
	case "shutdown":
		return rpcResponse{Result: map[string]any{}}
	default:
		return rpcResponse{Error: &rpcError{Code: codeMethodNotFound, Message: "unknown method: " + req.Method}}
	}
}

func (s *Server) handleInitialize() any {
	return map[string]any{
		"protocolVersion": ProtocolVersion,
		"serverInfo": map[string]any{
			"name":    ServerName,
			"version": ServerVersion,
		},
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
	}
}

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func (s *Server) handleToolsList() any {
	return map[string]any{"tools": toolDefinitions()}
}

func toolDefinitions() []toolDef {
	return []toolDef{
		{
			Name:        "lookup",
			Description: "Look up a single verse by free-form reference (e.g. \"John 3:16\", \"道德经第十一章\", \"Quran 2:255\"). Returns the canonical verse with checksum and source metadata.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ref": map[string]any{
						"type":        "string",
						"description": "Free-form scripture reference.",
					},
				},
				"required": []string{"ref"},
			},
		},
		{
			Name:        "search",
			Description: "Search for verses whose text contains the query substring. Optional `tradition` filter (bible|dao|sutra|quran). Returns up to `limit` verses (default 10).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":     map[string]any{"type": "string", "description": "Substring to search for."},
					"tradition": map[string]any{"type": "string", "description": "Restrict to one tradition: bible|dao|sutra|quran."},
					"limit":     map[string]any{"type": "integer", "description": "Max results (default 10, max 50)."},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "random",
			Description: "Return one randomly-selected verse. Optional `tradition` filter (bible|dao|sutra|quran).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tradition": map[string]any{"type": "string", "description": "Restrict to one tradition: bible|dao|sutra|quran."},
				},
			},
		},
		{
			Name:        "list_traditions",
			Description: "List the bundled traditions, with verse counts and inclusion mode.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func (s *Server) handleToolsCall(raw json.RawMessage) rpcResponse {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return rpcResponse{Error: &rpcError{Code: codeInvalidParams, Message: "invalid params: " + err.Error()}}
	}
	switch p.Name {
	case "lookup":
		return s.toolLookup(p.Arguments)
	case "search":
		return s.toolSearch(p.Arguments)
	case "random":
		return s.toolRandom(p.Arguments)
	case "list_traditions":
		return s.toolListTraditions()
	default:
		return rpcResponse{Error: &rpcError{Code: codeMethodNotFound, Message: "unknown tool: " + p.Name}}
	}
}

func (s *Server) toolLookup(raw json.RawMessage) rpcResponse {
	var args struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return rpcResponse{Error: &rpcError{Code: codeInvalidParams, Message: "invalid arguments: " + err.Error()}}
	}
	if strings.TrimSpace(args.Ref) == "" {
		return rpcResponse{Error: &rpcError{Code: codeInvalidParams, Message: "ref is required"}}
	}
	v, err := s.lookupByRef(args.Ref)
	if err != nil {
		return errorToolResult(err)
	}
	return verseToolResult(v)
}

func (s *Server) lookupByRef(ref string) (schema.Verse, error) {
	r, err := resolver.Resolve(ref)
	if err != nil {
		return schema.Verse{}, err
	}
	id, err := packs.ReferenceID(r)
	if err != nil {
		return schema.Verse{}, err
	}
	v, ok := s.registry.Lookup(id)
	if !ok {
		if r.Tradition == resolver.TraditionSutra || r.Tradition == resolver.TraditionQuran {
			return schema.Verse{}, fmt.Errorf("%w: %s", packs.ErrNotBundled, r.Tradition)
		}
		return schema.Verse{}, fmt.Errorf("verse not found: %s", id)
	}
	return v, nil
}

func (s *Server) toolSearch(raw json.RawMessage) rpcResponse {
	var args struct {
		Query     string `json:"query"`
		Tradition string `json:"tradition"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return rpcResponse{Error: &rpcError{Code: codeInvalidParams, Message: "invalid arguments: " + err.Error()}}
	}
	q := strings.TrimSpace(args.Query)
	if q == "" {
		return rpcResponse{Error: &rpcError{Code: codeInvalidParams, Message: "query is required"}}
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	matches := s.searchVerses(q, args.Tradition, limit)
	return versesToolResult(matches)
}

func (s *Server) searchVerses(query, tradition string, limit int) []schema.Verse {
	q := strings.ToLower(query)
	var out []schema.Verse
	for _, name := range s.registry.Names() {
		p := s.registry.Pack(name)
		if tradition != "" && p.Meta.Tradition != tradition {
			continue
		}
		for _, v := range p.Verses() {
			if strings.Contains(strings.ToLower(v.Text), q) {
				out = append(out, v)
				if len(out) >= limit {
					return out
				}
			}
		}
	}
	return out
}

func (s *Server) toolRandom(raw json.RawMessage) rpcResponse {
	var args struct {
		Tradition string `json:"tradition"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return rpcResponse{Error: &rpcError{Code: codeInvalidParams, Message: "invalid arguments: " + err.Error()}}
		}
	}
	v, ok := s.pickRandom(args.Tradition)
	if !ok {
		return errorToolResult(fmt.Errorf("no bundled verses available%s", traditionSuffix(args.Tradition)))
	}
	return verseToolResult(v)
}

func traditionSuffix(t string) string {
	if t == "" {
		return ""
	}
	return " for tradition " + t
}

func (s *Server) pickRandom(tradition string) (schema.Verse, bool) {
	var pool []schema.Verse
	for _, name := range s.registry.Names() {
		p := s.registry.Pack(name)
		if p.Meta.InclusionMode != "" && p.Meta.InclusionMode != "bundled" {
			continue
		}
		if tradition != "" && p.Meta.Tradition != tradition {
			continue
		}
		pool = append(pool, p.Verses()...)
	}
	if len(pool) == 0 {
		return schema.Verse{}, false
	}
	return pool[s.rng.Intn(len(pool))], true
}

func (s *Server) toolListTraditions() rpcResponse {
	type entry struct {
		Tradition     string `json:"tradition"`
		Work          string `json:"work"`
		Lang          string `json:"lang"`
		VerseCount    int    `json:"verse_count"`
		InclusionMode string `json:"inclusion_mode"`
		Source        string `json:"source"`
	}
	var entries []entry
	for _, name := range s.registry.Names() {
		p := s.registry.Pack(name)
		entries = append(entries, entry{
			Tradition:     p.Meta.Tradition,
			Work:          p.Meta.Work,
			Lang:          p.Meta.Lang,
			VerseCount:    len(p.Verses()),
			InclusionMode: p.Meta.InclusionMode,
			Source:        p.Meta.Provider,
		})
	}
	body, _ := json.MarshalIndent(map[string]any{"traditions": entries}, "", "  ")
	return rpcResponse{Result: textContent(string(body))}
}

func verseToolResult(v schema.Verse) rpcResponse {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return rpcResponse{Error: &rpcError{Code: codeInternalError, Message: err.Error()}}
	}
	return rpcResponse{Result: textContent(string(body))}
}

func versesToolResult(vs []schema.Verse) rpcResponse {
	body, err := json.MarshalIndent(map[string]any{"verses": vs, "count": len(vs)}, "", "  ")
	if err != nil {
		return rpcResponse{Error: &rpcError{Code: codeInternalError, Message: err.Error()}}
	}
	return rpcResponse{Result: textContent(string(body))}
}

// errorToolResult returns an MCP tool error using the result-with-isError
// convention: tool-level failures land as a tool result with isError=true,
// not as a JSON-RPC error. This matches how MCP clients distinguish "the
// transport failed" from "the tool ran and reported a problem".
func errorToolResult(err error) rpcResponse {
	return rpcResponse{Result: map[string]any{
		"content": []map[string]any{{"type": "text", "text": "error: " + err.Error()}},
		"isError": true,
	}}
}

func textContent(s string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": s}},
	}
}

