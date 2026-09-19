package snow

import (
	"errors"
	"fmt"
	"strings"
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
	if at.Line < 1 || at.Line > len(lines) {
		return msg
	}
	line := lines[at.Line-1]
	width := len(fmt.Sprintf("%d", at.Line))
	caret := at.Col - 1
	if caret < 0 {
		caret = 0
	}
	if caret > len(line) {
		caret = len(line)
	}
	return fmt.Sprintf("%s\n%*d | %s\n%s | %s^", msg, width, at.Line, line, strings.Repeat(" ", width), strings.Repeat(" ", caret))
}
