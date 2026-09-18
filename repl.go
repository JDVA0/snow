package snow

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// crnlWriter wraps an io.Writer and converts bare '\n' to '\r\n'.
// This is required when the terminal is in raw mode (used by golang.org/x/term)
// because raw mode disables the OS-level ONLCR translation, so without \r the
// cursor only moves down, causing every subsequent prompt to drift rightward.
type crnlWriter struct{ w io.Writer }

func (c crnlWriter) Write(p []byte) (int, error) {
	// Fast path: no bare newlines.
	if !bytes.Contains(p, []byte{'\n'}) {
		return c.w.Write(p)
	}
	// Replace each \n that is not already preceded by \r.
	var buf []byte
	for i := 0; i < len(p); i++ {
		if p[i] == '\n' && (i == 0 || p[i-1] != '\r') {
			buf = append(buf, '\r', '\n')
		} else {
			buf = append(buf, p[i])
		}
	}
	_, err := c.w.Write(buf)
	// Return the original length so callers don't think a short-write occurred.
	return len(p), err
}

const Version = "0.1"

// ANSI color codes for REPL output.
const (
	ansiReset   = "\033[0m"
	ansiCyan    = "\033[36m"
	ansiYellow  = "\033[33m"
	ansiMagenta = "\033[35m"
	ansiGray    = "\033[90m"
	ansiBlue    = "\033[34m"
	ansiRed     = "\033[31m"
)

// replColorize returns an ANSI-colored representation of a value for REPL display.
func replColorize(v Val) string {
	switch v.(type) {
	case Str:
		return ansiCyan + SnowRepr(v) + ansiReset
	case Int, Float:
		return ansiYellow + SnowStr(v) + ansiReset
	case Bool:
		return ansiMagenta + SnowStr(v) + ansiReset
	case NilT:
		return ansiGray + "nil" + ansiReset
	case List, *Dict:
		return ansiBlue + SnowStr(v) + ansiReset
	default:
		return ansiGray + SnowStr(v) + ansiReset
	}
}

// SnowRepr returns a quoted representation of a string, or falls back to SnowStr.
func SnowRepr(v Val) string {
	if s, ok := v.(Str); ok {
		return "\"" + strings.ReplaceAll(string(s), "\"", "\\\"") + "\""
	}
	return SnowStr(v)
}

// historyFile returns the path to the REPL history file.
func historyFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".snow_history")
}

