package snow

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokKind identifies a lexical token.
type TokKind int

const (
	tEOF TokKind = iota
	tNewline
	tIndent
	tDedent
	tIdent
	tInt
	tFlt
	tStr
	tFStr // f-string: Parts holds alternating literal/expr strings
	tLParen
	tRParen
	tLBrack
	tRBrack
	tLBrace
	tRBrace
	tComma
	tDot
	tColon
	tAssign
	tPlusEq
	tMinusEq
	tStarEq
	tSlashEq
	tSlashSlashEq
	tPctEq
	tPlus
	tMinus
	tStar
	tSlash
	tSlashSlash
	tPct
	tEq
	tNe
	tLt
	tLe
	tGt
	tGe
	tArrow  // ->
	tQQ     // ??
	tQDot   // ?.
	tQBrack // ?[
	tQQEq   // ??=
)

// Tok is a single lexical token.
type Tok struct {
	Kind  TokKind
	Text  string
	Num   int64
	Flt   float64
	Line  int
	Col   int
	Parts []string // for tFStr: [literal, expr, literal, expr, ..., literal]
}

func (t Tok) String() string {
	switch t.Kind {
	case tEOF:
		return "end of input"
	case tNewline:
		return "newline"
	case tIndent:
		return "indent"
	case tDedent:
		return "dedent"
	case tInt:
		return strconv.FormatInt(t.Num, 10)
	case tFlt:
		return FltStr(t.Flt)
	case tStr:
		return strconv.Quote(t.Text)
	default:
		return t.Text
	}
}

// Tokenize turns source into a token stream with indentation blocks.
// Tabs are not allowed in indentation; statements end at a newline unless
// they are inside parentheses, brackets or braces.
func Tokenize(src, name string) ([]Tok, error) {
	var toks []Tok
	indents := []int{0}
	depth := 0

	emit := func(t Tok) { toks = append(toks, t) }

	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSuffix(lines[i], "\r")
		ln := i + 1

		lead := 0
		for lead < len(line) && line[lead] == ' ' {
			lead++
		}
		if lead < len(line) && line[lead] == '\t' {
			return nil, &Errat{name, ln, lead + 1, fmt.Errorf("tabs are not allowed in indentation")}
		}
		content := line[lead:]
		if content == "" || strings.HasPrefix(strings.TrimLeft(content, " \t"), "#") {
			continue
		}

		if depth == 0 {
			indent := lead
			top := indents[len(indents)-1]
			switch {
			case indent > top:
				indents = append(indents, indent)
				emit(Tok{Kind: tIndent, Line: ln, Col: 1})
			case indent < top:
				for len(indents) > 1 && indents[len(indents)-1] > indent {
					indents = indents[:len(indents)-1]
					emit(Tok{Kind: tDedent, Line: ln, Col: 1})
				}
				if indents[len(indents)-1] != indent {
					return nil, &Errat{name, ln, 1, fmt.Errorf("unindent does not match any outer indentation level")}
				}
			}
		}

		lineToks, err := tokenizeLine(lines, &i, content, lead, name, &depth)
		if err != nil {
			return nil, err
		}
		toks = append(toks, lineToks...)
		if depth == 0 && len(lineToks) > 0 {
			emit(Tok{Kind: tNewline, Line: i + 1, Col: len(lines[i]) + 1})
		}
	}

	for len(indents) > 1 {
		indents = indents[:len(indents)-1]
		emit(Tok{Kind: tDedent, Line: len(lines), Col: 1})
	}
	emit(Tok{Kind: tEOF, Line: len(lines), Col: 1})
	return toks, nil
}

