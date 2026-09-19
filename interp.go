package snow

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	Name       string
	Params     []string
	ParamTypes []string
	Ret        string
	Body       []Op
	Env        *Env
	Line       int
	Col        int
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

// errFail is a Snow-level failure. The payload is a normal value
// (usually a string, sometimes a dict) — never a typed exception.
type errFail struct{ val Val }

func (e *errFail) Error() string { return SnowStr(e.val) }

// Env is a variable environment (scope chain).
type Env struct {
	parent *Env
	vars   map[string]Val
	types  map[string]string
}

// NewEnv creates an environment bound to a parent.
func NewEnv(parent *Env) *Env {
	return &Env{parent: parent, vars: map[string]Val{}, types: map[string]string{}}
}

// Interp is a Snow interpreter (a stack VM).
type Interp struct {
	stack     []Val
	env       *Env
	in        io.Reader
	inReader  *bufio.Reader
	out       io.Writer
	errOut    io.Writer
	src       string
	dir       string
	argv      []string
	depth     int
	frameBase int
	tryFrames []tryFrame
	api       *APIServer
	modules   map[string]Val
	loading   map[string]bool
	mu        sync.Mutex
}

type tryFrame struct {
	catchPC   int
	catchVar  string
	stackLen  int
	frameBase int
	env       *Env
	depth     int
}

const maxDepth = 10000

// New creates an interpreter with the standard library loaded.
func New() *Interp {
	i := &Interp{
		in:       os.Stdin,
		inReader: bufio.NewReader(os.Stdin),
		out:      os.Stdout,
		errOut:   os.Stderr,
		modules:  map[string]Val{},
		loading:  map[string]bool{},
	}
	i.env = NewEnv(nil)
	for name, fn := range stdBuiltins() {
		i.env.vars[name] = Native(fn)
	}
	return i
}

// In redirects input reader (for testing or embedding).
func (i *Interp) In(r io.Reader) {
	i.in = r
	if r != nil {
		i.inReader = bufio.NewReader(r)
	} else {
		i.inReader = nil
	}
}

