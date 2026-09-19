package snow

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Issue is a single finding from Check. Errors are definite problems
// such as an undefined name; Warnings are style or heuristics.
type Issue struct {
	Line   int
	Col    int
	Msg    string
	IsErr  bool
	Source string
}

func (i Issue) String() string {
	kind := "warning"
	if i.IsErr {
		kind = "error"
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s", i.Source, i.Line, i.Col, kind, i.Msg)
}

// Check parses, compiles and lints Snow source without running it. If the
// code does not parse or compile, the error is returned and no issues are
// produced.
func Check(src, name string) ([]Issue, error) {
	stdBuiltins() // fill stdNames so builtins are treated as known names
	p, err := Parse(src, name)
	if err != nil {
		return nil, err
	}
	if _, err := Compile(p); err != nil {
		return nil, err
	}
	c := &checker{name: name}
	if err := c.program(p); err != nil {
		return nil, err
	}
	return c.issues, nil
}

var stdMods = map[string]bool{
	"sys": true, "fs": true, "cli": true, "api": true, "http": true,
	"db": true, "time": true, "json": true, "crypto": true, "task": true,
	"env": true, "csv": true, "input": true,
}

var repoMods = map[string]bool{
	"arrays": true, "collections": true, "csvutil": true, "dict": true,
	"guards": true, "ids": true, "math": true, "numbers": true,
	"pagination": true, "query": true, "result": true, "stats": true,
	"strings": true, "template": true, "text": true, "validate": true,
}

type checker struct {
	name   string
	issues []Issue

	scopes []map[string]bool

	// used records every name read anywhere in the program.
	used map[string]bool
	// topVars records the first assignment position of a top-level name.
	topVars map[string]Pos
	// topPriv records top-level names that are explicitly private.
	topPriv     map[string]bool
	packageFile bool
	moduleFile  bool
}

func (c *checker) program(p *Program) error {
	c.used = map[string]bool{}
	c.topVars = map[string]Pos{}
	c.topPriv = map[string]bool{}
	c.packageFile = isPackageFile(c.name)
	c.moduleFile = c.packageFile
	c.push()
	for _, s := range p.Stmts {
		switch decl := s.(type) {
		case *FnStmt:
			if decl.Vis != "" {
				c.moduleFile = true
			}
			c.defineVisibility(decl.Name, decl.Pos, decl.Vis == "priv")
		case *AssignStmt:
			if decl.Vis != "" {
				c.moduleFile = true
			}
		}
	}

	// Collect every using alias so modules are known regardless of ordering.
	var uses []*UseStmt
	collectUsing(p.Stmts, &uses)
	for _, u := range uses {
		if len(u.Path) == 2 && u.Path[0] == "snow" && !stdMods[u.Path[1]] && !repoMods[u.Path[1]] {
			message := fmt.Sprintf("unknown standard module 'snow.%s'", u.Path[1])
			if suggestion := moduleSuggestion(u.Path[1]); suggestion != "" {
				message += fmt.Sprintf("; did you mean 'snow.%s'?", suggestion)
			}
			c.err(u.Pos, "%s", message)
		}
		c.scopes[len(c.scopes)-1][u.Alias] = true
	}
	// Mark reads: any name read while running the code counts as used.
	c.markUsed(p.Stmts)

	if err := c.walkStmts(p.Stmts, true); err != nil {
		return err
	}

	// Report unused top-level variables.
	for name, pos := range c.topVars {
		if name == "_" || (c.moduleFile && !c.topPriv[name]) {
			continue
		}
		if !c.used[name] {
			c.warn(pos, "variable '%s' is assigned but never used", name)
		}
	}
	return nil
}

func isPackageFile(name string) bool {
	path := filepath.ToSlash(name)
	return strings.Contains(path, "repo/packages/") || strings.Contains(path, "PkgsExamples/packages/")
}

func (c *checker) err(p Pos, format string, a ...any) {
	c.issues = append(c.issues, Issue{
		Line: p.Line, Col: p.Col, Source: c.name, IsErr: true,
		Msg: fmt.Sprintf(format, a...),
	})
}

func (c *checker) warn(p Pos, format string, a ...any) {
	c.issues = append(c.issues, Issue{
		Line: p.Line, Col: p.Col, Source: c.name,
		Msg: fmt.Sprintf(format, a...),
	})
}

func (c *checker) push() {
	c.scopes = append(c.scopes, map[string]bool{})
}

func (c *checker) pop() {
	c.scopes = c.scopes[:len(c.scopes)-1]
}

func (c *checker) define(name string, pos Pos) {
	c.defineVisibility(name, pos, false)
}

func (c *checker) defineVisibility(name string, pos Pos, private bool) {
	c.scopes[len(c.scopes)-1][name] = true
	if len(c.scopes) == 1 {
		if _, seen := c.topVars[name]; !seen {
			c.topVars[name] = pos
			c.topPriv[name] = private
		}
	}
}

func (c *checker) defined(name string) bool {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if c.scopes[i][name] {
			return true
		}
	}
	return false
}

