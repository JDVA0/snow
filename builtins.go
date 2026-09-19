package snow

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type bfunc func(i *Interp, args []Val) ([]Val, error)

func stdBuiltins() map[string]bfunc {
	m := map[string]bfunc{
		"print":       bPrint,
		"len":         bLen,
		"str":         bStr,
		"int":         bInt,
		"flt":         bFlt,
		"bool":        bBool,
		"type":        bType,
		"range":       bRange,
		"min":         bMin,
		"max":         bMax,
		"abs":         bAbs,
		"floor":       bFloor,
		"ceil":        bCeil,
		"round":       bRound,
		"upper":       bUpper,
		"lower":       bLower,
		"trim":        bTrim,
		"split":       bSplit,
		"join":        bJoin,
		"replace":     bReplace,
		"contains":    bContains,
		"has":         bHas,
		"keys":        bKeys,
		"values":      bValues,
		"append":      bAppend,
		"reverse":     bReverse,
		"sort":        bSort,
		"map":         bMap,
		"filter":      bFilter,
		"fold":        bFold,
		"call":        bCall,
		"get":         bGet,
		"json_encode": bJSONEncode,
		"json_decode": bJSONDecode,
		"env":         bEnv,
		"args":        bArgs,
		"now":         bNow,
		"sleep":       bSleep,
		"input":       bInput,
		"exit":        bExit,
		"fail":        bFail,
	}
	for name := range m {
		stdNames[name] = true
	}
	return m
}

func want(args []Val, name string, n int) error {
	if len(args) != n {
		return fmt.Errorf("%s expects %d argument(s), got %d", name, n, len(args))
	}
	return nil
}

func bPrint(i *Interp, args []Val) ([]Val, error) {
	parts := make([]string, len(args))
	for k, a := range args {
		parts[k] = SnowStr(a)
	}
	fmt.Fprintln(i.out, strings.Join(parts, " "))
	return []Val{}, nil
}

func bLen(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "len", 1); err != nil {
		return nil, err
	}
	n, err := lenVal(args[0])
	if err != nil {
		return nil, err
	}
	return []Val{Int(n)}, nil
}

func bStr(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "str", 1); err != nil {
		return nil, err
	}
	return []Val{Str(SnowStr(args[0]))}, nil
}

func bInt(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "int", 1); err != nil {
		return nil, err
	}
	switch t := args[0].(type) {
	case Int:
		return []Val{t}, nil
	case Float:
		return []Val{Int(t)}, nil
	case Bool:
		if bool(t) {
			return []Val{Int(1)}, nil
		}
		return []Val{Int(0)}, nil
	case Str:
		n, err := strconv.ParseInt(strings.TrimSpace(string(t)), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to an int", string(t))
		}
		return []Val{Int(n)}, nil
	}
	return nil, fmt.Errorf("cannot convert %s to an int", TypeName(args[0]))
}

func bFlt(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "flt", 1); err != nil {
		return nil, err
	}
	switch t := args[0].(type) {
	case Int:
		return []Val{Float(t)}, nil
	case Float:
		return []Val{t}, nil
	case Bool:
		if bool(t) {
			return []Val{Float(1)}, nil
		}
		return []Val{Float(0)}, nil
	case Str:
		f, err := strconv.ParseFloat(strings.TrimSpace(string(t)), 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to a float", string(t))
		}
		return []Val{Float(f)}, nil
	}
	return nil, fmt.Errorf("cannot convert %s to a float", TypeName(args[0]))
}

func bBool(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "bool", 1); err != nil {
		return nil, err
	}
	return []Val{Bool(Truthy(args[0]))}, nil
}

func bType(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "type", 1); err != nil {
		return nil, err
	}
	return []Val{Str(TypeName(args[0]))}, nil
}

func bRange(i *Interp, args []Val) ([]Val, error) {
	var start, stop, step int64 = 0, 0, 1
	switch len(args) {
	case 1:
		v, ok := args[0].(Int)
		if !ok {
			return nil, fmt.Errorf("range expects ints, got %s", TypeName(args[0]))
		}
		stop = int64(v)
	case 2, 3:
		a, ok1 := args[0].(Int)
		b, ok2 := args[1].(Int)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("range expects ints, got %s and %s", TypeName(args[0]), TypeName(args[1]))
		}
		start, stop = int64(a), int64(b)
		if len(args) == 3 {
			s, ok := args[2].(Int)
			if !ok {
				return nil, fmt.Errorf("range expects an int step, got %s", TypeName(args[2]))
			}
			if int64(s) == 0 {
				return nil, errors.New("range step cannot be zero")
			}
			step = int64(s)
		}
	default:
		return nil, fmt.Errorf("range expects 1 to 3 arguments, got %d", len(args))
	}
	var out List
	if step > 0 {
		for k := start; k < stop; k += step {
			out = append(out, Int(k))
		}
	} else {
		for k := start; k > stop; k += step {
			out = append(out, Int(k))
		}
	}
	return []Val{out}, nil
}