// InReader returns a persistent buffered reader for the interpreter's input stream.
func (i *Interp) InReader() *bufio.Reader {
	if i.inReader == nil {
		if i.in != nil {
			i.inReader = bufio.NewReader(i.in)
		} else {
			i.inReader = bufio.NewReader(os.Stdin)
		}
	}
	return i.inReader
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

func (i *Interp) setVar(name string, val Val, declared ...string) error {
	typ := ""
	if len(declared) > 0 {
		typ = declared[0]
	}
	for e := i.env; e != nil; e = e.parent {
		if _, ok := e.vars[name]; ok {
			if typ != "" {
				e.types[name] = typ
			}
			if typ = e.types[name]; typ != "" {
				if err := checkValType(val, typ); err != nil {
					return err
				}
			}
			e.vars[name] = val
			return nil
		}
	}
	if typ != "" {
		if err := checkValType(val, typ); err != nil {
			return err
		}
		i.env.types[name] = typ
	}
	i.env.vars[name] = val
	return nil
}

func (i *Interp) opErr(op Op, err error) error {
	var e *Errat
	if errors.As(err, &e) {
		return err
	}
	if _, ok := err.(*errReturn); ok {
		return err
	}
	var st *errStack
	if errors.As(err, &st) {
		return err
	}
	var ex *ExitError
	if errors.As(err, &ex) {
		return ex
	}
	return &Errat{i.src, op.Line, op.Col, err}
}

// checkValType validates a value against a declared type: a scalar
// (str, int, ...), a list of a type (str[]), or any.
func checkValType(v Val, typ string) error {
	if typ == "" || typ == "any" {
		return nil
	}
	if strings.Contains(typ, "|") {
		for _, option := range strings.Split(typ, "|") {
			if checkValType(v, option) == nil {
				return nil
			}
		}
		return fmt.Errorf("expected %s, got %s", strings.ReplaceAll(typ, "|", " or "), TypeName(v))
	}
	if strings.HasSuffix(typ, "[]") {
		return checkTypedList(v, strings.TrimSuffix(typ, "[]"))
	}
	if TypeName(v) != typ {
		return fmt.Errorf("expected %s, got %s", typ, TypeName(v))
	}
	return nil
}

// checkTypedList validates a value assigned to a typed-list variable
// (nombre: str[] = [...]). Every element must match the declared type.
func checkTypedList(v Val, elem string) error {
	l, ok := v.(List)
	if !ok {
		return fmt.Errorf("expected a list, got %s", TypeName(v))
	}
	if elem == "any" {
		return nil
	}
	for i, e := range l {
		if _, isNil := e.(NilT); isNil {
			continue
		}
		if TypeName(e) != elem {
			return fmt.Errorf("cannot hold %s in %s[] (element %d)", TypeName(e), elem, i)
		}
	}
	return nil
}

func failValue(err error) Val {
	var f *errFail
	if errors.As(err, &f) {
		return f.val
	}
	var er *Errat
	if errors.As(err, &er) && er.Err != nil {
		return Str(er.Err.Error())
	}
	return Str(err.Error())
}

// errStack carries the caller frames collected while an error propagated
// through function calls. Its Error() is the base error message.
type errStack struct {
	base   error
	frames []string
}

func (e *errStack) Error() string { return e.base.Error() }

func (e *errStack) Unwrap() error { return e.base }

// addStack appends a call frame to an error unless it is a return/exit.
func addStack(err error, name string, line, col int, src string) error {
	if _, ok := err.(*errReturn); ok {
		return err
	}
	var ex *ExitError
	if errors.As(err, &ex) {
		return err
	}
	frame := fmt.Sprintf("in %s (%s:%d:%d)", name, src, line, col)
	var st *errStack
	if errors.As(err, &st) {
		st.frames = append(st.frames, frame)
		return err
	}
	return &errStack{base: err, frames: []string{frame}}
}

// StackOf returns the frames recorded on a (possibly nested) error.
func StackOf(err error) []string {
	var st *errStack
	if errors.As(err, &st) {
		return st.frames
	}
	return nil
}

// FormatError renders an error message together with its stack trace.
func FormatError(err error) string {
	var st *errStack
	if errors.As(err, &st) && len(st.frames) > 0 {
		var b strings.Builder
		b.WriteString(err.Error())
		b.WriteString("\nstack trace:")
		for i := len(st.frames) - 1; i >= 0; i-- {
			b.WriteString("\n  ")
			b.WriteString(st.frames[i])
		}
		return b.String()
	}
	return err.Error()
}

func (i *Interp) catchError(err error) (int, bool) {
	if _, ok := err.(*errReturn); ok {
		return 0, false
	}
	var ex *ExitError
	if errors.As(err, &ex) {
		return 0, false
	}
	if len(i.tryFrames) == 0 {
		return 0, false
	}
	tf := i.tryFrames[len(i.tryFrames)-1]
	i.tryFrames = i.tryFrames[:len(i.tryFrames)-1]

	i.stack = i.stack[:tf.stackLen]
	i.frameBase = tf.frameBase
	i.env = tf.env
	i.depth = tf.depth

	if tf.catchVar != "" {
		_ = i.setVar(tf.catchVar, failValue(err))
	}
	return tf.catchPC, true
}

func (i *Interp) execOps(ops []Op) error {
	pc := 0
	tryBase := len(i.tryFrames)
	for pc < len(ops) {
		op := ops[pc]
		err := i.execSingleOp(op, ops, &pc)
		if err != nil {
			if _, ok := err.(*errReturn); ok {
				i.tryFrames = i.tryFrames[:tryBase]
				return err
			}
			var ex *ExitError
			if errors.As(err, &ex) {
				i.tryFrames = i.tryFrames[:tryBase]
				return err
			}
			// Only catch frames pushed by this ops chunk, so a fail()
			// inside a function does not jump to an outer catch PC.
			if len(i.tryFrames) > tryBase {
				if targetPC, ok := i.catchError(err); ok {
					pc = targetPC
					continue
				}
			}
			return err
		}
		pc++
	}
	return nil
}

func (i *Interp) execSingleOp(op Op, ops []Op, pc *int) error {
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
		i.push(&Fn{Name: op.Name, Params: op.Args, ParamTypes: op.ParamTypes, Ret: op.Ret, Body: op.Body, Env: i.env, Line: op.Line, Col: op.Col})
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
		name := "<anon>"
		if f, ok := callee.(*Fn); ok && f.Name != "" {
			name = f.Name
		}
		vals, err := i.invoke(callee, args)
		if err != nil {
			if _, isFn := callee.(*Fn); isFn {
				err = addStack(err, name, op.Line, op.Col, i.src)
			}
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
		*pc = int(op.Num) - 1 // -1 because execOps does pc++ after
	case OpJumpIfNot:
		v, err := i.popBool(op)
		if err != nil {
			return err
		}
		if !bool(v.(Bool)) {
			*pc = int(op.Num) - 1
		}
	case OpJumpIf:
		v, err := i.popBool(op)
		if err != nil {
			return err
		}
		if bool(v.(Bool)) {
			*pc = int(op.Num) - 1
		}
	case OpJumpIfNotNil:
		v, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		if v != Nil {
			*pc = int(op.Num) - 1
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
			if err := i.setVar(name, vals[k], op.Type); err != nil {
				return i.opErr(op, err)
			}
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
		if op.Num == int64(boQQEq) {
			if cur == Nil {
				if err := i.setVar(op.Name, v); err != nil {
					return i.opErr(op, err)
				}
			}
			return nil
		}
		res, err := binVal(cur, v, byte(op.Num))
		if err != nil {
			return i.opErr(op, err)
		}
		if err := i.setVar(op.Name, res); err != nil {
			return i.opErr(op, err)
		}
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
	case OpSafeIndex:
		// x?[key] — returns nil if container is nil, wrong type, or key is absent.
		key, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		box, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		if _, isNil := box.(NilT); isNil {
			i.push(Nil)
			break
		}
		v, err := indexVal(box, key)
		if err != nil {
			// Missing key or wrong container type → propagate nil
			i.push(Nil)
			break
		}
		i.push(v)
	case OpSlice:
		hi, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		lo, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		box, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		v, err := sliceVal(box, lo, hi)
		if err != nil {
			return i.opErr(op, err)
		}
		i.push(v)
	case OpIterKeys:
		c, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		ks, err := iterKeys(c)
		if err != nil {
			return i.opErr(op, err)
		}
		i.push(ks)
	case OpIterPair:
		idx, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		c, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		k, ok := idx.(Int)
		if !ok {
			return i.opErr(op, fmt.Errorf("iteration index must be an int, got %s", TypeName(idx)))
		}
		key, val, err := iterPair(c, int(k))
		if err != nil {
			return i.opErr(op, err)
		}
		i.push(key)
		i.push(val)
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
	case OpTrySetup:
		i.tryFrames = append(i.tryFrames, tryFrame{
			catchPC:   int(op.Num),
			catchVar:  op.Name,
			stackLen:  len(i.stack),
			frameBase: i.frameBase,
			env:       i.env,
			depth:     i.depth,
		})
	case OpTryEnd:
		if len(i.tryFrames) > 0 {
			i.tryFrames = i.tryFrames[:len(i.tryFrames)-1]
		}
	case OpClose:
		res, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		var closeFn Val
		switch r := res.(type) {
		case *Dict:
			if v, ok := r.Get("close"); ok {
				closeFn = v
			}
		case *Module:
			if v, ok := r.Get("close"); ok {
				closeFn = v
			}
		}
		if closeFn != nil {
			switch f := closeFn.(type) {
			case Native, *Fn:
				_, cerr := i.invoke(f, nil)
				if cerr != nil {
					return i.opErr(op, cerr)
				}
			}
		}
	case OpSetIndex:
		val, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		key, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		box, err := i.pop()
		if err != nil {
			return i.opErr(op, err)
		}
		switch c := box.(type) {
		case *Dict:
			ks, ok := key.(Str)
			if !ok {
				return i.opErr(op, fmt.Errorf("dict key must be a string, got %s", TypeName(key)))
			}
			c.Set(string(ks), val)
		case List:
			idx, ok := key.(Int)
			if !ok {
				return i.opErr(op, fmt.Errorf("list index must be an int, got %s", TypeName(key)))
			}
			n := int64(len(c))
			i2 := int64(idx)
			if i2 < 0 {
				i2 += n
			}
			if i2 < 0 || i2 >= n {
				return i.opErr(op, fmt.Errorf("list index out of range: %d (len %d)", int64(idx), n))
			}
			c[i2] = val
		default:
			return i.opErr(op, fmt.Errorf("cannot index-assign to %s", TypeName(box)))
		}
	default:
		return i.opErr(op, errors.New("unknown opcode"))
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
		for k, p := range f.Params {
			if k < len(f.ParamTypes) && f.ParamTypes[k] != "" {
				if err := checkValType(args[k], f.ParamTypes[k]); err != nil {
					return nil, fmt.Errorf("parameter '%s': %v", p, err)
				}
			}
		}
		if i.depth >= maxDepth {
			return nil, errors.New("recursion limit exceeded")
		}
		env := NewEnv(f.Env)
		for k, p := range f.Params {
			env.vars[p] = args[k]
			if k < len(f.ParamTypes) && f.ParamTypes[k] != "" {
				env.types[p] = f.ParamTypes[k]
			}
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
			vals := r.vals
			if len(vals) == 0 {
				vals = []Val{Nil}
			}
			if f.Ret != "" {
				for _, v := range vals {
					if cerr := checkValType(v, f.Ret); cerr != nil {
						return nil, fmt.Errorf("return of %s: %v", f.Name, cerr)
					}
				}
			}
			return vals, nil
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

// InvokeSafe executes a callable thread-safely using the interpreter mutex.
func (i *Interp) InvokeSafe(callable Val, args []Val) ([]Val, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.invoke(callable, args)
}

func (i *Interp) useOp(op Op) error {
	path := op.Args
	stdMod := ""
	if len(path) == 2 && path[0] == "snow" {
		switch path[1] {
		case "api", "sys", "fs", "cli", "http", "db", "time", "json", "crypto", "task", "input", "env", "csv":
			stdMod = path[1]
		}
	} else if len(path) == 1 {
		switch path[0] {
		case "api", "sys", "fs", "cli", "http", "db", "time", "json", "crypto", "task", "input", "env", "csv":
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
		case "http":
			m = newHTTPModule(i, op.Name)
		case "db":
			m = newDBModule(i, op.Name)
		case "time":
			m = newTimeModule(i, op.Name)
		case "json":
			m = newJSONModule(i, op.Name)
		case "crypto":
			m = newCryptoModule(i, op.Name)
		case "task":
			m = newTaskModule(i, op.Name)
		case "input":
			m = newInputModule(i, op.Name)
		case "env":
			m = newEnvModule(i, op.Name)
		case "csv":
			m = newCSVModule(i, op.Name)
		default:
			return fmt.Errorf("unknown standard module snow.%s", stdMod)
		}
		i.modules[k] = m
		i.env.vars[op.Name] = m
		return nil
	}
	rel := resolveImportPath(i.dir, path, op.Relative)
	if m, ok := i.modules[rel]; ok {
		i.env.vars[op.Name] = m
		return nil
	}
	if i.loading[rel] {
		return fmt.Errorf("import cycle detected while loading '%s'", rel)
	}
	b, err := os.ReadFile(rel)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("module '%s' not found", rel)
		}
		return err
	}
	sub := New()
	sub.modules = i.modules
	sub.loading = i.loading
	i.loading[rel] = true
	defer delete(i.loading, rel)
	sub.out, sub.errOut = i.out, i.errOut
	sub.argv = i.argv
	sub.dir = filepath.Dir(rel)
	sub.src = rel
	prog, err := Parse(string(b), rel)
	if err != nil {
		return err
	}
	privNames := collectPrivateNames(prog.Stmts)
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
		if privNames[name] {
			continue
		}
		exp.Set(name, v)
	}
	m := &Module{Name: rel, Dict: exp}
	i.modules[rel] = m
	i.env.vars[op.Name] = m
	return nil
}

func collectPrivateNames(sts []Stmt) map[string]bool {
	priv := map[string]bool{}
	for _, s := range sts {
		switch t := s.(type) {
		case *FnStmt:
			if t.Vis == "priv" {
				priv[t.Name] = true
			}
		case *AssignStmt:
			if t.Vis == "priv" {
				for _, n := range t.Names {
					priv[n] = true
				}
			}
		}
	}
	return priv
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
		return binValIn(a, b)
	case boNotIn:
		r, err := binValIn(a, b)
		if err != nil {
			return nil, err
		}
		return Bool(!bool(r.(Bool))), nil
	}
	return nil, errors.New("unknown operator")
}

func binValIn(a, b Val) (Val, error) {
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

// sliceVal implements x[lo:hi] for lists and strings. lo/hi may be Nil
// (open end) or an int; negative indices count from the end.
func sliceVal(box, lo, hi Val) (Val, error) {
	switch c := box.(type) {
	case List:
		s, e, err := sliceBounds(lo, hi, int64(len(c)))
		if err != nil {
			return nil, err
		}
		return c[s:e], nil
	case Str:
		rs := []rune(string(c))
		s, e, err := sliceBounds(lo, hi, int64(len(rs)))
		if err != nil {
			return nil, err
		}
		return Str(string(rs[s:e])), nil
	}
	return nil, fmt.Errorf("cannot slice a %s", TypeName(box))
}

func sliceBounds(lo, hi Val, n int64) (int64, int64, error) {
	var s, e int64 = 0, n
	if lo != nil {
		if _, isNil := lo.(NilT); isNil {
			// open start
		} else if k, ok := lo.(Int); ok {
			s = int64(k)
			if s < 0 {
				s += n
			}
		} else {
			return 0, 0, fmt.Errorf("slice start must be an int, got %s", TypeName(lo))
		}
	}
	if hi != nil {
		if _, isNil := hi.(NilT); isNil {
			// open end
		} else if k, ok := hi.(Int); ok {
			e = int64(k)
			if e < 0 {
				e += n
			}
		} else {
			return 0, 0, fmt.Errorf("slice end must be an int, got %s", TypeName(hi))
		}
	}
	if s < 0 {
		s = 0
	}
	if e > n {
		e = n
	}
	if s > e {
		return 0, 0, fmt.Errorf("slice start %d is after end %d", s, e)
	}
	return s, e, nil
}

// iterKeys returns the items a single-name for loop iterates:
// dict → its keys, list → itself, string → its characters.
func iterKeys(v Val) (List, error) {
	switch c := v.(type) {
	case *Dict:
		var ks List
		c.ForEach(func(k string, _ Val) { ks = append(ks, Str(k)) })
		return ks, nil
	case List:
		return c, nil
	case Str:
		rs := []rune(string(c))
		ks := make(List, len(rs))
		for k, r := range rs {
			ks[k] = Str(string(r))
		}
		return ks, nil
	}
	return nil, fmt.Errorf("cannot iterate a %s", TypeName(v))
}

// iterPair yields the key/or-index and value at position k for `for k, v in`.
func iterPair(v Val, k int) (Val, Val, error) {
	switch c := v.(type) {
	case *Dict:
		if k < 0 || k >= c.Len() {
			return nil, nil, fmt.Errorf("index %d out of range for a dict of %d entries", k, c.Len())
		}
		var keys []string
		c.ForEach(func(key string, _ Val) { keys = append(keys, key) })
		key := keys[k]
		val, _ := c.Get(key)
		return Str(key), val, nil
	case List:
		if k < 0 || k >= len(c) {
			return nil, nil, fmt.Errorf("index %d out of range for a list of length %d", k, len(c))
		}
		return Int(int64(k)), c[k], nil
	case Str:
		rs := []rune(string(c))
		if k < 0 || k >= len(rs) {
			return nil, nil, fmt.Errorf("index %d out of range for a string of length %d", k, len(rs))
		}
		return Int(int64(k)), Str(string(rs[k])), nil
	}
	return nil, nil, fmt.Errorf("cannot iterate a %s", TypeName(v))
}

// stdNames is the set of builtin names (excluded from module exports).
var stdNames = map[string]bool{}
