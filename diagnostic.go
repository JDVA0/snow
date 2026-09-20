package snow

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// FormatDiagnostic renders an error with a source line and a caret when a
// positioned Snow error is available. It is intended for command-line tools;
// Error() remains compact for embedders.
func FormatDiagnostic(err error, source string) string {
	var at *Errat
	if !errors.As(err, &at) {
		return fmt.Sprintf("error: %v", err)
	}
	msg := fmt.Sprintf("%s:%d:%d: error: %v", at.File, at.Line, at.Col, at.Err)
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
