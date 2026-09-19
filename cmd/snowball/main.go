// snowball is Snow's package manager. Official libraries are fetched from
// GitHub; get-local is reserved for local repository development.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const usage = `snowball - Snow package manager

Usage:
  snowball init <name>       create snow.toml, src/, and tests/
  snowball add <name> <path>         add a local package dependency
  snowball get snow/file.snow        install an official library from GitHub
  snowball get-local snow/file.snow  install a library from local repo/
	  snowball info snow/file.snow       show installed library metadata
  snowball remove <name>     remove a dependency
  snowball list              list dependencies
`

const officialRepoURL = "https://raw.githubusercontent.com/JDVA0/snow/main/repo"

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		return
	}
	var err error
	switch os.Args[1] {
	case "init":
		if len(os.Args) != 3 {
			err = fmt.Errorf("init expects a package name")
		} else {
			err = initProject(os.Args[2])
		}
	case "add":
		if len(os.Args) != 4 {
			err = fmt.Errorf("add expects a name and path")
		} else {
			err = setDependency(os.Args[2], os.Args[3])
		}
	case "get":
		if len(os.Args) != 3 {
			err = fmt.Errorf("get expects an official library path such as snow/text.snow")
		} else {
			err = getLibrary(os.Args[2], false)
		}
	case "get-local":
		if len(os.Args) != 3 {
			err = fmt.Errorf("get-local expects snow/file.snow")
		} else {
			err = getLibrary(os.Args[2], true)
		}
	case "info":
		if len(os.Args) != 3 {
			err = fmt.Errorf("info expects snow/file.snow")
		} else {
			err = showInfo(os.Args[2])
		}
	case "remove":
		if len(os.Args) != 3 {
			err = fmt.Errorf("remove expects a package name")
		} else {
			err = removeDependency(os.Args[2])
		}
	case "list":
		err = listDependencies()
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "snowball:", err)
		os.Exit(1)
	}
}

func manifest() string { return "snow.toml" }

func initProject(name string) error {
	if _, err := os.Stat(manifest()); err == nil {
		return fmt.Errorf("snow.toml already exists")
	}
	if err := os.MkdirAll("src", 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll("tests", 0o755); err != nil {
		return err
	}
	return os.WriteFile(manifest(), []byte("name = \""+name+"\"\nsource = \"src\"\n"), 0o644)
}

func readManifest() ([]string, error) {
	b, err := os.ReadFile(manifest())
	if err != nil {
		return nil, fmt.Errorf("run snowball init first: %w", err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n"), nil
}

func setDependency(name, path string) error {
	if _, err := os.Stat(filepath.Join(path, "snow.toml")); err != nil {
		return fmt.Errorf("%q is not a Snow package (missing snow.toml)", path)
	}
	lines, err := readManifest()
	if err != nil {
		return err
	}
	key := "dep." + name + " = "
	changed := false
	for n, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key) {
			lines[n] = key + fmt.Sprintf("%q", path)
			changed = true
		}
	}
	if !changed {
		lines = append(lines, key+fmt.Sprintf("%q", path))
	}
	if err := os.WriteFile(manifest(), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	return writeLock(lines)
}

func getLibrary(id string, local bool) error {
	parts := strings.Split(filepath.ToSlash(id), "/")
	if len(parts) < 2 || parts[0] != "snow" || !strings.HasSuffix(id, ".snow") {
		return fmt.Errorf("official libraries use snow/file.snow paths")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("invalid library path %q", id)
		}
	}
	var b []byte
	var err error
	if local {
		b, err = localLibrary(id)
	} else {
		b, err = githubLibrary(id)
	}
	if err != nil {
		return err
	}
	metadataID := strings.TrimSuffix(id, ".snow") + ".snowpkg"
	var metadata []byte
	if local {
		metadata, err = localLibrary(metadataID)
	} else {
		metadata, err = githubLibrary(metadataID)
	}
	if err != nil {
		return fmt.Errorf("package metadata for %q: %w", id, err)
	}
	packageRoot := filepath.Join("packages", parts[0])
	target := filepath.Join(packageRoot, "src", filepath.FromSlash(strings.Join(parts[1:], "/")))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, b, 0o644); err != nil {
		return err
	}
	metadataTarget := filepath.Join(packageRoot, filepath.FromSlash(strings.TrimPrefix(metadataID, parts[0]+"/")))
	if err := os.MkdirAll(filepath.Dir(metadataTarget), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(metadataTarget, metadata, 0o644); err != nil {
		return err
	}
	packageManifest := filepath.Join(packageRoot, "snow.toml")
	if _, err := os.Stat(packageManifest); os.IsNotExist(err) {
		manifestID := parts[0] + "/snow.toml"
		var packageMetadata []byte
		if local {
			packageMetadata, err = localLibrary(manifestID)
		} else {
			packageMetadata, err = githubLibrary(manifestID)
		}
		if err != nil {
			return fmt.Errorf("package manifest for %q: %w", id, err)
		}
		if err := os.WriteFile(packageManifest, packageMetadata, 0o644); err != nil {
			return err
		}
	}
	if err := setDependency(parts[0], packageRoot); err != nil {
		return err
	}
	installed, err := filepath.Abs(target)
	if err != nil {
		installed = target
	}
	if local {
		fmt.Printf("source: local repo/%s\n", id)
	} else {
		fmt.Printf("downloaded: %s/%s\n", officialRepoURL, id)
	}
	fmt.Printf("installed: %s\n", installed)
	printMetadata(metadata)
	return nil
}

func showInfo(id string) error {
	parts := strings.Split(filepath.ToSlash(id), "/")
	if len(parts) < 2 || parts[0] != "snow" || !strings.HasSuffix(id, ".snow") {
		return fmt.Errorf("installed libraries use snow/file.snow paths")
	}
	path := filepath.Join("packages", parts[0], filepath.FromSlash(strings.TrimSuffix(strings.Join(parts[1:], "/"), ".snow")+".snowpkg"))
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%q is not installed; run snowball get %s", id, id)
		}
		return err
	}
	printMetadata(b)
	return nil
}

func printMetadata(b []byte) {
	meta := metadata(b)
	for _, key := range []string{"name", "version", "description", "license", "repository", "keywords"} {
		if value := meta[key]; value != "" {
			fmt.Printf("%s: %s\n", key, value)
		}
	}
}

func metadata(b []byte) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), "\"")
	}
	return result
}

func officialRepo() (string, error) {
	if root := os.Getenv("SNOW_REPO"); root != "" {
		if info, err := os.Stat(filepath.Join(root, "snow")); err == nil && info.IsDir() {
			return root, nil
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "repo")
		if info, err := os.Stat(filepath.Join(candidate, "snow")); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("official repository not found; set SNOW_REPO to the repo directory")
}

func localLibrary(id string) ([]byte, error) {
	repo, err := officialRepo()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(id)))
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("local official library %q was not found", id)
	}
	return b, err
}

func githubLibrary(id string) ([]byte, error) {
	resp, err := http.Get(officialRepoURL + "/" + id)
	if err != nil {
		return nil, fmt.Errorf("could not download %q: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("official library %q was not found", id)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("official repository returned %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("official library %q is empty", id)
	}
	return b, nil
}

func removeDependency(name string) error {
	lines, err := readManifest()
	if err != nil {
		return err
	}
	key := "dep." + name + " = "
	out := lines[:0]
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), key) {
			out = append(out, line)
		}
	}
	if len(out) == len(lines) {
		return fmt.Errorf("dependency %q is not declared", name)
	}
	if err := os.WriteFile(manifest(), []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	return writeLock(out)
}

func writeLock(lines []string) error {
	deps := dependencies(lines)
	var out []string
	for name, path := range deps {
		out = append(out, fmt.Sprintf("%s = %q", name, path))
	}
	sort.Strings(out)
	return os.WriteFile("snow.lock", []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func dependencies(lines []string) map[string]string {
	deps := map[string]string{}
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if strings.HasPrefix(key, "dep.") {
			deps[strings.TrimPrefix(key, "dep.")] = strings.Trim(strings.TrimSpace(parts[1]), "\"")
		}
	}
	return deps
}

func listDependencies() error {
	lines, err := readManifest()
	if err != nil {
		return err
	}
	deps := dependencies(lines)
	if len(deps) == 0 {
		fmt.Println("no dependencies")
		return nil
	}
	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("%s %s\n", name, deps[name])
	}
	return nil
}
