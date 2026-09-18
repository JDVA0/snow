package snow

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Errat is a positioned error (file:line:col).
type Errat struct {
	File string
	Line int
	Col  int
	Err  error
}

func (e *Errat) Error() string {
	return fmt.Sprintf("%s:%d:%d: %v", e.File, e.Line, e.Col, e.Err)
}

func (e *Errat) Unwrap() error { return e.Err }

// ErrIncomplete marks input that needs more lines (used by the REPL).
var ErrIncomplete = errors.New("incomplete input")

// ExitError carries an exit code.
type ExitError struct{ Code int }

func (e *ExitError) Error() string { return fmt.Sprintf("exit %d", e.Code) }

// Fn is a Snow function value (a closure over its definition env).
type Fn struct {
	Name   string
	Params []string
	Body   []Op
	Env    *Env
	Line   int
	Col    int
}

// Native is a Go function exposed to Snow.
type Native func(i *Interp, args []Val) ([]Val, error)

// Module is a Snow module namespace.
type Module struct {
	Name string
	Dict *Dict
}

func (m *Module) Get(name string) (Val, bool) { return m.Dict.Get(name) }

type errReturn struct{ vals []Val }

func (e *errReturn) Error() string { return "return" }

// Env is a variable environment (scope chain).
type Env struct {
	parent *Env
	vars   map[string]Val
}

// NewEnv creates an environment bound to a parent.
func NewEnv(parent *Env) *Env {
	return &Env{parent: parent, vars: map[string]Val{}}
}

// Interp is a Snow interpreter (a stack VM).
type Interp struct {
	stack     []Val
	env       *Env
	out       io.Writer
	errOut    io.Writer
	src       string
	dir       string
	argv      []string
	depth     int
	frameBase int
	api       *APIServer
	modules   map[string]Val
}

const maxDepth = 10000

// New creates an interpreter with the standard library loaded.
func New() *Interp {
	i := &Interp{
		out:     os.Stdout,
		errOut:  os.Stderr,
		modules: map[string]Val{},
	}
	i.env = NewEnv(nil)
	for name, fn := range stdBuiltins() {
		i.env.vars[name] = Native(fn)
	}
	return i
}

// Out redirects output (for embedding).
func (i *Interp) Out(w io.Writer) { i.out = w; i.errOut = w }

// Args sets the program arguments visible to args().
func (i *Interp) Args(argv []string) { i.argv = argv }

// Run parses, compiles and executes source.
func (i *Interp) Run(src, name string) error {
	i.src = name
	p, err := Parse(src, name)
	if err != nil {
		return err
	}
	ops, err := Compile(p)
	if err != nil {
		return err
	}
	return i.Exec(ops)
}

// RunFile loads and runs a Snow file.
func (i *Interp) RunFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	i.dir = filepath.Dir(path)
	return i.Run(string(b), path)
}

// Eval runs source and leaves the last expression on the stack (REPL).
func (i *Interp) Eval(src string) error {
	p, err := Parse(src, "<repl>")
	if err != nil {
		return err
	}
	ops, err := CompileShow(p)
	if err != nil {
		return err
	}
	return i.Exec(ops)
}

// Exec runs precompiled ops (clearing the stack).
func (i *Interp) Exec(ops []Op) error {
	i.stack = i.stack[:0]
	return i.execOps(ops)
}

func (i *Interp) push(v Val) { i.stack = append(i.stack, v) }

func (i *Interp) pop() (Val, error) {
	if len(i.stack) == 0 {
		return nil, errors.New("stack underflow")
	}
	v := i.stack[len(i.stack)-1]
	i.stack = i.stack[:len(i.stack)-1]
	return v, nil
}

func (i *Interp) lookup(name string) (Val, bool) {
	for e := i.env; e != nil; e = e.parent {
		if v, ok := e.vars[name]; ok {
			return v, true
		}
	}
	return nil, false
}

func (i *Interp) opErr(op Op, err error) error {
	var e *Errat
	if errors.As(err, &e) {
		return err
	}
	if _, ok := err.(*errReturn); ok {
		return err
	}
	var ex *ExitError
	if errors.As(err, &ex) {
		return ex
	}
	return &Errat{i.src, op.Line, op.Col, err}
}

