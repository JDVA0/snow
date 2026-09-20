package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JDVA0/blizzard"
)

// SymbolKind mirrors LSP SymbolKind numeric values.
const (
	SymFile          = 1
	SymModule        = 2
	SymNamespace     = 3
	SymPackage       = 4
	SymClass         = 5
	SymMethod        = 6
	SymProperty      = 7
	SymField         = 8
	SymConstructor   = 9
	SymEnum          = 10
	SymInterface     = 11
	SymFunction      = 12
	SymVariable      = 13
	SymConstant      = 14
	SymString        = 15
	SymNumber        = 16
	SymBoolean       = 17
	SymArray         = 18
	SymObject        = 19
	SymKey           = 20
	SymNull          = 21
	SymEnumMember    = 22
	SymStruct        = 23
	SymEvent         = 24
	SymOperator      = 25
	SymTypeParameter = 26
)

// Symbol represents an extracted AST symbol.
type Symbol struct {
	Name       string
	Kind       int
	Detail     string
	Line       int
	Col        int
	EndLine    int
	EndCol     int
	Visibility string
	Children   []Symbol
}

// Analyzer wraps the Blizzard parser for LSP usage
type Analyzer struct{}

// NewAnalyzer creates a new analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// ParseResult represents the result of parsing
type ParseResult struct {
	Program     *blizzard.Program
	Diagnostics []Diagnostic
	Symbols     []Symbol
}

// Diagnostic represents a diagnostic message
type Diagnostic struct {
	Line      int
	Column    int
	EndLine   int
	EndColumn int
	Severity  string // "error", "warning", "info"
	Source    string
	Message   string
}

var positionPattern = regexp.MustCompile(`:(\d+):(\d+):\s*(.*)$`)

// Parse parses Blizzard source code and returns diagnostics
func (a *Analyzer) Parse(source, filename string) (*ParseResult, error) {
	program, err := blizzard.Parse(source, filename)

	result := &ParseResult{
		Program:     program,
		Diagnostics: []Diagnostic{},
		Symbols:     []Symbol{},
	}

	if err != nil {
		diag := a.errorToDiagnostic(err, source)
		result.Diagnostics = append(result.Diagnostics, diag)
	}

	if program != nil {
		result.Symbols = ExtractSymbols(program)
	}

	return result, nil
}

// ExtractSymbols walks the AST and extracts top-level and nested symbols.
func ExtractSymbols(program *blizzard.Program) []Symbol {
	symbols := []Symbol{}
	for _, stmt := range program.Stmts {
		if s := extractStmtSymbol(stmt); s != nil {
			symbols = append(symbols, *s)
		}
	}
	return symbols
}

func pos0(p blizzard.Pos) (int, int) { return p.Line - 1, p.Col - 1 }

func stmtEndLine(stmt blizzard.Stmt) int {
	switch s := stmt.(type) {
	case *blizzard.FnStmt:
		if len(s.Body) > 0 {
			return bodyEndLine(s.Body)
		}
		l, _ := pos0(s.Pos)
		return l
	case *blizzard.IfStmt:
		end := 0
		for _, body := range s.Bodies {
			if e := bodyEndLine(body); e > end {
				end = e
			}
		}
		if e := bodyEndLine(s.Else); e > end {
			end = e
		}
		if end > 0 {
			return end
		}
		l, _ := pos0(s.Pos)
		return l
	case *blizzard.ForStmt:
		if len(s.Body) > 0 {
			return bodyEndLine(s.Body)
		}
		l, _ := pos0(s.Pos)
		return l
	case *blizzard.WhileStmt:
		if len(s.Body) > 0 {
			return bodyEndLine(s.Body)
		}
		l, _ := pos0(s.Pos)
		return l
	case *blizzard.WithStmt:
		if len(s.Body) > 0 {
			return bodyEndLine(s.Body)
		}
		l, _ := pos0(s.Pos)
		return l
	case *blizzard.TryStmt:
		end := 0
		if e := bodyEndLine(s.TryBody); e > end {
			end = e
		}
		if e := bodyEndLine(s.CatchBody); e > end {
			end = e
		}
		if e := bodyEndLine(s.Always); e > end {
			end = e
		}
		if end > 0 {
			return end
		}
		l, _ := pos0(s.Pos)
		return l
	case *blizzard.MatchStmt:
		end := 0
		for _, c := range s.Cases {
			if e := bodyEndLine(c.Body); e > end {
				end = e
			}
		}
		if end > 0 {
			return end
		}
		l, _ := pos0(s.Pos)
		return l
	default:
		l, _ := pos0(stmtPos(stmt))
		return l
	}
}

