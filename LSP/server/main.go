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

	"github.com/JDVA0/snow/lsp/analyzer"
	"github.com/JDVA0/snow/lsp/protocol"
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
	Result  interface{} `json:"result,omitempty"`
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
	log.Println("Snow LSP Server starting...")

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
	}
	text := description[name]
	if text == "" {
		if kind := inferredDocumentType(doc.Text, name); kind != "" {
			text = fmt.Sprintf("%s: %s", name, kind)
		} else {
			text = "Snow symbol: " + name
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
	definition := regexp.MustCompile(`^\s*(?:(?:pub|priv)\s+)?(?:fn\s+)?` + regexp.QuoteMeta(name) + `\b`)
	for index, sourceLine := range strings.Split(doc.Text, "\n") {
		if definition.MatchString(sourceLine) {
			column := strings.Index(sourceLine, name)
			return &Response{Jsonrpc: "2.0", ID: req.ID, Result: protocol.Location{URI: uri, Range: protocol.Range{Start: protocol.Position{Line: index, Character: column}, End: protocol.Position{Line: index, Character: column + len(name)}}}}
		}
	}
	return &Response{Jsonrpc: "2.0", ID: req.ID, Result: nil}
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
	seen := map[string]bool{}
	pattern := regexp.MustCompile(`(?m)\b(?:fn|const)?\s*([A-Za-z_][A-Za-z0-9_]*)\s*=`)
	for _, match := range pattern.FindAllStringSubmatch(doc.Text, -1) {
		if len(match) > 1 {
			seen[match[1]] = true
		}
	}
	for _, match := range regexp.MustCompile(`(?m)^\s*(?:pub\s+|priv\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)`).FindAllStringSubmatch(doc.Text, -1) {
		if len(match) > 1 {
			seen[match[1]] = true
		}
	}
	items := make([]protocol.CompletionItem, 0, len(seen))
	for name := range seen {
		items = append(items, protocol.CompletionItem{Label: name, Kind: 6, Detail: "document symbol", InsertText: name})
	}
	sort.Slice(items, func(a, b int) bool { return items[a].Label < items[b].Label })
	return items
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

// getCompletionItems returns completion items based on the document context
func (s *Server) getCompletionItems(doc *Document, line, character int) []protocol.CompletionItem {
	prefix := completionWordPrefix(doc, line, character)
	if moduleName := detectModulePrefix(doc, line, character); moduleName != "" {
		if items := moduleMemberItems(moduleName); len(items) > 0 {
			return filterCompletionItems(items, prefix)
		}
	}

	items := []protocol.CompletionItem{}

	// Add Snow keywords
	keywords := []string{
		"using", "import", "pub", "priv", "const", "fn", "return",
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
		})
	}

	// Add built-in functions
	builtins := []string{
		"print", "len", "str", "int", "flt", "float", "bool", "list", "dict",
		"upper", "lower", "trim", "split", "join", "replace", "contains", "has",
		"keys", "values", "append", "first", "last", "take", "drop", "sum", "any", "all", "clamp",
		"enumerate", "zip", "reverse", "sort", "map", "filter", "fold", "range",
		"type", "min", "max", "abs", "floor", "ceil", "round", "fail", "exit", "assert",
	}

	for _, builtin := range builtins {
		items = append(items, protocol.CompletionItem{
			Label:      builtin,
			Kind:       3, // Function
			Detail:     "built-in function",
			InsertText: builtin + "()",
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
		})
	}
	items = append(items, documentNameItems(doc)...)
	return filterCompletionItems(items, prefix)
}
