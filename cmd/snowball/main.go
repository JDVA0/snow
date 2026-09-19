// snowball is Snow's package manager. Official libraries are fetched from
// GitHub; get-local is reserved for local repository development.
package main

import (
	"crypto/sha256"
	"encoding/hex"
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
  snowball get snow/file.snow[@version] install an official library from GitHub
  snowball get-local snow/file.snow[@version] install from local repo/
  snowball search [query]            search the official package index
  snowball index                     generate repo/index.toml and the web catalog index
  snowball update [package]          update locked official package(s)
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
	case "search":
		if len(os.Args) > 3 {
			err = fmt.Errorf("search accepts at most one query")
		} else {
			query := ""
			if len(os.Args) == 3 {
				query = os.Args[2]
			}
			err = searchPackages(query)
		}
	case "index":
		if len(os.Args) != 2 {
			err = fmt.Errorf("index does not accept arguments")
		} else {
			err = generateIndex()
		}
	case "update":
		if len(os.Args) > 3 {
			err = fmt.Errorf("update accepts at most one package")
		} else {
			id := ""
			if len(os.Args) == 3 {
				id = os.Args[2]
			}
			err = updatePackages(id)
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

type registryPackage struct {
	ID, Name, Version, Description, License, Repository, Keywords, Entry, Path string
}

func getLibrary(spec string, local bool) error {
	id, constraint, err := splitPackageSpec(spec)
	if err != nil {
		return err
	}
	packages, err := registry(local)
	if err != nil {
		return err
	}
	pkg, ok := packages[id]
	if !ok {
		return fmt.Errorf("official package %q was not found in the index", id)
	}
	if !versionMatches(pkg.Version, constraint) {
		return fmt.Errorf("%s is version %s, which does not satisfy %s", id, pkg.Version, constraint)
	}
	read := githubLibrary
	if local {
		read = localLibrary
	}
	b, err := read(pkg.Path + "/" + pkg.Entry)
	if err != nil {
		return err
	}
	metadata, err := read(pkg.Path + "/package.toml")
	if err != nil {
		return fmt.Errorf("package metadata for %q: %w", id, err)
	}
	packageRoot := filepath.Join("packages", "snow")
	target := filepath.Join(packageRoot, "src", filepath.Base(pkg.Entry))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, b, 0o644); err != nil {
		return err
	}
	metadataTarget := filepath.Join(packageRoot, "meta", filepath.Base(pkg.Path)+".toml")
	if err := os.MkdirAll(filepath.Dir(metadataTarget), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(metadataTarget, metadata, 0o644); err != nil {
		return err
	}
	packageManifest := filepath.Join(packageRoot, "snow.toml")
	if _, err := os.Stat(packageManifest); os.IsNotExist(err) {
		manifestID := "packages/snow.toml"
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
	if err := setDependency("snow", packageRoot); err != nil {
		return err
	}
	if err := recordOfficialLock(pkg, b, local); err != nil {
		return err
	}
	installed, err := filepath.Abs(target)
	if err != nil {
		installed = target
	}
	if local {
		fmt.Printf("source: local repo/%s\n", pkg.Path)
	} else {
		fmt.Printf("downloaded: %s/%s\n", officialRepoURL, pkg.Path)
	}
	fmt.Printf("installed: %s\n", installed)
	printMetadata(metadata)
	return nil
}

func showInfo(id string) error {
	id, _, err := splitPackageSpec(id)
	if err != nil {
		return err
	}
	path := filepath.Join("packages", "snow", "meta", strings.TrimSuffix(filepath.Base(id), ".snow")+".toml")
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

func splitPackageSpec(spec string) (string, string, error) {
	parts := strings.SplitN(filepath.ToSlash(spec), "@", 2)
	id := parts[0]
	path := strings.Split(id, "/")
	if len(path) != 2 || path[0] != "snow" || !strings.HasSuffix(path[1], ".snow") || strings.Contains(id, "..") {
		return "", "", fmt.Errorf("official packages use snow/file.snow[@version] paths")
	}
	constraint := ""
	if len(parts) == 2 {
		constraint = parts[1]
	}
	return id, constraint, nil
}

func versionMatches(version, constraint string) bool {
	if constraint == "" || constraint == "latest" {
		return true
	}
	if strings.HasPrefix(constraint, "^") {
		want := strings.Split(strings.TrimPrefix(constraint, "^"), ".")
		got := strings.Split(version, ".")
		return len(want) > 0 && len(got) > 0 && want[0] == got[0]
	}
	return version == constraint
}

func registry(local bool) (map[string]registryPackage, error) {
	var b []byte
	var err error
	if local {
		if err = generateIndex(); err != nil {
			return nil, err
		}
		b, err = localLibrary("index.toml")
	} else {
		b, err = githubLibrary("index.toml")
	}
	if err != nil {
		return nil, fmt.Errorf("official package index: %w", err)
	}
	result := map[string]registryPackage{}
	var current *registryPackage
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		if line == "[[package]]" {
			item := registryPackage{}
			current = &item
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if current == nil || len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch key {
		case "id":
			current.ID = value
		case "name":
			current.Name = value
		case "version":
			current.Version = value
		case "description":
			current.Description = value
		case "license":
			current.License = value
		case "repository":
			current.Repository = value
		case "keywords":
			current.Keywords = value
		case "entry":
			current.Entry = value
		case "path":
			current.Path = value
			if current.ID != "" {
				result[current.ID] = *current
			}
		}
	}
	if current != nil && current.ID != "" {
		result[current.ID] = *current
	}
	return result, nil
}

func generateIndex() error {
	repo, err := officialRepo()
	if err != nil {
		return err
	}
	root := filepath.Join(repo, "packages")
	dirs, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("package directory: %w", err)
	}
	var names []string
	for _, dir := range dirs {
		if dir.IsDir() {
			names = append(names, dir.Name())
		}
	}
	sort.Strings(names)
	var out []string
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(root, name, "package.toml"))
		if err != nil {
			return fmt.Errorf("package %q: %w", name, err)
		}
		meta := metadata(b)
		entry := meta["entry"]
		if entry == "" {
			entry = name + ".snow"
		}
		if _, err := os.Stat(filepath.Join(root, name, "src", entry)); err != nil {
			return fmt.Errorf("package %q entry %q: %w", name, entry, err)
		}
		out = append(out, "[[package]]", "id = \"snow/"+entry+"\"", "name = \""+meta["name"]+"\"", "version = \""+meta["version"]+"\"", "description = \""+meta["description"]+"\"", "license = \""+meta["license"]+"\"", "repository = \""+meta["repository"]+"\"", "keywords = \""+meta["keywords"]+"\"", "entry = \"src/"+entry+"\"", "path = \"packages/"+name+"\"", "")
	}
	index := []byte("# Generated by snowball index. Do not edit manually.\n\n" + strings.Join(out, "\n"))
	if err := os.WriteFile(filepath.Join(repo, "index.toml"), index, 0o644); err != nil {
		return err
	}
	// GitHub Pages serves docs/ rather than the repository root, so publish the
	// same generated index beside packages.html when this is the Snow repository.
	webIndex := filepath.Join(filepath.Dir(repo), "docs", "packages-index.toml")
	if _, err := os.Stat(filepath.Dir(webIndex)); err == nil {
		return os.WriteFile(webIndex, index, 0o644)
	}
	return nil
}

func searchPackages(query string) error {
	packages, err := registry(false)
	if err != nil {
		return err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	var list []registryPackage
	for _, pkg := range packages {
		text := strings.ToLower(pkg.ID + " " + pkg.Name + " " + pkg.Description + " " + pkg.Keywords)
		if query == "" || strings.Contains(text, query) {
			list = append(list, pkg)
		}
	}
	sort.Slice(list, func(a, b int) bool { return list[a].Name < list[b].Name })
	for _, pkg := range list {
		fmt.Printf("%s %s — %s\n", pkg.Name, pkg.Version, pkg.Description)
	}
	return nil
}

func recordOfficialLock(pkg registryPackage, source []byte, local bool) error {
	b, _ := os.ReadFile("snow.lock")
	content := string(b)
	var kept []string
	for _, section := range strings.Split(content, "[[package]]") {
		if strings.Contains(section, "id = \""+pkg.ID+"\"") {
			continue
		}
		if strings.TrimSpace(section) != "" {
			kept = append(kept, section)
		}
	}
	if len(kept) == 0 {
		kept = append(kept, "# Snowball lockfile v1\n")
	}
	sourceName := "github"
	if local {
		sourceName = "local"
	}
	checksum := sha256.Sum256(source)
	entry := "\n[[package]]\nid = \"" + pkg.ID + "\"\nname = \"" + pkg.Name + "\"\nversion = \"" + pkg.Version + "\"\nsource = \"" + sourceName + "\"\nchecksum = \"sha256:" + hex.EncodeToString(checksum[:]) + "\"\n"
	return os.WriteFile("snow.lock", []byte(strings.Join(kept, "[[package]]")+entry), 0o644)
}

func updatePackages(id string) error {
	b, err := os.ReadFile("snow.lock")
	if err != nil {
		return fmt.Errorf("read snow.lock: %w", err)
	}
	var ids []string
	for _, section := range strings.Split(string(b), "[[package]]") {
		for _, line := range strings.Split(section, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "id = ") {
				ids = append(ids, strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "id = ")), "\""))
			}
		}
	}
	if id != "" {
		parsed, _, err := splitPackageSpec(id)
		if err != nil {
			return err
		}
		ids = []string{parsed}
	}
	if len(ids) == 0 {
		return fmt.Errorf("no official packages are locked")
	}
	for _, item := range ids {
		if err := getLibrary(item, false); err != nil {
			return err
		}
	}
	return nil
}

func officialRepo() (string, error) {
	if root := os.Getenv("SNOW_REPO"); root != "" {
		if info, err := os.Stat(filepath.Join(root, "packages")); err == nil && info.IsDir() {
			return root, nil
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "repo")
		if info, err := os.Stat(filepath.Join(candidate, "packages")); err == nil && info.IsDir() {
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
	previous, _ := os.ReadFile("snow.lock")
	locked := ""
	if at := strings.Index(string(previous), "[[package]]"); at >= 0 {
		locked = "\n" + string(previous)[at:]
	}
	return os.WriteFile("snow.lock", []byte("# Snowball lockfile v1\n"+strings.Join(out, "\n")+"\n"+locked), 0o644)
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
