package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/JDVA0/blizzard/lsp/analyzer"
	"github.com/JDVA0/blizzard/lsp/protocol"
)

// Server represents the LSP server
type Server struct {
	documents map[string]*Document
	analyzer  *analyzer.Analyzer
	pending   []Notification
}

type Notification struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// Document represents an open document
type Document struct {
	URI         string
	Version     int
	Text        string
	Diagnostics []protocol.Diagnostic
	Analysis    *analyzer.ParseResult
}

// NewServer creates a new LSP server
func NewServer() *Server {
	return &Server{
		documents: make(map[string]*Document),
		analyzer:  analyzer.NewAnalyzer(),
	}
}

// Request represents an LSP request
type Request struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// Response represents an LSP response
type Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents an LSP error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// main entry point
func main() {
	server := NewServer()
	server.Run()
}

// Run starts the LSP server
func (s *Server) Run() {
	log.SetOutput(os.Stderr)
	log.Println("Blizzard LSP Server starting...")

	reader := bufio.NewReader(os.Stdin)

	for {
		message, err := readMessage(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error reading request: %v", err)
			continue
		}

		var req Request
		if err := json.Unmarshal(message, &req); err != nil {
			log.Printf("Error decoding request: %v", err)
			continue
		}

		log.Printf("Received request: %s with ID: %v", req.Method, req.ID)

		response := s.handleRequest(&req)
		if response == nil {
			log.Printf("Response is nil for method %s (notification)", req.Method)
		} else if err := writeMessage(os.Stdout, response); err != nil {
			log.Printf("Error encoding response: %v", err)
		} else {
			log.Printf("Sent response for method %s", req.Method)
		}

		for _, notification := range s.pending {
			if err := writeMessage(os.Stdout, notification); err != nil {
				log.Printf("Error publishing notification %s: %v", notification.Method, err)
			}
		}
		s.pending = s.pending[:0]
	}
}

func readMessage(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			contentLength, err = strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", err)
			}
		}
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	message := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, message); err != nil {
		return nil, err
	}
	return message, nil
}

func writeMessage(writer io.Writer, value interface{}) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n%s", len(payload), payload)
	return err
}

// handleRequest processes an LSP request
func (s *Server) handleRequest(req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		return s.handleInitialized(req)
	case "shutdown":
		return s.handleShutdown(req)
	case "exit":
		return s.handleExit(req)
	case "textDocument/didOpen":
		return s.handleDidOpen(req)
	case "textDocument/didChange":
		return s.handleDidChange(req)
	case "textDocument/didClose":
		return s.handleDidClose(req)
	case "textDocument/completion":
		return s.handleCompletion(req)
	case "textDocument/hover":
		return s.handleHover(req)
	case "textDocument/definition":
		return s.handleDefinition(req)
	case "textDocument/rename":
		return s.handleRename(req)
	case "textDocument/documentSymbol":
		return s.handleDocumentSymbol(req)
	default:
		return &Response{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *Request) *Response {
	var params protocol.InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    -32602,
				Message: fmt.Sprintf("Invalid params: %v", err),
			},
		}
	}

	result := protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    1, // Full sync
			},
			CompletionProvider: &protocol.CompletionOptions{
				ResolveProvider:   false,
				TriggerCharacters: []string{".", ":", " "},
			},
			HoverProvider:              true,
			DefinitionProvider:         true,
			ReferencesProvider:         false,
			DocumentFormattingProvider: false,
			RenameProvider:             true,
			DocumentSymbolProvider:     true,
		},
	}

	return &Response{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleInitialized handles the initialized notification
func (s *Server) handleInitialized(req *Request) *Response {
	// No response needed for notifications
	return nil
}

// handleShutdown handles the shutdown request
func (s *Server) handleShutdown(req *Request) *Response {
	return &Response{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  nil,
	}
}

// handleExit handles the exit notification
func (s *Server) handleExit(req *Request) *Response {
	os.Exit(0)
	return nil
}

// handleDidOpen handles the textDocument/didOpen notification
func (s *Server) handleDidOpen(req *Request) *Response {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		log.Printf("Error parsing didOpen params: %v", err)
		return nil
	}

	doc := &Document{
		URI:     params.TextDocument.URI,
		Version: params.TextDocument.Version,
		Text:    params.TextDocument.Text,
	}

	s.documents[params.TextDocument.URI] = doc

	doc.Diagnostics = s.analyzeDocument(doc)
	s.publishDiagnostics(doc)

	log.Printf("Opened document: %s (version %d)", params.TextDocument.URI, params.TextDocument.Version)

	return nil
}

