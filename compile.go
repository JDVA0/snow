package snow

import (
	"fmt"
)

// OpKind identifies a VM operation.
type OpKind int

const (
	OpPushInt OpKind = iota
	OpPushFloat
	OpPushStr
	OpPushBool
	OpPushNil
	OpLoad
	OpStore
	OpMakeFn
	OpCall
	OpReturn
	OpJump
	OpJumpIfNot
	OpJumpIf
	OpJumpIfNotNil // for ?? operator
	OpUse
	OpAssign
	OpStoreOp
	OpMakeList
	OpMakeDict
	OpIndex
	OpSafeIndex // x?[key]: returns nil when key is absent or container is nil
	OpSlice     // x[lo:hi]: pops hi, lo, container
	OpIterKeys  // container → list of keys/items (dict→keys, list→itself, str→chars)
	OpIterPair  // container,idx → pushes value, then key-or-idx (for `for k, v in`)
	OpQQEq      // ??= : pop rhs; if slot is nil store it, else discard
	OpLen
	OpDup
	OpPop
	OpBin
	OpUn
	OpTrySetup
	OpTryEnd
	OpClose
	OpSetIndex
)

// Op is a single VM instruction.
type Op struct {
	Kind       OpKind
	Name       string
	Num        int64
	Flt        float64
	Bol        bool
	Str        string
	Args       []string
	Body       []Op
	Type       string // declared gradual type for typed assignments (e.g. "str", "str[]")
	Const      bool
	ParamTypes []string // per-param types for OpMakeFn, "" when untyped
	Ret        string   // declared return type for OpMakeFn, "" when untyped
	Relative   int      // explicit relative import level for OpUse
	Line       int
	Col        int
}

// Binary op codes.
const (
	boAdd byte = 1 + iota
	boSub
	boMul
	boDiv
	boFloorDiv
	boMod
	boEq
	boNe
	boLt
	boLe
	boGt
	boGe
	boAnd
	boOr
	boIn
	boNotIn
)

// boQQEq is the sentinel op stored in AssignStmt.Op for the ??= assignment.
const boQQEq byte = 0xFE

// Unary op codes.
const (
	uoNeg byte = iota
	uoNot
)

var binCodes = map[string]byte{
	"+": boAdd, "-": boSub, "*": boMul, "/": boDiv, "//": boFloorDiv, "%": boMod,
	"==": boEq, "!=": boNe, "<": boLt, "<=": boLe, ">": boGt, ">=": boGe,
	"and": boAnd, "or": boOr, "in": boIn, "not in": boNotIn,
}

// boNullCoalesce is not a binVal opcode — it is handled via OpJumpIfNotNil.
// We set "??" to a sentinel that the compiler detects and handles specially.
const boDummy byte = 0xFF

type loopFrame struct {
	top        int
	isFor      bool
	tryDepth   int
	contJumps  []int
	breakJumps []int
}

type compiler struct {
	ops      []Op
	loops    []loopFrame
	tmpN     int
	tryDepth int
	keepLast bool
}

func (c *compiler) emit(op Op) int {
	c.ops = append(c.ops, op)
	return len(c.ops) - 1
}

func (c *compiler) patch(at, to int) {
	c.ops[at].Num = int64(to)
}

func (c *compiler) emitTryUnwind(loopTryDepth, line, col int) {
	for i := c.tryDepth; i > loopTryDepth; i-- {
		c.emit(Op{Kind: OpTryEnd, Line: line, Col: col})
	}
}

func (c *compiler) perr(ps Pos, format string, a ...any) error {
	return &Errat{"<compile>", ps.Line, ps.Col, fmt.Errorf(format, a...)}
}

// Compile turns a parsed program into VM ops.
func Compile(p *Program) ([]Op, error) {
	c := &compiler{}
	return c.finish(p)
}

func (c *compiler) finish(p *Program) ([]Op, error) {
	if err := c.stmts(p.Stmts, false); err != nil {
		return nil, err
	}
	return c.ops, nil
}

// CompileShow compiles a program keeping the value of its last
// top-level expression on the stack (used by the REPL).
func CompileShow(p *Program) ([]Op, error) {
	c := &compiler{keepLast: true}
	if err := c.stmts(p.Stmts, true); err != nil {
		return nil, err
	}
	return c.ops, nil
}

