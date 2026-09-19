package snow

import (
	"fmt"
	"strings"
)

// Pos is a source position.
type Pos struct{ Line, Col int }

// Stmt is a Snow statement.
type Stmt interface{ stmt() }

// Expr is a Snow expression.
type Expr interface{ expr() }

// Program is a parsed source file.
type Program struct {
	Stmts      []Stmt
	Incomplete bool
	SrcName    string
}

type UseStmt struct {
	Pos
	Path   []string // snow.api or a/b (file module)
	Alias  string
	Import bool // import loads a .snow file; using may load stdlib or a file
}
type AssignStmt struct {
	Pos
	Names []string
	Op    byte // 0 for '=', otherwise the binary op code
	Vals  []Expr
	Type  string // declared type, e.g. "str", "str[]"; "" keeps the variable dynamic
	Vis   string // "pub", "priv" or "" (default: pub)
}
type SetAttrStmt struct {
	Pos
	Container Expr
	Name      string
	Op        byte // 0 for '=', otherwise the binary op code
	Val       Expr
}
type SetIndexStmt struct {
	Pos
	Container Expr
	Key       Expr
	Op        byte // 0 for '=', otherwise the binary op code
	Val       Expr
}
type FnStmt struct {
	Pos
	Name       string
	Params     []string
	ParamTypes []string // len(Params), "" when a param is untyped
	Ret        string   // declared return type, "" when untyped
	Body       []Stmt
	Vis        string // "pub", "priv" or "" (default: pub)
}
type ReturnStmt struct {
	Pos
	Vals []Expr
}
type IfStmt struct {
	Pos
	Conds  []Expr
	Bodies [][]Stmt
	Else   []Stmt
}
type ForStmt struct {
	Pos
	Name  string
	Name2 string
	Iter  Expr
	Where Expr
	Body  []Stmt
}
type WithStmt struct {
	Pos
	Expr Expr
	Name string
	Body []Stmt
}
type WhileStmt struct {
	Pos
	Cond Expr
	Body []Stmt
}
type LoopStmt struct {
	Pos
	Break bool
}
type TryStmt struct {
	Pos
	TryBody   []Stmt
	CatchVar  string
	CatchBody []Stmt
}
type MatchStmt struct {
	Pos
	Target Expr
	Cases  []MatchCase
}
type MatchCase struct {
	Vals []Expr
	Body []Stmt
}
type ExprStmt struct {
	Pos
	X Expr
}

type NumLit struct {
	Pos
	IsFloat bool
	Int     int64
	Float   float64
}
type StrLit struct {
	Pos
	V string
}
type BoolLit struct {
	Pos
	V bool
}
type NilLit struct{ Pos }
type NameE struct {
	Pos
	X string
}
type BinE struct {
	Pos
	Op string
	X  Expr
	Y  Expr
}
type UnE struct {
	Pos
	Op string
	X  Expr
}
type CallE struct {
	Pos
	Fn   Expr
	Args []Expr
}
type IndexE struct {
	Pos
	X   Expr
	Key Expr
}

// SafeIndexE is a safe subscript: x?[key] returns nil if x is nil or key is missing.
type SafeIndexE struct {
	Pos
	X   Expr
	Key Expr
}
type AttrE struct {
	Pos
	X    Expr
	Name string
}

// SafeAttrE is a safe attribute access: x?.name returns nil if x is nil.
type SafeAttrE struct {
	Pos
	X    Expr
	Name string
}

// SliceE is a slice: x[lo:hi]. A nil Lo/Hi means an open end. Safe means x?[lo:hi].
type SliceE struct {
	Pos
	X    Expr
	Lo   Expr
	Hi   Expr
	Safe bool
}

// SafeChainE short-circuits an accessor chain at the first nil: the chain
// after the first safe accessor evaluates to nil whenever any intermediate
// value preceding the last step is nil. Prefix holds plain accessors before
// the first safe one; Steps holds the safe accessor and everything after it.
type SafeChainE struct {
	Pos
	Base   Expr
	Prefix []ChainStep
	Steps  []ChainStep
}

// ChainStep is one accessor in a chain: an attribute or a subscript.
type ChainStep struct {
	Safe  bool
	Idx   bool   // subscript (index or slice) instead of a dot attribute
	Slice bool   // subscript is a slice, using Lo/Hi
	Name  string // attribute name when !Idx
	Lo    Expr   // index key, or slice start (nil = omitted)
	Hi    Expr   // slice end (nil = omitted, or when not a slice)
}

type ListLit struct {
	Pos
	Items []Expr
}
type DictLit struct {
	Pos
	Pairs [][2]Expr
}

type FnExpr struct {
	Pos
	Params     []string
	ParamTypes []string
	Ret        string
	Body       []Stmt
}

// FStrLit is an f-string literal: Parts alternates [literal, expr, literal, ...].
type FStrLit struct {
	Pos
	Parts []FStrPart
}

type FStrPart struct {
	IsExpr bool
	Lit    string
	Expr   Expr
}

