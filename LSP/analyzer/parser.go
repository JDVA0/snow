package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JDVA0/snow"
)

// Analyzer wraps the Snow parser for LSP usage
type Analyzer struct{}

// NewAnalyzer creates a new analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// ParseResult represents the result of parsing
type ParseResult struct {
	Program     *snow.Program
	Diagnostics []Diagnostic
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

// Parse parses Snow source code and returns diagnostics
func (a *Analyzer) Parse(source, filename string) (*ParseResult, error) {
	program, err := snow.Parse(source, filename)

	result := &ParseResult{
		Program:     program,
		Diagnostics: []Diagnostic{},
	}

	if err != nil {
		// Convert Snow error to LSP diagnostic
		diag := a.errorToDiagnostic(err, source)
		result.Diagnostics = append(result.Diagnostics, diag)
	}

	return result, nil
}

// errorToDiagnostic converts a Snow error to an LSP diagnostic
func (a *Analyzer) errorToDiagnostic(err error, source string) Diagnostic {
	errStr := err.Error()

	diag := Diagnostic{
		Severity: "error",
		Source:   "snow",
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

// GetPositionFromSnowPos converts Snow Pos to LSP position
func (a *Analyzer) GetPositionFromSnowPos(pos snow.Pos) (line, col int) {
	return pos.Line - 1, pos.Col - 1 // Convert to 0-based
}

// Check performs static analysis on parsed code
func (a *Analyzer) Check(program *snow.Program, source string) []Diagnostic {
	diagnostics := []Diagnostic{}

	// Use Snow's built-in check functionality
	issues, err := snow.Check(source, program.SrcName)
	if err != nil {
		// This is a parsing error, already handled in Parse
		return diagnostics
	}

	// Convert Snow issues to LSP diagnostics
	for _, issue := range issues {
		diag := a.issueToDiagnostic(issue)
		diagnostics = append(diagnostics, diag)
	}

	return diagnostics
}

// issueToDiagnostic converts a Snow issue to an LSP diagnostic
func (a *Analyzer) issueToDiagnostic(issue snow.Issue) Diagnostic {
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