func (i *Interp) execOps(ops []Op) error {
	pc := 0
	for pc < len(ops) {
		op := ops[pc]
		switch op.Kind {
		case OpPushInt:
			i.push(Int(op.Num))
		case OpPushFloat:
			i.push(Float(op.Flt))
		case OpPushStr:
			i.push(Str(op.Str))
		case OpPushBool:
			i.push(Bool(op.Bol))
		case OpPushNil:
			i.push(Nil)
		case OpLoad:
			v, ok := i.lookup(op.Name)
			if !ok {
				return i.opErr(op, fmt.Errorf("undefined name '%s'", op.Name))
			}
			i.push(v)
		case OpStore:
			v, err := i.pop()
			if err != nil {
				return i.opErr(op, err)
			}
			i.env.vars[op.Name] = v
		case OpMakeFn:
			i.push(&Fn{Name: op.Name, Params: op.Args, Body: op.Body, Env: i.env, Line: op.Line, Col: op.Col})
		case OpCall:
			n := int(op.Num)
			if len(i.stack) < n {
				return i.opErr(op, fmt.Errorf("call expects %d argument(s), found %d on the stack", n, len(i.stack)))
			}
			args := make([]Val, n)
			for k := n - 1; k >= 0; k-- {
				args[k], _ = i.pop()
			}
			callee, err := i.pop()
			if err != nil {
				return i.opErr(op, err)
			}
			vals, err := i.invoke(callee, args)
			if err != nil {
				return i.opErr(op, err)
			}
			for _, v := range vals {
				i.push(v)
			}
		case OpReturn:
			n := int(op.Num)
			vals := make([]Val, n)
			for k := n - 1; k >= 0; k-- {
				v, err := i.pop()
				if err != nil {
					return i.opErr(op, err)
				}
				vals[k] = v
			}
			return &errReturn{vals}
		case OpJump:
			pc = int(op.Num)
			continue
		case OpJumpIfNot:
			v, err := i.popBool(op)
			if err != nil {
				return err
			}
			if !bool(v.(Bool)) {
				pc = int(op.Num)
				continue
			}
		case OpJumpIf:
			v, err := i.popBool(op)
			if err != nil {
				return err
			}
			if bool(v.(Bool)) {
				pc = int(op.Num)
				continue
			}
		case OpUse:
			if err := i.useOp(op); err != nil {
				return i.opErr(op, err)
			}
		case OpAssign:
			n := len(op.Args)
			if len(i.stack)-i.frameBase != n {
				return i.opErr(op, fmt.Errorf("assignment expects %d value(s), found %d", n, len(i.stack)-i.frameBase))
			}
			vals := make([]Val, n)
			for k := n - 1; k >= 0; k-- {
				vals[k], _ = i.pop()
			}
			for k, name := range op.Args {
				i.env.vars[name] = vals[k]
			}
		case OpStoreOp:
			v, err := i.pop()
			if err != nil {
				return i.opErr(op, err)
			}
			cur, ok := i.lookup(op.Name)
			if !ok {
				return i.opErr(op, fmt.Errorf("undefined name '%s'", op.Name))
			}
			res, err := binVal(cur, v, byte(op.Num))
			if err != nil {
				return i.opErr(op, err)
			}
			i.env.vars[op.Name] = res
		case OpMakeList:
			n := int(op.Num)
			if len(i.stack) < n {
				return i.opErr(op, errors.New("stack underflow building a list"))
			}
			l := make(List, n)
			for k := n - 1; k >= 0; k-- {
				l[k], _ = i.pop()
			}
			i.push(l)
		case OpMakeDict:
			n := int(op.Num)
			if len(i.stack) < 2*n {
				return i.opErr(op, errors.New("stack underflow building a dict"))
			}
			pairs := make([][2]Val, n)
			for k := n - 1; k >= 0; k-- {
				v, _ := i.pop()
				kk, _ := i.pop()
				pairs[k] = [2]Val{kk, v}
			}
			d := NewDict(n)
			for _, pr := range pairs {
				ks, ok := pr[0].(Str)
				if !ok {
					return i.opErr(op, fmt.Errorf("dict key must be a string, got %s", TypeName(pr[0])))
				}
				d.Set(string(ks), pr[1])
			}
			i.push(d)
		case OpIndex:
			key, err := i.pop()
			if err != nil {
				return i.opErr(op, err)
			}
			box, err := i.pop()
			if err != nil {
				return i.opErr(op, err)
			}
			v, err := indexVal(box, key)
			if err != nil {
				return i.opErr(op, err)
			}
			i.push(v)
		case OpLen:
			v, err := i.pop()
			if err != nil {
				return i.opErr(op, err)
			}
			n, err := lenVal(v)
			if err != nil {
				return i.opErr(op, err)
			}
			i.push(Int(n))
		case OpDup:
			if len(i.stack) == 0 {
				return i.opErr(op, errors.New("stack underflow"))
			}
			i.push(i.stack[len(i.stack)-1])
		case OpPop:
			if _, err := i.pop(); err != nil {
				return i.opErr(op, err)
			}
		case OpBin:
			b, err := i.pop()
			if err != nil {
				return i.opErr(op, err)
			}
			a, err := i.pop()
			if err != nil {
				return i.opErr(op, err)
			}
			v, err := binVal(a, b, byte(op.Num))
			if err != nil {
				return i.opErr(op, err)
			}
			i.push(v)
		case OpUn:
			a, err := i.pop()
			if err != nil {
				return i.opErr(op, err)
			}
			v, err := unVal(a, byte(op.Num))
			if err != nil {
				return i.opErr(op, err)
			}
			i.push(v)
		default:
			return i.opErr(op, errors.New("unknown opcode"))
		}
		pc++
	}
	return nil
}

