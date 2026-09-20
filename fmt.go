package blizzard

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const fmtIndent = "    "

type triviaKind int

const (
	trivBlank triviaKind = iota
	trivComment
)

type trivia struct {
	line int
	kind triviaKind
	text string
}

type formatter struct {
	b         strings.Builder
	orig      []string
	trivia    []trivia
	modern    bool
	declared  map[string]bool
	inFString bool
}

// Format pretty-prints Blizzard source with 4-space indentation.
// Full-line comments and blank lines are preserved in order.
func Format(src string) (string, error) {
	return formatSource(src, false)
}

// FormatModern formats source using the Blizzard 0.2 brace syntax, even when
// the input still uses the legacy colon-and-indentation form.
func FormatModern(src string) (string, error) {
	return formatSource(src, true)
}

func formatSource(src string, forceModern bool) (string, error) {
	p, err := Parse(src, "<fmt>")
	if err != nil {
		return "", err
	}
	modern := forceModern || strings.Contains(src, "import ") || strings.Contains(src, "let ") || strings.Contains(src, ") {")
	f := &formatter{orig: splitSrcLines(src), trivia: collectTrivia(src), modern: modern, declared: map[string]bool{}}
	f.program(p)
	out := f.b.String()
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
}

func splitSrcLines(src string) []string {
	return strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
}

func collectTrivia(src string) []trivia {
	lines := splitSrcLines(src)
	var out []trivia
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			out = append(out, trivia{line: i + 1, kind: trivBlank})
			continue
		}
		if strings.HasPrefix(trim, "#") {
			out = append(out, trivia{line: i + 1, kind: trivComment, text: trim})
		}
	}
	return out
}

func (f *formatter) program(p *Program) {
	for _, s := range p.Stmts {
		f.stmt(s, 0)
	}
	f.emitTriviaUntil(1<<30, 0)
	s := f.b.String()
	s = strings.TrimRight(s, "\n")
	f.b.Reset()
	f.b.WriteString(s)
	if s != "" {
		f.b.WriteByte('\n')
	}
}

func (f *formatter) emitTriviaUntil(line, indent int) {
	blankPending := false
	for len(f.trivia) > 0 && f.trivia[0].line < line {
		t := f.trivia[0]
		f.trivia = f.trivia[1:]
		if t.kind == trivBlank {
			blankPending = true
			continue
		}
		if blankPending && f.b.Len() > 0 {
			f.b.WriteByte('\n')
		}
		blankPending = false
		f.ind(indent)
		f.b.WriteString(t.text)
		f.b.WriteByte('\n')
	}
	if blankPending && f.b.Len() > 0 && !strings.HasSuffix(f.b.String(), "\n\n") {
		f.b.WriteByte('\n')
	}
}

func (f *formatter) ind(n int) {
	for i := 0; i < n; i++ {
		f.b.WriteString(fmtIndent)
	}
}

func (f *formatter) stmts(list []Stmt, indent int) {
	for _, s := range list {
		f.stmt(s, indent)
	}
}

func stmtLine(s Stmt) int {
	switch t := s.(type) {
	case *UseStmt:
		return t.Line
	case *AssignStmt:
		return t.Line
	case *FnStmt:
		return t.Line
	case *ReturnStmt:
		return t.Line
	case *IfStmt:
		return t.Line
	case *ForStmt:
		return t.Line
	case *WhileStmt:
		return t.Line
	case *LoopStmt:
		return t.Line
	case *TryStmt:
		return t.Line
	case *MatchStmt:
		return t.Line
	case *ExprStmt:
		return t.Line
	case *WithStmt:
		return t.Line
	case *SetAttrStmt:
		return t.Line
	case *SetIndexStmt:
		return t.Line
	}
	return 0
}

