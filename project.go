package blizzard

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Project describes the small, dependency-free package layout understood by
// Blizzard. A blizzard.toml file may contain:
//
//	name = "my_app"
//	source = "src" # optional; defaults to src
//
// A file can then import my_app.utils and Blizzard resolves src/utils.blizz from
// the project root. Relative imports continue to work unchanged.
type Project struct {
	Root         string
	Name         string
	Source       string
	Dependencies map[string]string
}

func findProject(start string) (*Project, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		manifest := filepath.Join(dir, "blizzard.toml")
		if b, err := os.ReadFile(manifest); err == nil {
			p := &Project{Root: dir, Source: "src", Dependencies: map[string]string{}}
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
				default:
					if strings.HasPrefix(key, "dep.") {
						p.Dependencies[strings.TrimPrefix(key, "dep.")] = val
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

func resolveImportPath(fromDir string, path []string, relative int) string {
	if relative > 0 {
		dir := fromDir
		for n := 1; n < relative; n++ {
			dir = filepath.Dir(dir)
		}
		return filepath.Join(dir, strings.Join(path, "/")+".blizz")
	}
	rel := strings.Join(path, "/") + ".blizz"
	local := filepath.Join(fromDir, rel)
	if _, err := os.Stat(local); err == nil {
		return local
	}
	p, err := findProject(fromDir)
	if err != nil || p == nil || len(path) < 2 {
		return local
	}
	if depRoot, ok := p.Dependencies[path[0]]; ok {
		depRoot = filepath.Join(p.Root, depRoot)
		return filepath.Join(depRoot, projectSource(depRoot), strings.Join(path[1:], "/")+".blizz")
	}
	if path[0] != p.Name {
		return local
	}
	return filepath.Join(p.Root, p.Source, strings.Join(path[1:], "/")+".blizz")
}

func projectSource(root string) string {
	p, err := findProject(root)
	if err == nil && p != nil {
		return p.Source
	}
	return "src"
}
