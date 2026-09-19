package snow

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Project describes the small, dependency-free package layout understood by
// Snow. A snow.toml file may contain:
//
//	name = "my_app"
//	source = "src" # optional; defaults to src
//
// A file can then import my_app.utils and Snow resolves src/utils.snow from
// the project root. Relative imports continue to work unchanged.
type Project struct {
	Root   string
	Name   string
	Source string
}

func findProject(start string) (*Project, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		manifest := filepath.Join(dir, "snow.toml")
		if b, err := os.ReadFile(manifest); err == nil {
			p := &Project{Root: dir, Source: "src"}
			s := bufio.NewScanner(strings.NewReader(string(b)))
			for s.Scan() {
				line := strings.TrimSpace(strings.SplitN(s.Text(), "#", 2)[0])
				parts := strings.SplitN(line, "=", 2)
				if len(parts) != 2 {
					continue
				}
				key := strings.TrimSpace(parts[0])
				val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
				switch key {
				case "name":
					p.Name = val
				case "source":
					if val != "" {
						p.Source = val
					}
				}
			}
			if p.Name != "" {
				return p, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}
}

func resolveImportPath(fromDir string, path []string) string {
	rel := strings.Join(path, "/") + ".snow"
	local := filepath.Join(fromDir, rel)
	if _, err := os.Stat(local); err == nil {
		return local
	}
	p, err := findProject(fromDir)
	if err != nil || p == nil || len(path) < 2 || path[0] != p.Name {
		return local
	}
	return filepath.Join(p.Root, p.Source, strings.Join(path[1:], "/")+".snow")
}