// loadHistory reads up to maxLines entries from the history file.
func loadHistory(maxLines int) []string {
	path := historyFile()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	var out []string
	for _, l := range lines {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// saveHistory appends new entries to the history file, keeping at most maxLines.
func saveHistory(newEntries []string, maxLines int) {
	path := historyFile()
	if path == "" {
		return
	}
	existing := loadHistory(maxLines)
	all := append(existing, newEntries...)
	if len(all) > maxLines {
		all = all[len(all)-maxLines:]
	}
	_ = os.WriteFile(path, []byte(strings.Join(all, "\n")+"\n"), 0600)
}

// Repl runs an interactive session on the interpreter.
// If stdin is a terminal, it uses raw mode with line editing (arrow keys, history).
// Otherwise, it falls back to basic bufio.Reader.
func Repl(i *Interp) {
	fmt.Fprintf(i.out, "snow %s  type help for commands, .exit to quit\n", Version)

	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		replTerminal(i, fd)
		return
	}
	replFallback(i)
}

func replTerminal(i *Interp, fd int) {
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		replFallback(i)
		return
	}
	defer term.Restore(fd, oldState)

	// In raw mode the OS ONLCR flag is disabled, so we must translate \n→\r\n
	// ourselves for any output that Snow scripts produce (print, cli.info, etc.).
	rawOut := crnlWriter{os.Stdout}
	rawErr := crnlWriter{os.Stderr}

	// Save and restore the interpreter's output writers.
	prevOut, prevErr := i.out, i.errOut
	i.out = rawOut
	i.errOut = rawErr
	defer func() { i.out = prevOut; i.errOut = prevErr }()

	t := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}, "snow> ")

	// Load and seed history
	const maxHistory = 500
	histEntries := loadHistory(maxHistory)
	for _, h := range histEntries {
		t.Write([]byte{}) // warm-up write; history populated via readline API below
		_ = h
	}
	// golang.org/x/term Terminal has no exported SetHistory, so we replay
	// by calling ReadLine on a fake source isn't feasible. We instead track
	// new entries ourselves and persist on exit.
	// Handle Tab keypresses: insert 4 spaces (standard Snow block indentation)
	// or complete common built-ins if completing a word.
	t.AutoCompleteCallback = func(line string, pos int, key rune) (string, int, bool) {
		if key == '\t' {
			// If cursor is at or after leading whitespace, or pressing tab:
			// insert 4 spaces for easy Python-style indentation
			prefix := line[:pos]
			suffix := line[pos:]
			// If prefix is just whitespace, indent by 4 spaces
			if strings.TrimSpace(prefix) == "" {
				return prefix + "    " + suffix, pos + 4, true
			}
			// Keyword autocompletion
			lastWord := ""
			for i := pos - 1; i >= 0; i-- {
				if (line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z') || line[i] == '_' || (line[i] >= '0' && line[i] <= '9') {
					lastWord = string(line[i]) + lastWord
				} else {
					break
				}
			}
			if lastWord != "" {
				candidates := []string{
					"using", "fn", "return", "if", "elif", "else", "for", "while", "break",
					"api", "http", "db", "sys", "fs", "cli", "print", "len", "str", "int",
					"float", "bool", "type", "range", "append", "keys", "values", "trim",
					"split", "join", "contains", "help", "exit", "clear",
				}
				for _, c := range candidates {
					if strings.HasPrefix(c, lastWord) && len(c) > len(lastWord) {
						added := c[len(lastWord):]
						return prefix + added + suffix, pos + len(added), true
					}
				}
			}
			// Default fallback on tab: insert 4 spaces
			return prefix + "    " + suffix, pos + 4, true
		}
		return "", 0, false
	}

	var sessionHistory []string

	defer func() {
		if len(sessionHistory) > 0 {
			saveHistory(sessionHistory, maxHistory)
		}
	}()

	var buf string
	for {
		if buf == "" {
			t.SetPrompt("snow> ")
		} else {
			t.SetPrompt("...> ")
		}

		line, err := t.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(i.out)
				return
			}
			return
		}

		trimmed := strings.TrimSpace(line)
		if buf == "" {
			if trimmed == ".exit" || trimmed == "exit" || trimmed == "quit" {
				return
			}
			if trimmed == ".clear" || trimmed == "clear" {
				fmt.Fprint(i.out, "\033[H\033[2J")
				continue
			}
			if trimmed == ".help" || trimmed == "help" {
				fmt.Fprintln(i.out, "Snow REPL commands:")
				fmt.Fprintln(i.out, "  help / .help    show this help")
				fmt.Fprintln(i.out, "  .exit / exit    quit the REPL")
				fmt.Fprintln(i.out, "  .clear / clear  clear the terminal")
				fmt.Fprintln(i.out, "  .history        show command history")
				continue
			}
			if trimmed == ".history" {
				h := loadHistory(50)
				for k, e := range h {
					fmt.Fprintf(i.out, "  %3d  %s\n", k+1, e)
				}
				continue
			}
			if trimmed != "" {
				sessionHistory = append(sessionHistory, trimmed)
			}
		}

		// Auto-indent helper: if the previous statement ended with ':' (a new block)
		// and the user provided code without leading indentation, auto-indent with 4 spaces.
		if buf != "" {
			prevTrimmed := strings.TrimRight(buf, " \t\r\n")
			if strings.HasSuffix(prevTrimmed, ":") && line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				line = "    " + line
			}
		}

		buf += line + "\n"

		prog, err := Parse(buf, "<repl>")
		if err != nil {
			var e *Errat
			if errors.As(err, &e) && errors.Is(e.Err, ErrIncomplete) {
				continue
			}
			fmt.Fprintf(i.errOut, "%s%s%s\n", ansiRed, err, ansiReset)
			buf = ""
			continue
		}
		if prog.Incomplete {
			continue
		}

		ops, err := CompileShow(prog)
		if err != nil {
			fmt.Fprintf(i.errOut, "%s%s%s\n", ansiRed, err, ansiReset)
			buf = ""
			continue
		}
		if err := i.Exec(ops); err != nil {
			fmt.Fprintf(i.errOut, "%s%s%s\n", ansiRed, err, ansiReset)
			buf = ""
			continue
		}
		if len(prog.Stmts) > 0 {
			if _, ok := prog.Stmts[len(prog.Stmts)-1].(*ExprStmt); ok && len(i.stack) > 0 {
				v := i.stack[len(i.stack)-1]
				i.stack = i.stack[:0]
				if v != Nil {
					fmt.Fprintln(i.out, replColorize(v))
				}
			}
		}
		buf = ""
	}
}

func replFallback(i *Interp) {
	in := bufio.NewReader(os.Stdin)
	var buf string
	for {
		if buf == "" {
			fmt.Fprint(i.out, "snow> ")
		} else {
			fmt.Fprint(i.out, "...> ")
		}
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			fmt.Fprintln(i.out)
			return
		}
		trimmed := strings.TrimSpace(line)
		if buf == "" {
			if trimmed == ".exit" || trimmed == "exit" || trimmed == "quit" {
				return
			}
			if trimmed == ".clear" || trimmed == "clear" {
				continue
			}
			if trimmed == ".help" || trimmed == "help" {
				fmt.Fprintln(i.out, "Snow REPL commands:")
				fmt.Fprintln(i.out, "  help / .help    show this help")
				fmt.Fprintln(i.out, "  .exit / exit    quit the REPL")
				fmt.Fprintln(i.out, "  .clear / clear  clear the terminal")
				continue
			}
		}
		if buf != "" {
			prevTrimmed := strings.TrimRight(buf, " \t\r\n")
			if strings.HasSuffix(prevTrimmed, ":") && trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				line = "    " + line
			}
		}
		buf += line

		prog, err := Parse(buf, "<repl>")
		if err != nil {
			var e *Errat
			if errors.As(err, &e) && errors.Is(e.Err, ErrIncomplete) {
				continue
			}
			fmt.Fprintln(i.errOut, err)
			buf = ""
			continue
		}
		if prog.Incomplete {
			continue
		}

		ops, err := CompileShow(prog)
		if err != nil {
			fmt.Fprintln(i.errOut, err)
			buf = ""
			continue
		}
		if err := i.Exec(ops); err != nil {
			fmt.Fprintln(i.errOut, err)
			buf = ""
			continue
		}
		if len(prog.Stmts) > 0 {
			if _, ok := prog.Stmts[len(prog.Stmts)-1].(*ExprStmt); ok && len(i.stack) > 0 {
				v := i.stack[len(i.stack)-1]
				i.stack = i.stack[:0]
				if v != Nil {
					fmt.Fprintln(i.out, SnowStr(v))
				}
			}
		}
		buf = ""
	}
}
