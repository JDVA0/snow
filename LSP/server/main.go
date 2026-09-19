package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
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
			HoverProvider:              false,
			DefinitionProvider:         false,
			ReferencesProvider:         false,
			DocumentFormattingProvider: false,
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

// getCompletionItems returns completion items based on the document context
func (s *Server) getCompletionItems(doc *Document, line, character int) []protocol.CompletionItem {
	if moduleName := detectModulePrefix(doc, line, character); moduleName != "" {
		if items := moduleMemberItems(moduleName); len(items) > 0 {
			return items
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
		"print", "len", "str", "int", "float", "bool", "list", "dict",
		"upper", "lower", "split", "join", "enumerate", "zip", "range",
		"type", "fail", "exit",
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

	return items
}