func (*UseStmt) stmt()      {}
func (*AssignStmt) stmt()   {}
func (*SetAttrStmt) stmt()  {}
func (*SetIndexStmt) stmt() {}
func (*FnStmt) stmt()       {}
func (*ReturnStmt) stmt()   {}
func (*IfStmt) stmt()       {}
func (*ForStmt) stmt()      {}
func (*WhileStmt) stmt()    {}
func (*LoopStmt) stmt()     {}
func (*TryStmt) stmt()      {}
func (*MatchStmt) stmt()    {}
func (*ExprStmt) stmt()     {}
func (*WithStmt) stmt()     {}
func (*NumLit) expr()       {}
func (*StrLit) expr()       {}
func (*BoolLit) expr()      {}
func (*NilLit) expr()       {}
func (*NameE) expr()        {}
func (*BinE) expr()         {}
func (*UnE) expr()          {}
func (*CallE) expr()        {}
func (*IndexE) expr()       {}
func (*SafeIndexE) expr()   {}
func (*AttrE) expr()        {}
func (*SafeAttrE) expr()    {}
func (*SliceE) expr()       {}
func (*SafeChainE) expr()   {}
func (*ListLit) expr()      {}
func (*DictLit) expr()      {}
func (*FnExpr) expr()       {}
func (*FStrLit) expr()      {}

type parser struct {
	toks       []Tok
	pos        int
	name       string
	src        string
	fnN        int
	loopN      int
	incomplete bool
	atTop      bool // true while parsing the file's top level
}

// Parse parses Snow source into a program.
func Parse(src, name string) (*Program, error) {
	toks, err := Tokenize(src, name)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks, name: name, src: src}
	return p.parseProgram()
}

func (p *parser) errf(t Tok, format string, a ...any) error {
	return &Errat{p.name, t.Line, t.Col, fmt.Errorf(format, a...)}
}

func (p *parser) peek() Tok {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return p.toks[len(p.toks)-1]
}

func (p *parser) next() Tok {
	t := p.peek()
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *parser) parseProgram() (*Program, error) {
	prog := &Program{SrcName: p.name}
	p.atTop = true
	defer func() { p.atTop = false }()
	for {
		k := p.peek()
		switch k.Kind {
		case tEOF:
			prog.Incomplete = p.incomplete || sourceOpen(p.src)
			return prog, nil
		case tNewline:
			p.next()
		case tIndent, tDedent:
			return nil, p.errf(k, "unexpected indentation")
		default:
			s, err := p.parseStmt()
			if err != nil {
				return nil, err
			}
			prog.Stmts = append(prog.Stmts, s)
		}
	}
}

func sourceOpen(src string) bool {
	trimmed := strings.TrimRight(src, " \t\r\n")
	if strings.HasSuffix(trimmed, ":") {
		return true
	}
	var pCount, bCount, cCount int
	inStr := false
	var strQuote byte
	escaped := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inStr {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == strQuote {
				inStr = false
			}
			continue
		}
		if c == '#' {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '"' || c == '\'' {
			inStr = true
			strQuote = c
			continue
		}
		switch c {
		case '(':
			pCount++
		case ')':
			if pCount > 0 {
				pCount--
			}
		case '[':
			bCount++
		case ']':
			if bCount > 0 {
				bCount--
			}
		case '{':
			cCount++
		case '}':
			if cCount > 0 {
				cCount--
			}
		}
	}
	if inStr || pCount > 0 || bCount > 0 || cCount > 0 {
		return true
	}

	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if strings.HasSuffix(src, "\n\n") {
		return false
	}
	for i := len(lines) - 1; i >= 0; i-- {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		if strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t") {
			return true
		}
		break
	}
	return false
}

func (p *parser) parseStmt() (Stmt, error) {
	k := p.peek()
	if k.Kind == tIdent {
		switch k.Text {
		case "using":
			return p.parseUsing()
		case "import":
			return p.parseImport()
		case "pub", "priv":
			return p.parseVisStmt(k.Text)
		case "fn":
			return p.parseFn()
		case "return":
			return p.parseReturn()
		case "if":
			return p.parseIf()
		case "elif", "else":
			return nil, p.errf(k, "'%s' without matching 'if'", k.Text)
		case "for":
			return p.parseFor()
		case "while":
			return p.parseWhile()
		case "break":
			return p.parseLoopCtrl(true)
		case "continue":
			return p.parseLoopCtrl(false)
		case "try":
			return p.parseTry()
		case "match":
			return p.parseMatch()
		case "with":
			return p.parseWith()
		case "catch":
			return nil, p.errf(k, "'catch' without matching 'try'")
		}
	}
	return p.parseSimple()
}

var assignTokens = map[TokKind]byte{
	tAssign:       0,
	tPlusEq:       boAdd,
	tMinusEq:      boSub,
	tStarEq:       boMul,
	tSlashEq:      boDiv,
	tSlashSlashEq: boFloorDiv,
	tPctEq:        boMod,
	tQQEq:         boQQEq,
}

func boToStr(op byte) string {
	switch op {
	case boAdd:
		return "+"
	case boSub:
		return "-"
	case boMul:
		return "*"
	case boDiv:
		return "/"
	case boFloorDiv:
		return "//"
	case boMod:
		return "%"
	}
	return ""
}