func (c *checker) walkStmts(list []Stmt, topLevel bool) error {
	terminated := false
	for _, s := range list {
		if terminated {
			c.warn(stmtPos(s), "unreachable code")
			continue
		}
		if terminates(s) {
			terminated = true
		}
		if err := c.walkStmt(s, topLevel); err != nil {
			return err
		}
	}
	return nil
}

// terminates reports whether a statement never lets control fall through.
func terminates(s Stmt) bool {
	switch t := s.(type) {
	case *ReturnStmt, *LoopStmt:
		return true
	case *ExprStmt:
		if call, ok := t.X.(*CallE); ok {
			if n, ok := call.Fn.(*NameE); ok && (n.X == "fail" || n.X == "exit") {
				return true
			}
		}
	}
	return false
}

func stmtPos(s Stmt) Pos {
	switch t := s.(type) {
	case *UseStmt:
		return t.Pos
	case *AssignStmt:
		return t.Pos
	case *FnStmt:
		return t.Pos
	case *ReturnStmt:
		return t.Pos
	case *IfStmt:
		return t.Pos
	case *ForStmt:
		return t.Pos
	case *WhileStmt:
		return t.Pos
	case *LoopStmt:
		return t.Pos
	case *TryStmt:
		return t.Pos
	case *MatchStmt:
		return t.Pos
	case *ExprStmt:
		return t.Pos
	case *WithStmt:
		return t.Pos
	case *SetAttrStmt:
		return t.Pos
	case *SetIndexStmt:
		return t.Pos
	}
	return Pos{}
}

func (c *checker) walkStmt(s Stmt, topLevel bool) error {
	switch t := s.(type) {
	case *UseStmt:
		// already known from the first pass

	case *AssignStmt:
		for _, v := range t.Vals {
			if err := c.walkExpr(v); err != nil {
				return err
			}
		}
		if t.Type != "" && len(t.Vals) == 1 {
			if got := staticExprType(t.Vals[0]); got != "" && !staticTypeAccepts(t.Type, got) {
				name := "value"
				if len(t.Names) > 0 {
					name = t.Names[0]
				}
				c.err(t.Pos, "type mismatch for %q: expected %s, received %s", name, t.Type, got)
			}
		}
		for _, n := range t.Names {
			c.defineVisibility(n, t.Pos, t.Vis == "priv")
		}

	case *SetAttrStmt:
		if err := c.walkExpr(t.Container); err != nil {
			return err
		}
		if err := c.walkExpr(t.Val); err != nil {
			return err
		}

	case *SetIndexStmt:
		if err := c.walkExpr(t.Container); err != nil {
			return err
		}
		if err := c.walkExpr(t.Key); err != nil {
			return err
		}
		if err := c.walkExpr(t.Val); err != nil {
			return err
		}

	case *FnStmt:
		c.defineVisibility(t.Name, t.Pos, t.Vis == "priv")
		c.push()
		for _, p := range t.Params {
			c.define(p, t.Pos)
		}
		// the function name is visible inside its own body (recursion)
		c.scopes[len(c.scopes)-1][t.Name] = true
		if err := c.walkStmts(t.Body, false); err != nil {
			return err
		}
		c.pop()

	case *ReturnStmt:
		for _, v := range t.Vals {
			if err := c.walkExpr(v); err != nil {
				return err
			}
		}

	case *IfStmt:
		for _, cond := range t.Conds {
			if err := c.walkExpr(cond); err != nil {
				return err
			}
		}
		for _, body := range t.Bodies {
			if err := c.walkStmts(body, topLevel); err != nil {
				return err
			}
		}
		if err := c.walkStmts(t.Else, topLevel); err != nil {
			return err
		}

	case *ForStmt:
		if err := c.walkExpr(t.Iter); err != nil {
			return err
		}
		c.define(t.Name, t.Pos)
		if t.Name2 != "" && t.Name2 != "_" {
			c.define(t.Name2, t.Pos)
		}
		if t.Where != nil {
			if err := c.walkExpr(t.Where); err != nil {
				return err
			}
		}
		if err := c.walkStmts(t.Body, topLevel); err != nil {
			return err
		}

	case *WhileStmt:
		if err := c.walkExpr(t.Cond); err != nil {
			return err
		}
		if err := c.walkStmts(t.Body, topLevel); err != nil {
			return err
		}

	case *LoopStmt:

	case *TryStmt:
		if err := c.walkStmts(t.TryBody, topLevel); err != nil {
			return err
		}
		c.push()
		if t.CatchVar != "" && t.CatchVar != "_" {
			c.define(t.CatchVar, t.Pos)
		}
		if err := c.walkStmts(t.CatchBody, topLevel); err != nil {
			return err
		}
		c.pop()

	case *MatchStmt:
		if err := c.walkExpr(t.Target); err != nil {
			return err
		}
		for _, cs := range t.Cases {
			for _, v := range cs.Vals {
				if name, ok := v.(*NameE); ok && name.X == "_" {
					continue // wildcard case: matches anything
				}
				if err := c.walkExpr(v); err != nil {
					return err
				}
			}
			if err := c.walkStmts(cs.Body, topLevel); err != nil {
				return err
			}
		}
	case *WithStmt:
		if err := c.walkExpr(t.Expr); err != nil {
			return err
		}
		if t.Name != "" && t.Name != "_" {
			c.define(t.Name, t.Pos)
		}
		if err := c.walkStmts(t.Body, topLevel); err != nil {
			return err
		}
	case *ExprStmt:
		if err := c.walkExpr(t.X); err != nil {
			return err
		}
	}
	return nil
}

