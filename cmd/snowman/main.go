package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/JDVA0/snow"
)

const usage = `snowman - the Snow language runner

Usage:
  snowman [file] [args...]         run a .snow file
  snowman run <file> [args...]     run a .snow file
  snowman eval <code>              evaluate a snippet
  snowman -e <code>                same as eval
  snowman fmt [-w] [file...]       format Snow source (stdout, or -w in place)
  snowman check <file>             lint a .snow file (exit 1 when issues found)
  snowman init [dir]               scaffold a new project (app.snow + README)
  snowman repl                     start an interactive session
  snowman -h, --help               show this help
  snowman -v, --version            print version

Standard modules (use with 'using'):
  sys     shell commands, environment, process, filesystem helpers
  fs      file operations: read, write, append, list, stat, mkdir
  cli     terminal colors, tables, boxes, prompts, flag parsing
  api     HTTP server with routing, middleware, static files
  http    HTTP client: get, post, put, delete, patch — resp.json()
  db      JSON key-value store persisted to disk
  time    sleep, timestamps, date formatting (time.now, time.sleep)
  json    parse, stringify with indent, valid validation
  crypto  sha256, md5, random_token, jwt_sign, jwt_verify
  task    scheduled and delayed background jobs (task.every, task.after)
  env     environment variables and .env files
  csv     parse, stringify, read and write CSV
  input   typed console prompts

Language features:
  f-strings     f"Hello {name}, age {age}"
  ?? operator   value ?? "default"  (returns right side when left is nil)
  ?. / ?[]      safe chaining: nil when missing, never an error
  ??= operator  assign only when the variable is nil
  slicing       list[1:3], list[:2], list[2:], "texto"[1:-1]
  not in        check an element is absent: 3 not in [1, 2]
  base literals 0b1010, 0o17, 0xFF
  match/case    switch-like statement match x: case ...
  for k, v in   iterate dicts and lists with index
  try/catch     errors as values; fail(valor) to raise
  typed lists   nombres: str[] = ["Julian", "Ana"]  (validates elements)
  fn types      fn sumar(a: int, b: int) -> int:    (optional, validated at call)
  multi-line    """..."""  or  '''...'''

Examples:
  snowman app.snow
  snowman fmt -w app.snow
  snowman -e 'using sys; print(sys.platform)'
  snowman run server.snow --port 8080
  snowman -e 'using http; r = http.get("https://httpbin.org/get"); print(r.status)'
`

func main() {
	args := os.Args[1:]
	i := snow.New()

	if len(args) == 0 {
		snow.Repl(i)
		return
	}

	switch args[0] {
	case "-h", "--help":
		fmt.Print(usage)
	case "-v", "--version":
		fmt.Printf("snow %s\n", snow.Version)
	case "repl":
		snow.Repl(i)
	case "fmt":
		if err := runFmt(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "check":
		if err := runCheck(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "-e", "eval":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: eval expects a code string")
			os.Exit(1)
		}
		i.Args(args[2:])
		runSrc(i, args[1], "<eval>")
	case "init":
		if err := runInit(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: run expects a file path")
			os.Exit(1)
		}
		i.Args(args[2:])
		runFile(i, args[1])
	default:
		i.Args(args[1:])
		runFile(i, args[0])
	}
}

func runSrc(i *snow.Interp, src, name string) {
	if err := i.Run(src, name); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var ex *snow.ExitError
		if errors.As(err, &ex) {
			os.Exit(ex.Code)
		}
		os.Exit(1)
	}
}

func runFile(i *snow.Interp, path string) {
	if err := i.RunFile(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var ex *snow.ExitError
		if errors.As(err, &ex) {
			os.Exit(ex.Code)
		}
		os.Exit(1)
	}
}

func runCheck(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("error: check expects a file path")
	}
	had := false
	for _, path := range args {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		issues, err := snow.Check(string(b), path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for _, is := range issues {
			fmt.Println(is)
		}
		if len(issues) > 0 {
			had = true
		}
	}
	if had {
		return fmt.Errorf("found issues in %d file(s)", len(args))
	}
	return nil
}

func runFmt(args []string) error {
	write := false
	var files []string
	for _, a := range args {
		if a == "-w" {
			write = true
			continue
		}
		files = append(files, a)
	}
	if len(files) == 0 {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		out, err := snow.Format(string(b))
		if err != nil {
			return err
		}
		_, err = io.WriteString(os.Stdout, out)
		return err
	}
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out, err := snow.Format(string(b))
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if write {
			if err := os.WriteFile(path, []byte(out), 0644); err != nil {
				return err
			}
			continue
		}
		if _, err := io.WriteString(os.Stdout, out); err != nil {
			return err
		}
	}
	return nil
}
