package blizzard

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// DiagnosticCode identifies a stable class of Blizzard diagnostic.
type DiagnosticCode string

// DiagnosticError carries a diagnostic code at the point where an error is
// created, rather than requiring tools to infer it from human text.
type DiagnosticError struct {
	Code DiagnosticCode
	Err  error
}

func (e *DiagnosticError) Error() string { return e.Err.Error() }
func (e *DiagnosticError) Unwrap() error { return e.Err }

const (
	CodeSyntax    DiagnosticCode = "B001"
	CodeUndefined DiagnosticCode = "B002"
	CodeModule    DiagnosticCode = "B003"
	CodeType      DiagnosticCode = "B004"
	CodeRuntime   DiagnosticCode = "B005"
	CodeWarning   DiagnosticCode = "B000"
	CodeLegacy    DiagnosticCode = "B100"
	CodeInternal  DiagnosticCode = "B999"
)

// CodeFor returns the public code for an error without changing the compact
// error strings used by embedders and existing scripts.
func CodeFor(err error) DiagnosticCode {
	if err == nil {
		return ""
	}
	var at *Errat
	if errors.As(err, &at) {
		err = at.Err
	}
	var coded *DiagnosticError
	if errors.As(err, &coded) {
		return coded.Code
	}
	message := err.Error()
	switch {
	case errors.Is(err, ErrIncomplete), strings.Contains(message, "expected "), strings.Contains(message, "unexpected "), strings.Contains(message, "unterminated"), strings.Contains(message, "unbalanced"):
		return CodeSyntax
	case strings.Contains(message, "undefined name"):
		return CodeUndefined
	case strings.Contains(message, "module '") && strings.Contains(message, "not found"):
		return CodeModule
	case strings.Contains(message, "type mismatch") || strings.Contains(message, "expected ") && strings.Contains(message, "received"):
		return CodeType
	default:
		return CodeRuntime
	}
}

// FormatDiagnostic renders an error with a source line and a caret when a
// positioned Blizzard error is available. It is intended for command-line tools;
// Error() remains compact for embedders.
func FormatDiagnostic(err error, source string) string {
	var at *Errat
	if !errors.As(err, &at) {
		return fmt.Sprintf("error: %v", err)
	}
	code := CodeFor(err)
	msg := fmt.Sprintf("%s:%d:%d: %s: %v", at.File, at.Line, at.Col, code, at.Err)
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	// A trailing newline produces an empty last element; drop it so line
	// numbers line up with at.Line.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	ln := at.Line
	if ln < 1 {
		return msg
	}
	if ln > len(lines) {
		ln = len(lines)
	}
	if len(lines) == 0 {
		return msg
	}
	line := lines[ln-1]
	// Errors that land on a blank line (for example a missing indented block
	// reported near the end of input) are shown against the nearest previous
	// line of code so the caret still has context.
	for ln > 1 && line == "" {
		ln--
		line = lines[ln-1]
	}
	width := len(fmt.Sprintf("%d", ln))
	gutter := strings.Repeat(" ", width)
	if line == "" {
		return fmt.Sprintf("%s\n%*d | (end of input)", msg, width, ln)
	}
	caret := at.Col - 1
	if n := utf8.RuneCountInString(line); caret > n {
		caret = n
	}
	if caret < 0 {
		caret = 0
	}
	return fmt.Sprintf("%s\n%*d | %s\n%s | %s^", msg, width, ln, line, gutter, strings.Repeat(" ", caret))
}
