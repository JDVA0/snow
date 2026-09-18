package snow

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func newJSONModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(4)}
	m.Dict.Set("parse", Native(jsonParse))
	m.Dict.Set("stringify", Native(jsonStringify))
	m.Dict.Set("valid", Native(jsonValid))
	return m
}

func jsonParse(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("json.parse expects 1 argument")
	}
	s, err := strArg(args[0], "json.parse")
	if err != nil {
		return nil, err
	}
	v, err := DecodeJSON(string(s))
	if err != nil {
		return nil, fmt.Errorf("json.parse: %v", err)
	}
	return []Val{v}, nil
}

func jsonStringify(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("json.stringify expects at least 1 argument")
	}
	raw, err := EncodeJSON(args[0])
	if err != nil {
		return nil, fmt.Errorf("json.stringify: %v", err)
	}
	// Check if indent is requested
	if len(args) >= 2 {
		indent := ""
		switch ind := args[1].(type) {
		case Bool:
			if bool(ind) {
				indent = "  "
			}
		case Int:
			if ind > 0 {
				indent = fmt.Sprintf("%*s", int(ind), "")
			}
		case Str:
			indent = string(ind)
		}
		if indent != "" {
			var buf bytes.Buffer
			if err := json.Indent(&buf, []byte(raw), "", indent); err == nil {
				return []Val{Str(buf.String())}, nil
			}
		}
	}
	return []Val{Str(raw)}, nil
}

func jsonValid(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("json.valid expects 1 argument")
	}
	s, err := strArg(args[0], "json.valid")
	if err != nil {
		return nil, err
	}
	_, err = DecodeJSON(string(s))
	return []Val{Bool(err == nil)}, nil
}