func (i *Interp) popBool(op Op) (Val, error) {
	v, err := i.pop()
	if err != nil {
		return nil, i.opErr(op, err)
	}
	if _, ok := v.(Bool); !ok {
		return nil, i.opErr(op, fmt.Errorf("condition must be a bool, got %s", TypeName(v)))
	}
	return v, nil
}

func (i *Interp) invoke(callable Val, args []Val) ([]Val, error) {
	switch f := callable.(type) {
	case *Fn:
		if len(args) != len(f.Params) {
			return nil, fmt.Errorf("%s expects %d argument(s), got %d", f.Name, len(f.Params), len(args))
		}
		if i.depth >= maxDepth {
			return nil, errors.New("recursion limit exceeded")
		}
		env := NewEnv(f.Env)
		for k, p := range f.Params {
			env.vars[p] = args[k]
		}
		saved := i.env
		prevBase := i.frameBase
		i.env = env
		i.frameBase = len(i.stack)
		base := len(i.stack)
		i.depth++
		err := i.execOps(f.Body)
		i.depth--
		i.env = saved
		i.frameBase = prevBase
		if r, ok := err.(*errReturn); ok {
			i.stack = i.stack[:base]
			if len(r.vals) == 0 {
				return []Val{Nil}, nil
			}
			return r.vals, nil
		}
		if err != nil {
			return nil, err
		}
		i.stack = i.stack[:base]
		return []Val{Nil}, nil
	case Native:
		vals, err := f(i, args)
		if err != nil {
			return nil, err
		}
		if len(vals) == 0 {
			vals = []Val{Nil}
		}
		return vals, nil
	default:
		return nil, fmt.Errorf("cannot call a %s", TypeName(callable))
	}
}

func (i *Interp) useOp(op Op) error {
	path := op.Args
	stdMod := ""
	if len(path) == 2 && path[0] == "snow" {
		stdMod = path[1]
	} else if len(path) == 1 {
		switch path[0] {
		case "api", "sys", "fs", "cli":
			stdMod = path[0]
		}
	}

	if stdMod != "" {
		k := "snow:" + stdMod
		if m, ok := i.modules[k]; ok {
			i.env.vars[op.Name] = m
			return nil
		}
		var m *Module
		switch stdMod {
		case "api":
			m = newAPIModule(i, op.Name)
		case "sys":
			m = newSysModule(i, op.Name)
		case "fs":
			m = newFSModule(i, op.Name)
		case "cli":
			m = newCLIModule(i, op.Name)
		default:
			return fmt.Errorf("unknown standard module snow.%s", stdMod)
		}
		i.modules[k] = m
		i.env.vars[op.Name] = m
		return nil
	}
	rel := strings.Join(path, "/") + ".snow"
	if !filepath.IsAbs(rel) {
		rel = filepath.Join(i.dir, rel)
	}
	if m, ok := i.modules[rel]; ok {
		i.env.vars[op.Name] = m
		return nil
	}
	b, err := os.ReadFile(rel)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("module '%s' not found", rel)
		}
		return err
	}
	sub := New()
	sub.out, sub.errOut = i.out, i.errOut
	sub.argv = i.argv
	sub.dir = filepath.Dir(rel)
	sub.src = rel
	prog, err := Parse(string(b), rel)
	if err != nil {
		return err
	}
	ops, err := Compile(prog)
	if err != nil {
		return err
	}
	if err := sub.Exec(ops); err != nil {
		return err
	}
	exp := NewDict(len(sub.env.vars))
	for name, v := range sub.env.vars {
		if stdNames[name] {
			continue
		}
		exp.Set(name, v)
	}
	m := &Module{Name: rel, Dict: exp}
	i.modules[rel] = m
	i.env.vars[op.Name] = m
	return nil
}