func tokenizeLine(lines []string, lineIdx *int, s string, lead int, name string, depth *int) ([]Tok, error) {
	var toks []Tok
	i := 0
	for i < len(s) {
		ln := *lineIdx + 1
		c := s[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '#':
			return toks, nil

		// f-string: f"..." or f'...'
		case c == 'f' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\''):
			quote := s[i+1]
			tripleStart := i + 1
			if tripleStart+2 < len(s) && s[tripleStart] == quote && s[tripleStart+1] == quote && s[tripleStart+2] == quote {
				quote3 := s[tripleStart : tripleStart+3]
				rest := s[tripleStart+3:]
				closeIdx := strings.Index(rest, quote3)
				if closeIdx >= 0 {
					raw := rest[:closeIdx]
					parts, err := splitFString(raw, name, ln)
					if err != nil {
						return nil, err
					}
					toks = append(toks, Tok{Kind: tFStr, Parts: parts, Line: ln, Col: i + 1 + lead})
					i = tripleStart + 3 + closeIdx + 3
					continue
				}
				// Multi-line f-string
				var sb strings.Builder
				sb.WriteString(rest)
				startLine := ln
				startCol := i + 1 + lead
				found := false
				for *lineIdx+1 < len(lines) {
					*lineIdx++
					sb.WriteByte('\n')
					l := strings.TrimSuffix(lines[*lineIdx], "\r")
					cIdx := strings.Index(l, quote3)
					if cIdx >= 0 {
						sb.WriteString(l[:cIdx])
						s = l
						i = cIdx + 3
						lead = 0
						found = true
						break
					}
					sb.WriteString(l)
				}
				if !found {
					return nil, &Errat{name, startLine, startCol, fmt.Errorf("unterminated f-string")}
				}
				parts, err := splitFString(sb.String(), name, startLine)
				if err != nil {
					return nil, err
				}
				toks = append(toks, Tok{Kind: tFStr, Parts: parts, Line: startLine, Col: startCol})
				continue
			}
			j := i + 2
			var sb strings.Builder
			for j < len(s) && s[j] != quote {
				if s[j] == '\\' && j+1 < len(s) {
					k := s[j+1]
					switch k {
					case 'n':
						sb.WriteByte('\n')
					case 't':
						sb.WriteByte('\t')
					case 'r':
						sb.WriteByte('\r')
					case '\\':
						sb.WriteByte('\\')
					case '"':
						sb.WriteByte('"')
					case '\'':
						sb.WriteByte('\'')
					default:
						return nil, &Errat{name, ln, j + 1 + lead, fmt.Errorf("unknown escape sequence \\%c", k)}
					}
					j += 2
					continue
				}
				sb.WriteByte(s[j])
				j++
			}
			if j >= len(s) {
				return nil, &Errat{name, ln, i + 1 + lead, fmt.Errorf("unterminated f-string")}
			}
			raw := sb.String()
			parts, err := splitFString(raw, name, ln)
			if err != nil {
				return nil, err
			}
			toks = append(toks, Tok{Kind: tFStr, Parts: parts, Line: ln, Col: i + 1 + lead})
			i = j + 1

		case c == '"' || c == '\'':
			quote := c
			// Triple-quote: """...""" or '''...'''
			if i+2 < len(s) && s[i+1] == quote && s[i+2] == quote {
				quote3 := s[i : i+3]
				rest := s[i+3:]
				closeIdx := strings.Index(rest, quote3)
				if closeIdx >= 0 {
					toks = append(toks, Tok{Kind: tStr, Text: rest[:closeIdx], Line: ln, Col: i + 1 + lead})
					i = i + 3 + closeIdx + 3
					continue
				}
				// Multi-line triple-quoted string spanning multiple lines
				var sb strings.Builder
				sb.WriteString(rest)
				startLine := ln
				startCol := i + 1 + lead
				found := false
				for *lineIdx+1 < len(lines) {
					*lineIdx++
					sb.WriteByte('\n')
					l := strings.TrimSuffix(lines[*lineIdx], "\r")
					cIdx := strings.Index(l, quote3)
					if cIdx >= 0 {
						sb.WriteString(l[:cIdx])
						s = l
						i = cIdx + 3
						lead = 0
						toks = append(toks, Tok{Kind: tStr, Text: sb.String(), Line: startLine, Col: startCol})
						found = true
						break
					}
					sb.WriteString(l)
				}
				if !found {
					return nil, &Errat{name, startLine, startCol, fmt.Errorf("unterminated triple-quoted string")}
				}
				continue
			}
			j := i + 1
			var sb strings.Builder
			for j < len(s) && s[j] != quote {
				if s[j] == '\\' && j+1 < len(s) {
					k := s[j+1]
					switch k {
					case 'n':
						sb.WriteByte('\n')
					case 't':
						sb.WriteByte('\t')
					case 'r':
						sb.WriteByte('\r')
					case '\\':
						sb.WriteByte('\\')
					case '"':
						sb.WriteByte('"')
					case '\'':
						sb.WriteByte('\'')
					default:
						return nil, &Errat{name, ln, j + 1, fmt.Errorf("unknown escape sequence \\%c", k)}
					}
					j += 2
					continue
				}
				sb.WriteByte(s[j])
				j++
			}
			if j >= len(s) {
				return nil, &Errat{name, ln, i + 1, fmt.Errorf("unterminated string")}
			}
			toks = append(toks, Tok{Kind: tStr, Text: sb.String(), Line: ln, Col: i + 1})
			i = j + 1
		case c >= '0' && c <= '9':
			j := i
			isFloat := false
			if c == '0' && j+1 < len(s) {
				var base int
				var ok func(byte) bool
				switch s[j+1] {
				case 'x', 'X':
					base, ok = 16, isHex
				case 'b', 'B':
					base, ok = 2, isBin
				case 'o', 'O':
					base, ok = 8, isOct
				}
				if base != 0 {
					k := j + 2
					for k < len(s) && ok(s[k]) {
						k++
					}
					if k == j+2 {
						return nil, &Errat{name, ln, j + 1, fmt.Errorf("invalid base-%d literal", base)}
					}
					// A decimal digit that is invalid for this base means the
					// whole literal is malformed (e.g. 0b102, 0o8).
					if k < len(s) && s[k] >= '0' && s[k] <= '9' {
						return nil, &Errat{name, ln, j + 1, fmt.Errorf("invalid base-%d literal", base)}
					}
					n, err := strconv.ParseInt(s[j+2:k], base, 64)
					if err != nil {
						return nil, &Errat{name, ln, j + 1, fmt.Errorf("invalid base-%d literal", base)}
					}
					toks = append(toks, Tok{Kind: tInt, Num: n, Line: ln, Col: j + 1})
					i = k
					continue
				}
			}
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			if j < len(s) && s[j] == '.' {
				isFloat = true
				j++
				for j < len(s) && s[j] >= '0' && s[j] <= '9' {
					j++
				}
			}
			if isFloat {
				f, err := strconv.ParseFloat(s[i:j], 64)
				if err != nil {
					return nil, &Errat{name, ln, i + 1, fmt.Errorf("invalid float literal")}
				}
				toks = append(toks, Tok{Kind: tFlt, Flt: f, Line: ln, Col: i + 1})
			} else {
				n, err := strconv.ParseInt(s[i:j], 10, 64)
				if err != nil {
					return nil, &Errat{name, ln, i + 1, fmt.Errorf("invalid integer literal")}
				}
				toks = append(toks, Tok{Kind: tInt, Num: n, Line: ln, Col: i + 1})
			}
			i = j
		case isIdentStart(c):
			j := i
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			toks = append(toks, Tok{Kind: tIdent, Text: s[i:j], Line: ln, Col: i + 1})
			i = j
		case c == '(':
			toks = append(toks, Tok{Kind: tLParen, Line: ln, Col: i + 1})
			*depth++
			i++
		case c == ')':
			toks = append(toks, Tok{Kind: tRParen, Line: ln, Col: i + 1})
			*depth--
			if *depth < 0 {
				return nil, &Errat{name, ln, i + 1, fmt.Errorf("unbalanced ')'")}
			}
			i++
		case c == '[':
			toks = append(toks, Tok{Kind: tLBrack, Line: ln, Col: i + 1})
			*depth++
			i++
		case c == ']':
			toks = append(toks, Tok{Kind: tRBrack, Line: ln, Col: i + 1})
			*depth--
			if *depth < 0 {
				return nil, &Errat{name, ln, i + 1, fmt.Errorf("unbalanced ']'")}
			}
			i++
		case c == '{':
			toks = append(toks, Tok{Kind: tLBrace, Line: ln, Col: i + 1})
			*depth++
			i++
		case c == '}':
			toks = append(toks, Tok{Kind: tRBrace, Line: ln, Col: i + 1})
			*depth--
			if *depth < 0 {
				return nil, &Errat{name, ln, i + 1, fmt.Errorf("unbalanced '}'")}
			}
			i++
		case c == ',':
			toks = append(toks, Tok{Kind: tComma, Line: ln, Col: i + 1})
			i++
		case c == '.':
			toks = append(toks, Tok{Kind: tDot, Line: ln, Col: i + 1})
			i++
		case c == ';':
			toks = append(toks, Tok{Kind: tNewline, Line: ln, Col: i + 1})
			i++
		case c == ':':
			toks = append(toks, Tok{Kind: tColon, Line: ln, Col: i + 1})
			i++
		case isOp(c):
			two := ""
			if i+1 < len(s) {
				two = s[i : i+2]
			}
			three := ""
			if i+2 < len(s) {
				three = s[i : i+3]
			}
			var kind TokKind
			n := 1
			switch three {
			case "//=":
				kind, n = tSlashSlashEq, 3
			case "??=":
				kind, n = tQQEq, 3
			}
			if kind == 0 {
				switch two {
				case "//":
					kind, n = tSlashSlash, 2
				case "==":
					kind, n = tEq, 2
				case "!=":
					kind, n = tNe, 2
				case "<=":
					kind, n = tLe, 2
				case ">=":
					kind, n = tGe, 2
				case "+=":
					kind, n = tPlusEq, 2
				case "-=":
					kind, n = tMinusEq, 2
				case "*=":
					kind, n = tStarEq, 2
				case "/=":
					kind, n = tSlashEq, 2
				case "%=":
					kind, n = tPctEq, 2
				case "??":
					kind, n = tQQ, 2
				case "?.":
					kind, n = tQDot, 2
				case "?[":
					kind, n = tQBrack, 2
					*depth++ // ?[ opens a bracket scope, matched by ]
				case "->":
					kind, n = tArrow, 2
				}
			}
			if kind == 0 {
				switch c {
				case '+':
					kind = tPlus
				case '-':
					kind = tMinus
				case '*':
					kind = tStar
				case '/':
					kind = tSlash
				case '%':
					kind = tPct
				case '=':
					kind = tAssign
				case '<':
					kind = tLt
				case '>':
					kind = tGt
				}
			}
			toks = append(toks, Tok{Kind: kind, Text: s[i : i+n], Line: ln, Col: i + 1})
			i += n
		default:
			r, _ := utf8.DecodeRuneInString(s[i:])
			if unicode.IsLetter(r) {
				i += utf8.RuneLen(r)
				continue
			}
			return nil, &Errat{name, ln, i + 1, fmt.Errorf("unexpected character %q", s[i])}
		}
	}
	return toks, nil
}