func (s *Server) analyzeDocument(doc *Document) []protocol.Diagnostic {
	result, err := s.analyzer.Parse(doc.Text, doc.URI)
	doc.Analysis = result
	if err != nil {
		log.Printf("Error parsing document: %v", err)
		return nil
	}

	diagnostics := append([]analyzer.Diagnostic{}, result.Diagnostics...)
	if result.Program != nil {
		diagnostics = append(diagnostics, s.analyzer.Check(result.Program, doc.URI)...)
	}
	return s.convertDiagnostics(diagnostics)
}

func (s *Server) publishDiagnostics(doc *Document) {
	s.pending = append(s.pending, Notification{
		Jsonrpc: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: protocol.PublishDiagnosticsParams{
			URI:         doc.URI,
			Diagnostics: doc.Diagnostics,
		},
	})
}

// convertDiagnostics converts analyzer diagnostics to protocol diagnostics
func (s *Server) convertDiagnostics(diagnostics []analyzer.Diagnostic) []protocol.Diagnostic {
	protocolDiagnostics := make([]protocol.Diagnostic, len(diagnostics))

	for i, diag := range diagnostics {
		severity := 1 // Error
		if diag.Severity == "warning" {
			severity = 2 // Warning
		} else if diag.Severity == "info" {
			severity = 3 // Information
		}

		protocolDiagnostics[i] = protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      diag.Line,
					Character: diag.Column,
				},
				End: protocol.Position{
					Line:      diag.EndLine,
					Character: diag.EndColumn,
				},
			},
			Severity: severity,
			Source:   diag.Source,
			Message:  diag.Message,
		}
	}

	return protocolDiagnostics
}

// handleDidChange handles the textDocument/didChange notification
func (s *Server) handleDidChange(req *Request) *Response {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		log.Printf("Error parsing didChange params: %v", err)
		return nil
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		log.Printf("Document not found: %s", params.TextDocument.URI)
		return nil
	}

	// For now, we only support full document sync
	if len(params.ContentChanges) > 0 {
		doc.Text = params.ContentChanges[0].Text
		doc.Version = params.TextDocument.Version
	}

	doc.Diagnostics = s.analyzeDocument(doc)
	s.publishDiagnostics(doc)

	log.Printf("Changed document: %s (version %d)", params.TextDocument.URI, params.TextDocument.Version)

	return nil
}

// handleDidClose handles the textDocument/didClose notification
func (s *Server) handleDidClose(req *Request) *Response {
	var params protocol.DidCloseTextDocumentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		log.Printf("Error parsing didClose params: %v", err)
		return nil
	}

	delete(s.documents, params.TextDocument.URI)
	s.pending = append(s.pending, Notification{
		Jsonrpc: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: protocol.PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: []protocol.Diagnostic{},
		},
	})
	log.Printf("Closed document: %s", params.TextDocument.URI)

	return nil
}