func bMin(i *Interp, args []Val) ([]Val, error) {
	if len(args) == 1 {
		if l, ok := args[0].(List); ok {
			args = l
		}
	}
	if len(args) == 0 {
		return nil, errors.New("min needs at least one value")
	}
	best := args[0]
	for _, a := range args[1:] {
		c, err := Cmp(a, best)
		if err != nil {
			return nil, err
		}
		if c < 0 {
			best = a
		}
	}
	return []Val{best}, nil
}

func bMax(i *Interp, args []Val) ([]Val, error) {
	if len(args) == 1 {
		if l, ok := args[0].(List); ok {
			args = l
		}
	}
	if len(args) == 0 {
		return nil, errors.New("max needs at least one value")
	}
	best := args[0]
	for _, a := range args[1:] {
		c, err := Cmp(a, best)
		if err != nil {
			return nil, err
		}
		if c > 0 {
			best = a
		}
	}
	return []Val{best}, nil
}

func bAbs(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "abs", 1); err != nil {
		return nil, err
	}
	switch t := args[0].(type) {
	case Int:
		if t < 0 {
			return []Val{Int(-t)}, nil
		}
		return []Val{t}, nil
	case Float:
		return []Val{Float(math.Abs(float64(t)))}, nil
	}
	return nil, fmt.Errorf("abs expects a number, got %s", TypeName(args[0]))
}

func bFloor(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "floor", 1); err != nil {
		return nil, err
	}
	switch t := args[0].(type) {
	case Int:
		return []Val{t}, nil
	case Float:
		return []Val{Int(math.Floor(float64(t)))}, nil
	}
	return nil, fmt.Errorf("floor expects a number, got %s", TypeName(args[0]))
}

func bCeil(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "ceil", 1); err != nil {
		return nil, err
	}
	switch t := args[0].(type) {
	case Int:
		return []Val{t}, nil
	case Float:
		return []Val{Int(math.Ceil(float64(t)))}, nil
	}
	return nil, fmt.Errorf("ceil expects a number, got %s", TypeName(args[0]))
}

func bRound(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "round", 1); err != nil {
		return nil, err
	}
	switch t := args[0].(type) {
	case Int:
		return []Val{t}, nil
	case Float:
		return []Val{Int(math.Round(float64(t)))}, nil
	}
	return nil, fmt.Errorf("round expects a number, got %s", TypeName(args[0]))
}

func strArg(a Val, name string) (Str, error) {
	s, ok := a.(Str)
	if !ok {
		return "", fmt.Errorf("%s expects strings, got %s", name, TypeName(a))
	}
	return s, nil
}

func bUpper(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "upper", 1); err != nil {
		return nil, err
	}
	s, err := strArg(args[0], "upper")
	if err != nil {
		return nil, err
	}
	return []Val{Str(strings.ToUpper(string(s)))}, nil
}

func bLower(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "lower", 1); err != nil {
		return nil, err
	}
	s, err := strArg(args[0], "lower")
	if err != nil {
		return nil, err
	}
	return []Val{Str(strings.ToLower(string(s)))}, nil
}

func bTrim(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "trim", 1); err != nil {
		return nil, err
	}
	s, err := strArg(args[0], "trim")
	if err != nil {
		return nil, err
	}
	return []Val{Str(strings.TrimSpace(string(s)))}, nil
}

func bSplit(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "split", 2); err != nil {
		return nil, err
	}
	s, err := strArg(args[0], "split")
	if err != nil {
		return nil, err
	}
	sep, err := strArg(args[1], "split")
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(s), string(sep))
	out := make(List, len(parts))
	for k, p := range parts {
		out[k] = Str(p)
	}
	return []Val{out}, nil
}

func bJoin(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "join", 2); err != nil {
		return nil, err
	}
	l, ok := args[0].(List)
	if !ok {
		return nil, fmt.Errorf("join expects a list, got %s", TypeName(args[0]))
	}
	sep, err := strArg(args[1], "join")
	if err != nil {
		return nil, err
	}
	parts := make([]string, len(l))
	for k, e := range l {
		s, ok := e.(Str)
		if !ok {
			return nil, fmt.Errorf("join expects a list of strings, found %s", TypeName(e))
		}
		parts[k] = string(s)
	}
	return []Val{Str(strings.Join(parts, string(sep)))}, nil
}