func bodyEndLine(body []blizzard.Stmt) int {
	if len(body) == 0 {
		return 0
	}
	return stmtEndLine(body[len(body)-1])
}

func stmtPos(stmt blizzard.Stmt) blizzard.Pos {
	switch s := stmt.(type) {
	case *blizzard.UseStmt:
		return s.Pos
	case *blizzard.AssignStmt:
		return s.Pos
	case *blizzard.SetAttrStmt:
		return s.Pos
	case *blizzard.SetIndexStmt:
		return s.Pos
	case *blizzard.FnStmt:
		return s.Pos
	case *blizzard.ReturnStmt:
		return s.Pos
	case *blizzard.IfStmt:
		return s.Pos
	case *blizzard.ForStmt:
		return s.Pos
	case *blizzard.WithStmt:
		return s.Pos
	case *blizzard.WhileStmt:
		return s.Pos
	case *blizzard.LoopStmt:
		return s.Pos
	case *blizzard.TryStmt:
		return s.Pos
	case *blizzard.MatchStmt:
		return s.Pos
	case *blizzard.ExprStmt:
		return s.Pos
	}
	return blizzard.Pos{}
}

func extractStmtSymbol(stmt blizzard.Stmt) *Symbol {
	switch s := stmt.(type) {
	case *blizzard.UseStmt:
		name := s.Alias
		if name == "" && len(s.Path) > 0 {
			name = s.Path[len(s.Path)-1]
		}
		l, c := pos0(s.Pos)
		return &Symbol{
			Name:       name,
			Kind:       SymModule,
			Detail:     strings.Join(s.Path, "."),
			Line:       l,
			Col:        c,
			EndLine:    l,
			EndCol:     c + len(name),
			Visibility: "pub",
		}
	case *blizzard.FnStmt:
		l, c := pos0(s.Pos)
		vis := s.Vis
		if vis == "" {
			vis = "pub"
		}
		sig := "fn " + s.Name + "(" + strings.Join(s.Params, ", ") + ")"
		if s.Ret != "" {
			sig += " -> " + s.Ret
		}
		children := extractBodySymbols(s.Body)
		return &Symbol{
			Name:       s.Name,
			Kind:       SymFunction,
			Detail:     sig,
			Line:       l,
			Col:        c,
			EndLine:    stmtEndLine(s),
			EndCol:     c + len(s.Name),
			Visibility: vis,
			Children:   children,
		}
	case *blizzard.AssignStmt:
		if len(s.Names) == 0 {
			return nil
		}
		l, c := pos0(s.Pos)
		vis := s.Vis
		if vis == "" {
			vis = "pub"
		}
		kind := SymVariable
		if s.Const {
			kind = SymConstant
		}
		// If all names are single, create one symbol per name on the same line.
		if len(s.Names) == 1 {
			name := s.Names[0]
			detail := "var"
			if s.Const {
				detail = "const"
			}
			if s.Type != "" {
				detail += ": " + s.Type
			}
			return &Symbol{
				Name:       name,
				Kind:       kind,
				Detail:     detail,
				Line:       l,
				Col:        c,
				EndLine:    l,
				EndCol:     c + len(name),
				Visibility: vis,
			}
		}
		// Multiple names (destructuring): return a parent container with children.
		children := make([]Symbol, 0, len(s.Names))
		for i, name := range s.Names {
			children = append(children, Symbol{
				Name:       name,
				Kind:       kind,
				Detail:     fmt.Sprintf("(destructured #%d)", i+1),
				Line:       l,
				Col:        c + i*2,
				EndLine:    l,
				EndCol:     c + i*2 + len(name),
				Visibility: vis,
			})
		}
		return &Symbol{
			Name:       "= (destructuring)",
			Kind:       SymArray,
			Detail:     fmt.Sprintf("%d names", len(s.Names)),
			Line:       l,
			Col:        c,
			EndLine:    l,
			EndCol:     c + 1,
			Visibility: vis,
			Children:   children,
		}
	case *blizzard.ForStmt:
		l, c := pos0(s.Pos)
		children := extractBodySymbols(s.Body)
		// Add iteration variable as child.
		if s.Name != "" {
			child := Symbol{
				Name:    s.Name,
				Kind:    SymVariable,
				Detail:  "for index/value",
				Line:    l,
				Col:     c + 4,
				EndLine: l,
				EndCol:  c + 4 + len(s.Name),
			}
			children = append([]Symbol{child}, children...)
		}
		return &Symbol{
			Name:     "for",
			Kind:     SymNamespace,
			Detail:   "loop block",
			Line:     l,
			Col:      c,
			EndLine:  stmtEndLine(s),
			EndCol:   c + 3,
			Children: children,
		}
	}
	return nil
}

