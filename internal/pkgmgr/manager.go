package pkgmgr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/term"
)

const officialAPIURL = "https://api.github.com/repos/JDVA0/snow/commits/main"

type Package struct {
	ID, Name, Version, Description, License, Repository, Keywords, Entry, Path string
}

func Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("package command required (init, get, search, list, info, add, remove, index, update)")
	}
	switch args[0] {
	case "init":
		if len(args) != 2 {
			return fmt.Errorf("init expects a project name")
		}
		return initProject(args[1])
	case "get":
		if len(args) != 2 {
			return fmt.Errorf("get expects snow/name.snow[@version]")
		}
		return get(args[1], false)
	case "get-local":
		if len(args) != 2 {
			return fmt.Errorf("get-local expects snow/name.snow[@version]")
		}
		return get(args[1], true)
	case "search":
		if len(args) > 2 {
			return fmt.Errorf("search accepts at most one query")
		}
		query := ""
		if len(args) == 2 {
			query = args[1]
		}
		return search(query)
	case "list":
		return list()
	case "info":
		if len(args) != 2 {
			return fmt.Errorf("info expects snow/name.snow")
		}
		return info(args[1])
	case "add":
		if len(args) != 3 {
			return fmt.Errorf("add expects a dependency name and path")
		}
		return add(args[1], args[2])
	case "remove":
		if len(args) != 2 {
			return fmt.Errorf("remove expects a dependency name")
		}
		return remove(args[1])
	case "index":
		return index()
	case "update":
		return update()
	default:
		return fmt.Errorf("unknown package command %q", args[0])
	}
}

func initProject(name string) error {
	if _, err := os.Stat("snow.toml"); err == nil {
		return fmt.Errorf("snow.toml already exists")
	}
	if err := os.MkdirAll("src", 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll("tests", 0o755); err != nil {
		return err
	}
	return os.WriteFile("snow.toml", []byte("name = \""+name+"\"\nsource = \"src\"\n"), 0o644)
}

func readManifest() ([]string, error) {
	b, err := os.ReadFile("snow.toml")
	if err != nil {
		return nil, fmt.Errorf("snow.toml not found; run snowman init first")
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n"), nil
}

func writeDependency(name, path string) error {
	lines, err := readManifest()
	if err != nil {
		return err
	}
	key := "dep." + name + " = "
	found := false
	for n, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key) {
			lines[n] = key + fmt.Sprintf("%q", path)
			found = true
		}
	}
	if !found {
		lines = append(lines, key+fmt.Sprintf("%q", path))
	}
	return os.WriteFile("snow.toml", []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func add(name, path string) error {
	if _, err := os.Stat(filepath.Join(path, "snow.toml")); err != nil {
		return fmt.Errorf("%q is not a Snow package", path)
	}
	if err := writeDependency(name, path); err != nil {
		return err
	}
	status("added", fmt.Sprintf("%s = %s", name, path), "32")
	return nil
}

func remove(name string) error {
	lines, err := readManifest()
	if err != nil {
		return err
	}
	prefix := "dep." + name + " = "
	out := lines[:0]
	removed := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			removed = true
			continue
		}
		out = append(out, line)
	}
	if !removed {
		return fmt.Errorf("dependency %q is not declared", name)
	}
	return os.WriteFile("snow.toml", []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func parseSpec(spec string) (string, string, error) {
	parts := strings.SplitN(filepath.ToSlash(spec), "@", 2)
	id := parts[0]
	bits := strings.Split(id, "/")
	if len(bits) != 2 || bits[0] != "snow" || !strings.HasSuffix(bits[1], ".snow") || strings.Contains(id, "..") {
		return "", "", fmt.Errorf("official packages use snow/name.snow[@version]")
	}
	constraint := ""
	if len(parts) == 2 {
		constraint = parts[1]
	}
	return id, constraint, nil
}

func get(spec string, local bool) error {
	id, constraint, err := parseSpec(spec)
	if err != nil {
		return err
	}
	useLocal := local || os.Getenv("SNOW_REPO") != ""
	packages, err := registry(useLocal)
	if err != nil {
		return err
	}
	pkg, ok := packages[id]
	if !ok {
		return fmt.Errorf("package %q was not found", id)
	}
	if constraint != "" && constraint != "latest" && constraint != pkg.Version {
		return fmt.Errorf("%s is version %s, not %s", id, pkg.Version, constraint)
	}
	read := fetch
	if useLocal {
		read = localRead
	}
	source, err := read(pkg.Path + "/" + pkg.Entry)
	if err != nil {
		return err
	}
	meta, err := read(pkg.Path + "/package.toml")
	if err != nil {
		return err
	}
	root := filepath.Join("packages", "snow")
	target := filepath.Join(root, "src", filepath.Base(pkg.Entry))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, source, 0o644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "meta"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "meta", filepath.Base(pkg.Path)+".toml"), meta, 0o644); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(root, "snow.toml")); os.IsNotExist(err) {
		manifest, readErr := read("packages/snow.toml")
		if readErr != nil {
			return readErr
		}
		if err := os.WriteFile(filepath.Join(root, "snow.toml"), manifest, 0o644); err != nil {
			return err
		}
	}
	if err := writeDependency("snow", root); err != nil {
		return err
	}
	if err := lock(pkg, source, local); err != nil {
		return err
	}
	status("installed", pkg.ID+" "+pkg.Version, "32")
	return nil
}