func bReplace(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "replace", 3); err != nil {
		return nil, err
	}
	s, err := strArg(args[0], "replace")
	if err != nil {
		return nil, err
	}
	old, err := strArg(args[1], "replace")
	if err != nil {
		return nil, err
	}
	nu, err := strArg(args[2], "replace")
	if err != nil {
		return nil, err
	}
	return []Val{Str(strings.ReplaceAll(string(s), string(old), string(nu)))}, nil
}

func bContains(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "contains", 2); err != nil {
		return nil, err
	}
	got, err := binVal(args[1], args[0], boIn)
	if err != nil {
		return nil, err
	}
	return []Val{got}, nil
}

func bHas(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "has", 2); err != nil {
		return nil, err
	}
	d, ok := args[0].(*Dict)
	if !ok {
		return nil, fmt.Errorf("has expects a dict, got %s", TypeName(args[0]))
	}
	s, err := strArg(args[1], "has")
	if err != nil {
		return nil, err
	}
	return []Val{Bool(d.Has(string(s)))}, nil
}

func bKeys(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "keys", 1); err != nil {
		return nil, err
	}
	d, ok := args[0].(*Dict)
	if !ok {
		return nil, fmt.Errorf("keys expects a dict, got %s", TypeName(args[0]))
	}
	ks := d.Keys()
	out := make(List, len(ks))
	for k, v := range ks {
		out[k] = Str(v)
	}
	return []Val{out}, nil
}

func bValues(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "values", 1); err != nil {
		return nil, err
	}
	d, ok := args[0].(*Dict)
	if !ok {
		return nil, fmt.Errorf("values expects a dict, got %s", TypeName(args[0]))
	}
	out := make(List, 0, d.Len())
	d.ForEach(func(k string, v Val) { out = append(out, v) })
	return []Val{out}, nil
}

func bAppend(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "append", 2); err != nil {
		return nil, err
	}
	l, ok := args[0].(List)
	if !ok {
		return nil, fmt.Errorf("append expects a list, got %s", TypeName(args[0]))
	}
	out := make(List, len(l)+1)
	copy(out, l)
	out[len(l)] = args[1]
	return []Val{out}, nil
}

func bReverse(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "reverse", 1); err != nil {
		return nil, err
	}
	switch t := args[0].(type) {
	case List:
		out := make(List, len(t))
		for k, e := range t {
			out[len(t)-1-k] = e
		}
		return []Val{out}, nil
	case Str:
		rs := []rune(string(t))
		for a, b := 0, len(rs)-1; a < b; a, b = a+1, b-1 {
			rs[a], rs[b] = rs[b], rs[a]
		}
		return []Val{Str(string(rs))}, nil
	}
	return nil, fmt.Errorf("reverse expects a list or string, got %s", TypeName(args[0]))
}

func bSort(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "sort", 1); err != nil {
		return nil, err
	}
	l, ok := args[0].(List)
	if !ok {
		return nil, fmt.Errorf("sort expects a list, got %s", TypeName(args[0]))
	}
	out := make(List, len(l))
	copy(out, l)
	var sortErr error
	less := func(a, b Val) bool {
		c, err := Cmp(a, b)
		if err != nil {
			sortErr = err
			return false
		}
		return c < 0
	}
	sort.SliceStable(out, func(p, q int) bool { return less(out[p], out[q]) })
	if sortErr != nil {
		return nil, sortErr
	}
	return []Val{out}, nil
}

func bMap(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "map", 2); err != nil {
		return nil, err
	}
	f, l, err := fnAndList(args)
	if err != nil {
		return nil, err
	}
	out := make(List, len(l))
	for k, e := range l {
		vals, err := i.invoke(f, []Val{e})
		if err != nil {
			return nil, err
		}
		out[k] = vals[0]
	}
	return []Val{out}, nil
}

func bFilter(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "filter", 2); err != nil {
		return nil, err
	}
	f, l, err := fnAndList(args)
	if err != nil {
		return nil, err
	}
	var out List
	for _, e := range l {
		vals, err := i.invoke(f, []Val{e})
		if err != nil {
			return nil, err
		}
		v := vals[0]
		switch x := v.(type) {
		case Bool:
			if bool(x) {
				out = append(out, e)
			}
		default:
			return nil, fmt.Errorf("filter predicate must return a bool, got %s", TypeName(v))
		}
	}
	return []Val{out}, nil
}