// handleCompletion handles the textDocument/completion request
func (s *Server) handleCompletion(req *Request) *Response {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"position"`
		Context struct {
			TriggerKind      int    `json:"triggerKind"`
			TriggerCharacter string `json:"triggerCharacter"`
		} `json:"context"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		log.Printf("Error parsing completion params: %v", err)
		return &Response{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    -32602,
				Message: fmt.Sprintf("Invalid params: %v", err),
			},
		}
	}

	doc, exists := s.documents[params.TextDocument.URI]
	if !exists {
		log.Printf("Document not found: %s", params.TextDocument.URI)
		return &Response{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    -32602,
				Message: "Document not found",
			},
		}
	}

	log.Printf("Completion request for document %s at line %d, char %d", params.TextDocument.URI, params.Position.Line, params.Position.Character)

	items := s.getCompletionItems(doc, params.Position.Line, params.Position.Character)
	log.Printf("Generated %d completion items", len(items))

	if len(items) == 0 {
		log.Printf("No completion items generated")
		return &Response{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result: protocol.CompletionList{
				IsIncomplete: false,
				Items:        []protocol.CompletionItem{},
			},
		}
	}

	result := protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}

	log.Printf("Returning completion list with %d items", len(result.Items))

	return &Response{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func moduleMemberItems(module string) []protocol.CompletionItem {
	members := map[string][]string{
		"http":   {"get", "post", "put", "delete", "patch", "request"},
		"fs":     {"read", "write", "append", "exists", "remove", "mkdir", "list", "is_file", "is_dir", "stat", "copy"},
		"sys":    {"exec", "sh", "env", "set_env", "envs", "args", "cwd", "cd", "hostname", "pid", "platform", "arch", "exit", "now", "sleep"},
		"db":     {"open", "set", "get", "all", "save", "delete", "has"},
		"time":   {"unix", "unix_ms", "sleep", "iso", "now"},
		"json":   {"encode", "decode"},
		"cli":    {"box", "input", "confirm", "menu", "table"},
		"api":    {"route", "get", "post", "put", "delete", "patch", "serve"},
		"csv":    {"read", "write", "parse"},
		"env":    {"get", "set", "has", "all"},
		"task":   {"run", "schedule", "cancel"},
		"input":  {"read", "read_line", "read_secret"},
		"crypto": {"sha256", "md5", "hash"},
	}

	items := []protocol.CompletionItem{}
	for _, name := range members[module] {
		items = append(items, protocol.CompletionItem{
			Label:      name,
			Kind:       3,
			Detail:     module + "." + name,
			InsertText: name,
		})
	}
	return items
}

func wordAt(text string, line, character int) (string, protocol.Range) {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return "", protocol.Range{}
	}
	value := lines[line]
	if character > len(value) {
		character = len(value)
	}
	start := character
	for start > 0 && isWordByte(value[start-1]) {
		start--
	}
	end := character
	for end < len(value) && isWordByte(value[end]) {
		end++
	}
	if start == end {
		return "", protocol.Range{}
	}
	return value[start:end], protocol.Range{Start: protocol.Position{Line: line, Character: start}, End: protocol.Position{Line: line, Character: end}}
}

func isWordByte(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func positionParams(req *Request) (string, int, int, error) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"position"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return "", 0, 0, err
	}
	return params.TextDocument.URI, params.Position.Line, params.Position.Character, nil
}

func (s *Server) handleHover(req *Request) *Response {
	uri, line, character, err := positionParams(req)
	if err != nil {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Error: &Error{Code: -32602, Message: err.Error()}}
	}
	doc := s.documents[uri]
	if doc == nil {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Result: nil}
	}
	name, span := wordAt(doc.Text, line, character)
	if name == "" {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Result: nil}
	}
	description := map[string]string{
		"first":     "first(value) returns the first item of a list or string.",
		"last":      "last(value) returns the last item of a list or string.",
		"sum":       "sum(values) adds numeric list values.",
		"any":       "any(values) returns true when one value is truthy.",
		"all":       "all(values) returns true when every value is truthy.",
		"clamp":     "clamp(value, low, high) limits a number to a range.",
		"enumerate": "enumerate(values) returns index/value pairs.",
		"zip":       "zip(left, right) combines two lists.",
		"print":     "print(...) writes values to standard output.",
		"len":       "len(value) returns the length of a list, string, or dictionary.",
		"upper":     "upper(text) returns an uppercase string; also available as text.upper().",
		"lower":     "lower(text) returns a lowercase string; also available as text.lower().",
		"trim":      "trim(text) removes surrounding whitespace; also available as text.trim().",
		"split":     "split(text, separator) splits text; also available as text.split(separator).",
		"join":      "join(items, separator) joins strings; also available as items.join(separator).",
	}
	text := description[name]
	if text == "" {
		if kind := inferredDocumentType(doc.Text, name); kind != "" {
			text = fmt.Sprintf("%s: %s", name, kind)
		} else {
			text = "Blizzard symbol: " + name
		}
	}
	return &Response{Jsonrpc: "2.0", ID: req.ID, Result: protocol.Hover{Contents: text, Range: span}}
}

