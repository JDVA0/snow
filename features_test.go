package snow

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWhereClause(t *testing.T) {
	src := `
nums = [1, 2, 3, 4, 5, 6]
result = []
for x in nums where x > 3:
    result = append(result, x)
print(result)
`
	i := New()
	buf := &bytes.Buffer{}
	i.Out(buf)
	if err := i.Run(src, "test"); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	want := "[4, 5, 6]"
	if got != want {
		t.Errorf("where clause: got %q, want %q", got, want)
	}
}

func TestWhereWithDict(t *testing.T) {
	src := `
users = [{"name": "Alice", "active": true}, {"name": "Bob", "active": false}, {"name": "Charlie", "active": true}]
result = []
for user in users where user.active:
    result = append(result, user.name)
print(result)
`
	i := New()
	buf := &bytes.Buffer{}
	i.Out(buf)
	if err := i.Run(src, "test"); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	if got != "[Alice, Charlie]" && !strings.Contains(got, "Alice") || !strings.Contains(got, "Charlie") || strings.Contains(got, "Bob") {
		t.Errorf("where with dict: expected only Alice and Charlie, got %q", got)
	}
}

func TestWithStatement(t *testing.T) {
	// Test basic with statement structure. Build resource dict with close method as part of its literal.
	src := `
make_res = fn():
    # close method is a no-op native-style function; will be called by with statement
    close_fn = fn():
        nil
    res = {"opened": true, "data": "hello resource data", "close": close_fn}
    return res

with make_res() as r:
    x = r.data
    y = r.opened
    print(x)
    print(y)
print("done")
`
	i := New()
	buf := &bytes.Buffer{}
	i.Out(buf)
	if err := i.Run(src, "test"); err != nil {
		t.Fatalf("with statement failed: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	if !strings.Contains(got, "hello resource data") {
		t.Errorf("expected resource data, got output: %q", got)
	}
	if !strings.Contains(got, "true") {
		t.Errorf("expected opened=true, got output: %q", got)
	}
	if !strings.Contains(got, "done") {
		t.Errorf("expected block to complete and print done, got output: %q", got)
	}
}

func TestWithAndWhereCombined(t *testing.T) {
	// Test the combination from the user's example: with + for where
	src := `
get_users = fn():
    close_fn = fn():
        nil
    all_fn = fn():
        return [{"name": "Ana", "active": true}, {"name": "Luis", "active": false}, {"name": "Mia", "active": true}]
    db = {"all": all_fn, "close": close_fn}
    return db

names = []
with get_users() as db:
    for user in db.all() where user.active:
        names = append(names, user.name)
print(names)
`
	i := New()
	buf := &bytes.Buffer{}
	i.Out(buf)
	if err := i.Run(src, "test"); err != nil {
		t.Fatalf("with+where combined failed: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	if !strings.Contains(got, "Ana") || !strings.Contains(got, "Mia") {
		t.Errorf("expected active users (Ana, Mia), got %q", got)
	}
	if strings.Contains(got, "Luis") {
		t.Errorf("did not expect inactive user Luis, got %q", got)
	}
}

func TestPubPrivExport(t *testing.T) {
	// Vamos a probar la recolección de nombres privados directamente
	src := `
pub pub_var = 1
priv priv_var = 2
x = 3
pub fn pub_f():
    return 10
priv fn priv_f():
    return 20
fn normal_f():
    return 30
`
	prog, err := Parse(src, "test.snow")
	if err != nil {
		t.Fatal(err)
	}
	priv := collectPrivateNames(prog.Stmts)
	if !priv["priv_var"] {
		t.Error("priv_var should be private")
	}
	if !priv["priv_f"] {
		t.Error("priv_f should be private")
	}
	if priv["pub_var"] {
		t.Error("pub_var should not be private")
	}
	if priv["pub_f"] {
		t.Error("pub_f should not be private")
	}
	if priv["x"] {
		t.Error("x should not be private (no modifier = pub by default)")
	}
	if priv["normal_f"] {
		t.Error("normal_f should not be private (no modifier = pub by default)")
	}
}

func TestImportPubPriv(t *testing.T) {
	// Verificamos el flujo completo con archivos temporales
	tmp, err := os.MkdirTemp("", "snow-import-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	// lib.snow: contiene pub y priv symbols
	lib := `
pub pub_version = "1.0.0"
priv internal_secret = "abc123"
default_pub = "hello"

pub fn add(a, b):
    return a + b

priv fn internal_helper(x):
    return x * 2

fn double(x):
    return internal_helper(x)
`
	if err := os.WriteFile(filepath.Join(tmp, "lib.snow"), []byte(lib), 0644); err != nil {
		t.Fatal(err)
	}

	// main.snow: importa lib y accede a los symbols
	mainGood := `
import lib as lib

# Los símbolos pub (incluidos default sin modificador) se acceden como lib.ATRIBUTO
print(lib.pub_version)
print(lib.default_pub)
print(lib.add(2, 3))
print(lib.double(5))
print("ok")
`
	mainPath := filepath.Join(tmp, "main.snow")
	if err := os.WriteFile(mainPath, []byte(mainGood), 0644); err != nil {
		t.Fatal(err)
	}
	i := New()
	buf := &bytes.Buffer{}
	i.Out(buf)
	i.dir = tmp
	if err := i.Run(mainGood, mainPath); err != nil {
		t.Fatalf("expected no error for pub access, got: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "1.0.0") || !strings.Contains(got, "hello") || !strings.Contains(got, "5") || !strings.Contains(got, "10") || !strings.Contains(got, "ok") {
		t.Errorf("missing expected output for pub symbols, got:\n%s", got)
	}

	// Import without an alias uses the module filename as its binding.
	mainNoAlias := `
import lib
print(lib.add(3, 4))
`
	iNoAlias := New()
	bufNoAlias := &bytes.Buffer{}
	iNoAlias.Out(bufNoAlias)
	iNoAlias.dir = tmp
	if err := iNoAlias.Run(mainNoAlias, mainPath); err != nil {
		t.Fatalf("expected unaliased import to work, got: %v", err)
	}
	if got := strings.TrimSpace(bufNoAlias.String()); got != "7" {
		t.Errorf("unaliased import: got %q, want %q", got, "7")
	}

	// Ahora probamos que acceder a priv symbols da error runtime (attribute not found)
	mainBadPriv := `
import lib as lib
print(lib.internal_secret)
`
	i2 := New()
	i2.dir = tmp
	if err := i2.Run(mainBadPriv, mainPath); err == nil {
		t.Fatal("expected error when accessing private symbol lib.internal_secret, got nil")
	} else if !strings.Contains(err.Error(), "attribute not found") {
		t.Fatalf("expected 'attribute not found' error, got: %v", err)
	}

	// Probar que acceder a una función privada también da error
	mainBadFn := `
import lib as lib
print(lib.internal_helper(5))
`
	i3 := New()
	i3.dir = tmp
	if err := i3.Run(mainBadFn, mainPath); err == nil {
		t.Fatal("expected error when accessing private fn lib.internal_helper, got nil")
	} else if !strings.Contains(err.Error(), "attribute not found") {
		t.Fatalf("expected 'attribute not found' error for private fn, got: %v", err)
	}
}

func TestProjectPackageImport(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "snow.toml"), []byte("name = \"demo\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "src", "math.snow"), []byte("pub fn twice(n):\n    return n * 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(tmp, "app.snow")
	src := "import demo.math\nprint(math.twice(21))\n"
	if err := os.WriteFile(main, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	i := New()
	buf := &bytes.Buffer{}
	i.Out(buf)
	if err := i.RunFile(main); err != nil {
		t.Fatalf("package import failed: %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "42" {
		t.Fatalf("package import output: got %q, want 42", got)
	}
}
