package blizzard

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

func readLineStdin(i *Interp, prompt string) (string, error) {
	if prompt != "" {
		fmt.Fprint(i.out, prompt)
	}
	reader := i.InReader()
	line, err := readLineFrom(reader)
	if err != nil && len(line) == 0 {
		return "", err
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}

func newInputModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(8)}
	m.Dict.Set("str", Native(inputStr))
	m.Dict.Set("line", Native(inputStr))
	m.Dict.Set("int", Native(inputInt))
	m.Dict.Set("float", Native(inputFloat))
	m.Dict.Set("bool", Native(inputBool))
	m.Dict.Set("hidden", Native(inputHidden))
	return m
}

// input.str([prompt], [default]) -> str
func inputStr(i *Interp, args []Val) ([]Val, error) {
	prompt := ""
	defaultVal := ""
	if len(args) >= 1 {
		prompt = BlizzardStr(args[0])
	}
	if len(args) >= 2 {
		defaultVal = BlizzardStr(args[1])
	}

	line, err := readLineStdin(i, prompt)
	if err != nil && line == "" {
		if defaultVal != "" {
			return []Val{Str(defaultVal)}, nil
		}
		return []Val{Nil}, nil
	}
	if strings.TrimSpace(line) == "" && defaultVal != "" {
		return []Val{Str(defaultVal)}, nil
	}
	return []Val{Str(line)}, nil
}

// input.int([prompt], [default]) -> int
func inputInt(i *Interp, args []Val) ([]Val, error) {
	prompt := ""
	var defaultVal *int64
	if len(args) >= 1 {
		prompt = BlizzardStr(args[0])
	}
	if len(args) >= 2 {
		switch d := args[1].(type) {
		case Int:
			v := int64(d)
			defaultVal = &v
		case Float:
			v := int64(d)
			defaultVal = &v
		case Str:
			if n, err := strconv.ParseInt(strings.TrimSpace(string(d)), 10, 64); err == nil {
				defaultVal = &n
			}
		}
	}

	for {
		line, err := readLineStdin(i, prompt)
		if err != nil && line == "" {
			if defaultVal != nil {
				return []Val{Int(*defaultVal)}, nil
			}
			return []Val{Nil}, nil
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && defaultVal != nil {
			return []Val{Int(*defaultVal)}, nil
		}
		n, err := strconv.ParseInt(trimmed, 10, 64)
		if err == nil {
			return []Val{Int(n)}, nil
		}
		// If default was given and input failed, retry or warn
		fmt.Fprintf(i.errOut, "Entrada inválida. Ingrese un número entero válido.\n")
	}
}

// input.float([prompt], [default]) -> float
func inputFloat(i *Interp, args []Val) ([]Val, error) {
	prompt := ""
	var defaultVal *float64
	if len(args) >= 1 {
		prompt = BlizzardStr(args[0])
	}
	if len(args) >= 2 {
		switch d := args[1].(type) {
		case Float:
			v := float64(d)
			defaultVal = &v
		case Int:
			v := float64(d)
			defaultVal = &v
		case Str:
			if f, err := strconv.ParseFloat(strings.TrimSpace(string(d)), 64); err == nil {
				defaultVal = &f
			}
		}
	}

	for {
		line, err := readLineStdin(i, prompt)
		if err != nil && line == "" {
			if defaultVal != nil {
				return []Val{Float(*defaultVal)}, nil
			}
			return []Val{Nil}, nil
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && defaultVal != nil {
			return []Val{Float(*defaultVal)}, nil
		}
		f, err := strconv.ParseFloat(trimmed, 64)
		if err == nil {
			return []Val{Float(f)}, nil
		}
		fmt.Fprintf(i.errOut, "Entrada inválida. Ingrese un número decimal válido.\n")
	}
}

// input.bool([prompt], [default]) -> bool
func inputBool(i *Interp, args []Val) ([]Val, error) {
	prompt := ""
	defaultVal := false
	if len(args) >= 1 {
		prompt = BlizzardStr(args[0])
	}
	if len(args) >= 2 {
		defaultVal = Truthy(args[1])
	}

	line, err := readLineStdin(i, prompt)
	if err != nil && line == "" {
		return []Val{Bool(defaultVal)}, nil
	}
	trimmed := strings.ToLower(strings.TrimSpace(line))
	if trimmed == "" {
		return []Val{Bool(defaultVal)}, nil
	}
	if trimmed == "y" || trimmed == "yes" || trimmed == "s" || trimmed == "si" || trimmed == "sí" || trimmed == "true" || trimmed == "1" {
		return []Val{Bool(true)}, nil
	}
	if trimmed == "n" || trimmed == "no" || trimmed == "false" || trimmed == "0" {
		return []Val{Bool(false)}, nil
	}
	return []Val{Bool(defaultVal)}, nil
}

// input.hidden([prompt], [default]) -> str (masked terminal input or line read)
func inputHidden(i *Interp, args []Val) ([]Val, error) {
	prompt := ""
	defaultVal := ""
	if len(args) >= 1 {
		prompt = BlizzardStr(args[0])
	}
	if len(args) >= 2 {
		defaultVal = BlizzardStr(args[1])
	}

	if prompt != "" {
		fmt.Fprint(i.out, prompt)
	}

	// Check if reading from an actual interactive terminal to mask characters
	if f, ok := i.in.(*os.File); (ok && term.IsTerminal(int(f.Fd()))) || (i.in == nil && term.IsTerminal(int(os.Stdin.Fd()))) {
		fd := int(os.Stdin.Fd())
		if ok {
			fd = int(f.Fd())
		}
		passBytes, err := termReadPassword(fd)
		fmt.Fprintln(i.out)
		if err != nil && len(passBytes) == 0 {
			if defaultVal != "" {
				return []Val{Str(defaultVal)}, nil
			}
			return []Val{Nil}, nil
		}
		line := string(passBytes)
		if strings.TrimSpace(line) == "" && defaultVal != "" {
			return []Val{Str(defaultVal)}, nil
		}
		return []Val{Str(line)}, nil
	}

	// Fallback for piped stdin or test mocks
	line, err := readLineStdin(i, "")
	if err != nil && line == "" {
		if defaultVal != "" {
			return []Val{Str(defaultVal)}, nil
		}
		return []Val{Nil}, nil
	}
	if strings.TrimSpace(line) == "" && defaultVal != "" {
		return []Val{Str(defaultVal)}, nil
	}
	return []Val{Str(line)}, nil
}
