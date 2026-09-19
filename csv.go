package snow

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

func newCSVModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(8)}
	m.Dict.Set("parse", Native(csvParse))
	m.Dict.Set("dicts", Native(csvDicts))
	m.Dict.Set("stringify", Native(csvStringify))
	m.Dict.Set("read", Native(csvRead))
	m.Dict.Set("write", Native(csvWrite))
	return m
}

// csv.parse(text, [delimiter]) -> list of lists
func csvParse(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("csv.parse requires at least 1 argument (text)")
	}
	text := SnowStr(args[0])
	r := csv.NewReader(strings.NewReader(text))
	if len(args) >= 2 {
		delimStr := SnowStr(args[1])
		if len(delimStr) > 0 {
			r.Comma = rune(delimStr[0])
		}
	}
	r.FieldsPerRecord = -1 // Allow variable lengths smoothly

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv parse error: %w", err)
	}

	result := make(List, len(records))
	for idx, row := range records {
		rowList := make(List, len(row))
		for cIdx, cell := range row {
			rowList[cIdx] = Str(cell)
		}
		result[idx] = rowList
	}
	return []Val{result}, nil
}

// csv.dicts(text, [delimiter]) -> list of dicts using first row as headers
func csvDicts(i *Interp, args []Val) ([]Val, error) {
	parsed, err := csvParse(i, args)
	if err != nil {
		return nil, err
	}
	matrix := parsed[0].(List)
	if len(matrix) == 0 {
		return []Val{List{}}, nil
	}

	headerRow := matrix[0].(List)
	headers := make([]string, len(headerRow))
	for idx, h := range headerRow {
		headers[idx] = strings.TrimSpace(SnowStr(h))
	}

	result := make(List, 0, len(matrix)-1)
	for r := 1; r < len(matrix); r++ {
		row := matrix[r].(List)
		d := NewDict(len(headers))
		for c, h := range headers {
			if h == "" {
				continue
			}
			val := Str("")
			if c < len(row) {
				val = row[c].(Str)
			}
			d.Set(h, val)
		}
		result = append(result, d)
	}
	return []Val{result}, nil
}

// csv.stringify(rows_or_dicts, [delimiter]) -> str
func csvStringify(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("csv.stringify requires at least 1 argument (list)")
	}
	data, ok := args[0].(List)
	if !ok {
		return nil, fmt.Errorf("csv.stringify expects a list of lists or list of dicts")
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if len(args) >= 2 {
		delimStr := SnowStr(args[1])
		if len(delimStr) > 0 {
			w.Comma = rune(delimStr[0])
		}
	}

	if len(data) == 0 {
		w.Flush()
		return []Val{Str("")}, nil
	}

	// Check if first element is a Dict
	if firstDict, isDict := data[0].(*Dict); isDict {
		headers := firstDict.keys
		if err := w.Write(headers); err != nil {
			return nil, err
		}
		for _, item := range data {
			if d, ok := item.(*Dict); ok {
				row := make([]string, len(headers))
				for idx, h := range headers {
					val, _ := d.Get(h)
					row[idx] = SnowStr(val)
				}
				if err := w.Write(row); err != nil {
					return nil, err
				}
			}
		}
	} else {
		// List of lists
		for _, item := range data {
			if rowList, ok := item.(List); ok {
				row := make([]string, len(rowList))
				for idx, cell := range rowList {
					row[idx] = SnowStr(cell)
				}
				if err := w.Write(row); err != nil {
					return nil, err
				}
			}
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return []Val{Str(buf.String())}, nil
}

// csv.read(path, [delimiter]) -> list of lists
func csvRead(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("csv.read requires at least 1 argument (path)")
	}
	path := SnowStr(args[0])
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read csv file '%s': %w", path, err)
	}
	parseArgs := []Val{Str(string(content))}
	if len(args) >= 2 {
		parseArgs = append(parseArgs, args[1])
	}
	return csvParse(i, parseArgs)
}

// csv.write(path, rows_or_dicts, [delimiter]) -> nil
func csvWrite(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("csv.write requires 2 arguments (path, data)")
	}
	path := SnowStr(args[0])
	strArgs := []Val{args[1]}
	if len(args) >= 3 {
		strArgs = append(strArgs, args[2])
	}
	res, err := csvStringify(i, strArgs)
	if err != nil {
		return nil, err
	}
	outStr := string(res[0].(Str))
	if err := os.WriteFile(path, []byte(outStr), 0644); err != nil {
		return nil, fmt.Errorf("failed to write csv file '%s': %w", path, err)
	}
	return []Val{Nil}, nil
}