// splitFString splits an f-string body into alternating [literal, expr, literal, ...] parts.
// Always starts and ends with a literal (which may be empty).
func splitFString(s, name string, ln int) ([]string, error) {
	var parts []string
	i := 0
	var lit strings.Builder
	for i < len(s) {
		if s[i] == '{' {
			if i+1 < len(s) && s[i+1] == '{' {
				// Escaped brace
				lit.WriteByte('{')
				i += 2
				continue
			}
			// Start of expression
			parts = append(parts, lit.String())
			lit.Reset()
			i++
			depth := 1
			start := i
			for i < len(s) && depth > 0 {
				if s[i] == '{' {
					depth++
				} else if s[i] == '}' {
					depth--
				}
				if depth > 0 {
					i++
				}
			}
			if depth != 0 {
				return nil, &Errat{name, ln, 0, fmt.Errorf("unclosed '{' in f-string")}
			}
			expr := strings.TrimSpace(s[start:i])
			if expr == "" {
				return nil, &Errat{name, ln, 0, fmt.Errorf("empty expression in f-string")}
			}
			parts = append(parts, expr)
			i++ // skip closing '}'
		} else if s[i] == '}' {
			if i+1 < len(s) && s[i+1] == '}' {
				lit.WriteByte('}')
				i += 2
				continue
			}
			return nil, &Errat{name, ln, i + 1, fmt.Errorf("unexpected '}' in f-string")}
		} else {
			lit.WriteByte(s[i])
			i++
		}
	}
	parts = append(parts, lit.String())
	return parts, nil
}

func isOp(c byte) bool {
	switch c {
	case '+', '-', '*', '/', '%', '=', '<', '>', '!', '?':
		return true
	}
	return false
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= utf8.RuneSelf
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isBin(c byte) bool {
	return c == '0' || c == '1'
}

func isOct(c byte) bool {
	return c >= '0' && c <= '7'
}