func (f *formatter) stmt(s Stmt, indent int) {
	f.emitTriviaUntil(stmtLine(s), indent)
	switch t := s.(type) {
	case *UseStmt:
		f.ind(indent)
		if f.modern {
			f.b.WriteString("import ")
		} else {
			f.b.WriteString("using ")
		}
		f.b.WriteString(strings.Join(t.Path, "."))
		if t.Alias != "" && (len(t.Path) == 0 || t.Alias != t.Path[len(t.Path)-1]) {
			f.b.WriteString(" as ")
			f.b.WriteString(t.Alias)
		}
		if f.modern {
			f.b.WriteString(";\n")
		} else {
			f.b.WriteByte('\n')
		}
	case *AssignStmt:
		f.ind(indent)
		declare := t.Let
		if f.modern && !declare && t.Op == 0 {
			for _, name := range t.Names {
				if !f.declared[name] {
					declare = true
				}
			}
		}
		if f.modern && declare {
			f.b.WriteString("let ")
		}
		for _, name := range t.Names {
			f.declared[name] = true
		}
		if t.Type != "" {
			f.b.WriteString(t.Names[0])
			f.b.WriteString(": ")
			f.b.WriteString(t.Type)
		} else {
			f.b.WriteString(strings.Join(t.Names, ", "))
		}
		f.b.WriteByte(' ')
		f.b.WriteString(assignOp(t.Op))
		f.b.WriteByte(' ')
		for i, v := range t.Vals {
			if i > 0 {
				f.b.WriteString(", ")
			}
			f.expr(v, 0)
		}
		if f.modern {
			f.b.WriteString(";\n")
		} else {
			f.b.WriteByte('\n')
		}
	case *SetAttrStmt:
		f.ind(indent)
		f.expr(t.Container, 0)
		f.b.WriteByte('.')
		f.b.WriteString(t.Name)
		f.b.WriteByte(' ')
		f.b.WriteString(assignOp(t.Op))
		f.b.WriteByte(' ')
		f.expr(t.Val, 0)
		if f.modern {
			f.b.WriteString(";\n")
		} else {
			f.b.WriteByte('\n')
		}
	case *SetIndexStmt:
		f.ind(indent)
		f.expr(t.Container, 0)
		f.b.WriteByte('[')
		f.expr(t.Key, 0)
		f.b.WriteString("] ")
		f.b.WriteString(assignOp(t.Op))
		f.b.WriteByte(' ')
		f.expr(t.Val, 0)
		if f.modern {
			f.b.WriteString(";\n")
		} else {
			f.b.WriteByte('\n')
		}
	case *FnStmt:
		previous := f.declared
		f.declared = map[string]bool{}
		for _, param := range t.Params {
			f.declared[param] = true
		}
		f.ind(indent)
		f.b.WriteString("fn ")
		f.b.WriteString(t.Name)
		f.writeFnSig(t.Params, t.ParamTypes, t.Rest, t.Ret)
		if f.modern {
			f.b.WriteString(" {\n")
		} else {
			f.b.WriteString(":\n")
		}
		f.stmts(t.Body, indent+1)
		if f.modern {
			f.ind(indent)
			f.b.WriteString("}\n")
		}
		f.declared = previous
	case *ReturnStmt:
		f.ind(indent)
		f.b.WriteString("return")
		for i, v := range t.Vals {
			if i == 0 {
				f.b.WriteByte(' ')
			} else {
				f.b.WriteString(", ")
			}
			f.expr(v, 0)
		}
		if f.modern {
			f.b.WriteString(";\n")
		} else {
			f.b.WriteByte('\n')
		}
	case *IfStmt:
		for i, cond := range t.Conds {
			if !(f.modern && i > 0) {
				f.ind(indent)
			}
			if i == 0 {
				f.b.WriteString("if ")
			} else {
				f.b.WriteString("elif ")
			}
			if f.modern {
				f.b.WriteByte('(')
			}
			f.expr(cond, 0)
			if f.modern {
				f.b.WriteByte(')')
			}
			if f.modern {
				f.b.WriteString(" {\n")
			} else {
				f.b.WriteString(":\n")
			}
			f.stmts(t.Bodies[i], indent+1)
			if f.modern {
				f.ind(indent)
				f.b.WriteString("}")
				if i+1 < len(t.Conds) || len(t.Else) > 0 {
					f.b.WriteByte(' ')
				} else {
					f.b.WriteByte('\n')
				}
			}
		}
		if len(t.Else) > 0 {
			if !(f.modern && len(t.Conds) > 0) {
				f.ind(indent)
			}
			if f.modern {
				f.b.WriteString("else {\n")
			} else {
				f.b.WriteString("else:\n")
			}
			f.stmts(t.Else, indent+1)
			if f.modern {
				f.ind(indent)
				f.b.WriteString("}\n")
			}
		}
	case *ForStmt:
		f.ind(indent)
		f.b.WriteString("for ")
		f.b.WriteString(t.Name)
		if t.Name2 != "" {
			f.b.WriteString(", ")
			f.b.WriteString(t.Name2)
		}
		f.b.WriteString(" in ")
		f.expr(t.Iter, 0)
		if f.modern {
			f.b.WriteString(" {")
			f.b.WriteByte('\n')
		} else {
			f.b.WriteString(":\n")
		}
		f.stmts(t.Body, indent+1)
		if f.modern {
			f.ind(indent)
			f.b.WriteString("}\n")
		}
	case *WhileStmt:
		f.ind(indent)
		f.b.WriteString("while ")
		if f.modern {
			f.b.WriteByte('(')
		}
		f.expr(t.Cond, 0)
		if f.modern {
			f.b.WriteByte(')')
		}
		if f.modern {
			f.b.WriteString(" {\n")
		} else {
			f.b.WriteString(":\n")
		}
		f.stmts(t.Body, indent+1)
		if f.modern {
			f.ind(indent)
			f.b.WriteString("}\n")
		}
	case *LoopStmt:
		f.ind(indent)
		if t.Break {
			if f.modern {
				f.b.WriteString("break;\n")
			} else {
				f.b.WriteString("break\n")
			}
		} else {
			if f.modern {
				f.b.WriteString("continue;\n")
			} else {
				f.b.WriteString("continue\n")
			}
		}
	case *TryStmt:
		f.ind(indent)
		if f.modern {
			f.b.WriteString("try {\n")
		} else {
			f.b.WriteString("try:\n")
		}
		f.stmts(t.TryBody, indent+1)
		if f.modern {
			f.ind(indent)
			f.b.WriteString("} ")
		}
		if !f.modern {
			f.ind(indent)
		}
		f.b.WriteString("catch ")
		f.b.WriteString(t.CatchVar)
		if f.modern {
			f.b.WriteString(" {\n")
		} else {
			f.b.WriteString(":\n")
		}
		f.stmts(t.CatchBody, indent+1)
		if f.modern {
			f.ind(indent)
			f.b.WriteString("}")
			if len(t.Always) > 0 {
				f.b.WriteByte(' ')
			} else {
				f.b.WriteByte('\n')
			}
		}
		if len(t.Always) > 0 {
			if f.modern {
				f.b.WriteString("always {\n")
			} else {
				f.ind(indent)
				f.b.WriteString("always:\n")
			}
			f.stmts(t.Always, indent+1)
			if f.modern {
				f.ind(indent)
				f.b.WriteString("}\n")
			}
		}
	case *MatchStmt:
		f.ind(indent)
		f.b.WriteString("match ")
		if f.modern {
			f.b.WriteByte('(')
		}
		f.expr(t.Target, 0)
		if f.modern {
			f.b.WriteByte(')')
		}
		if f.modern {
			f.b.WriteString(" {\n")
		} else {
			f.b.WriteString(":\n")
		}
		for _, cs := range t.Cases {
			f.ind(indent + 1)
			f.b.WriteString("case ")
			for j, v := range cs.Vals {
				if j > 0 {
					f.b.WriteString(", ")
				}
				f.expr(v, 0)
			}
			if f.modern {
				f.b.WriteString(" {\n")
			} else {
				f.b.WriteString(":\n")
			}
			f.stmts(cs.Body, indent+2)
			if f.modern {
				f.ind(indent + 1)
				f.b.WriteString("}\n")
			}
		}
		if f.modern {
			f.ind(indent)
			f.b.WriteString("}\n")
		}
	case *WithStmt:
		f.ind(indent)
		f.b.WriteString("with ")
		f.expr(t.Expr, 0)
		if t.Name != "" {
			f.b.WriteString(" as ")
			f.b.WriteString(t.Name)
		}
		if f.modern {
			f.b.WriteString(" {\n")
		} else {
			f.b.WriteString(":\n")
		}
		f.stmts(t.Body, indent+1)
		if f.modern {
			f.ind(indent)
			f.b.WriteString("}\n")
		}
	case *ExprStmt:
		f.ind(indent)
		f.expr(t.X, 0)
		if f.modern {
			f.b.WriteString(";\n")
		} else {
			f.b.WriteByte('\n')
		}
	}
}