func (p *parser) parseSimple() (Stmt, error) {
	vals, err := p.parseExprList()
	if err != nil {
		return nil, err
	}
	if p.peek().Kind == tColon {
		return p.parseTypedAssign(vals)
	}
	if op, ok := assignTokens[p.peek().Kind]; ok {
		eq := p.next()
		if len(vals) == 1 {
			switch tgt := vals[0].(type) {
			case *AttrE:
				rhs, err := p.parseValueList()
				if err != nil {
					return nil, err
				}
				if len(rhs) != 1 {
					return nil, p.errf(eq, "attribute assignment expects a single value")
				}
				val := rhs[0]
				if op != 0 {
					val = &BinE{Pos: Pos{eq.Line, eq.Col}, Op: boToStr(op), X: tgt, Y: rhs[0]}
				}
				if err := p.lineEnd(); err != nil {
					return nil, err
				}
				return &SetAttrStmt{Pos: Pos{eq.Line, eq.Col}, Container: tgt.X, Name: tgt.Name, Op: 0, Val: val}, nil
			case *IndexE:
				rhs, err := p.parseValueList()
				if err != nil {
					return nil, err
				}
				if len(rhs) != 1 {
					return nil, p.errf(eq, "index assignment expects a single value")
				}
				val := rhs[0]
				if op != 0 {
					val = &BinE{Pos: Pos{eq.Line, eq.Col}, Op: boToStr(op), X: tgt, Y: rhs[0]}
				}
				if err := p.lineEnd(); err != nil {
					return nil, err
				}
				return &SetIndexStmt{Pos: Pos{eq.Line, eq.Col}, Container: tgt.X, Key: tgt.Key, Op: 0, Val: val}, nil
			}
		}
		names := make([]string, len(vals))
		for i, v := range vals {
			n, ok := v.(*NameE)
			if !ok {
				ps := posOf(v)
				return nil, p.errf(Tok{Line: ps.Line, Col: ps.Col}, "invalid assignment target")
			}
			names[i] = n.X
		}
		rhs, err := p.parseValueList()
		if err != nil {
			return nil, err
		}
		if op != 0 && len(names) != 1 {
			return nil, p.errf(eq, "augmented assignment needs a single name")
		}
		if op != 0 && len(rhs) != 1 {
			return nil, p.errf(eq, "augmented assignment expects a single value")
		}
		if err := p.lineEnd(); err != nil {
			return nil, err
		}
		return &AssignStmt{Pos: Pos{eq.Line, eq.Col}, Names: names, Op: op, Vals: rhs}, nil
	}
	if len(vals) > 1 {
		ps := posOf(vals[0])
		return nil, p.errf(Tok{Line: ps.Line, Col: ps.Col}, "multiple values need an assignment (name = ...)")
	}
	if err := p.lineEnd(); err != nil {
		return nil, err
	}
	return &ExprStmt{posOf(vals[0]), vals[0]}, nil
}

// parseTypedAssign handles an optional gradual type annotation:
// name: int = 1, names: str[] = ["Ada"]. An annotation is checked at runtime;
// unannotated variables remain fully dynamic.
func (p *parser) parseTypedAssign(vals []Expr) (Stmt, error) {
	peek := p.peek()
	if len(vals) != 1 {
		return nil, p.errf(peek, "invalid type declaration: type applies to a single name")
	}
	n, ok := vals[0].(*NameE)
	if !ok {
		ps := posOf(vals[0])
		return nil, p.errf(Tok{Line: ps.Line, Col: ps.Col}, "invalid assignment target")
	}
	p.next() // consume ':'
	typeName, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if p.peek().Kind != tAssign {
		return nil, p.errf(p.peek(), "expected '=' in type declaration")
	}
	p.next()
	rhs, err := p.parseValueList()
	if err != nil {
		return nil, err
	}
	if err := p.lineEnd(); err != nil {
		return nil, err
	}
	return &AssignStmt{Pos: Pos{n.Line, n.Col}, Names: []string{n.X}, Vals: rhs, Type: typeName}, nil
}

func validTypeName(e string) bool {
	switch e {
	case "str", "int", "float", "bool", "dict", "list", "any":
		return true
	}
	return false
}

// lineEnd consumes the newline that ends a simple statement and
// rejects leftover tokens on the same line.
func (p *parser) lineEnd() error {
	if p.pos > 0 && p.toks[p.pos-1].Kind == tDedent {
		return nil
	}
	k := p.peek()
	switch k.Kind {
	case tNewline:
		p.next()
		return nil
	case tEOF, tDedent:
		return nil
	}
	return p.errf(k, "unexpected %s", k.String())
}

func (p *parser) parseUsing() (Stmt, error) {
	st := p.next()
	first, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	path := []string{first}
	for p.peek().Kind == tDot {
		p.next()
		seg, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		path = append(path, seg)
	}
	alias := path[len(path)-1]
	if p.peek().Kind == tIdent && p.peek().Text == "as" {
		p.next()
		a, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		alias = a
	}
	return &UseStmt{Pos: Pos{st.Line, st.Col}, Path: path, Alias: alias}, nil
}

// parseImport parses an import statement. It follows the same dotted
// path syntax as using, but always resolves to a .snow file relative
// to the current file's directory:
//
//	import lib.utils
//	import lib.utils as u
func (p *parser) parseImport() (Stmt, error) {
	st := p.next()
	first, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	path := []string{first}
	for p.peek().Kind == tDot {
		p.next()
		seg, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		path = append(path, seg)
	}
	alias := path[len(path)-1]
	if p.peek().Kind == tIdent && p.peek().Text == "as" {
		p.next()
		a, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		alias = a
	}
	return &UseStmt{Pos{st.Line, st.Col}, path, alias, true}, nil
}

// parseVisStmt parses a 'pub' or 'priv' visibility modifier applied to a
// top-level function or variable declaration. Anything else is an error.
func (p *parser) parseVisStmt(mod string) (Stmt, error) {
	st := p.next() // consume 'pub' / 'priv'
	if !p.atTop {
		return nil, p.errf(st, "'%s' is only allowed at the top level", mod)
	}
	k := p.peek()
	var s Stmt
	var err error
	if k.Kind == tIdent && k.Text == "fn" {
		s, err = p.parseFn()
	} else {
		s, err = p.parseSimple()
	}
	if err != nil {
		return nil, err
	}
	switch t := s.(type) {
	case *FnStmt:
		t.Vis = mod
	case *AssignStmt:
		t.Vis = mod
	default:
		return nil, p.errf(k, "'%s' only applies to functions and variables", mod)
	}
	return s, nil
}