func (c *compiler) stmts(list []Stmt, last bool) error {
	for i, s := range list {
		if err := c.stmt(s, last && i == len(list)-1); err != nil {
			return err
		}
	}
	return nil
}

func (c *compiler) stmt(s Stmt, last bool) error {
	switch t := s.(type) {
	case *ExprStmt:
		if err := c.expr(t.X); err != nil {
			return err
		}
		if !(c.keepLast && last) {
			c.emit(Op{Kind: OpPop, Line: t.Line, Col: t.Col})
		}
	case *AssignStmt:
		for _, v := range t.Vals {
			if err := c.expr(v); err != nil {
				return err
			}
		}
		if t.Op == 0 {
			dyn := false
			for _, v := range t.Vals {
				if _, ok := v.(*CallE); ok {
					dyn = true
				}
			}
			if !dyn && len(t.Vals) != len(t.Names) {
				return c.perr(t.Pos, "assignment expects %d value(s), found %d", len(t.Names), len(t.Vals))
			}
			c.emit(Op{Kind: OpAssign, Args: t.Names, Type: t.Type, Const: t.Const, Line: t.Line, Col: t.Col})
		} else {
			c.emit(Op{Kind: OpStoreOp, Name: t.Names[0], Num: int64(t.Op), Line: t.Line, Col: t.Col})
		}
	case *SetAttrStmt:
		if err := c.expr(t.Container); err != nil {
			return err
		}
		c.emit(Op{Kind: OpPushStr, Str: t.Name, Line: t.Line, Col: t.Col})
		if err := c.expr(t.Val); err != nil {
			return err
		}
		c.emit(Op{Kind: OpSetIndex, Line: t.Line, Col: t.Col})
	case *SetIndexStmt:
		if err := c.expr(t.Container); err != nil {
			return err
		}
		if err := c.expr(t.Key); err != nil {
			return err
		}
		if err := c.expr(t.Val); err != nil {
			return err
		}
		c.emit(Op{Kind: OpSetIndex, Line: t.Line, Col: t.Col})
	case *FnStmt:
		bc := &compiler{}
		if err := bc.stmts(t.Body, false); err != nil {
			return err
		}
		op := Op{Kind: OpMakeFn, Name: t.Name, Args: t.Params, ParamTypes: t.ParamTypes, Ret: t.Ret, Body: bc.ops, Line: t.Line, Col: t.Col}
		c.emit(op)
		// bind the function name in the current scope
		c.emit(Op{Kind: OpAssign, Args: []string{t.Name}, Line: t.Line, Col: t.Col})
	case *ReturnStmt:
		for _, v := range t.Vals {
			if err := c.expr(v); err != nil {
				return err
			}
		}
		c.emit(Op{Kind: OpReturn, Num: int64(len(t.Vals)), Line: t.Line, Col: t.Col})
	case *IfStmt:
		var endJumps []int
		for i := range t.Conds {
			if err := c.expr(t.Conds[i]); err != nil {
				return err
			}
			skip := c.emit(Op{Kind: OpJumpIfNot, Line: posLine(t.Conds[i]), Col: posCol(t.Conds[i])})
			if err := c.stmts(t.Bodies[i], false); err != nil {
				return err
			}
			endJumps = append(endJumps, c.emit(Op{Kind: OpJump, Line: t.Line, Col: t.Col}))
			c.patch(skip, len(c.ops))
		}
		if err := c.stmts(t.Else, false); err != nil {
			return err
		}
		for _, j := range endJumps {
			c.patch(j, len(c.ops))
		}
	case *WhileStmt:
		top := len(c.ops)
		c.loops = append(c.loops, loopFrame{top: top, tryDepth: c.tryDepth})
		cond := t.Cond
		if err := c.expr(cond); err != nil {
			return err
		}
		skip := c.emit(Op{Kind: OpJumpIfNot, Line: posLine(cond), Col: posCol(cond)})
		if err := c.stmts(t.Body, false); err != nil {
			return err
		}
		fr := c.loops[len(c.loops)-1]
		c.loops = c.loops[:len(c.loops)-1]
		c.emit(Op{Kind: OpJump, Num: int64(top), Line: t.Line, Col: t.Col})
		c.patch(skip, len(c.ops))
		for _, b := range fr.breakJumps {
			c.patch(b, len(c.ops))
		}
	case *ForStmt:
		tmp := fmt.Sprintf("_for%d", c.tmpN)
		idx := fmt.Sprintf("_forI%d", c.tmpN)
		keys := fmt.Sprintf("_forK%d", c.tmpN)
		c.tmpN++
		if err := c.expr(t.Iter); err != nil {
			return err
		}
		c.emit(Op{Kind: OpStore, Name: tmp, Line: posLine(t.Iter), Col: posCol(t.Iter)})
		if t.Name2 == "" {
			// single name: iterate keys/items of dicts, chars of strings, items of lists
			c.emit(Op{Kind: OpLoad, Name: tmp, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpIterKeys, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpStore, Name: keys, Line: t.Line, Col: t.Col})
		}
		c.emit(Op{Kind: OpPushInt, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpStore, Name: idx, Line: t.Line, Col: t.Col})
		top := len(c.ops)
		c.loops = append(c.loops, loopFrame{top: top, isFor: true, tryDepth: c.tryDepth})
		c.emit(Op{Kind: OpLoad, Name: idx, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpLoad, Name: tmp, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpLen, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpBin, Num: int64(boLt), Line: t.Line, Col: t.Col})
		skip := c.emit(Op{Kind: OpJumpIfNot, Line: t.Line, Col: t.Col})
		if t.Name2 == "" {
			c.emit(Op{Kind: OpLoad, Name: keys, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpLoad, Name: idx, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpIndex, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpStore, Name: t.Name, Line: t.Line, Col: t.Col})
		} else {
			c.emit(Op{Kind: OpLoad, Name: tmp, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpLoad, Name: idx, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpIterPair, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpStore, Name: t.Name2, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpStore, Name: t.Name, Line: t.Line, Col: t.Col})
		}
		var whereSkip int
		if t.Where != nil {
			if err := c.expr(t.Where); err != nil {
				return err
			}
			whereSkip = c.emit(Op{Kind: OpJumpIfNot, Line: posLine(t.Where), Col: posCol(t.Where)})
		}
		if err := c.stmts(t.Body, false); err != nil {
			return err
		}
		if t.Where != nil {
			c.patch(whereSkip, len(c.ops))
		}
		fr := c.loops[len(c.loops)-1]
		c.loops = c.loops[:len(c.loops)-1]
		cont := len(c.ops)
		c.emit(Op{Kind: OpLoad, Name: idx, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpPushInt, Num: 1, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpBin, Num: int64(boAdd), Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpStore, Name: idx, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpJump, Num: int64(top), Line: t.Line, Col: t.Col})
		c.patch(skip, len(c.ops))
		for _, b := range fr.breakJumps {
			c.patch(b, len(c.ops))
		}
		for _, j := range fr.contJumps {
			c.patch(j, cont)
		}
	case *LoopStmt:
		if len(c.loops) == 0 {
			return c.perr(t.Pos, "break/continue outside of a loop")
		}
		fr := &c.loops[len(c.loops)-1]
		c.emitTryUnwind(fr.tryDepth, t.Line, t.Col)
		if t.Break {
			fr.breakJumps = append(fr.breakJumps, c.emit(Op{Kind: OpJump, Line: t.Line, Col: t.Col}))
		} else {
			if fr.isFor {
				fr.contJumps = append(fr.contJumps, c.emit(Op{Kind: OpJump, Line: t.Line, Col: t.Col}))
			} else {
				c.emit(Op{Kind: OpJump, Num: int64(fr.top), Line: t.Line, Col: t.Col})
			}
		}
	case *TryStmt:
		setupIdx := c.emit(Op{Kind: OpTrySetup, Name: t.CatchVar, Line: t.Line, Col: t.Col})
		c.tryDepth++
		if err := c.stmts(t.TryBody, false); err != nil {
			return err
		}
		c.tryDepth--
		c.emit(Op{Kind: OpTryEnd, Line: t.Line, Col: t.Col})
		jumpPastCatch := c.emit(Op{Kind: OpJump, Line: t.Line, Col: t.Col})
		catchOffset := len(c.ops)
		c.patch(setupIdx, catchOffset)
		if err := c.stmts(t.CatchBody, false); err != nil {
			return err
		}
		c.patch(jumpPastCatch, len(c.ops))
		if err := c.stmts(t.Always, false); err != nil {
			return err
		}
	case *UseStmt:
		c.emit(Op{Kind: OpUse, Name: t.Alias, Args: t.Path, Relative: t.Relative, Line: t.Line, Col: t.Col})
	case *MatchStmt:
		tmp := fmt.Sprintf("_match%d", c.tmpN)
		c.tmpN++
		if err := c.expr(t.Target); err != nil {
			return err
		}
		c.emit(Op{Kind: OpStore, Name: tmp, Line: t.Line, Col: t.Col})
		var endJumps []int
		for _, cs := range t.Cases {
			if err := c.caseCond(tmp, cs.Vals, t.Line, t.Col); err != nil {
				return err
			}
			skip := c.emit(Op{Kind: OpJumpIfNot, Line: t.Line, Col: t.Col})
			if err := c.stmts(cs.Body, false); err != nil {
				return err
			}
			endJumps = append(endJumps, c.emit(Op{Kind: OpJump, Line: t.Line, Col: t.Col}))
			c.patch(skip, len(c.ops))
		}
		for _, j := range endJumps {
			c.patch(j, len(c.ops))
		}
	case *WithStmt:
		withTmp := fmt.Sprintf("_with%d", c.tmpN)
		c.tmpN++
		if err := c.expr(t.Expr); err != nil {
			return err
		}
		c.emit(Op{Kind: OpStore, Name: withTmp, Line: posLine(t.Expr), Col: posCol(t.Expr)})
		if t.Name != "" {
			c.emit(Op{Kind: OpLoad, Name: withTmp, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpStore, Name: t.Name, Line: t.Line, Col: t.Col})
		}
		if err := c.stmts(t.Body, false); err != nil {
			return err
		}
		c.emit(Op{Kind: OpLoad, Name: withTmp, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpClose, Line: t.Line, Col: t.Col})
	default:
		return c.perr(Pos{}, "unsupported statement")
	}
	return nil
}