// writeFnSig writes "(a: str, b: int[]) -> ret" without the surrounding 'fn'.
func (f *formatter) writeFnSig(params, types []string, rest bool, ret string) {
	f.b.WriteByte('(')
	for i, p := range params {
		if i > 0 {
			f.b.WriteString(", ")
		}
		if rest && i == len(params)-1 {
			f.b.WriteString("...")
		}
		f.b.WriteString(p)
		if i < len(types) && types[i] != "" {
			f.b.WriteString(": ")
			f.b.WriteString(types[i])
		}
	}
	f.b.WriteByte(')')
	if ret != "" {
		f.b.WriteString(" -> ")
		f.b.WriteString(ret)
	}
}

func assignOp(op byte) string {
	switch op {
	case 0:
		return "="
	case boAdd:
		return "+="
	case boSub:
		return "-="
	case boMul:
		return "*="
	case boDiv:
		return "/="
	case boFloorDiv:
		return "//="
	case boMod:
		return "%="
	case boQQEq:
		return "??="
	}
	return "="
}

func precOf(op string) int {
	switch op {
	case "or":
		return 1
	case "??":
		return 2
	case "and":
		return 3
	case "==", "!=", "<", "<=", ">", ">=", "in", "not in":
		return 4
	case "+", "-":
		return 5
	case "*", "/", "//", "%":
		return 6
	}
	return 0
}