func (p *parser) expectIdent() (string, error) {
	k := p.peek()
	if k.Kind != tIdent || isReserved(k.Text) {
		return "", p.errf(k, "expected a name")
	}
	p.next()
	return k.Text, nil
}

func (p *parser) parseFn() (Stmt, error) {
	st := p.next()
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if k := p.peek(); k.Kind != tLParen {
		return nil, p.errf(k, "expected '(' after function name")
	}
	p.next()
	params, ptypes, err := p.parseParams()
	if err != nil {
		return nil, err
	}
	if k := p.peek(); k.Kind != tRParen {
		return nil, p.errf(k, "expected ')' after parameters")
	}
	p.next()
	ret, err := p.parseOptReturn()
	if err != nil {
		return nil, err
	}
	if _, err := p.expectColon(); err != nil {
		return nil, err
	}
	p.fnN++
	body, err := p.parseSuite()
	p.fnN--
	if err != nil {
		return nil, err
	}
	return &FnStmt{Pos: Pos{st.Line, st.Col}, Name: name, Params: params, ParamTypes: ptypes, Ret: ret, Body: body}, nil
}

func (p *parser) parseParams() ([]string, []string, error) {
	var names, types []string
	if p.peek().Kind == tRParen {
		return names, types, nil
	}
	for {
		n, err := p.expectIdent()
		if err != nil {
			return nil, nil, err
		}
		typ := ""
		if p.peek().Kind == tColon {
			p.next()
			if typ, err = p.parseType(); err != nil {
				return nil, nil, err
			}
		}
		names = append(names, n)
		types = append(types, typ)
		k := p.peek()
		switch k.Kind {
		case tRParen:
			return names, types, nil
		case tComma:
			p.next()
			if p.peek().Kind == tRParen {
				return names, types, nil
			}
		case tIdent:
			// optional commas: fn f(a b):
		default:
			return nil, nil, p.errf(k, "expected a parameter name or ')'")
		}
	}
}

// parseOptReturn parses an optional "-> type" after a function signature.
func (p *parser) parseOptReturn() (string, error) {
	if p.peek().Kind != tArrow {
		return "", nil
	}
	p.next()
	return p.parseType()
}

// parseType parses a type: a scalar (str, int, ...) or a list of them (str[]).
func (p *parser) parseType() (string, error) {
	start := p.peek()
	elem, err := p.expectIdent()
	if err != nil {
		return "", err
	}
	if !validTypeName(elem) {
		return "", p.errf(start, "unknown type %q (use str, int, float, bool, dict, list or any)", elem)
	}
	if p.peek().Kind == tLBrack {
		p.next()
		if p.peek().Kind != tRBrack {
			return "", p.errf(p.peek(), "expected ']' in type")
		}
		p.next()
		return elem + "[]", nil
	}
	return elem, nil
}

func (p *parser) expectColon() (Tok, error) {
	k := p.peek()
	if k.Kind != tColon {
		return k, p.errf(k, "expected ':'")
	}
	p.next()
	return k, nil
}

// parseSuite parses the block that follows the ':' of a compound statement.
func (p *parser) parseSuite() ([]Stmt, error) {
	k := p.peek()
	if k.Kind == tNewline {
		p.next()
		k = p.peek()
		if k.Kind == tIndent {
			p.next()
			return p.parseBlockStmts()
		}
		if k.Kind == tEOF {
			return nil, &Errat{p.name, k.Line, k.Col, ErrIncomplete}
		}
		return nil, p.errf(k, "expected an indented block")
	}
	s, err := p.parseStmt()
	if err != nil {
		return nil, err
	}
	return []Stmt{s}, nil
}

func (p *parser) parseBlockStmts() ([]Stmt, error) {
	var stmts []Stmt
	for {
		k := p.peek()
		switch k.Kind {
		case tDedent:
			p.next()
			return stmts, nil
		case tEOF:
			p.incomplete = true
			return stmts, nil
		case tNewline:
			p.next()
		case tIndent:
			return nil, p.errf(k, "unexpected indentation")
		case tRParen:
			// closing paren of outer call — stop block, don't consume
			return stmts, nil
		default:
			s, err := p.parseStmt()
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, s)
			if p.peek().Kind == tNewline {
				p.next()
			}
		}
	}
}

func (p *parser) parseIf() (Stmt, error) {
	st := p.next()
	cond, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expectColon(); err != nil {
		return nil, err
	}
	body, err := p.parseSuite()
	if err != nil {
		return nil, err
	}
	conds := []Expr{cond}
	bodies := [][]Stmt{body}
	var els []Stmt
	for p.peek().Kind == tIdent && p.peek().Text == "elif" {
		p.next()
		c, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expectColon(); err != nil {
			return nil, err
		}
		b, err := p.parseSuite()
		if err != nil {
			return nil, err
		}
		conds = append(conds, c)
		bodies = append(bodies, b)
	}
	if p.peek().Kind == tIdent && p.peek().Text == "else" {
		p.next()
		if _, err := p.expectColon(); err != nil {
			return nil, err
		}
		if els, err = p.parseSuite(); err != nil {
			return nil, err
		}
	}
	return &IfStmt{Pos{st.Line, st.Col}, conds, bodies, els}, nil
}