// caseCond compiles the equality chain for a match case: (v1 == tmp) or (v2 == tmp) ...
// A single identifier "_" is a wildcard that matches any value.
func (c *compiler) caseCond(tmp string, vals []Expr, line, col int) error {
	for _, v := range vals {
		if name, ok := v.(*NameE); ok && name.X == "_" {
			// wildcard: the whole case always matches
			c.emit(Op{Kind: OpPushBool, Bol: true, Line: line, Col: col})
			return nil
		}
	}
	for i, v := range vals {
		if err := c.expr(v); err != nil {
			return err
		}
		c.emit(Op{Kind: OpLoad, Name: tmp, Line: line, Col: col})
		c.emit(Op{Kind: OpBin, Num: int64(boEq), Line: line, Col: col})
		if i > 0 {
			c.emit(Op{Kind: OpBin, Num: int64(boOr), Line: line, Col: col})
		}
	}
	return nil
}

func (c *compiler) expr(e Expr) error {
	switch t := e.(type) {
	case *NumLit:
		if t.IsFloat {
			c.emit(Op{Kind: OpPushFloat, Flt: t.Float, Line: t.Line, Col: t.Col})
		} else {
			c.emit(Op{Kind: OpPushInt, Num: t.Int, Line: t.Line, Col: t.Col})
		}
	case *StrLit:
		c.emit(Op{Kind: OpPushStr, Str: t.V, Line: t.Line, Col: t.Col})
	case *BoolLit:
		c.emit(Op{Kind: OpPushBool, Bol: t.V, Line: t.Line, Col: t.Col})
	case *NilLit:
		c.emit(Op{Kind: OpPushNil, Line: t.Line, Col: t.Col})
	case *NameE:
		c.emit(Op{Kind: OpLoad, Name: t.X, Line: t.Line, Col: t.Col})
	case *BinE:
		code, ok := binCodes[t.Op]
		if t.Op == "??" {
			// Nullish coalescing: left ?? right
			// Evaluate left, duplicate on stack, jump past right if not nil.
			if err := c.expr(t.X); err != nil {
				return err
			}
			c.emit(Op{Kind: OpDup, Line: t.Line, Col: t.Col})
			skip := c.emit(Op{Kind: OpJumpIfNotNil, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpPop, Line: t.Line, Col: t.Col})
			if err := c.expr(t.Y); err != nil {
				return err
			}
			c.patch(skip, len(c.ops))
			return nil
		}
		if !ok {
			return c.perr(t.Pos, "unknown operator %q", t.Op)
		}
		if t.Op == "and" || t.Op == "or" {
			if err := c.expr(t.X); err != nil {
				return err
			}
			c.emit(Op{Kind: OpDup, Line: t.Line, Col: t.Col})
			jmp := OpJumpIfNot
			if t.Op == "or" {
				jmp = OpJumpIf
			}
			skip := c.emit(Op{Kind: jmp, Line: t.Line, Col: t.Col})
			c.emit(Op{Kind: OpPop, Line: t.Line, Col: t.Col})
			if err := c.expr(t.Y); err != nil {
				return err
			}
			c.patch(skip, len(c.ops))
			return nil
		}
		if err := c.expr(t.X); err != nil {
			return err
		}
		if err := c.expr(t.Y); err != nil {
			return err
		}
		c.emit(Op{Kind: OpBin, Num: int64(code), Line: t.Line, Col: t.Col})
	case *CondE:
		if err := c.expr(t.Cond); err != nil {
			return err
		}
		otherwise := c.emit(Op{Kind: OpJumpIfNot, Line: t.Line, Col: t.Col})
		if err := c.expr(t.Yes); err != nil {
			return err
		}
		end := c.emit(Op{Kind: OpJump, Line: t.Line, Col: t.Col})
		c.patch(otherwise, len(c.ops))
		if err := c.expr(t.No); err != nil {
			return err
		}
		c.patch(end, len(c.ops))
	case *UnE:
		if err := c.expr(t.X); err != nil {
			return err
		}
		var code byte
		switch t.Op {
		case "-":
			code = uoNeg
		case "not":
			code = uoNot
		default:
			return c.perr(t.Pos, "unknown unary operator %q", t.Op)
		}
		c.emit(Op{Kind: OpUn, Num: int64(code), Line: t.Line, Col: t.Col})
	case *CallE:
		if err := c.expr(t.Fn); err != nil {
			return err
		}
		for _, a := range t.Args {
			if err := c.expr(a); err != nil {
				return err
			}
		}
		c.emit(Op{Kind: OpCall, Num: int64(len(t.Args)), Line: t.Line, Col: t.Col})
	case *IndexE:
		if err := c.expr(t.X); err != nil {
			return err
		}
		if err := c.expr(t.Key); err != nil {
			return err
		}
		c.emit(Op{Kind: OpIndex, Line: t.Line, Col: t.Col})
	case *SafeIndexE:
		if err := c.expr(t.X); err != nil {
			return err
		}
		if err := c.expr(t.Key); err != nil {
			return err
		}
		c.emit(Op{Kind: OpSafeIndex, Line: t.Line, Col: t.Col})
	case *AttrE:
		if err := c.expr(t.X); err != nil {
			return err
		}
		c.emit(Op{Kind: OpPushStr, Str: t.Name, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpIndex, Line: t.Line, Col: t.Col})
	case *SafeAttrE:
		if err := c.expr(t.X); err != nil {
			return err
		}
		c.emit(Op{Kind: OpPushStr, Str: t.Name, Line: t.Line, Col: t.Col})
		c.emit(Op{Kind: OpSafeIndex, Line: t.Line, Col: t.Col})
	case *SliceE:
		if t.Safe {
			// x?[lo:hi]: nil if x is nil
			if err := c.expr(t.X); err != nil {
				return err
			}
			// OpJumpIfNotNil pops its test value, so this leaves the
			// container on the stack when it is non-nil.
			c.emit(Op{Kind: OpDup, Line: t.Line, Col: t.Col})
			use := c.emit(Op{Kind: OpJumpIfNotNil, Line: t.Line, Col: t.Col})
			end := c.emit(Op{Kind: OpJump, Line: t.Line, Col: t.Col})
			c.patch(use, len(c.ops))
			if err := c.emitSliceBody(t.Lo, t.Hi, t.Line, t.Col); err != nil {
				return err
			}
			c.patch(end, len(c.ops))
			return nil
		}
		if err := c.expr(t.X); err != nil {
			return err
		}
		if err := c.emitSliceBody(t.Lo, t.Hi, t.Line, t.Col); err != nil {
			return err
		}
	case *SafeChainE:
		if err := c.expr(t.Base); err != nil {
			return err
		}
		for _, st := range t.Prefix {
			if err := c.emitChainStep(st, t.Line, t.Col); err != nil {
				return err
			}
		}
		var endJumps []int
		for _, st := range t.Steps {
			// decision: if the current value is nil, the whole chain yields
			// nil. OpJumpIfNotNil pops its test value, leaving the container
			// on the stack when it is non-nil.
			c.emit(Op{Kind: OpDup, Line: t.Line, Col: t.Col})
			use := c.emit(Op{Kind: OpJumpIfNotNil, Line: t.Line, Col: t.Col})
			endJumps = append(endJumps, c.emit(Op{Kind: OpJump, Line: t.Line, Col: t.Col}))
			c.patch(use, len(c.ops))
			if err := c.emitChainStep(st, t.Line, t.Col); err != nil {
				return err
			}
		}
		for _, j := range endJumps {
			c.patch(j, len(c.ops))
		}
	case *ListLit:
		for _, it := range t.Items {
			if err := c.expr(it); err != nil {
				return err
			}
		}
		c.emit(Op{Kind: OpMakeList, Num: int64(len(t.Items)), Line: t.Line, Col: t.Col})
	case *DictLit:
		for _, pr := range t.Pairs {
			if err := c.expr(pr[0]); err != nil {
				return err
			}
			if err := c.expr(pr[1]); err != nil {
				return err
			}
		}
		c.emit(Op{Kind: OpMakeDict, Num: int64(len(t.Pairs)), Line: t.Line, Col: t.Col})
	case *FnExpr:
		bc := &compiler{}
		if err := bc.stmts(t.Body, false); err != nil {
			return err
		}
		c.emit(Op{Kind: OpMakeFn, Name: "<anon>", Args: t.Params, ParamTypes: t.ParamTypes, Ret: t.Ret, Body: bc.ops, Line: t.Line, Col: t.Col})
	case *FStrLit:
		// Compile f-string as a series of str() calls joined with '+'
		if len(t.Parts) == 0 {
			c.emit(Op{Kind: OpPushStr, Str: "", Line: t.Line, Col: t.Col})
			return nil
		}
		// Emit first part
		first := true
		for _, part := range t.Parts {
			var partEmpty bool
			if !part.IsExpr && part.Lit == "" {
				if first {
					c.emit(Op{Kind: OpPushStr, Str: "", Line: t.Line, Col: t.Col})
					first = false
				}
				partEmpty = true
			}
			if !partEmpty {
				if !part.IsExpr {
					c.emit(Op{Kind: OpPushStr, Str: part.Lit, Line: t.Line, Col: t.Col})
				} else {
					// Load str builtin and call it
					c.emit(Op{Kind: OpLoad, Name: "str", Line: t.Line, Col: t.Col})
					if err := c.expr(part.Expr); err != nil {
						return err
					}
					c.emit(Op{Kind: OpCall, Num: 1, Line: t.Line, Col: t.Col})
				}
				if !first {
					c.emit(Op{Kind: OpBin, Num: int64(boAdd), Line: t.Line, Col: t.Col})
				}
				first = false
			}
		}
		if first {
			// all parts empty
			c.emit(Op{Kind: OpPushStr, Str: "", Line: t.Line, Col: t.Col})
		}
	default:
		return c.perr(Pos{}, "unsupported expression")
	}
	return nil
}

func (c *compiler) emitSliceBody(lo, hi Expr, line, col int) error {
	if lo == nil {
		c.emit(Op{Kind: OpPushNil, Line: line, Col: col})
	} else if err := c.expr(lo); err != nil {
		return err
	}
	if hi == nil {
		c.emit(Op{Kind: OpPushNil, Line: line, Col: col})
	} else if err := c.expr(hi); err != nil {
		return err
	}
	c.emit(Op{Kind: OpSlice, Line: line, Col: col})
	return nil
}

func (c *compiler) emitChainStep(st ChainStep, line, col int) error {
	if st.Idx && st.Slice {
		return c.emitSliceBody(st.Lo, st.Hi, line, col)
	}
	if st.Idx {
		if err := c.expr(st.Lo); err != nil {
			return err
		}
		op := OpIndex
		if st.Safe {
			op = OpSafeIndex
		}
		c.emit(Op{Kind: op, Line: line, Col: col})
		return nil
	}
	c.emit(Op{Kind: OpPushStr, Str: st.Name, Line: line, Col: col})
	op := OpIndex
	if st.Safe {
		op = OpSafeIndex
	}
	c.emit(Op{Kind: op, Line: line, Col: col})
	return nil
}