// binVal implements binary operators.
func binVal(a, b Val, code byte) (Val, error) {
	switch code {
	case boAdd:
		switch x := a.(type) {
		case Int:
			switch y := b.(type) {
			case Int:
				return Int(x + y), nil
			case Float:
				return Float(float64(x) + float64(y)), nil
			}
		case Float:
			switch y := b.(type) {
			case Int:
				return x + Float(y), nil
			case Float:
				return x + y, nil
			}
		case Str:
			if y, ok := b.(Str); ok {
				return x + y, nil
			}
		case List:
			if y, ok := b.(List); ok {
				n := make(List, 0, len(x)+len(y))
				n = append(n, x...)
				n = append(n, y...)
				return n, nil
			}
		case *Dict:
			if y, ok := b.(*Dict); ok {
				n := x.Clone()
				y.ForEach(func(k string, v Val) { n.Set(k, v) })
				return n, nil
			}
		}
		return nil, fmt.Errorf("cannot add %s and %s", TypeName(a), TypeName(b))
	case boSub:
		af, ok := asFlt(a)
		if !ok {
			return nil, fmt.Errorf("cannot subtract %s", TypeName(a))
		}
		bf, ok := asFlt(b)
		if !ok {
			return nil, fmt.Errorf("cannot subtract %s", TypeName(b))
		}
		if _, ai := a.(Int); ai {
			if _, bi := b.(Int); bi {
				return Int(af - bf), nil
			}
		}
		return Float(af - bf), nil
	case boMul:
		switch x := a.(type) {
		case Str:
			if n, ok := b.(Int); ok && n >= 0 {
				return Str(strings.Repeat(string(x), int(n))), nil
			}
		case List:
			if n, ok := b.(Int); ok && n >= 0 {
				var out List
				for k := Int(0); k < n; k++ {
					out = append(out, x...)
				}
				return out, nil
			}
		}
		af, ok := asFlt(a)
		if !ok {
			return nil, fmt.Errorf("cannot multiply %s and %s", TypeName(a), TypeName(b))
		}
		bf, ok := asFlt(b)
		if !ok {
			return nil, fmt.Errorf("cannot multiply %s and %s", TypeName(a), TypeName(b))
		}
		if _, ai := a.(Int); ai {
			if _, bi := b.(Int); bi {
				return Int(af * bf), nil
			}
		}
		return Float(af * bf), nil
	case boDiv:
		af, ok := asFlt(a)
		if !ok {
			return nil, fmt.Errorf("cannot divide %s", TypeName(a))
		}
		bf, ok := asFlt(b)
		if !ok {
			return nil, fmt.Errorf("cannot divide by %s", TypeName(b))
		}
		if bf == 0 {
			return nil, errors.New("division by zero")
		}
		return Float(af / bf), nil
	case boFloorDiv:
		af, ok := asFlt(a)
		if !ok {
			return nil, fmt.Errorf("cannot divide %s", TypeName(a))
		}
		bf, ok := asFlt(b)
		if !ok {
			return nil, fmt.Errorf("cannot divide by %s", TypeName(b))
		}
		if bf == 0 {
			return nil, errors.New("division by zero")
		}
		if _, ai := a.(Int); ai {
			if _, bi := b.(Int); bi {
				q := af / bf
				if math.Mod(af, bf) != 0 && (af < 0) != (bf < 0) {
					q--
				}
				return Int(q), nil
			}
		}
		return Float(math.Floor(af / bf)), nil
	case boMod:
		af, ok := asFlt(a)
		if !ok {
			return nil, fmt.Errorf("cannot take %% of %s", TypeName(a))
		}
		bf, ok := asFlt(b)
		if !ok {
			return nil, fmt.Errorf("cannot take %% of %s", TypeName(b))
		}
		if bf == 0 {
			return nil, errors.New("division by zero")
		}
		if _, ai := a.(Int); ai {
			if _, bi := b.(Int); bi {
				r := math.Mod(af, bf)
				if r != 0 && (r < 0) != (bf < 0) {
					r += bf
				}
				return Int(r), nil
			}
		}
		return Float(math.Mod(af, bf)), nil
	case boEq:
		return Bool(Eql(a, b)), nil
	case boNe:
		return Bool(!Eql(a, b)), nil
	case boLt, boLe, boGt, boGe:
		c, err := Cmp(a, b)
		if err != nil {
			return nil, err
		}
		switch code {
		case boLt:
			return Bool(c < 0), nil
		case boLe:
			return Bool(c <= 0), nil
		case boGt:
			return Bool(c > 0), nil
		default:
			return Bool(c >= 0), nil
		}
	case boAnd:
		x, ok1 := a.(Bool)
		y, ok2 := b.(Bool)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("'and' needs bools, got %s and %s", TypeName(a), TypeName(b))
		}
		return Bool(bool(x) && bool(y)), nil
	case boOr:
		x, ok1 := a.(Bool)
		y, ok2 := b.(Bool)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("'or' needs bools, got %s and %s", TypeName(a), TypeName(b))
		}
		return Bool(bool(x) || bool(y)), nil
	case boIn:
		switch c := b.(type) {
		case Str:
			s, ok := a.(Str)
			if !ok {
				return nil, fmt.Errorf("left side of 'in' must be a string, got %s", TypeName(a))
			}
			return Bool(strings.Contains(string(c), string(s))), nil
		case List:
			for _, e := range c {
				if Eql(a, e) {
					return Bool(true), nil
				}
			}
			return Bool(false), nil
		case *Dict:
			s, ok := a.(Str)
			if !ok {
				return nil, fmt.Errorf("left side of 'in' must be a string, got %s", TypeName(a))
			}
			return Bool(c.Has(string(s))), nil
		case *Module:
			s, ok := a.(Str)
			if !ok {
				return nil, fmt.Errorf("left side of 'in' must be a string, got %s", TypeName(a))
			}
			return Bool(c.Dict.Has(string(s))), nil
		}
		return nil, fmt.Errorf("right side of 'in' must be a string, list or dict, got %s", TypeName(b))
	}
	return nil, errors.New("unknown operator")
}