func inferredDocumentType(text, name string) string {
	pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(name) + `\s*=\s*(.*)$`)
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	value := strings.TrimSpace(match[1])
	switch {
	case strings.HasPrefix(value, "["):
		return "list"
	case strings.HasPrefix(value, "{"):
		return "dict"
	case strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "'"):
		return "str"
	case value == "true" || value == "false":
		return "bool"
	case regexp.MustCompile(`^-?[0-9]`).MatchString(value):
		return "number"
	default:
		return ""
	}
}

func (s *Server) handleDefinition(req *Request) *Response {
	uri, line, character, err := positionParams(req)
	if err != nil {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Error: &Error{Code: -32602, Message: err.Error()}}
	}
	doc := s.documents[uri]
	if doc == nil {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Result: nil}
	}
	name, _ := wordAt(doc.Text, line, character)
	if name == "" {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Result: nil}
	}
	// First, try AST-based lookup (more accurate).
	if doc.Analysis != nil {
		if loc := findSymbolLocation(doc.Analysis.Symbols, name, uri); loc != nil {
			return &Response{Jsonrpc: "2.0", ID: req.ID, Result: *loc}
		}
	}
	// Fallback: regex-based search.
	definition := regexp.MustCompile(`^\s*(?:(?:pub|priv)\s+)?(?:fn\s+)?` + regexp.QuoteMeta(name) + `\b`)
	for index, sourceLine := range strings.Split(doc.Text, "\n") {
		if definition.MatchString(sourceLine) {
			column := strings.Index(sourceLine, name)
			return &Response{Jsonrpc: "2.0", ID: req.ID, Result: protocol.Location{URI: uri, Range: protocol.Range{Start: protocol.Position{Line: index, Character: column}, End: protocol.Position{Line: index, Character: column + len(name)}}}}
		}
	}
	return &Response{Jsonrpc: "2.0", ID: req.ID, Result: nil}
}

func findSymbolLocation(symbols []analyzer.Symbol, name, uri string) *protocol.Location {
	for _, sym := range symbols {
		if sym.Name == name {
			return &protocol.Location{
				URI: uri,
				Range: protocol.Range{
					Start: protocol.Position{Line: sym.Line, Character: sym.Col},
					End:   protocol.Position{Line: sym.Line, Character: sym.EndCol},
				},
			}
		}
		if len(sym.Children) > 0 {
			if loc := findSymbolLocation(sym.Children, name, uri); loc != nil {
				return loc
			}
		}
	}
	return nil
}