func bFold(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "fold", 3); err != nil {
		return nil, err
	}
	var f Val
	var l List
	var acc Val
	if list, ok := args[0].(List); ok {
		l = list
		acc = args[1]
		f = args[2]
	} else if list, ok := args[2].(List); ok {
		f = args[0]
		acc = args[1]
		l = list
	} else {
		return nil, fmt.Errorf("fold expects a list, got %s", TypeName(args[0]))
	}
	switch f.(type) {
	case *Fn, Native:
	default:
		return nil, fmt.Errorf("fold expects a function, got %s", TypeName(f))
	}
	for _, e := range l {
		v, err := i.invoke(f, []Val{acc, e})
		if err != nil {
			return nil, err
		}
		acc = v[0]
	}
	return []Val{acc}, nil
}

func fnAndList(args []Val) (Val, List, error) {
	l, ok := args[1].(List)
	if !ok {
		return nil, nil, fmt.Errorf("expected a list, got %s", TypeName(args[1]))
	}
	switch args[0].(type) {
	case *Fn, Native:
		return args[0], l, nil
	}
	return nil, nil, fmt.Errorf("expected a function, got %s", TypeName(args[0]))
}

func bCall(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, errors.New("call expects a function")
	}
	return i.invoke(args[0], args[1:])
}

func bGet(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "get", 2); err != nil {
		return nil, err
	}
	d, ok := args[0].(*Dict)
	if !ok {
		return nil, fmt.Errorf("get expects a dict, got %s", TypeName(args[0]))
	}
	s, err := strArg(args[1], "get")
	if err != nil {
		return nil, err
	}
	if v, has := d.Get(string(s)); has {
		return []Val{v}, nil
	}
	return []Val{Nil}, nil
}

func bJSONEncode(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "json_encode", 1); err != nil {
		return nil, err
	}
	s, err := EncodeJSON(args[0])
	if err != nil {
		return nil, err
	}
	return []Val{Str(s)}, nil
}

func bJSONDecode(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "json_decode", 1); err != nil {
		return nil, err
	}
	s, err := strArg(args[0], "json_decode")
	if err != nil {
		return nil, err
	}
	v, err := DecodeJSON(string(s))
	if err != nil {
		return nil, err
	}
	return []Val{v}, nil
}

func bEnv(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "env", 1); err != nil {
		return nil, err
	}
	s, err := strArg(args[0], "env")
	if err != nil {
		return nil, err
	}
	if v, ok := os.LookupEnv(string(s)); ok {
		return []Val{Str(v)}, nil
	}
	return []Val{Nil}, nil
}

func bArgs(i *Interp, args []Val) ([]Val, error) {
	if len(args) != 0 {
		return nil, errors.New("args expects no arguments")
	}
	out := make(List, len(i.argv))
	for k, a := range i.argv {
		out[k] = Str(a)
	}
	return []Val{out}, nil
}

func bNow(i *Interp, args []Val) ([]Val, error) {
	if len(args) != 0 {
		return nil, errors.New("now expects no arguments")
	}
	return []Val{Float(float64(time.Now().UnixNano()) / 1e9)}, nil
}

func bSleep(i *Interp, args []Val) ([]Val, error) {
	if err := want(args, "sleep", 1); err != nil {
		return nil, err
	}
	v, ok := args[0].(Int)
	if !ok {
		return nil, fmt.Errorf("sleep expects milliseconds, got %s", TypeName(args[0]))
	}
	time.Sleep(time.Duration(v) * time.Millisecond)
	return []Val{}, nil
}

var stdinReader *bufio.Reader

func bInput(i *Interp, args []Val) ([]Val, error) {
	if len(args) == 1 {
		s, err := strArg(args[0], "input")
		if err != nil {
			return nil, err
		}
		fmt.Fprint(i.out, string(s))
	}
	reader := i.InReader()
	line, err := readLineFrom(reader)
	if err != nil && len(line) == 0 {
		return []Val{Nil}, nil
	}
	return []Val{Str(strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"))}, nil
}

func bExit(i *Interp, args []Val) ([]Val, error) {
	code := 0
	if len(args) == 1 {
		v, ok := args[0].(Int)
		if !ok {
			return nil, fmt.Errorf("exit expects an int code, got %s", TypeName(args[0]))
		}
		code = int(v)
	} else if len(args) > 1 {
		return nil, errors.New("exit expects at most one argument")
	}
	return nil, &ExitError{code}
}

func bFail(i *Interp, args []Val) ([]Val, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("fail expects 1 argument, got %d", len(args))
	}
	return nil, &errFail{val: args[0]}
}
