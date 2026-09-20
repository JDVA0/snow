package blizzard

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Val is any Blizzard value held on the stack.
type Val any

// Language types.
type (
	Int   int64
	Float float64
	Bool  bool
	Str   string
	NilT  struct{}
	List  []Val
	Dict  struct {
		keys []string
		vals []Val
		idx  map[string]int
	}
)

// Nil is the language's null value.
var Nil = NilT{}

// TypeName returns the language-level type name of a value.
func TypeName(v Val) string {
	switch v.(type) {
	case Int:
		return "int"
	case Float:
		return "float"
	case Bool:
		return "bool"
	case Str:
		return "str"
	case NilT:
		return "nil"
	case List:
		return "list"
	case *Dict:
		return "dict"
	case *Fn:
		return "fn"
	case Native:
		return "native"
	case *Module:
		return "module"
	case *Resp:
		return "response"
	}
	return "unknown"
}

// ---- Dict ----

func NewDict(cap int) *Dict {
	return &Dict{idx: make(map[string]int, cap)}
}

func (d *Dict) Len() int { return len(d.keys) }

func (d *Dict) Get(k string) (Val, bool) {
	i, ok := d.idx[k]
	if !ok {
		return Nil, false
	}
	return d.vals[i], true
}

func (d *Dict) Has(k string) bool { _, ok := d.idx[k]; return ok }

func (d *Dict) Set(k string, v Val) {
	if d.idx == nil {
		d.idx = make(map[string]int)
	}
	if i, ok := d.idx[k]; ok {
		d.vals[i] = v
		return
	}
	d.idx[k] = len(d.keys)
	d.keys = append(d.keys, k)
	d.vals = append(d.vals, v)
}

func (d *Dict) Keys() []string { return d.keys }

func (d *Dict) ForEach(fn func(k string, v Val)) {
	for i, k := range d.keys {
		fn(k, d.vals[i])
	}
}

// Clone returns a shallow copy of the dict.
func (d *Dict) Clone() *Dict {
	n := NewDict(d.Len())
	for i, k := range d.keys {
		n.Set(k, d.vals[i])
	}
	return n
}

// Delete removes a key from the dict. No-op if the key doesn't exist.
func (d *Dict) Delete(k string) {
	i, ok := d.idx[k]
	if !ok {
		return
	}
	last := len(d.keys) - 1
	if i != last {
		// Swap with last element
		d.keys[i] = d.keys[last]
		d.vals[i] = d.vals[last]
		d.idx[d.keys[i]] = i
	}
	d.keys = d.keys[:last]
	d.vals = d.vals[:last]
	delete(d.idx, k)
}

// ---- Formatting ----

// FltStr formats a float without noise.
func FltStr(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }

// BlizzardStr renders a value for print.
func BlizzardStr(v Val) string {
	switch t := v.(type) {
	case Int:
		return strconv.FormatInt(int64(t), 10)
	case Float:
		return FltStr(float64(t))
	case Bool:
		if bool(t) {
			return "true"
		}
		return "false"
	case Str:
		return string(t)
	case NilT:
		return "nil"
	case List:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = BlizzardStr(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Dict:
		var b strings.Builder
		b.WriteByte('{')
		for i, k := range t.keys {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(BlizzardStr(t.vals[i]))
		}
		b.WriteByte('}')
		return b.String()
	case *Fn:
		return "<fn " + t.Name + "(" + strings.Join(t.Params, ", ") + ")>"
	case Native:
		return "<native>"
	case *Module:
		return "<module " + t.Name + ">"
	case *Resp:
		return t.Body
	}
	return fmt.Sprintf("%v", v)
}

// ---- Truthiness ----

// Truthy decides what counts as true for bool(v).
func Truthy(v Val) bool {
	switch t := v.(type) {
	case NilT:
		return false
	case Bool:
		return bool(t)
	case Int:
		return t != 0
	case Float:
		return t != 0
	case Str:
		return len(t) > 0
	case List:
		return len(t) > 0
	case *Dict:
		return t.Len() > 0
	}
	return true
}

// ---- Numbers ----

func asFlt(v Val) (float64, bool) {
	switch t := v.(type) {
	case Int:
		return float64(t), true
	case Float:
		return float64(t), true
	}
	return 0, false
}

func isNum(v Val) bool {
	switch v.(type) {
	case Int, Float:
		return true
	}
	return false
}

// ---- Comparison ----

// Eql decides deep equality.
func Eql(a, b Val) bool {
	switch x := a.(type) {
	case Int:
		switch y := b.(type) {
		case Int:
			return x == y
		case Float:
			return float64(x) == float64(y)
		}
	case Float:
		switch y := b.(type) {
		case Float:
			return x == y
		case Int:
			return float64(x) == float64(y)
		}
	case Bool:
		y, ok := b.(Bool)
		return ok && x == y
	case Str:
		y, ok := b.(Str)
		return ok && x == y
	case NilT:
		_, ok := b.(NilT)
		return ok
	case List:
		y, ok := b.(List)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !Eql(x[i], y[i]) {
				return false
			}
		}
		return true
	case *Dict:
		y, ok := b.(*Dict)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for _, k := range x.keys {
			xv, ok1 := x.Get(k)
			yv, ok2 := y.Get(k)
			if !ok1 || !ok2 || !Eql(xv, yv) {
				return false
			}
		}
		return true
	case *Fn:
		y, ok := b.(*Fn)
		return ok && x == y
	}
	return false
}