func (f *formatter) expr(e Expr, parentPrec int) {
	switch t := e.(type) {
	case *NumLit:
		if t.IsFloat {
			f.b.WriteString(FltStr(t.Float))
		} else {
			f.b.WriteString(strconv.FormatInt(t.Int, 10))
		}
	case *StrLit:
		if f.inFString {
			f.writeSingleStr(t.V)
		} else {
			f.writeStr(t.V)
		}
	case *BoolLit:
		if t.V {
			f.b.WriteString("true")
		} else {
			f.b.WriteString("false")
		}
	case *NilLit:
		f.b.WriteString("nil")
	case *NameE:
		f.b.WriteString(t.X)
	case *BinE:
		p := precOf(t.Op)
		if p < parentPrec {
			f.b.WriteByte('(')
		}
		f.expr(t.X, p)
		f.b.WriteByte(' ')
		f.b.WriteString(t.Op)
		f.b.WriteByte(' ')
		right := p
		if t.Op == "-" || t.Op == "/" || t.Op == "//" || t.Op == "%" {
			right = p + 1
		}
		f.expr(t.Y, right)
		if p < parentPrec {
			f.b.WriteByte(')')
		}
	case *UnE:
		if t.Op == "not" {
			f.b.WriteString("not ")
			f.expr(t.X, 4)
		} else {
			f.b.WriteString(t.Op)
			_, innerUn := t.X.(*UnE)
			if innerUn {
				f.b.WriteByte('(')
				f.expr(t.X, 0)
				f.b.WriteByte(')')
			} else {
				f.expr(t.X, 7)
			}
		}
	case *CallE:
		f.expr(t.Fn, 8)
		f.b.WriteByte('(')
		for i, a := range t.Args {
			if i > 0 {
				f.b.WriteString(", ")
			}
			f.expr(a, 0)
		}
		f.b.WriteByte(')')
	case *SpreadE:
		f.b.WriteString("...")
		f.expr(t.X, 0)
	case *IndexE:
		f.expr(t.X, 8)
		f.b.WriteByte('[')
		f.expr(t.Key, 0)
		f.b.WriteByte(']')
	case *SafeIndexE:
		f.expr(t.X, 8)
		f.b.WriteString("?[")
		f.expr(t.Key, 0)
		f.b.WriteByte(']')
	case *AttrE:
		f.expr(t.X, 8)
		f.b.WriteByte('.')
		f.b.WriteString(t.Name)
	case *SafeAttrE:
		f.expr(t.X, 8)
		f.b.WriteString("?.")
		f.b.WriteString(t.Name)
	case *SliceE:
		f.expr(t.X, 8)
		if t.Safe {
			f.b.WriteString("?[")
		} else {
			f.b.WriteByte('[')
		}
		f.writeSlicePart(t.Lo)
		f.b.WriteByte(':')
		f.writeSlicePart(t.Hi)
		f.b.WriteByte(']')
	case *SafeChainE:
		f.expr(t.Base, 8)
		for _, st := range t.Prefix {
			f.writeChainStep(st)
		}
		for _, st := range t.Steps {
			f.writeChainStep(st)
		}
	case *ListLit:
		f.b.WriteByte('[')
		for i, it := range t.Items {
			if i > 0 {
				f.b.WriteString(", ")
			}
			f.expr(it, 0)
		}
		f.b.WriteByte(']')
	case *DictLit:
		f.b.WriteByte('{')
		for i, pr := range t.Pairs {
			if i > 0 {
				f.b.WriteString(", ")
			}
			if pr[0] == nil {
				f.expr(pr[1], 0)
				continue
			}
			f.dictKey(pr[0])
			f.b.WriteString(": ")
			f.expr(pr[1], 0)
		}
		f.b.WriteByte('}')
	case *FnExpr:
		previous := f.declared
		f.declared = map[string]bool{}
		for _, param := range t.Params {
			f.declared[param] = true
		}
		baseIndent := 0
		if newline := strings.LastIndex(f.b.String(), "\n"); newline >= 0 {
			tail := f.b.String()[newline+1:]
			baseIndent = strings.Count(tail, fmtIndent)
		}
		f.b.WriteString("fn")
		f.writeFnSig(t.Params, t.ParamTypes, t.Rest, t.Ret)
		if f.modern {
			f.b.WriteString(" {")
		} else {
			f.b.WriteString(":")
		}
		if len(t.Body) == 1 {
			if r, ok := t.Body[0].(*ReturnStmt); ok && len(r.Vals) == 1 {
				if f.modern {
					f.b.WriteString(" return ")
				} else {
					f.b.WriteByte(' ')
				}
				f.expr(r.Vals[0], 0)
				if f.modern {
					f.b.WriteString("; }")
				}
				f.declared = previous
				return
			}
		}
		f.b.WriteByte('\n')
		f.stmts(t.Body, baseIndent+1)
		if f.modern {
			f.ind(baseIndent)
			f.b.WriteString("}")
		}
		f.declared = previous
	case *FStrLit:
		f.writeFStr(t)
	}
}