func (s *Server) handleRename(req *Request) *Response {
	uri, line, character, err := positionParams(req)
	if err != nil {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Error: &Error{Code: -32602, Message: err.Error()}}
	}
	var params struct {
		NewName string `json:"newName"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil || params.NewName == "" {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Error: &Error{Code: -32602, Message: "rename expects newName"}}
	}
	doc := s.documents[uri]
	if doc == nil {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Result: nil}
	}
	name, _ := wordAt(doc.Text, line, character)
	if name == "" {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Result: nil}
	}
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
	var edits []protocol.TextEdit
	for lineNumber, sourceLine := range strings.Split(doc.Text, "\n") {
		for _, match := range pattern.FindAllStringIndex(sourceLine, -1) {
			edits = append(edits, protocol.TextEdit{Range: protocol.Range{Start: protocol.Position{Line: lineNumber, Character: match[0]}, End: protocol.Position{Line: lineNumber, Character: match[1]}}, NewText: params.NewName})
		}
	}
	return &Response{Jsonrpc: "2.0", ID: req.ID, Result: protocol.WorkspaceEdit{Changes: map[string][]protocol.TextEdit{uri: edits}}}
}

func detectModulePrefix(doc *Document, line, character int) string {
	lines := strings.Split(doc.Text, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	prefix := lines[line]
	if character > len(prefix) {
		character = len(prefix)
	}
	prefix = prefix[:character]
	re := regexp.MustCompile(`(^|[^A-Za-z0-9_])([A-Za-z_][A-Za-z0-9_]*)\.$`)
	matches := re.FindStringSubmatch(prefix)
	if len(matches) < 3 {
		return ""
	}
	return matches[2]
}

func detectDotAccessContext(doc *Document, line, character int) (baseVar string, typ string) {
	lines := strings.Split(doc.Text, "\n")
	if line < 0 || line >= len(lines) {
		return "", ""
	}
	prefix := lines[line]
	if character > len(prefix) {
		character = len(prefix)
	}
	match := regexp.MustCompile(`(?:^|[^A-Za-z0-9_])([A-Za-z_][A-Za-z0-9_]*)\.[A-Za-z_0-9]*$`).FindStringSubmatch(prefix[:character])
	if len(match) < 2 {
		return "", ""
	}
	baseVar = match[1]

	// AST-based inference is best; fall back to regex heuristic.
	if doc.Analysis != nil {
		if t := inferSymbolTypeFromAST(doc.Analysis.Symbols, baseVar); t != "" {
			return baseVar, t
		}
	}
	return baseVar, inferredDocumentType(doc.Text, baseVar)
}

func inferSymbolTypeFromAST(symbols []analyzer.Symbol, name string) string {
	for _, sym := range symbols {
		if sym.Name == name {
			switch sym.Kind {
			case analyzer.SymFunction:
				return "fn"
			case analyzer.SymModule:
				return "module"
			case analyzer.SymConstant, analyzer.SymVariable:
				if strings.Contains(sym.Detail, "str") {
					return "str"
				}
			}
		}
		if len(sym.Children) > 0 {
			if t := inferSymbolTypeFromAST(sym.Children, name); t != "" {
				return t
			}
		}
	}
	return ""
}

func stringMethodItems() []protocol.CompletionItem {
	methods := []struct {
		name   string
		detail string
		doc    string
	}{
		{"length", "str.length", "Returns the number of characters (string length)."},
		{"upper", "str.upper()", "Returns the string converted to uppercase."},
		{"lower", "str.lower()", "Returns the string converted to lowercase."},
		{"trim", "str.trim()", "Removes surrounding whitespace."},
		{"trim_left", "str.trim_left()", "Removes leading whitespace."},
		{"trim_right", "str.trim_right()", "Removes trailing whitespace."},
		{"split", "str.split(sep)", "Splits the string by a separator into a list."},
		{"replace", "str.replace(old, new)", "Replaces all occurrences of old with new."},
		{"contains", "str.contains(sub)", "Reports whether the substring is present."},
		{"starts_with", "str.starts_with(prefix)", "Reports whether the string starts with prefix."},
		{"ends_with", "str.ends_with(suffix)", "Reports whether the string ends with suffix."},
		{"count", "str.count(sub)", "Counts occurrences of a substring."},
		{"index_of", "str.index_of(sub)", "Returns the first index of the substring, or -1."},
		{"repeat", "str.repeat(n)", "Repeats the string n times."},
		{"join", "str.join(list)", "Joins list elements using the string as separator."},
	}
	items := make([]protocol.CompletionItem, 0, len(methods))
	for _, m := range methods {
		items = append(items, protocol.CompletionItem{
			Label:         m.name,
			Kind:          2, // Method
			Detail:        m.detail,
			Documentation: m.doc,
			InsertText:    m.name + "()",
			SortText:      "1_" + m.name,
		})
	}
	return items
}

func listMethodItems() []protocol.CompletionItem {
	methods := []struct {
		name   string
		detail string
		doc    string
	}{
		{"length", "list.length", "Returns the number of elements in the list."},
		{"append", "list.append(value)", "Adds an element to the end of the list."},
		{"first", "list.first", "Returns the first element or nil."},
		{"last", "list.last", "Returns the last element or nil."},
		{"take", "list.take(n)", "Returns the first n elements."},
		{"drop", "list.drop(n)", "Skips the first n elements."},
		{"reverse", "list.reverse()", "Returns a reversed copy."},
		{"sort", "list.sort()", "Returns a sorted copy."},
		{"map", "list.map(fn)", "Applies fn to every element."},
		{"filter", "list.filter(fn)", "Keeps elements where fn returns true."},
		{"fold", "list.fold(init, fn)", "Reduces the list using fn."},
		{"enumerate", "list.enumerate()", "Returns index/value pairs."},
		{"zip", "list.zip(other)", "Zips two lists into pairs."},
		{"join", "list.join(sep)", "Joins list elements into a string."},
		{"contains", "list.contains(value)", "Reports whether value is in the list."},
		{"has", "list.has(value)", "Alias for contains."},
	}
	items := make([]protocol.CompletionItem, 0, len(methods))
	for _, m := range methods {
		items = append(items, protocol.CompletionItem{
			Label:         m.name,
			Kind:          2, // Method
			Detail:        m.detail,
			Documentation: m.doc,
			InsertText:    m.name + "()",
			SortText:      "2_" + m.name,
		})
	}
	return items
}

func dictMethodItems() []protocol.CompletionItem {
	methods := []struct {
		name   string
		detail string
		doc    string
	}{
		{"length", "dict.length", "Returns the number of entries in the dictionary."},
		{"has", "dict.has(key)", "Reports whether the key exists."},
		{"keys", "dict.keys()", "Returns a list of keys."},
		{"values", "dict.values()", "Returns a list of values."},
	}
	items := make([]protocol.CompletionItem, 0, len(methods))
	for _, m := range methods {
		items = append(items, protocol.CompletionItem{
			Label:         m.name,
			Kind:          2, // Method
			Detail:        m.detail,
			Documentation: m.doc,
			InsertText:    m.name + "()",
			SortText:      "3_" + m.name,
		})
	}
	return items
}

func stringLiteralMethodAccess(doc *Document, line, character int) bool {
	lines := strings.Split(doc.Text, "\n")
	if line < 0 || line >= len(lines) {
		return false
	}
	prefix := lines[line]
	if character > len(prefix) {
		character = len(prefix)
	}
	match := regexp.MustCompile(`(?:"[^"\\]*"|'[^'\\]*')\.[A-Za-z_0-9]*$`).MatchString(prefix[:character])
	return match
}

