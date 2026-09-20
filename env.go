package blizzard

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func newEnvModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(8)}
	m.Dict.Set("get", Native(envGet))
	m.Dict.Set("set", Native(envSet))
	m.Dict.Set("int", Native(envInt))
	m.Dict.Set("float", Native(envFloat))
	m.Dict.Set("bool", Native(envBool))
	m.Dict.Set("has", Native(envHas))
	m.Dict.Set("all", Native(envAll))
	m.Dict.Set("load", Native(envLoad))
	return m
}

// env.get(key, [default]) -> str | nil
func envGet(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("env.get requires at least 1 argument (key)")
	}
	key := BlizzardStr(args[0])
	val, ok := os.LookupEnv(key)
	if !ok {
		if len(args) >= 2 {
			return []Val{args[1]}, nil
		}
		return []Val{Nil}, nil
	}
	return []Val{Str(val)}, nil
}

// env.has(key) -> bool
func envHas(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("env.has requires 1 argument (key)")
	}
	key := BlizzardStr(args[0])
	_, ok := os.LookupEnv(key)
	return []Val{Bool(ok)}, nil
}

// env.set(key, val) -> nil
func envSet(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("env.set requires 2 arguments (key, val)")
	}
	key := BlizzardStr(args[0])
	val := BlizzardStr(args[1])
	if err := os.Setenv(key, val); err != nil {
		return nil, err
	}
	return []Val{Nil}, nil
}

// env.int(key, [default]) -> int
func envInt(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("env.int requires at least 1 argument (key)")
	}
	key := BlizzardStr(args[0])
	val, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(val) == "" {
		if len(args) >= 2 {
			switch d := args[1].(type) {
			case Int:
				return []Val{d}, nil
			case Float:
				return []Val{Int(int64(d))}, nil
			case Str:
				if n, err := strconv.ParseInt(strings.TrimSpace(string(d)), 10, 64); err == nil {
					return []Val{Int(n)}, nil
				}
			}
		}
		return []Val{Nil}, nil
	}
	n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
	if err != nil {
		if len(args) >= 2 {
			if d, ok := args[1].(Int); ok {
				return []Val{d}, nil
			}
		}
		return nil, fmt.Errorf("env.int: %q is not an int", val)
	}
	return []Val{Int(n)}, nil
}

// env.float(key, [default]) -> float
func envFloat(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("env.float requires at least 1 argument (key)")
	}
	key := BlizzardStr(args[0])
	val, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(val) == "" {
		if len(args) >= 2 {
			switch d := args[1].(type) {
			case Float:
				return []Val{d}, nil
			case Int:
				return []Val{Float(float64(d))}, nil
			}
		}
		return []Val{Nil}, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
	if err != nil {
		if len(args) >= 2 {
			if d, ok := args[1].(Float); ok {
				return []Val{d}, nil
			}
		}
		return nil, fmt.Errorf("env.float: %q is not a float", val)
	}
	return []Val{Float(f)}, nil
}

// env.bool(key, [default]) -> bool
func envBool(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("env.bool requires at least 1 argument (key)")
	}
	key := BlizzardStr(args[0])
	val, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(val) == "" {
		if len(args) >= 2 {
			return []Val{Bool(Truthy(args[1]))}, nil
		}
		return []Val{Bool(false)}, nil
	}
	trimmed := strings.ToLower(strings.TrimSpace(val))
	if trimmed == "1" || trimmed == "true" || trimmed == "yes" || trimmed == "y" || trimmed == "si" || trimmed == "on" {
		return []Val{Bool(true)}, nil
	}
	return []Val{Bool(false)}, nil
}

// env.all() -> dict
func envAll(i *Interp, args []Val) ([]Val, error) {
	envVars := os.Environ()
	d := NewDict(len(envVars))
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			d.Set(parts[0], Str(parts[1]))
		}
	}
	return []Val{d}, nil
}

// env.load([path]) -> int (count of vars loaded)
func envLoad(i *Interp, args []Val) ([]Val, error) {
	path := ".env"
	if len(args) >= 1 {
		path = BlizzardStr(args[0])
	}
	file, err := os.Open(path)
	if err != nil {
		return []Val{Int(0)}, nil // If file doesn't exist, return 0 smoothly
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := int64(0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// support "export KEY=VAL"
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		// Remove wrapping quotes if present
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		os.Setenv(k, v)
		count++
	}
	return []Val{Int(count)}, nil
}