// writeSlicePart writes a slice endpoint, skipping an open (nil) end.
func (f *formatter) writeSlicePart(e Expr) {
	if _, isNil := e.(*NilLit); isNil || e == nil {
		return
	}
	f.expr(e, 0)
}

func (f *formatter) writeChainStep(st ChainStep) {
	if st.Idx {
		if st.Safe {
			f.b.WriteString("?[")
		} else {
			f.b.WriteByte('[')
		}
		if st.Slice {
			f.writeSlicePart(st.Lo)
			f.b.WriteByte(':')
			f.writeSlicePart(st.Hi)
		} else {
			f.expr(st.Lo, 0)
		}
		f.b.WriteByte(']')
		return
	}
	if st.Safe {
		f.b.WriteString("?.")
	} else {
		f.b.WriteByte('.')
	}
	f.b.WriteString(st.Name)
}

func (f *formatter) dictKey(e Expr) {
	if s, ok := e.(*StrLit); ok && identKey(s.V) {
		f.b.WriteString(s.V)
		return
	}
	f.expr(e, 0)
}

func identKey(s string) bool {
	if s == "" || isReserved(s) {
		return false
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
		} else if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
		i += size
	}
	return true
}

func (f *formatter) writeStr(s string) {
	if strings.Contains(s, "\n") && !strings.Contains(s, `"""`) {
		f.b.WriteString(`"""`)
		f.b.WriteString(s)
		f.b.WriteString(`"""`)
		return
	}
	f.b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			f.b.WriteString(`\\`)
		case '"':
			f.b.WriteString(`\"`)
		case '\n':
			f.b.WriteString(`\n`)
		case '\t':
			f.b.WriteString(`\t`)
		case '\r':
			f.b.WriteString(`\r`)
		default:
			f.b.WriteRune(r)
		}
	}
	f.b.WriteByte('"')
}

func (f *formatter) writeFStr(t *FStrLit) {
	f.b.WriteString(`f"`)
	for _, part := range t.Parts {
		if !part.IsExpr {
			for _, r := range part.Lit {
				switch r {
				case '\\':
					f.b.WriteString(`\\`)
				case '"':
					f.b.WriteString(`\"`)
				case '{':
					f.b.WriteString(`{{`)
				case '}':
					f.b.WriteString(`}}`)
				case '\n':
					f.b.WriteString(`\n`)
				case '\t':
					f.b.WriteString(`\t`)
				default:
					f.b.WriteRune(r)
				}
			}
			continue
		}
		f.b.WriteByte('{')
		previous := f.inFString
		f.inFString = true
		f.expr(part.Expr, 0)
		f.inFString = previous
		f.b.WriteByte('}')
	}
	f.b.WriteByte('"')
}

func (f *formatter) writeSingleStr(s string) {
	f.b.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\\':
			f.b.WriteString(`\\`)
		case '\'':
			f.b.WriteString(`\'`)
		case '\n':
			f.b.WriteString(`\n`)
		case '\r':
			f.b.WriteString(`\r`)
		case '\t':
			f.b.WriteString(`\t`)
		default:
			f.b.WriteRune(r)
		}
	}
	f.b.WriteByte('\'')
}