// Cmp compares two orderable values, returning -1, 0 or 1.
func Cmp(a, b Val) (int, error) {
	if af, ok := asFlt(a); ok {
		if bf, ok := asFlt(b); ok {
			switch {
			case af < bf:
				return -1, nil
			case af > bf:
				return 1, nil
			default:
				return 0, nil
			}
		}
		return 0, fmt.Errorf("cannot compare %s with %s", TypeName(a), TypeName(b))
	}
	switch x := a.(type) {
	case Str:
		y, ok := b.(Str)
		if !ok {
			return 0, fmt.Errorf("cannot compare str with %s", TypeName(b))
		}
		return strings.Compare(string(x), string(y)), nil
	case Bool:
		y, ok := b.(Bool)
		if !ok {
			return 0, fmt.Errorf("cannot compare bool with %s", TypeName(b))
		}
		if x == y {
			return 0, nil
		}
		if !bool(x) {
			return -1, nil
		}
		return 1, nil
	}
	return 0, fmt.Errorf("cannot compare %s", TypeName(a))
}

// ---- JSON ----

// EncodeJSON serializes a value as JSON.
func EncodeJSON(v Val) (string, error) {
	var b strings.Builder
	if err := writeJSON(&b, v); err != nil {
		return "", err
	}
	return b.String(), nil
}

func writeJSON(b *strings.Builder, v Val) error {
	switch t := v.(type) {
	case Int:
		b.WriteString(strconv.FormatInt(int64(t), 10))
	case Float:
		f := float64(t)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("cannot serialize a non-finite number")
		}
		b.WriteString(strconv.FormatFloat(f, 'g', -1, 64))
	case Bool:
		if bool(t) {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case Str:
		b.WriteString(strconv.Quote(string(t)))
	case NilT:
		b.WriteString("null")
	case List:
		b.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writeJSON(b, e); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case *Dict:
		b.WriteByte('{')
		for i, k := range t.keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Quote(k))
			b.WriteByte(':')
			if err := writeJSON(b, t.vals[i]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	case *Resp:
		return fmt.Errorf("a response is not serializable")
	case *Fn, Native, *Module:
		return fmt.Errorf("a %s is not serializable", TypeName(v))
	default:
		return fmt.Errorf("type is not serializable")
	}
	return nil
}

// DecodeJSON turns JSON text into Blizzard values.
func DecodeJSON(s string) (Val, error) {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var out Val
	if err := decInto(dec, &out); err != nil {
		return Nil, err
	}
	return out, nil
}

func decInto(dec *json.Decoder, out *Val) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	switch tok := t.(type) {
	case json.Delim:
		switch tok {
		case '{':
			d := NewDict(4)
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return err
				}
				k := kt.(string)
				var v Val
				if err := decInto(dec, &v); err != nil {
					return err
				}
				d.Set(k, v)
			}
			if _, err := dec.Token(); err != nil { // consume closing }
				return err
			}
			*out = d
		case '[':
			var l List
			for dec.More() {
				var v Val
				if err := decInto(dec, &v); err != nil {
					return err
				}
				l = append(l, v)
			}
			if _, err := dec.Token(); err != nil { // consume closing ]
				return err
			}
			*out = l
		default:
			return fmt.Errorf("unexpected json token: %v", tok)
		}
	case json.Number:
		s := tok.String()
		if strings.ContainsAny(s, ".eE") {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return err
			}
			*out = Float(f)
		} else {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return err
			}
			*out = Int(n)
		}
	case string:
		*out = Str(tok)
	case bool:
		*out = Bool(tok)
	case nil:
		*out = Nil
	default:
		return fmt.Errorf("unexpected json token: %v", tok)
	}
	return nil
}