func (p *parser) parseFor() (Stmt, error) {
	st := p.next()
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	name2 := ""
	if p.peek().Kind == tComma {
		p.next()
		name2, err = p.expectIdent()
		if err != nil {
			return nil, err
		}
	}
	k := p.peek()
	if k.Kind != tIdent || k.Text != "in" {
		return nil, p.errf(k, "expected 'in' in for statement")
	}
	p.next()
	iter, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	var whereExpr Expr
	if p.peek().Kind == tIdent && p.peek().Text == "where" {
		p.next()
		whereExpr, err = p.parseOr()
		if err != nil {
			return nil, err
		}
	}
	if _, err := p.expectColon(); err != nil {
		return nil, err
	}
	p.loopN++
	body, err := p.parseSuite()
	p.loopN--
	if err != nil {
		return nil, err
	}
	return &ForStmt{Pos{st.Line, st.Col}, name, name2, iter, whereExpr, body}, nil
}

func (p *parser) parseWhile() (Stmt, error) {
	st := p.next()
	cond, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expectColon(); err != nil {
		return nil, err
	}
	p.loopN++
	body, err := p.parseSuite()
	p.loopN--
	if err != nil {
		return nil, err
	}
	return &WhileStmt{Pos{st.Line, st.Col}, cond, body}, nil
}

func (p *parser) parseLoopCtrl(break_ bool) (Stmt, error) {
	st := p.next()
	if p.loopN == 0 {
		if break_ {
			return nil, p.errf(st, "'break' outside of a loop")
		}
		return nil, p.errf(st, "'continue' outside of a loop")
	}
	return &LoopStmt{Pos{st.Line, st.Col}, break_}, nil
}

// parseMatch parses a match statement:
//
//	match value:
//	  case 1, 2:
//	    ...
//	  case "x":
//	    ...
//	  case _:
//	    ...
func (p *parser) parseMatch() (Stmt, error) {
	p.next() // consume 'match'
	target, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if _, err := p.expectColon(); err != nil {
		return nil, err
	}
	if k := p.peek(); k.Kind != tNewline {
		return nil, p.errf(k, "expected a newline after 'match ...:'")
	}
	p.next()
	if k := p.peek(); k.Kind != tIndent {
		return nil, p.errf(k, "expected an indented block in match")
	}
	p.next()
	ms := &MatchStmt{Pos: posOf(target), Target: target}
	seenWildcard := false
	for {
		k := p.peek()
		switch k.Kind {
		case tDedent:
			p.next()
			return ms, nil
		case tEOF:
			p.incomplete = true
			return ms, nil
		case tNewline:
			p.next()
		case tRParen:
			// closing paren of an enclosing call — stop, don't consume
			return ms, nil
		default:
			if k.Kind != tIdent || k.Text != "case" {
				if k.Kind == tIdent && k.Text == "else" {
					return nil, p.errf(k, "'else' is not allowed in match; use 'case _:' for the default branch")
				}
				return nil, p.errf(k, "expected 'case' in match")
			}
			p.next()
			if seenWildcard {
				return nil, p.errf(k, "'case _' must be the last case in match")
			}
			var vals []Expr
			for {
				v, err := p.parseOr()
				if err != nil {
					return nil, err
				}
				vals = append(vals, v)
				if p.peek().Kind == tComma {
					p.next()
					continue
				}
				break
			}
			for _, v := range vals {
				if name, ok := v.(*NameE); ok && name.X == "_" {
					if len(vals) != 1 {
						return nil, p.errf(k, "'_' wildcard must be the only value in a case")
					}
					seenWildcard = true
				}
			}
			if _, err := p.expectColon(); err != nil {
				return nil, err
			}
			body, err := p.parseSuite()
			if err != nil {
				return nil, err
			}
			ms.Cases = append(ms.Cases, MatchCase{Vals: vals, Body: body})
			if p.peek().Kind == tNewline {
				p.next()
			}
		}
	}
}

func (p *parser) parseTry() (Stmt, error) {
	st := p.next() // consume 'try'
	if _, err := p.expectColon(); err != nil {
		return nil, err
	}
	tryBody, err := p.parseSuite()
	if err != nil {
		return nil, err
	}
	k := p.peek()
	if k.Kind != tIdent || k.Text != "catch" {
		return nil, p.errf(k, "expected 'catch' after try block")
	}
	p.next() // consume 'catch'
	if p.peek().Kind != tIdent || isReserved(p.peek().Text) {
		return nil, p.errf(p.peek(), "expected a name after 'catch'")
	}
	catchVar := p.next().Text
	if _, err := p.expectColon(); err != nil {
		return nil, err
	}
	catchBody, err := p.parseSuite()
	if err != nil {
		return nil, err
	}
	return &TryStmt{Pos{st.Line, st.Col}, tryBody, catchVar, catchBody}, nil
}

func (p *parser) parseWith() (Stmt, error) {
	st := p.next() // consume 'with'
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	name := ""
	if p.peek().Kind == tIdent && p.peek().Text == "as" {
		p.next()
		if p.peek().Kind != tIdent || isReserved(p.peek().Text) {
			return nil, p.errf(p.peek(), "expected a name after 'as'")
		}
		name = p.next().Text
	}
	if _, err := p.expectColon(); err != nil {
		return nil, err
	}
	body, err := p.parseSuite()
	if err != nil {
		return nil, err
	}
	return &WithStmt{Pos{st.Line, st.Col}, expr, name, body}, nil
}

