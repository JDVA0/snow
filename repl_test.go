package blizzard

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// replWithInput runs the fallback REPL with a fixed stdin pipe and returns
// everything it wrote to its output.
func replWithInput(t *testing.T, input string) string {
	t.Helper()
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.WriteString(input); err != nil {
		t.Fatal(err)
	}
	pw.Close()
	old := os.Stdin
	os.Stdin = pr
	defer func() { os.Stdin = old }()

	var out bytes.Buffer
	i := New()
	i.Out(&out)
	Repl(i)
	pr.Close()
	return out.String()
}

func TestReplMultiLineBlock(t *testing.T) {
	got := replWithInput(t, "pub fn cal(a,b):\nfor i in range(a,b) where i % 2 == 0:\nprint(i)\n\nprint(cal(4, 10))\n")
	for _, want := range []string{"4", "6", "8"} {
		if !strings.Contains(got, want+"\n") {
			t.Fatalf("REPL should compute even numbers, got:\n%s", got)
		}
	}
	if strings.Contains(got, "error:") {
		t.Fatalf("REPL reported an error for a valid multi-line block:\n%s", got)
	}
}

func TestReplBlockHeaderKeepsReading(t *testing.T) {
	// A header with no body yet ('for ...:') must not error out; the REPL
	// waits for the indented body before evaluating.
	got := replWithInput(t, "fn sums():\nfor i in [1, 2, 3]:\nprint(i)\n\nsums()\n")
	if strings.Contains(got, "error:") || strings.Contains(got, "expected an indented block") {
		t.Fatalf("REPL should wait for the block body, got:\n%s", got)
	}
	if !strings.Contains(got, "1\n") || !strings.Contains(got, "3\n") {
		t.Fatalf("REPL should run sums(), got:\n%s", got)
	}
}

func TestAutoIndentNested(t *testing.T) {
	if got := autoIndent("fn f():\n"); got != "    " {
		t.Fatalf("top-level block should indent 4, got %q", got)
	}
	if got := autoIndent("fn f():\n    for i in [1, 2]:\n"); got != "        " {
		t.Fatalf("nested block should indent 8, got %q", got)
	}
	if got := autoIndent("x = 1\n"); got != "" {
		t.Fatalf("plain line should not auto-indent, got %q", got)
	}
}