func listLiteralMethodAccess(doc *Document, line, character int) bool {
	lines := strings.Split(doc.Text, "\n")
	if line < 0 || line >= len(lines) {
		return false
	}
	prefix := lines[line]
	if character > len(prefix) {
		character = len(prefix)
	}
	match := regexp.MustCompile(`\]\.[A-Za-z_0-9]*$`).MatchString(prefix[:character])
	return match
}

func dictLiteralMethodAccess(doc *Document, line, character int) bool {
	lines := strings.Split(doc.Text, "\n")
	if line < 0 || line >= len(lines) {
		return false
	}
	prefix := lines[line]
	if character > len(prefix) {
		character = len(prefix)
	}
	match := regexp.MustCompile(`\}\.[A-Za-z_0-9]*$`).MatchString(prefix[:character])
	return match
}

func completionWordPrefix(doc *Document, line, character int) string {
	lines := strings.Split(doc.Text, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	text := lines[line]
	if character > len(text) {
		character = len(text)
	}
	match := regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*$`).FindString(text[:character])
	return match
}

func documentNameItems(doc *Document) []protocol.CompletionItem {
	seen := map[string]protocol.CompletionItem{}
	collectFromAST(doc.Analysis, seen)
	// Fallback: regex scan for symbols AST might have missed.
	fallbackRegex(doc.Text, seen)
	items := make([]protocol.CompletionItem, 0, len(seen))
	for _, it := range seen {
		items = append(items, it)
	}
	sort.Slice(items, func(a, b int) bool { return items[a].Label < items[b].Label })
	return items
}

func collectFromAST(result *analyzer.ParseResult, out map[string]protocol.CompletionItem) {
	if result == nil {
		return
	}
	walk(result.Symbols, out)
}

func walk(symbols []analyzer.Symbol, out map[string]protocol.CompletionItem) {
	for _, sym := range symbols {
		if _, exists := out[sym.Name]; !exists {
			kind := 6 // Variable (default fallback)
			switch sym.Kind {
			case analyzer.SymFunction:
				kind = 3 // Function
			case analyzer.SymConstant:
				kind = 21 // Constant
			case analyzer.SymModule:
				kind = 9 // Module
			case analyzer.SymVariable:
				kind = 6
			case analyzer.SymClass, analyzer.SymStruct:
				kind = 5 // Class/Struct
			}
			insert := sym.Name
			if sym.Kind == analyzer.SymFunction {
				insert += "()"
			}
			out[sym.Name] = protocol.CompletionItem{
				Label:      sym.Name,
				Kind:       kind,
				Detail:     sym.Detail,
				InsertText: insert,
			}
		}
		if len(sym.Children) > 0 {
			walk(sym.Children, out)
		}
	}
}

func fallbackRegex(text string, out map[string]protocol.CompletionItem) {
	pattern := regexp.MustCompile(`(?m)\b(?:fn|const)?\s*([A-Za-z_][A-Za-z0-9_]*)\s*=`)
	for _, match := range pattern.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if _, exists := out[name]; !exists {
			out[name] = protocol.CompletionItem{Label: name, Kind: 6, Detail: "document symbol", InsertText: name}
		}
	}
	for _, match := range regexp.MustCompile(`(?m)^\s*(?:pub\s+|priv\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)`).FindAllStringSubmatch(text, -1) {
		name := match[1]
		if _, exists := out[name]; !exists {
			out[name] = protocol.CompletionItem{Label: name, Kind: 3, Detail: "document function", InsertText: name + "()"}
		}
	}
}

func filterCompletionItems(items []protocol.CompletionItem, prefix string) []protocol.CompletionItem {
	if prefix == "" {
		return items
	}
	filtered := make([]protocol.CompletionItem, 0, len(items))
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item.Label), strings.ToLower(prefix)) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// handleDocumentSymbol returns hierarchical document symbols using the AST.
func (s *Server) handleDocumentSymbol(req *Request) *Response {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Error: &Error{Code: -32602, Message: err.Error()}}
	}
	doc := s.documents[params.TextDocument.URI]
	if doc == nil || doc.Analysis == nil {
		return &Response{Jsonrpc: "2.0", ID: req.ID, Result: []protocol.DocumentSymbol{}}
	}
	out := make([]protocol.DocumentSymbol, 0, len(doc.Analysis.Symbols))
	for _, sym := range doc.Analysis.Symbols {
		out = append(out, toDocumentSymbol(sym))
	}
	return &Response{Jsonrpc: "2.0", ID: req.ID, Result: out}
}

func toDocumentSymbol(sym analyzer.Symbol) protocol.DocumentSymbol {
	children := make([]protocol.DocumentSymbol, 0, len(sym.Children))
	for _, c := range sym.Children {
		children = append(children, toDocumentSymbol(c))
	}
	sel := protocol.Range{
		Start: protocol.Position{Line: sym.Line, Character: sym.Col},
		End:   protocol.Position{Line: sym.Line, Character: sym.EndCol},
	}
	rng := protocol.Range{
		Start: protocol.Position{Line: sym.Line, Character: sym.Col},
		End:   protocol.Position{Line: sym.EndLine, Character: max(sym.EndCol, sym.Col+1)},
	}
	return protocol.DocumentSymbol{
		Name:           sym.Name,
		Detail:         sym.Detail,
		Kind:           sym.Kind,
		Range:          rng,
		SelectionRange: sel,
		Children:       children,
	}
}

// getCompletionItems returns completion items based on the document context
func (s *Server) getCompletionItems(doc *Document, line, character int) []protocol.CompletionItem {
	prefix := completionWordPrefix(doc, line, character)
	if moduleName := detectModulePrefix(doc, line, character); moduleName != "" {
		if items := moduleMemberItems(moduleName); len(items) > 0 {
			return filterCompletionItems(items, prefix)
		}
	}

	// 1) Literal dot access detection: "abc". | [1,2]. | {k:v}.
	if stringLiteralMethodAccess(doc, line, character) {
		return filterCompletionItems(stringMethodItems(), prefix)
	}
	if listLiteralMethodAccess(doc, line, character) {
		return filterCompletionItems(listMethodItems(), prefix)
	}
	if dictLiteralMethodAccess(doc, line, character) {
		return filterCompletionItems(dictMethodItems(), prefix)
	}

	// 2) Variable dot access detection by inferred type.
	if _, typ := detectDotAccessContext(doc, line, character); typ != "" {
		switch typ {
		case "str":
			return filterCompletionItems(stringMethodItems(), prefix)
		case "list":
			return filterCompletionItems(listMethodItems(), prefix)
		case "dict":
			return filterCompletionItems(dictMethodItems(), prefix)
		}
	}

	items := []protocol.CompletionItem{}

	// Add Blizzard keywords (sorted first)
	keywords := []string{
		"import", "let", "pub", "priv", "const", "fn", "return",
		"if", "elif", "else", "for", "while", "break", "continue",
		"try", "catch", "always", "match", "case", "where", "with",
		"not", "in", "and", "or", "as",
	}

	for _, keyword := range keywords {
		items = append(items, protocol.CompletionItem{
			Label:      keyword,
			Kind:       14, // Keyword
			Detail:     "keyword",
			InsertText: keyword,
			SortText:   "0_k_" + keyword,
		})
	}

	// Add standard library modules
	stdModules := []string{
		"sys", "fs", "cli", "api", "http", "db", "time", "json",
		"crypto", "task", "env", "csv", "input",
	}

	for _, module := range stdModules {
		items = append(items, protocol.CompletionItem{
			Label:      module,
			Kind:       9, // Module
			Detail:     "standard module",
			InsertText: module,
			SortText:   "0_m_" + module,
		})
	}

	// Add built-in functions (category: string)
	strBuiltins := []string{
		"upper", "lower", "trim", "trim_left", "trim_right", "split", "join", "replace", "contains", "starts_with", "ends_with", "count", "index_of", "repeat",
	}
	for _, builtin := range strBuiltins {
		items = append(items, protocol.CompletionItem{
			Label:         builtin,
			Kind:          3, // Function
			Detail:        "str built-in",
			Documentation: fmt.Sprintf("%s(value, ...) — string helper; also available as value.%s()", builtin, builtin),
			InsertText:    builtin + "()",
			SortText:      "4_s_" + builtin,
		})
	}

	// Add built-in functions (category: collections/list)
	listBuiltins := []string{
		"len", "has", "keys", "values", "append", "first", "last", "take", "drop",
		"enumerate", "zip", "reverse", "sort", "map", "filter", "fold",
	}
	for _, builtin := range listBuiltins {
		items = append(items, protocol.CompletionItem{
			Label:         builtin,
			Kind:          3, // Function
			Detail:        "collection built-in",
			Documentation: fmt.Sprintf("%s(collection, ...) — list/dict helper", builtin),
			InsertText:    builtin + "()",
			SortText:      "5_c_" + builtin,
		})
	}

	// Add built-in functions (category: math / general)
	genBuiltins := []string{
		"print", "str", "int", "flt", "float", "bool", "list", "dict",
		"sum", "any", "all", "clamp", "range", "type", "min", "max",
		"abs", "floor", "ceil", "round", "fail", "exit", "assert",
	}
	for _, builtin := range genBuiltins {
		items = append(items, protocol.CompletionItem{
			Label:      builtin,
			Kind:       3, // Function
			Detail:     "built-in function",
			InsertText: builtin + "()",
			SortText:   "6_g_" + builtin,
		})
	}

	// Add constants
	constants := []string{"true", "false", "nil"}
	for _, constant := range constants {
		items = append(items, protocol.CompletionItem{
			Label:      constant,
			Kind:       12, // Constant
			Detail:     "constant",
			InsertText: constant,
			SortText:   "7_v_" + constant,
		})
	}
	items = append(items, documentNameItems(doc)...)
	return filterCompletionItems(items, prefix)
}