func (p *parser) parseReturn() (Stmt, error) {
	st := p.next()
	if p.fnN == 0 {
		return nil, p.errf(st, "'return' outside of a function")
	}
	var vals []Expr
	if exprStart(p.peek()) {
		var err error
		if vals, err = p.parseExprList(); err != nil {
			return nil, err
		}
	}
	if err := p.lineEnd(); err != nil {
		return nil, err
	}
	return &ReturnStmt{Pos{st.Line, st.Col}, vals}, nil
}

// ---- expressions ----

func (p *parser) parseExprList() ([]Expr, error) {
	var xs []Expr
	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	xs = append(xs, e)
	for {
		k := p.peek()
		switch {
		case k.Kind == tComma:
			p.next()
			if !exprStart(p.peek()) {
				return xs, nil
			}
			e, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			xs = append(xs, e)
		case exprStart(k):
			if p.pos > 0 && p.toks[p.pos-1].Kind == tDedent {
				return xs, nil
			}
			e, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			xs = append(xs, e)
		default:
			return xs, nil
		}
	}
}

func (p *parser) parseValueList() ([]Expr, error) {
	return p.parseExprList()
}

func (p *parser) parseOr() (Expr, error) {
	l, err := p.parseNullCoalesce()
	if err != nil {
		return nil, err
	}
	for p.peek().Kind == tIdent && p.peek().Text == "or" {
		p.next()
		r, err := p.parseNullCoalesce()
		if err != nil {
			return nil, err
		}
		l = &BinE{Pos{posLine(l), posCol(l)}, "or", l, r}
	}
	return l, nil
}

// parseNullCoalesce handles the ?? operator (lower precedence than and/or, higher than or).
func (p *parser) parseNullCoalesce() (Expr, error) {
	l, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Kind == tQQ {
		p.next()
		r, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		l = &BinE{Pos{posLine(l), posCol(l)}, "??", l, r}
	}
	return l, nil
}

func (p *parser) parseAnd() (Expr, error) {
	l, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().Kind == tIdent && p.peek().Text == "and" {
		p.next()
		r, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		l = &BinE{Pos{posLine(l), posCol(l)}, "and", l, r}
	}
	return l, nil
}

func (p *parser) parseNot() (Expr, error) {
	k := p.peek()
	if k.Kind == tIdent && k.Text == "not" {
		p.next()
		x, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &UnE{Pos{posLine(x), posCol(x)}, "not", x}, nil
	}
	return p.parseCmp()
}

var cmpOps = map[TokKind]string{tEq: "==", tNe: "!=", tLt: "<", tLe: "<=", tGt: ">", tGe: ">="}

func (p *parser) parseCmp() (Expr, error) {
	x, err := p.parseAdd()
	if err != nil {
		return nil, err
	}
	var ops []string
	vals := []Expr{x}
	for {
		op := ""
		if o, ok := cmpOps[p.peek().Kind]; ok {
			op = o
			p.next()
		} else if p.peek().Kind == tIdent && p.peek().Text == "in" {
			op = "in"
			p.next()
		} else if p.peek().Kind == tIdent && p.peek().Text == "not" &&
			p.pos+1 < len(p.toks) && p.toks[p.pos+1].Kind == tIdent && p.toks[p.pos+1].Text == "in" {
			op = "not in"
			p.next()
			p.next()
		} else {
			break
		}
		v, err := p.parseAdd()
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
		vals = append(vals, v)
	}
	if len(ops) == 0 {
		return x, nil
	}
	res := &BinE{Pos{posLine(x), posCol(x)}, ops[0], vals[0], vals[1]}
	for i := 1; i < len(ops); i++ {
		res = &BinE{Pos{posLine(vals[i]), posCol(vals[i])}, "and", res,
			&BinE{Pos{posLine(vals[i]), posCol(vals[i])}, ops[i], vals[i], vals[i+1]}}
	}
	return res, nil
}

func (p *parser) parseAdd() (Expr, error) {
	l, err := p.parseMul()
	if err != nil {
		return nil, err
	}
	for {
		var op string
		switch p.peek().Kind {
		case tPlus:
			op = "+"
		case tMinus:
			op = "-"
		default:
			return l, nil
		}
		p.next()
		r, err := p.parseMul()
		if err != nil {
			return nil, err
		}
		l = &BinE{Pos{posLine(l), posCol(l)}, op, l, r}
	}
}

func (p *parser) parseMul() (Expr, error) {
	l, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	for {
		var op string
		switch p.peek().Kind {
		case tStar:
			op = "*"
		case tSlash:
			op = "/"
		case tSlashSlash:
			op = "//"
		case tPct:
			op = "%"
		default:
			return l, nil
		}
		p.next()
		r, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		l = &BinE{Pos{posLine(l), posCol(l)}, op, l, r}
	}
}

