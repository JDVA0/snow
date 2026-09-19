package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/term"

	"github.com/JDVA0/snow"
	"github.com/JDVA0/snow/internal/pkgmgr"
)

const usage = `snowman - the Snow language runner

Usage:
  snowman [file] [args...]         run a .snow file
  snowman run <file> [args...]     run a .snow file
  snowman eval <code>              evaluate a snippet
  snowman -e <code>                same as eval
  snowman fmt [-w] [file...]       format Snow source (stdout, or -w in place)
  snowman check <file>             lint a .snow file (exit 1 when issues found)
  snowman test [--filter text] [path...] run *_test.snow files
	snowman init <name>               create a Snow project
	snowman get snow/name.snow        install an official package
	snowman search [query]            search official packages
	snowman add <name> <path>         add a local dependency
	snowman remove <name>             remove a dependency
	snowman list                      list dependencies
	snowman info snow/name.snow       show installed package metadata
	snowman index                     inspect package registry
	snowman update                    update locked packages
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
	power         2 ** 3
	collections   first, last, take, drop, sum, any, all, clamp
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
			printError(err)
			os.Exit(1)
		}
	case "check":
		if err := runCheck(args[1:]); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "test":
		if err := runTests(args[1:]); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "-e", "eval":
		if len(args) < 2 {
			printError(errors.New("eval expects a code string"))
			os.Exit(1)
		}
		i.Args(args[2:])
		runSrc(i, args[1], "<eval>")
	case "init":
		if err := pkgmgr.Run(args); err != nil {
			printError(fmt.Errorf("snowman: %w", err))
			os.Exit(1)
		}
	case "get", "get-local", "search", "add", "remove", "list", "info", "index", "update":
		if err := pkgmgr.Run(args); err != nil {
			printError(fmt.Errorf("snowman: %w", err))
			os.Exit(1)
		}
	case "run":
		if len(args) < 2 {
			printError(errors.New("run expects a file path"))
			os.Exit(1)
		}
		i.Args(args[2:])
		runFile(i, args[1])
	default:
		i.Args(args[1:])
		runFile(i, args[0])
	}
}

func printError(err error) {
	message := err.Error()
	if term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("NO_COLOR") == "" {
		message = "\033[31;1merror:\033[0m " + message
	} else {
		message = "error: " + message
	}
	fmt.Fprintln(os.Stderr, message)
}

func printDiagnostic(message string) {
	if term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("NO_COLOR") == "" {
		message = "\033[31;1m" + message + "\033[0m"
	}
	fmt.Fprintln(os.Stderr, message)
}

func runSrc(i *snow.Interp, src, name string) {
	if err := i.Run(src, name); err != nil {
		printDiagnostic(snow.FormatDiagnostic(err, src))
		var ex *snow.ExitError
		if errors.As(err, &ex) {
			os.Exit(ex.Code)
		}
		os.Exit(1)
	}
}

func runFile(i *snow.Interp, path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		printError(err)
		os.Exit(1)
	}
	if err := i.RunFile(path); err != nil {
		printDiagnostic(snow.FormatDiagnostic(err, string(b)))
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
			return fmt.Errorf("%s", snow.FormatDiagnostic(err, string(b)))
		}
		for _, is := range issues {
			message := is.String()
			if term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == "" {
				color := "\033[33m"
				if is.IsErr {
					color = "\033[31;1m"
				}
				message = color + message + "\033[0m"
			}
			fmt.Println(message)
			if is.IsErr {
				had = true
			}
		}
	}
	if had {
		return fmt.Errorf("found issues in %d file(s)", len(args))
	}
	return nil
}

func runTests(args []string) error {
	filter := ""
	var paths []string
	for n := 0; n < len(args); n++ {
		if args[n] == "--filter" {
			if n+1 == len(args) {
				return fmt.Errorf("test: --filter expects text")
			}
			filter = args[n+1]
			n++
			continue
		}
		paths = append(paths, args[n])
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
			continue
		}
		err = filepath.WalkDir(path, func(file string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.snow") {
				files = append(files, file)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	sort.Strings(files)
	if filter != "" {
		filtered := files[:0]
		for _, file := range files {
			if strings.Contains(file, filter) {
				filtered = append(filtered, file)
			}
		}
		files = filtered
	}
	if len(files) == 0 {
		return fmt.Errorf("no Snow test files found (expected *_test.snow)")
	}
	failed := 0
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err == nil {
			i := snow.New()
			err = i.RunFile(file)
		}
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "FAIL %s\n%s\n", file, snow.FormatDiagnostic(err, string(b)))
			continue
		}
		fmt.Printf("PASS %s\n", file)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d test file(s) failed", failed, len(files))
	}
	fmt.Printf("PASS %d test file(s)\n", len(files))
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