func unVal(a Val, code byte) (Val, error) {
	switch code {
	case uoNeg:
		switch x := a.(type) {
		case Int:
			return Int(-x), nil
		case Float:
			return Float(-x), nil
		}
		return nil, fmt.Errorf("cannot negate a %s", TypeName(a))
	case uoNot:
		x, ok := a.(Bool)
		if !ok {
			return nil, fmt.Errorf("'not' needs a bool, got %s", TypeName(a))
		}
		return Bool(!bool(x)), nil
	}
	return nil, errors.New("unknown operator")
}

func indexVal(box, key Val) (Val, error) {
	switch c := box.(type) {
	case List:
		idx, ok := key.(Int)
		if !ok {
			return nil, fmt.Errorf("list index must be an int, got %s", TypeName(key))
		}
		k := int(idx)
		if k < 0 {
			k += len(c)
		}
		if k < 0 || k >= len(c) {
			return nil, fmt.Errorf("index %d out of range for a list of length %d", int(idx), len(c))
		}
		return c[k], nil
	case Str:
		idx, ok := key.(Int)
		if !ok {
			return nil, fmt.Errorf("string index must be an int, got %s", TypeName(key))
		}
		rs := []rune(string(c))
		k := int(idx)
		if k < 0 {
			k += len(rs)
		}
		if k < 0 || k >= len(rs) {
			return nil, fmt.Errorf("index %d out of range for a string of length %d", int(idx), len(rs))
		}
		return Str(string(rs[k])), nil
	case *Dict:
		ks, ok := key.(Str)
		if !ok {
			return nil, fmt.Errorf("dict key must be a string, got %s", TypeName(key))
		}
		v, has := c.Get(string(ks))
		if !has {
			return nil, fmt.Errorf("key not found: %s", SnowStr(ks))
		}
		return v, nil
	case *Module:
		ks, ok := key.(Str)
		if !ok {
			return nil, fmt.Errorf("attribute name must be a string, got %s", TypeName(key))
		}
		v, has := c.Get(string(ks))
		if !has {
			return nil, fmt.Errorf("attribute not found: %s", SnowStr(ks))
		}
		return v, nil
	}
	return nil, fmt.Errorf("cannot index a %s", TypeName(box))
}

func lenVal(v Val) (int64, error) {
	switch c := v.(type) {
	case Str:
		return int64(utf8.RuneCountInString(string(c))), nil
	case List:
		return int64(len(c)), nil
	case *Dict:
		return int64(c.Len()), nil
	}
	return 0, fmt.Errorf("len() does not accept %s", TypeName(v))
}

// stdNames is the set of builtin names (excluded from module exports).
var stdNames = map[string]bool{}
