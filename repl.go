package snow

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

const Version = "0.1"

// Repl runs an interactive session on the interpreter.
func Repl(i *Interp) {
	in := bufio.NewReader(os.Stdin)
	fmt.Fprintf(i.out, "snow %s  type .exit to quit\n", Version)
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
