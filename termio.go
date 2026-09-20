package blizzard

import (
	"bufio"

	"golang.org/x/term"
)

// replTerm tracks the terminal state installed by the interactive REPL so
// that script reads from stdin (input.*, input(), cli.ask, cli.confirm) can
// briefly leave raw mode, read an echoed line, and return to raw mode.
// Without this, a script read blocks forever: in raw mode Enter sends '\r',
// not '\n', so ReadString('\n') never returns, and two buffered readers
// fight over the same file descriptor.
var (
	replTermFD     int
	replTermActive bool
	replTermSaved  *term.State
)

// setReplTerm registers the original (pre-raw) terminal state for stdin.
// It must be called while the REPL holds the terminal in raw mode.
func setReplTerm(fd int, saved *term.State) {
	replTermFD = fd
	replTermSaved = saved
	replTermActive = true
}

// endReplTerm clears the registration when the REPL exits.
func endReplTerm() {
	replTermActive = false
	replTermSaved = nil
}

// readLineFrom reads one line from reader. Inside the terminal REPL the line
// is read with the terminal restored to cooked mode (echo + line editing) and
// raw mode is re-enabled right after, so scripts and the REPL prompt do not
// contend over the same raw stdin.
func readLineFrom(reader *bufio.Reader) (string, error) {
	if replTermActive {
		term.Restore(replTermFD, replTermSaved)
	}
	line, err := reader.ReadString('\n')
	if replTermActive {
		replTermSaved, _ = term.MakeRaw(replTermFD)
	}
	return line, err
}

// termReadPassword wraps term.ReadPassword so masked input works the same way
// inside the terminal REPL.
func termReadPassword(fd int) ([]byte, error) {
	if replTermActive {
		term.Restore(replTermFD, replTermSaved)
	}
	b, err := term.ReadPassword(fd)
	if replTermActive {
		replTermSaved, _ = term.MakeRaw(replTermFD)
	}
	return b, err
}
