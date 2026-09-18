package snow

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const Version = "0.1"

// Repl runs an interactive session on the interpreter.
// If stdin is a terminal, it uses raw mode with line editing (arrow keys, history).
// Otherwise, it falls back to basic bufio.Reader.
func Repl(i *Interp) {
	fmt.Fprintf(i.out, "snow %s  type .exit to quit\n", Version)

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

	t := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}, "snow> ")

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
			if trimmed == ".help" {
				fmt.Fprintln(i.out, "Snow REPL commands:")
				fmt.Fprintln(i.out, "  .exit   quit the REPL")
				fmt.Fprintln(i.out, "  .clear  clear the terminal")
				fmt.Fprintln(i.out, "  .help   show this help")
				continue
			}
		}

		buf += line + "\n"

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
			if trimmed == ".help" {
				fmt.Fprintln(i.out, "Snow REPL commands:")
				fmt.Fprintln(i.out, "  .exit   quit the REPL")
				fmt.Fprintln(i.out, "  .clear  clear the terminal")
				fmt.Fprintln(i.out, "  .help   show this help")
				continue
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