func (p *parser) parseFactor() (Expr, error) {
	k := p.peek()
	if k.Kind == tMinus {
		p.next()
		x, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &UnE{Pos{posLine(x), posCol(x)}, "-", x}, nil
	}
	if k.Kind == tPlus {
		p.next()
		return p.parseFactor()
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (Expr, error) {
	x, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	var steps []ChainStep
	for {
		k := p.peek()
		switch k.Kind {
		case tLParen:
			x = p.applySteps(x, steps)
			steps = steps[:0]
			p.next()
			args, err := p.parseCallArgs()
			if err != nil {
				return nil, err
			}
			if k2 := p.peek(); k2.Kind != tRParen {
				return nil, p.errf(k2, "expected ')'")
			}
			p.next()
			x = &CallE{Pos{posLine(x), posCol(x)}, x, args}
		case tLBrack, tQBrack:
			safe := k.Kind == tQBrack
			p.next()
			st := ChainStep{Safe: safe, Idx: true}
			var first Expr
			if p.peek().Kind != tColon {
				var err error
				first, err = p.parseOr()
				if err != nil {
					return nil, err
				}
			}
			if p.peek().Kind == tColon {
				// slice: x[a:b], x[:b], x[a:], x[:]
				p.next()
				st.Slice = true
				if first != nil {
					st.Lo = first
				}
				if p.peek().Kind != tRBrack {
					hi, err := p.parseOr()
					if err != nil {
						return nil, err
					}
					st.Hi = hi
				}
			} else {
				st.Lo = first
			}
			if k2 := p.peek(); k2.Kind != tRBrack {
				return nil, p.errf(k2, "expected ']'")
			}
			p.next()
			steps = append(steps, st)
		case tDot, tQDot:
			safe := k.Kind == tQDot
			p.next()
			n, err := p.expectIdent()
			if err != nil {
				return nil, err
			}
			steps = append(steps, ChainStep{Safe: safe, Name: n})
		default:
			return p.finishChain(x, steps)
		}
	}
}

// finishChain assembles the accessor chain, flattening any chain that goes
// through a safe accessor into a SafeChainE (or a single safe accessor).
func (p *parser) finishChain(x Expr, steps []ChainStep) (Expr, error) {
	if len(steps) == 0 {
		return x, nil
	}
	firstSafe := -1
	for i, st := range steps {
		if st.Safe {
			firstSafe = i
			break
		}
	}
	if firstSafe < 0 {
		for _, st := range steps {
			x = p.applyStep(x, st)
		}
		return x, nil
	}
	for _, st := range steps[:firstSafe] {
		x = p.applyStep(x, st)
	}
	rest := steps[firstSafe:]
	if len(rest) == 1 {
		return p.applySafe(x, rest[0]), nil
	}
	return &SafeChainE{Pos: posOf(x), Base: x, Steps: rest}, nil
}

func (p *parser) applyStep(x Expr, st ChainStep) Expr {
	pos := posOf(x)
	if st.Idx {
		if st.Slice {
			return &SliceE{Pos: pos, X: x, Lo: st.Lo, Hi: st.Hi}
		}
		return &IndexE{Pos: pos, X: x, Key: st.Lo}
	}
	return &AttrE{Pos: pos, X: x, Name: st.Name}
}

func (p *parser) applySteps(x Expr, steps []ChainStep) Expr {
	for _, st := range steps {
		x = p.applyStep(x, st)
	}
	return x
}

func (p *parser) applySafe(x Expr, st ChainStep) Expr {
	pos := posOf(x)
	if st.Idx {
		if st.Slice {
			return &SliceE{Pos: pos, X: x, Lo: st.Lo, Hi: st.Hi, Safe: true}
		}
		return &SafeIndexE{Pos: pos, X: x, Key: st.Lo}
	}
	return &SafeAttrE{Pos: pos, X: x, Name: st.Name}
}

func (p *parser) parseCallArgs() ([]Expr, error) {
	var args []Expr
	// skip any leading newlines/indent inside the call parens
	for p.peek().Kind == tNewline || p.peek().Kind == tIndent || p.peek().Kind == tDedent {
		p.next()
	}
	if p.peek().Kind == tRParen {
		return args, nil
	}
	for {
		// skip whitespace before each argument
		for p.peek().Kind == tNewline || p.peek().Kind == tIndent {
			p.next()
		}
		a, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		args = append(args, a)
		// skip whitespace/newlines/dedents after argument
		for p.peek().Kind == tNewline || p.peek().Kind == tDedent {
			p.next()
		}
		k := p.peek()
		switch k.Kind {
		case tRParen:
			return args, nil
		case tComma:
			p.next()
			for p.peek().Kind == tNewline || p.peek().Kind == tIndent || p.peek().Kind == tDedent {
				p.next()
			}
			if p.peek().Kind == tRParen {
				return args, nil
			}
		default:
			if !exprStart(k) {
				return nil, p.errf(k, "expected ')' or another argument")
			}
		}
	}
}

func (p *parser) parsePrimary() (Expr, error) {
	k := p.peek()
	switch k.Kind {
	case tInt:
		p.next()
		return &NumLit{Pos{k.Line, k.Col}, false, k.Num, 0}, nil
	case tFlt:
		p.next()
		return &NumLit{Pos{k.Line, k.Col}, true, 0, k.Flt}, nil
	case tStr:
		p.next()
		return &StrLit{Pos{k.Line, k.Col}, k.Text}, nil
	case tFStr:
		p.next()
		return p.buildFStr(k)
	case tIdent:
		if k.Text == "fn" && p.pos+1 < len(p.toks) && p.toks[p.pos+1].Kind == tLParen {
			p.next() // consume 'fn'
			p.next() // consume '('
			params, ptypes, err := p.parseParams()
			if err != nil {
				return nil, err
			}
			if p.peek().Kind != tRParen {
				return nil, p.errf(p.peek(), "expected ')'")
			}
			p.next()
			ret, err := p.parseOptReturn()
			if err != nil {
				return nil, err
			}
			if _, err := p.expectColon(); err != nil {
				return nil, err
			}
			if p.peek().Kind != tNewline {
				expr, err := p.parseOr()
				if err != nil {
					return nil, err
				}
				body := []Stmt{&ReturnStmt{Pos: posOf(expr), Vals: []Expr{expr}}}
				return &FnExpr{Pos: Pos{k.Line, k.Col}, Params: params, ParamTypes: ptypes, Ret: ret, Body: body}, nil
			}
			p.fnN++
			body, err := p.parseSuite()
			p.fnN--
			if err != nil {
				return nil, err
			}
			return &FnExpr{Pos: Pos{k.Line, k.Col}, Params: params, ParamTypes: ptypes, Ret: ret, Body: body}, nil
		}
		switch k.Text {
		case "true":
			p.next()
			return &BoolLit{Pos{k.Line, k.Col}, true}, nil
		case "false":
			p.next()
			return &BoolLit{Pos{k.Line, k.Col}, false}, nil
		case "nil":
			p.next()
			return &NilLit{Pos{k.Line, k.Col}}, nil
		case "and", "or", "not", "in", "using", "import", "pub", "priv", "as", "fn", "return", "if", "elif", "else", "for", "while", "break", "continue", "try", "catch", "match", "case", "where", "with":
			return nil, p.errf(k, "unexpected %q", k.Text)
		}
		p.next()
		return &NameE{Pos{k.Line, k.Col}, k.Text}, nil
	case tLParen:
		p.next()
		e, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if k2 := p.peek(); k2.Kind != tRParen {
			return nil, p.errf(k2, "expected ')'")
		}
		p.next()
		return e, nil
	case tLBrack:
		p.next()
		var items []Expr
		if p.peek().Kind != tRBrack {
			var err error
			items, err = p.parseExprList()
			if err != nil {
				return nil, err
			}
		}
		if k2 := p.peek(); k2.Kind != tRBrack {
			return nil, p.errf(k2, "expected ']'")
		}
		p.next()
		return &ListLit{Pos{k.Line, k.Col}, items}, nil
	case tLBrace:
		p.next()
		var pairs [][2]Expr
		for p.peek().Kind != tRBrace {
			key, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			if n, ok := key.(*NameE); ok {
				key = &StrLit{Pos{n.Line, n.Col}, n.X}
			}
			if k2 := p.peek(); k2.Kind != tColon {
				return nil, p.errf(k2, "expected ':' in dict literal")
			}
			p.next()
			val, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, [2]Expr{key, val})
			k := p.peek()
			switch k.Kind {
			case tComma:
				p.next()
			case tRBrace:
			case tIdent, tInt, tFlt, tStr, tLParen, tLBrack, tLBrace:
				// optional commas between pairs
			default:
				return nil, p.errf(k, "expected '}' or another pair")
			}
		}
		if k2 := p.peek(); k2.Kind != tRBrace {
			return nil, p.errf(k2, "expected '}'")
		}
		p.next()
		return &DictLit{Pos{k.Line, k.Col}, pairs}, nil
	}
	return nil, p.errf(k, "expected an expression, got %s", k.String())
}

// buildFStr converts a tFStr token into an FStrLit AST node by sub-parsing expressions.
func (p *parser) buildFStr(k Tok) (Expr, error) {
	var fparts []FStrPart
	// Parts alternates: [lit, expr, lit, expr, ..., lit]
	for j, part := range k.Parts {
		if j%2 == 0 {
			// Literal segment
			fparts = append(fparts, FStrPart{IsExpr: false, Lit: part})
		} else {
			// Expression segment — sub-parse
			toks, err := Tokenize(part+"\n", "<fstr>")
			if err != nil {
				return nil, &Errat{p.name, k.Line, k.Col, fmt.Errorf("in f-string expression %q: %v", part, err)}
			}
			sub := &parser{toks: toks, name: "<fstr>"}
			expr, err := sub.parseOr()
			if err != nil {
				return nil, &Errat{p.name, k.Line, k.Col, fmt.Errorf("in f-string expression %q: %v", part, err)}
			}
			fparts = append(fparts, FStrPart{IsExpr: true, Expr: expr})
		}
	}
	return &FStrLit{Pos{k.Line, k.Col}, fparts}, nil
}

func exprStart(k Tok) bool {
	switch k.Kind {
	case tIdent:
		return !isReserved(k.Text) || k.Text == "true" || k.Text == "false" || k.Text == "nil" || k.Text == "fn"
	case tInt, tFlt, tStr, tFStr, tLParen, tLBrack, tLBrace:
		return true
	}
	return false
}

func isReserved(w string) bool {
	switch w {
	case "using", "import", "pub", "priv", "as", "fn", "return", "if", "elif", "else", "for", "in", "while", "break", "continue", "and", "or", "not", "try", "catch", "match", "case", "where", "with":
		return true
	}
	return false
}

func posLine(e Expr) int { return posOf(e).Line }

func posCol(e Expr) int { return posOf(e).Col }

func posOf(e Expr) Pos {
	switch x := e.(type) {
	case *NumLit:
		return x.Pos
	case *StrLit:
		return x.Pos
	case *BoolLit:
		return x.Pos
	case *NilLit:
		return x.Pos
	case *NameE:
		return x.Pos
	case *BinE:
		return x.Pos
	case *UnE:
		return x.Pos
	case *CallE:
		return x.Pos
	case *IndexE:
		return x.Pos
	case *SafeIndexE:
		return x.Pos
	case *AttrE:
		return x.Pos
	case *SafeAttrE:
		return x.Pos
	case *SliceE:
		return x.Pos
	case *SafeChainE:
		return x.Pos
	case *ListLit:
		return x.Pos
	case *DictLit:
		return x.Pos
	case *FnExpr:
		return x.Pos
	case *FStrLit:
		return x.Pos
	}
	return Pos{}
}