func extractBodySymbols(body []blizzard.Stmt) []Symbol {
	out := []Symbol{}
	for _, st := range body {
		if s := extractStmtSymbol(st); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// errorToDiagnostic converts a Blizzard error to an LSP diagnostic
func (a *Analyzer) errorToDiagnostic(err error, source string) Diagnostic {
	errStr := err.Error()

	diag := Diagnostic{
		Severity: "error",
		Source:   "blizzard",
		Message:  errStr,
	}

	// Match from the end so URI schemes such as file:/// do not shift the position.
	if match := positionPattern.FindStringSubmatch(errStr); len(match) == 4 {
		var line, col int
		if _, scanErr := fmt.Sscanf(match[1], "%d", &line); scanErr == nil {
			diag.Line = max(line-1, 0)
		}
		if _, scanErr := fmt.Sscanf(match[2], "%d", &col); scanErr == nil {
			diag.Column = max(col-1, 0)
		}
		diag.Message = match[3]
	}
	diag.EndLine = diag.Line
	diag.EndColumn = diag.Column + 1

	return diag
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

// GetPositionFromBlizzardPos converts Blizzard Pos to LSP position
func (a *Analyzer) GetPositionFromBlizzardPos(pos blizzard.Pos) (line, col int) {
	return pos.Line - 1, pos.Col - 1 // Convert to 0-based
}

// Check performs static analysis on parsed code
func (a *Analyzer) Check(program *blizzard.Program, source string) []Diagnostic {
	diagnostics := []Diagnostic{}

	// Use Blizzard's built-in check functionality
	issues, err := blizzard.Check(source, program.SrcName)
	if err != nil {
		// This is a parsing error, already handled in Parse
		return diagnostics
	}

	// Convert Blizzard issues to LSP diagnostics
	for _, issue := range issues {
		diag := a.issueToDiagnostic(issue)
		diagnostics = append(diagnostics, diag)
	}

	return diagnostics
}

// issueToDiagnostic converts a Blizzard issue to an LSP diagnostic
func (a *Analyzer) issueToDiagnostic(issue blizzard.Issue) Diagnostic {
	severity := "warning"
	if issue.IsErr {
		severity = "error"
	}

	diag := Diagnostic{
		Line:      issue.Line - 1, // Convert to 0-based
		Column:    issue.Col - 1,  // Convert to 0-based
		EndLine:   issue.Line - 1,
		EndColumn: issue.Col, // End one character after start
		Severity:  severity,
		Source:    issue.Source,
		Message:   issue.Msg,
	}

	return diag
}

// FormatIssues formats diagnostics for display
func FormatIssues(diagnostics []Diagnostic) string {
	var b strings.Builder
	for _, diag := range diagnostics {
		kind := diag.Severity
		b.WriteString(fmt.Sprintf("%s:%d:%d: %s: %s\n", diag.Source, diag.Line+1, diag.Column+1, kind, diag.Message))
	}
	return strings.TrimRight(b.String(), "\n")
}