// staticExprType intentionally only identifies values whose type is certain
// without executing code. Runtime checks remain the authority for dynamic
// expressions.
func staticExprType(e Expr) string {
	switch t := e.(type) {
	case *NumLit:
		if t.IsFloat {
			return "float"
		}
		return "int"
	case *StrLit, *FStrLit:
		return "str"
	case *BoolLit:
		return "bool"
	case *NilLit:
		return "nil"
	case *ListLit:
		return "list"
	case *DictLit:
		return "dict"
	}
	return ""
}

func staticTypeAccepts(want, got string) bool {
	if want == "any" || want == got {
		return true
	}
	for _, option := range strings.Split(want, "|") {
		if option == got {
			return true
		}
	}
	return false
}

func (c *checker) walkExpr(e Expr) error {
	switch t := e.(type) {
	case *NameE:
		if !stdNames[t.X] && !c.defined(t.X) {
			c.err(t.Pos, "undefined name '%s'", t.X)
		}
	case *BinE:
		if err := c.walkExpr(t.X); err != nil {
			return err
		}
		if err := c.walkExpr(t.Y); err != nil {
			return err
		}
	case *UnE:
		if err := c.walkExpr(t.X); err != nil {
			return err
		}
	case *CallE:
		if err := c.walkExpr(t.Fn); err != nil {
			return err
		}
		for _, a := range t.Args {
			if err := c.walkExpr(a); err != nil {
				return err
			}
		}
	case *IndexE:
		if err := c.walkExpr(t.X); err != nil {
			return err
		}
		if err := c.walkExpr(t.Key); err != nil {
			return err
		}
	case *SafeIndexE:
		if err := c.walkExpr(t.X); err != nil {
			return err
		}
		if err := c.walkExpr(t.Key); err != nil {
			return err
		}
	case *AttrE:
		if err := c.walkExpr(t.X); err != nil {
			return err
		}
	case *SafeAttrE:
		if err := c.walkExpr(t.X); err != nil {
			return err
		}
	case *SliceE:
		if err := c.walkExpr(t.X); err != nil {
			return err
		}
		if t.Lo != nil {
			if err := c.walkExpr(t.Lo); err != nil {
				return err
			}
		}
		if t.Hi != nil {
			if err := c.walkExpr(t.Hi); err != nil {
				return err
			}
		}
	case *SafeChainE:
		if err := c.walkExpr(t.Base); err != nil {
			return err
		}
		for _, st := range t.Prefix {
			if err := c.walkStep(st); err != nil {
				return err
			}
		}
		for _, st := range t.Steps {
			if err := c.walkStep(st); err != nil {
				return err
			}
		}
	case *ListLit:
		for _, it := range t.Items {
			if err := c.walkExpr(it); err != nil {
				return err
			}
		}
	case *DictLit:
		for _, pr := range t.Pairs {
			if err := c.walkExpr(pr[0]); err != nil {
				return err
			}
			if err := c.walkExpr(pr[1]); err != nil {
				return err
			}
		}
	case *FnExpr:
		c.push()
		for _, p := range t.Params {
			c.define(p, t.Pos)
		}
		if err := c.walkStmts(t.Body, false); err != nil {
			return err
		}
		c.pop()
	case *FStrLit:
		for _, part := range t.Parts {
			if part.IsExpr {
				if err := c.walkExpr(part.Expr); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *checker) walkStep(st ChainStep) error {
	if st.Lo != nil {
		if err := c.walkExpr(st.Lo); err != nil {
			return err
		}
	}
	if st.Hi != nil {
		if err := c.walkExpr(st.Hi); err != nil {
			return err
		}
	}
	return nil
}

// markUsed walks the whole program (bodies included) recording reads.
func (c *checker) markUsed(sts []Stmt) {
	for _, s := range sts {
		c.markStmt(s)
	}
}

func (c *checker) markRead(name string) {
	c.used[name] = true
}

func (c *checker) markStmt(s Stmt) {
	switch t := s.(type) {
	case *AssignStmt:
		for _, v := range t.Vals {
			c.markExpr(v)
		}
	case *SetAttrStmt:
		c.markExpr(t.Container)
		c.markExpr(t.Val)
	case *SetIndexStmt:
		c.markExpr(t.Container)
		c.markExpr(t.Key)
		c.markExpr(t.Val)
	case *ReturnStmt:
		for _, v := range t.Vals {
			c.markExpr(v)
		}
	case *ExprStmt:
		c.markExpr(t.X)
	case *IfStmt:
		for _, cond := range t.Conds {
			c.markExpr(cond)
		}
		for _, b := range t.Bodies {
			c.markUsed(b)
		}
		c.markUsed(t.Else)
	case *ForStmt:
		c.markExpr(t.Iter)
		if t.Where != nil {
			c.markExpr(t.Where)
		}
		c.markUsed(t.Body)
	case *WhileStmt:
		c.markExpr(t.Cond)
		c.markUsed(t.Body)
	case *FnStmt:
		c.markUsed(t.Body)
	case *TryStmt:
		c.markUsed(t.TryBody)
		c.markUsed(t.CatchBody)
	case *MatchStmt:
		c.markExpr(t.Target)
		for _, cs := range t.Cases {
			for _, v := range cs.Vals {
				c.markExpr(v)
			}
			c.markUsed(cs.Body)
		}
	case *WithStmt:
		c.markExpr(t.Expr)
		c.markUsed(t.Body)
	}
}

func (c *checker) markExpr(e Expr) {
	switch t := e.(type) {
	case *NameE:
		c.markRead(t.X)
	case *BinE:
		c.markExpr(t.X)
		c.markExpr(t.Y)
	case *UnE:
		c.markExpr(t.X)
	case *CallE:
		c.markExpr(t.Fn)
		for _, a := range t.Args {
			c.markExpr(a)
		}
	case *IndexE:
		c.markExpr(t.X)
		c.markExpr(t.Key)
	case *SafeIndexE:
		c.markExpr(t.X)
		c.markExpr(t.Key)
	case *AttrE:
		c.markExpr(t.X)
	case *SafeAttrE:
		c.markExpr(t.X)
	case *SliceE:
		c.markExpr(t.X)
		if t.Lo != nil {
			c.markExpr(t.Lo)
		}
		if t.Hi != nil {
			c.markExpr(t.Hi)
		}
	case *SafeChainE:
		c.markExpr(t.Base)
		c.markSteps(t.Prefix)
		c.markSteps(t.Steps)
	case *ListLit:
		for _, it := range t.Items {
			c.markExpr(it)
		}
	case *DictLit:
		for _, pr := range t.Pairs {
			c.markExpr(pr[0])
			c.markExpr(pr[1])
		}
	case *FnExpr:
		c.markUsed(t.Body)
	case *FStrLit:
		for _, part := range t.Parts {
			if part.IsExpr {
				c.markExpr(part.Expr)
			}
		}
	}
}

func (c *checker) markSteps(steps []ChainStep) {
	for _, st := range steps {
		if st.Lo != nil {
			c.markExpr(st.Lo)
		}
		if st.Hi != nil {
			c.markExpr(st.Hi)
		}
	}
}

func collectUsing(sts []Stmt, out *[]*UseStmt) {
	for _, s := range sts {
		switch t := s.(type) {
		case *UseStmt:
			*out = append(*out, t)
		case *FnStmt:
			collectUsing(t.Body, out)
		case *IfStmt:
			for _, b := range t.Bodies {
				collectUsing(b, out)
			}
			collectUsing(t.Else, out)
		case *ForStmt:
			collectUsing(t.Body, out)
		case *WhileStmt:
			collectUsing(t.Body, out)
		case *TryStmt:
			collectUsing(t.TryBody, out)
			collectUsing(t.CatchBody, out)
		case *MatchStmt:
			for _, cs := range t.Cases {
				collectUsing(cs.Body, out)
			}
		case *WithStmt:
			collectUsing(t.Body, out)
		}
	}
}

// checkSource is a tiny helper used by tests and the CLI to format issues
// with the offending line; it returns a multi-line error.
func formatIssues(issues []Issue) string {
	var b strings.Builder
	for _, is := range issues {
		b.WriteString(is.String())
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