func status(label, message, color string) {
	if term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == "" {
		fmt.Printf("\033[%sm%s\033[0m %s\n", color, label, message)
		return
	}
	fmt.Printf("%s %s\n", label, message)
}

func fetch(path string) ([]byte, error) {
	revision, err := currentRevision()
	if err != nil {
		return nil, err
	}
	url := "https://raw.githubusercontent.com/JDVA0/snow/" + revision + "/repo/" + strings.TrimPrefix(path, "/") + fmt.Sprintf("?v=%d", time.Now().UnixNano())
	client := &http.Client{Timeout: 20 * time.Second}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("User-Agent", "snowman-package-manager")
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func currentRevision() (string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	request, err := http.NewRequest(http.MethodGet, officialAPIURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "snowman-package-manager")
	resp, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("resolve official repository revision: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve official repository revision: %s", resp.Status)
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return "", fmt.Errorf("decode official repository revision: %w", err)
	}
	if commit.SHA == "" {
		return "", fmt.Errorf("official repository returned an empty revision")
	}
	return commit.SHA, nil
}

func localRead(path string) ([]byte, error) {
	root := os.Getenv("SNOW_REPO")
	if root == "" {
		root = "repo"
	}
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
}

func registry(local bool) (map[string]Package, error) {
	read := fetch
	if local {
		read = localRead
	}
	b, err := read("index.toml")
	if err != nil {
		return nil, err
	}
	result := map[string]Package{}
	var current *Package
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		if line == "[[package]]" {
			item := Package{}
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
	return result, nil
}

func search(query string) error {
	packages, err := registry(os.Getenv("SNOW_REPO") != "")
	if err != nil {
		return err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	list := make([]Package, 0, len(packages))
	for _, pkg := range packages {
		text := strings.ToLower(pkg.ID + " " + pkg.Name + " " + pkg.Description + " " + pkg.Keywords)
		if query == "" || strings.Contains(text, query) {
			list = append(list, pkg)
		}
	}
	sort.Slice(list, func(a, b int) bool { return list[a].Name < list[b].Name })
	for _, pkg := range list {
		status(pkg.Name, pkg.Version+" - "+pkg.Description, "36")
	}
	return nil
}

func list() error {
	lines, err := readManifest()
	if err != nil {
		return err
	}
	found := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "dep.") {
			fmt.Println(strings.TrimSpace(line))
			found = true
		}
	}
	if !found {
		fmt.Println("no dependencies")
	}
	return nil
}

func info(spec string) error {
	id, _, err := parseSpec(spec)
	if err != nil {
		return err
	}
	path := filepath.Join("packages", "snow", "meta", strings.TrimSuffix(filepath.Base(id), ".snow")+".toml")
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s is not installed", id)
	}
	fmt.Print(string(b))
	return nil
}

func lock(pkg Package, source []byte, local bool) error {
	sum := sha256.Sum256(source)
	origin := "github"
	if local {
		origin = "local"
	}
	entry := "# Snow lockfile v1\n\n[[package]]\nid = \"" + pkg.ID + "\"\nversion = \"" + pkg.Version + "\"\nsource = \"" + origin + "\"\nchecksum = \"sha256:" + hex.EncodeToString(sum[:]) + "\"\n"
	return os.WriteFile("snow.lock", []byte(entry), 0o644)
}

func index() error {
	root := os.Getenv("SNOW_REPO")
	if root == "" {
		root = "repo"
	}
	dirs, err := os.ReadDir(filepath.Join(root, "packages"))
	if err != nil {
		return err
	}
	var names []string
	for _, dir := range dirs {
		if dir.IsDir() {
			names = append(names, dir.Name())
		}
	}
	sort.Strings(names)
	var blocks []string
	for _, name := range names {
		metaPath := filepath.Join(root, "packages", name, "package.toml")
		b, err := os.ReadFile(metaPath)
		if err != nil {
			return err
		}
		meta := parseMetadata(b)
		entry := meta["entry"]
		if entry == "" {
			entry = name + ".snow"
		}
		if _, err := os.Stat(filepath.Join(root, "packages", name, "src", entry)); err != nil {
			return err
		}
		blocks = append(blocks, "[[package]]", "id = \"snow/"+entry+"\"", "name = \""+meta["name"]+"\"", "version = \""+meta["version"]+"\"", "description = \""+meta["description"]+"\"", "license = \""+meta["license"]+"\"", "repository = \""+meta["repository"]+"\"", "keywords = \""+meta["keywords"]+"\"", "entry = \"src/"+entry+"\"", "path = \"packages/"+name+"\"", "")
	}
	return os.WriteFile(filepath.Join(root, "index.toml"), []byte("# Generated by snowman index.\n\n"+strings.Join(blocks, "\n")), 0o644)
}

func update() error {
	b, err := os.ReadFile("snow.lock")
	if err != nil {
		return fmt.Errorf("snow.lock not found")
	}
	var ids []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "id = ") {
			ids = append(ids, strings.Trim(strings.TrimPrefix(line, "id = "), "\""))
		}
	}
	if len(ids) == 0 {
		return fmt.Errorf("no packages are locked")
	}
	for _, id := range ids {
		if err := get(id, false); err != nil {
			return err
		}
	}
	return nil
}

func parseMetadata(b []byte) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		parts := strings.SplitN(strings.TrimSpace(strings.SplitN(line, "#", 2)[0]), "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), "\"")
		}
	}
	return result
}
