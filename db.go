package snow

import (
	"fmt"
	"os"
	"path/filepath"
)

func newDBModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(4)}
	m.Dict.Set("open", Native(dbOpen))
	return m
}

// dbOpen(path) — loads a JSON key-value store from disk (creates if absent).
// Returns a dict augmented with: get, set, delete, keys, save, all methods.
func dbOpen(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("db.open expects a file path")
	}
	path, err := strArg(args[0], "db.open")
	if err != nil {
		return nil, err
	}

	filePath := string(path)
	var data *Dict

	if _, serr := os.Stat(filePath); os.IsNotExist(serr) {
		data = NewDict(8)
	} else {
		raw, rerr := os.ReadFile(filePath)
		if rerr != nil {
			return nil, fmt.Errorf("db.open: %v", rerr)
		}
		if len(raw) == 0 {
			data = NewDict(8)
		} else {
			v, derr := DecodeJSON(string(raw))
			if derr != nil {
				return nil, fmt.Errorf("db.open: invalid JSON in %s: %v", filePath, derr)
			}
			d, ok := v.(*Dict)
			if !ok {
				return nil, fmt.Errorf("db.open: expected a JSON object in %s", filePath)
			}
			data = d
		}
	}

	store := NewDict(8)

	// store.get(key)  or  store.get(key, default)
	store.Set("get", Native(func(interp *Interp, a []Val) ([]Val, error) {
		if len(a) < 1 || len(a) > 2 {
			return nil, fmt.Errorf("store.get expects 1 or 2 arguments")
		}
		key, err := strArg(a[0], "store.get")
		if err != nil {
			return nil, err
		}
		if v, ok := data.Get(string(key)); ok {
			return []Val{v}, nil
		}
		if len(a) == 2 {
			return []Val{a[1]}, nil
		}
		return []Val{Nil}, nil
	}))

	// store.set(key, value)
	store.Set("set", Native(func(interp *Interp, a []Val) ([]Val, error) {
		if len(a) != 2 {
			return nil, fmt.Errorf("store.set expects 2 arguments")
		}
		key, err := strArg(a[0], "store.set")
		if err != nil {
			return nil, err
		}
		data.Set(string(key), a[1])
		return []Val{Nil}, nil
	}))

	// store.delete(key)
	store.Set("delete", Native(func(interp *Interp, a []Val) ([]Val, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("store.delete expects 1 argument")
		}
		key, err := strArg(a[0], "store.delete")
		if err != nil {
			return nil, err
		}
		data.Delete(string(key))
		return []Val{Nil}, nil
	}))

	// store.keys()
	store.Set("keys", Native(func(interp *Interp, a []Val) ([]Val, error) {
		ks := data.Keys()
		out := make(List, len(ks))
		for k, v := range ks {
			out[k] = Str(v)
		}
		return []Val{out}, nil
	}))

	// store.has(key)
	store.Set("has", Native(func(interp *Interp, a []Val) ([]Val, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("store.has expects 1 argument")
		}
		key, err := strArg(a[0], "store.has")
		if err != nil {
			return nil, err
		}
		return []Val{Bool(data.Has(string(key)))}, nil
	}))

	// store.all() — returns a snapshot dict
	store.Set("all", Native(func(interp *Interp, a []Val) ([]Val, error) {
		return []Val{data.Clone()}, nil
	}))

	// store.save() — persists to disk atomically
	store.Set("save", Native(func(interp *Interp, a []Val) ([]Val, error) {
		enc, err := EncodeJSON(data)
		if err != nil {
			return nil, fmt.Errorf("store.save: %v", err)
		}
		// Atomic write via temp file
		dir := filepath.Dir(filePath)
		tmp, err := os.CreateTemp(dir, ".snowdb-*.tmp")
		if err != nil {
			return nil, fmt.Errorf("store.save: %v", err)
		}
		tmpName := tmp.Name()
		if _, werr := tmp.WriteString(enc); werr != nil {
			tmp.Close()
			os.Remove(tmpName)
			return nil, fmt.Errorf("store.save: %v", werr)
		}
		if cerr := tmp.Close(); cerr != nil {
			os.Remove(tmpName)
			return nil, fmt.Errorf("store.save: %v", cerr)
		}
		if rerr := os.Rename(tmpName, filePath); rerr != nil {
			os.Remove(tmpName)
			return nil, fmt.Errorf("store.save: %v", rerr)
		}
		return []Val{Nil}, nil
	}))

	// store.path — the file path as a string field
	store.Set("path", Str(filePath))

	return []Val{store}, nil
}
