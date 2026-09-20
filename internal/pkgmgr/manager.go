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
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

const officialAPIURL = "https://api.github.com/repos/JDVA0/blizzard/commits/main"

type Package struct {
	ID, Name, Version, Description, License, Repository, Keywords, Entry, Path string
}

func Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("package command required (init, install, get, search, list, info, add, remove, index, update)")
	}
	switch args[0] {
	case "init":
		if len(args) != 2 {
			return fmt.Errorf("init expects a project name")
		}
		return initProject(args[1])
	case "get":
		if len(args) != 2 {
			return fmt.Errorf("get expects blizzard/name.blizz[@version]")
		}
		return get(args[1], false)
	case "install":
		if len(args) != 1 {
			return fmt.Errorf("install does not accept arguments")
		}
		return install()
	case "get-local":
		if len(args) != 2 {
			return fmt.Errorf("get-local expects blizzard/name.blizz[@version]")
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
			return fmt.Errorf("info expects blizzard/name.blizz")
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
	if _, err := os.Stat("blizzard.toml"); err == nil {
		return fmt.Errorf("blizzard.toml already exists")
	}
	if err := os.MkdirAll("src", 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll("tests", 0o755); err != nil {
		return err
	}
	return os.WriteFile("blizzard.toml", []byte("name = \""+name+"\"\nsource = \"src\"\n"), 0o644)
}

func readManifest() ([]string, error) {
	b, err := os.ReadFile("blizzard.toml")
	if err != nil {
		return nil, fmt.Errorf("blizzard.toml not found; run blizzard init first")
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
	return os.WriteFile("blizzard.toml", []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func add(name, path string) error {
	if _, err := os.Stat(filepath.Join(path, "blizzard.toml")); err != nil {
		return fmt.Errorf("%q is not a Blizzard package", path)
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
	return os.WriteFile("blizzard.toml", []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func parseSpec(spec string) (string, string, error) {
	parts := strings.SplitN(filepath.ToSlash(spec), "@", 2)
	id := parts[0]
	bits := strings.Split(id, "/")
	if len(bits) != 2 || bits[0] != "blizzard" || !strings.HasSuffix(bits[1], ".blizz") || strings.Contains(id, "..") {
		return "", "", fmt.Errorf("official packages use blizzard/name.blizz[@version]")
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
	useLocal := local || os.Getenv("BLIZZARD_REPO") != ""
	packages, err := registry(useLocal)
	if err != nil {
		return err
	}
	pkg, ok := packages[id]
	if !ok {
		return fmt.Errorf("package %q was not found", id)
	}
	if constraint != "" && constraint != "latest" && !satisfiesVersion(pkg.Version, constraint) {
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
	root := filepath.Join("packages", "blizzard")
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
	if _, err := os.Stat(filepath.Join(root, "blizzard.toml")); os.IsNotExist(err) {
		manifest, readErr := read("packages/blizzard.toml")
		if readErr != nil {
			return readErr
		}
		if err := os.WriteFile(filepath.Join(root, "blizzard.toml"), manifest, 0o644); err != nil {
			return err
		}
	}
	if err := writeDependency("blizzard", root); err != nil {
		return err
	}
	if err := lock(pkg, source, local); err != nil {
		return err
	}
	status("installed", pkg.ID+" "+pkg.Version, "32")
	return nil
}

type version struct{ major, minor, patch int }

func parseVersion(value string) (version, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.SplitN(value, ".", 3)
	if len(parts) != 3 {
		return version{}, false
	}
	var out version
	var err error
	if out.major, err = strconv.Atoi(parts[0]); err != nil {
		return version{}, false
	}
	if out.minor, err = strconv.Atoi(parts[1]); err != nil {
		return version{}, false
	}
	if out.patch, err = strconv.Atoi(parts[2]); err != nil {
		return version{}, false
	}
	return out, true
}

func compareVersion(left, right version) int {
	if left.major != right.major {
		if left.major < right.major {
			return -1
		}
		return 1
	}
	if left.minor != right.minor {
		if left.minor < right.minor {
			return -1
		}
		return 1
	}
	if left.patch < right.patch {
		return -1
	}
	if left.patch > right.patch {
		return 1
	}
	return 0
}

func satisfiesVersion(actual, constraint string) bool {
	actualVersion, ok := parseVersion(actual)
	if !ok {
		return false
	}
	constraint = strings.TrimSpace(constraint)
	operator := "="
	for _, candidate := range []string{"^", "~", ">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(constraint, candidate) {
			operator = candidate
			constraint = strings.TrimSpace(strings.TrimPrefix(constraint, candidate))
			break
		}
	}
	wanted, ok := parseVersion(constraint)
	if !ok {
		return false
	}
	comparison := compareVersion(actualVersion, wanted)
	switch operator {
	case "=":
		return comparison == 0
	case ">=":
		return comparison >= 0
	case "<=":
		return comparison <= 0
	case ">":
		return comparison > 0
	case "<":
		return comparison < 0
	case "~":
		return actualVersion.major == wanted.major && actualVersion.minor == wanted.minor && comparison >= 0
	case "^":
		return actualVersion.major == wanted.major && comparison >= 0
	default:
		return false
	}
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
	url := "https://raw.githubusercontent.com/JDVA0/blizzard/" + revision + "/repo/" + strings.TrimPrefix(path, "/") + fmt.Sprintf("?v=%d", time.Now().UnixNano())
	client := &http.Client{Timeout: 20 * time.Second}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("User-Agent", "blizzard-package-manager")
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
	request.Header.Set("User-Agent", "blizzard-package-manager")
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
	root := os.Getenv("BLIZZARD_REPO")
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
	packages, err := registry(os.Getenv("BLIZZARD_REPO") != "")
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
	path := filepath.Join("packages", "blizzard", "meta", strings.TrimSuffix(filepath.Base(id), ".blizz")+".toml")
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
	entry := "[[package]]\nid = \"" + pkg.ID + "\"\nversion = \"" + pkg.Version + "\"\nsource = \"" + origin + "\"\nchecksum = \"sha256:" + hex.EncodeToString(sum[:]) + "\"\n"
	old, _ := os.ReadFile("blizzard.lock")
	var kept []string
	for _, section := range strings.Split(string(old), "[[package]]") {
		if strings.Contains(section, "id = \""+pkg.ID+"\"") || strings.TrimSpace(section) == "" {
			continue
		}
		kept = append(kept, strings.TrimSpace(section))
	}
	content := "# Blizzard lockfile v1\n\n"
	for _, section := range kept {
		content += "[[package]]\n" + section + "\n\n"
	}
	content += entry
	return os.WriteFile("blizzard.lock", []byte(content), 0o644)
}

func lockedIDs() (map[string]string, error) {
	b, err := os.ReadFile("blizzard.lock")
	if err != nil {
		return nil, fmt.Errorf("blizzard.lock not found; run blizzard get first")
	}
	locked := map[string]string{}
	for _, section := range strings.Split(string(b), "[[package]]") {
		var id, version string
		for _, line := range strings.Split(section, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "id = ") {
				id = strings.Trim(strings.TrimPrefix(line, "id = "), "\"")
			}
			if strings.HasPrefix(line, "version = ") {
				version = strings.Trim(strings.TrimPrefix(line, "version = "), "\"")
			}
		}
		if id != "" {
			locked[id] = version
		}
	}
	return locked, nil
}

func install() error {
	locked, err := lockedIDs()
	if err != nil {
		return err
	}
	if len(locked) == 0 {
		return fmt.Errorf("no packages are locked")
	}
	ids := make([]string, 0, len(locked))
	for id := range locked {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := get(id+"@"+locked[id], false); err != nil {
			return err
		}
	}
	return nil
}

func index() error {
	root := os.Getenv("BLIZZARD_REPO")
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
			entry = name + ".blizz"
		}
		if _, err := os.Stat(filepath.Join(root, "packages", name, "src", entry)); err != nil {
			return err
		}
		blocks = append(blocks, "[[package]]", "id = \"blizzard/"+entry+"\"", "name = \""+meta["name"]+"\"", "version = \""+meta["version"]+"\"", "description = \""+meta["description"]+"\"", "license = \""+meta["license"]+"\"", "repository = \""+meta["repository"]+"\"", "keywords = \""+meta["keywords"]+"\"", "entry = \"src/"+entry+"\"", "path = \"packages/"+name+"\"", "")
	}
	return os.WriteFile(filepath.Join(root, "index.toml"), []byte("# Generated by blizzard index.\n\n"+strings.Join(blocks, "\n")), 0o644)
}

func update() error {
	locked, err := lockedIDs()
	if err != nil {
		return err
	}
	packages, err := registry(false)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(locked))
	for id := range locked {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		pkg, ok := packages[id]
		if !ok {
			return fmt.Errorf("locked package %q was not found", id)
		}
		if pkg.Version == locked[id] {
			status("unchanged", id+" "+pkg.Version, "36")
			continue
		}
		if err := get(id, false); err != nil {
			return err
		}
		status("updated", id+" "+locked[id]+" -> "+pkg.Version, "32")
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
