package snow

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
)

// run executes src and returns what it printed.
func run(t *testing.T, src string) (string, error) {
	t.Helper()
	i := New()
	var buf bytes.Buffer
	i.Out(&buf)
	err := i.Run(src, "<test>")
	return buf.String(), err
}

func mustRun(t *testing.T, src string) string {
	t.Helper()
	out, err := run(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return out
}

func check(t *testing.T, src, expected string) {
	t.Helper()
	got := mustRun(t, src)
	if strings.TrimRight(got, "\n") != strings.TrimRight(expected, "\n") {
		t.Fatalf("source:\n%s\ngot:      %q\nexpected: %q", src, got, expected)
	}
}

func TestArithmetic(t *testing.T) {
	check(t, `print(1 + 2 * 3)`, "7")
	check(t, `print(7 / 2)`, "3.5")
	check(t, `print(7 // 2)`, "3")
	check(t, `print(-7 // 2)`, "-4")
	check(t, `print(-7 % 2)`, "1")
	check(t, `print(2 * 3 - 4)`, "2")
	check(t, `print((1 + 2) * 3)`, "9")
	check(t, `print(-(3 + 4))`, "-7")
	check(t, `x = 10
x += 5
x -= 2
x *= 2
print(x)`, "26")
}

func TestStringsAndLists(t *testing.T) {
	check(t, `print("a" + "b")`, "ab")
	check(t, `print("ab" * 3)`, "ababab")
	check(t, `print(len("hello"))`, "5")
	check(t, `print("hello"[1])`, "e")
	check(t, `print("hello"[-1])`, "o")
	check(t, `print(upper("snow"))`, "SNOW")
	check(t, `print(join(split("a,b,c", ","), "-"))`, "a-b-c")
	check(t, `print(contains("snowflake", "snow"))`, "true")
	check(t, `print([1, 2] + [3])`, "[1, 2, 3]")
	check(t, `print(len([1, 2, 3]))`, "3")
	check(t, `print([1, 2, 3][1])`, "2")
	check(t, `print(sort([3, 1, 2]))`, "[1, 2, 3]")
	check(t, `print(reverse([1, 2, 3]))`, "[3, 2, 1]")
}

func TestDicts(t *testing.T) {
	check(t, `d = {a: 1, b: 2}
print(d.a + d.b)`, "3")
	check(t, `print({x: 1}["x"])`, "1")
	check(t, `print("k" in {k: 1})`, "true")
	check(t, `print(keys({a: 1}))`, "[a]")
	check(t, `print(has({a: 1}, "a"))`, "true")
}

func TestFunctions(t *testing.T) {
	check(t, `fn add(a, b):
    return a + b
print(add(2, 3))`, "5")
	check(t, `fn fib(n):
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)
print(fib(10))`, "55")
	check(t, `fn noop():
    return
print(type(noop()))`, "nil")
	check(t, `fn pair():
    return 1, 2
a, b = pair()
print(a, b)`, "1 2")
}

func TestControlFlow(t *testing.T) {
	check(t, `if 1 < 2:
    print("yes")
else:
    print("no")`, "yes")
	check(t, `x = 3
if x == 1:
    print("one")
elif x == 3:
    print("three")
else:
    print("other")`, "three")
	check(t, `s = 0
for i in range(4):
    s = s + i
print(s)`, "6")
	check(t, `s = 0
i = 0
while i < 4:
    s = s + i
    i = i + 1
print(s)`, "6")
	check(t, `for i in range(10):
    if i == 2:
        continue
    if i == 5:
        break
    print(i)`, "0\n1\n3\n4")
}

func TestShortCircuit(t *testing.T) {
	check(t, `print(true and false)`, "false")
	check(t, `print(false or true)`, "true")
	check(t, `print(not false)`, "true")
	check(t, `fn boom():
    return 1 // 0
print(false and boom())`, "false")
	check(t, `print(true or boom())`, "true")
}

func TestMultiAssign(t *testing.T) {
	check(t, `a, b = 1, 2
print(a, b)`, "1 2")
	check(t, `a, b = 1, 2
a, b = b, a
print(a, b)`, "2 1")
}

func TestStrictTypes(t *testing.T) {
	if _, err := run(t, "if 1:\n    print(1)"); err == nil {
		t.Fatal("expected a non-bool condition to fail")
	}
	if _, err := run(t, `print(1 + "a")`); err == nil {
		t.Fatal("expected mixed arithmetic to fail")
	}
	if _, err := run(t, "x = nope"); err == nil {
		t.Fatal("expected an undefined name to fail")
	}
	if _, err := run(t, "print(1 // 0)"); err == nil {
		t.Fatal("expected division by zero to fail")
	}
	if _, err := run(t, "fn f(a):\n    return a\nf(1, 2)"); err == nil {
		t.Fatal("expected a wrong argument count to fail")
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{
		"if true\n    print(1)", // missing colon
		"return 1",              // return outside a function
		"break",                 // break outside a loop
		"x = ",                  // missing expression
		"fn f(:\n    return 1",  // malformed params
		"x = 1 y = 2",           // two statements on a line
	}
	for _, src := range cases {
		if _, err := run(t, src); err == nil {
			t.Fatalf("expected a parse error for %q", src)
		}
	}
}

func TestJSON(t *testing.T) {
	check(t, `print(json_encode({ok: true, n: 3}))`, `{"ok":true,"n":3}`)
	check(t, `print(json_decode("{\"a\": [1, 2]}"))`, "{a: [1, 2]}")
	if _, err := run(t, `print(json_decode("{"))`); err == nil {
		t.Fatal("expected invalid json to fail")
	}
}

func TestBuiltins(t *testing.T) {
	check(t, `print(min([3, 1, 2]))`, "1")
	check(t, `print(max([3, 1, 2]))`, "3")
	check(t, `print(abs(-5))`, "5")
	check(t, `print(floor(2.7), ceil(2.1), round(2.5))`, "2 3 3")
	check(t, `fn double(x):
    return x * 2
print(map(double, [1, 2, 3]))`, "[2, 4, 6]")
	check(t, `fn add(a, b):
    return a + b
print(fold([1, 2, 3], 0, add))`, "6")
	check(t, `fn is_even(x):
    return x % 2 == 0
print(filter(is_even, [1, 2, 3, 4]))`, "[2, 4]")
}

func TestIncomplete(t *testing.T) {
	if _, err := Parse("if true:\n    print(1)\n", "<t>"); err != nil {
		t.Fatalf("complete program: %v", err)
	}
	_, err := Parse("if true:\n", "<t>")
	if err == nil {
		t.Fatal("expected an incomplete program to be flagged")
	}
	p, err := Parse("fn f():\n    x = 1\n", "<t>")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !p.Incomplete {
		t.Fatal("expected Program.Incomplete")
	}
}

func TestExit(t *testing.T) {
	_, err := run(t, "exit(3)")
	ex, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if ex.Code != 3 {
		t.Fatalf("expected code 3, got %d", ex.Code)
	}
}

func TestAnonymousFunctions(t *testing.T) {
	check(t, `double = fn(x): x * 2
print(double(5))`, "10")
	check(t, `print(map(fn(x): x * 3, [1, 2, 3]))`, "[3, 6, 9]")
	check(t, `print(filter(fn(x): x > 2, [1, 2, 3, 4]))`, "[3, 4]")
	check(t, `add = fn(a, b):
    return a + b
print(add(10, 20))`, "30")
}

func TestSysAndFS(t *testing.T) {
	check(t, `using sys
res = sys.sh("echo hello-snow")
print(trim(res.stdout), res.ok)`, "hello-snow true")

	check(t, `using sys
sys.set_env("SNOW_VAR", "snow_val_123")
print(sys.env("SNOW_VAR"))`, "snow_val_123")

	check(t, `using fs
test_file = "temp_test_snow.txt"
fs.write(test_file, "hello ")
fs.append(test_file, "world")
print(fs.exists(test_file), fs.read(test_file))
fs.remove(test_file)
print(fs.exists(test_file))`, "true hello world\nfalse")
}

func TestCLIModule(t *testing.T) {
	check(t, `using cli
p = cli.parse(["--port", "8080", "--verbose", "serve", "app.snow"])
print(p.flags.port, p.flags.verbose, p.args[0], p.args[1])`, "8080 true serve app.snow")

	check(t, `using cli
colored = cli.green("OK")
print(contains(colored, "OK"))`, "true")
}

func TestAPIRouting(t *testing.T) {
	check(t, `using api
api.get("/users/:id", fn(req): {id: req.params.id, status: "active"})
api.post("/echo", fn(req): api.json({got: req.json}, 201))
print("api initialized")`, "api initialized")
}

func TestAPIHttpHandler(t *testing.T) {
	i := New()
	src := `
using api
api.cors()
api.get("/hello", fn(req): "hi there")
api.post("/echo", fn(req): api.json({got: req.json, ok: true}, 201))
api.get("/users/:id", fn(req): {id: req.params.id})
`
	if err := i.Run(src, "<test_api>"); err != nil {
		t.Fatalf("api run error: %v", err)
	}
	s := i.api
	if s == nil {
		t.Fatal("api server is nil")
	}

	// 1. Test GET /hello
	req := httptest.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()
	s.handle(w, req)
	if w.Code != 200 || w.Body.String() != "hi there" {
		t.Fatalf("unexpected GET /hello: code %d, body %q", w.Code, w.Body.String())
	}

	// 2. Test POST /echo with JSON body
	req = httptest.NewRequest("POST", "/echo", strings.NewReader(`{"name":"snow"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.handle(w, req)
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"name":"snow"`) {
		t.Fatalf("unexpected POST /echo: code %d, body %q", w.Code, w.Body.String())
	}

	// 3. Test GET /users/:id parameter matching
	req = httptest.NewRequest("GET", "/users/99", nil)
	w = httptest.NewRecorder()
	s.handle(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id":"99"`) {
		t.Fatalf("unexpected GET /users/:id: code %d, body %q", w.Code, w.Body.String())
	}

	// 4. Test CORS OPTIONS
	req = httptest.NewRequest("OPTIONS", "/anything", nil)
	w = httptest.NewRecorder()
	s.handle(w, req)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("unexpected OPTIONS CORS: code %d, header %v", w.Code, w.Header())
	}
}

func TestFStrings(t *testing.T) {
	check(t, `
name = "Snow"
ver = 0.2
print(f"Welcome to {name} v{ver}!")
`, "Welcome to Snow v0.2!")

	check(t, `
a = 10
b = 20
print(f"{a} + {b} = {a + b}")
`, "10 + 20 = 30")

	check(t, `
print(f"simple string without interpolations")
`, "simple string without interpolations")
}

func TestNullCoalesce(t *testing.T) {
	check(t, `
x = nil
print(x ?? "fallback")
`, "fallback")

	check(t, `
x = "actual"
print(x ?? "fallback")
`, "actual")

	check(t, `
a = nil
b = nil
c = "found"
print(a ?? b ?? c ?? "never")
`, "found")

	check(t, `
# 0 and false are not nil, should be preserved
print(0 ?? 99)
print(false ?? true)
`, "0\nfalse")
}

func TestMultilineStrings(t *testing.T) {
	check(t, `
s = """line 1
line 2
line 3"""
print(s)
`, "line 1\nline 2\nline 3")
}

func TestDBModule(t *testing.T) {
	src := `
using db
using fs

tmp = "test_run_db.json"
if fs.exists(tmp):
    fs.remove(tmp)

store = db.open(tmp)
store.set("greeting", "hello")
store.set("n", 42)
store.save()

re = db.open(tmp)
print(re.get("greeting"))
print(re.get("n"))
print(re.get("nonexistent", "default"))
print(re.has("greeting"))
re.delete("greeting")
re.save()
print(re.has("greeting"))

if fs.exists(tmp):
    fs.remove(tmp)
`
	check(t, src, "hello\n42\ndefault\ntrue\nfalse")
}

func TestCLIBox(t *testing.T) {
	testCases := []string{
		`
using cli
cli.box("❄️ SNOW CLI TOOLKIT", "A modern, concise & elegant CLI experience\nPlatform: linux (amd64) | PID: 76365")
`,
		`
using cli
cli.box("hola", "mundo")
`,
		`
using cli
cli.box("solo_contenido")
`,
		`
using cli
cli.box("TITULO MUY LARGO", "corto")
`,
	}

	for _, src := range testCases {
		out := mustRun(t, src)
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) < 3 {
			t.Fatalf("expected at least 3 lines in box output, got %d", len(lines))
		}
		w0 := stringWidth(lines[0])
		for idx, l := range lines {
			w := stringWidth(l)
			if w != w0 {
				t.Fatalf("line %d width %d != line 0 width %d: %q\nFull output:\n%s", idx, w, w0, l, out)
			}
		}
	}
}

func TestTimeModule(t *testing.T) {
	src := `
using time
t0 = time.unix()
time.sleep("20ms")
iso = time.iso()
now = time.now("%Y")
print(len(now) == 4)
print(time.unix_ms() > 0)
`
	check(t, src, "true\ntrue")
}

func TestJSONModule(t *testing.T) {
	src := `
using json
obj = json.parse('{"name": "snow", "count": 10, "active": true}')
print(obj.name)
print(obj.count)
print(obj.active)
s = json.stringify(obj)
print(json.valid(s))
print(json.valid("invalid json"))
`
	check(t, src, "snow\n10\ntrue\ntrue\nfalse")
}

func TestCryptoModule(t *testing.T) {
	src := `
using crypto
h = crypto.sha256("hello")
print(h == "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
tok = crypto.random_token(16)
print(len(tok) == 16)

# JWT
token = crypto.jwt_sign({"user": "snow", "role": "admin"}, "secret123")
payload = crypto.jwt_verify(token, "secret123")
print(payload.user)
print(payload.role)

# Bad secret returns nil
bad = crypto.jwt_verify(token, "wrong_secret")
print(bad == nil)
`
	check(t, src, "true\ntrue\nsnow\nadmin\ntrue")
}

func TestTaskModule(t *testing.T) {
	src := `
using task
using time

count = 0
fn tick():
    count = count + 1

job = task.every("10ms", tick)
time.sleep("35ms")
job.stop()
c1 = count
time.sleep("25ms")
print(c1 >= 2)
print(count == c1)
`
	check(t, src, "true\ntrue")
}

func TestAPIStatusReturn(t *testing.T) {
	i := New()
	src := `
using api

fn handle_ok(req):
    return 200, {status: "ok"}

fn handle_nf(req):
    return 404, {error: "missing"}

fn handle_del(req):
    return 204

api.get("/ok", handle_ok)
api.get("/notfound", handle_nf)
api.delete("/item", handle_del)
`
	if err := i.Run(src, "<test_api_status>"); err != nil {
		t.Fatalf("api status run error: %v", err)
	}
	s := i.api
	if s == nil {
		t.Fatal("api server is nil")
	}

	// 1. Test 200 with body
	req := httptest.NewRequest("GET", "/ok", nil)
	w := httptest.NewRecorder()
	s.handle(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected GET /ok: code %d, body %q", w.Code, w.Body.String())
	}

	// 2. Test 404 with body
	req = httptest.NewRequest("GET", "/notfound", nil)
	w = httptest.NewRecorder()
	s.handle(w, req)
	if w.Code != 404 || !strings.Contains(w.Body.String(), `"error":"missing"`) {
		t.Fatalf("unexpected GET /notfound: code %d, body %q", w.Code, w.Body.String())
	}

	// 3. Test 204 No Content
	req = httptest.NewRequest("DELETE", "/item", nil)
	w = httptest.NewRecorder()
	s.handle(w, req)
	if w.Code != 204 || w.Body.String() != "" {
		t.Fatalf("unexpected DELETE /item: code %d, body %q", w.Code, w.Body.String())
	}
}

func TestInputModule(t *testing.T) {
	src := `
using input

nombre = input.str()
edad = input.int()
precio = input.float()
activo = input.bool()

print(nombre)
print(edad)
print(precio)
print(activo)
`
	i := New()
	i.In(strings.NewReader("Julian\n28\n19.95\nyes\n"))
	var buf bytes.Buffer
	i.Out(&buf)
	err := i.Run(src, "<test>")
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines output, got: %v", lines)
	}
	if lines[0] != "Julian" || lines[1] != "28" || lines[2] != "19.95" || lines[3] != "true" {
		t.Fatalf("unexpected input module outputs: %v", lines)
	}
}

func TestInputModuleDefaults(t *testing.T) {
	src := `
using input

s = input.str("", "def_name")
n = input.int("", 99)
f = input.float("", 1.5)
b = input.bool("", false)
h = input.hidden("", "def_pass")

print(s)
print(n)
print(f)
print(b)
print(h)
`
	i := New()
	i.In(strings.NewReader("\n\n\n\n\n"))
	var buf bytes.Buffer
	i.Out(&buf)
	err := i.Run(src, "<test>")
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines output, got: %v", lines)
	}
	if lines[0] != "def_name" || lines[1] != "99" || lines[2] != "1.5" || lines[3] != "false" || lines[4] != "def_pass" {
		t.Fatalf("unexpected input module default outputs: %v", lines)
	}
}
func TestSafeIndex(t *testing.T) {
	// Basic safe access on a real key returns the value
	check(t, `
d = {"a": {"b": 42}}
print(d?["a"]?["b"] ?? "missing")
`, "42")

	// Missing inner key returns nil → ?? kicks in
	check(t, `
d = {"a": {"b": 42}}
print(d?["a"]?["x"] ?? "missing")
`, "missing")

	// Missing outer key returns nil, chain short-circuits, ?? provides default
	check(t, `
d = {"a": {"b": 42}}
print(d?["z"]?["b"] ?? "missing")
`, "missing")

	// Safe access on a nil value returns nil → ?? default
	check(t, `
d = nil
print(d?["key"] ?? "none")
`, "none")

	// Deep chain: three levels, last key missing
	check(t, `
data = {"user": {"profile": {"age": 30}}}
print(data?["user"]?["profile"]?["name"] ?? "anonymous")
`, "anonymous")

	// Deep chain: all keys present
	check(t, `
data = {"user": {"profile": {"name": "snow"}}}
print(data?["user"]?["profile"]?["name"] ?? "anonymous")
`, "snow")

	// Safe access on a list index (out of bounds) returns nil
	check(t, `
items = [1, 2, 3]
print(items?[10] ?? "oob")
`, "oob")

	// Safe access on a list index that exists
	check(t, `
items = [10, 20, 30]
print(items?[1] ?? "oob")
`, "20")
}

func TestEnvModule(t *testing.T) {
	check(t, `
using env
env.set("SNOW_TEST_VAR", "hello_snow")
print(env.get("SNOW_TEST_VAR"))
print(env.has("SNOW_TEST_VAR"))
print(env.get("SNOW_NON_EXISTENT", "default_val"))
`, "hello_snow\ntrue\ndefault_val")

	check(t, `
using env
env.set("SNOW_TEST_NUM", "42")
env.set("SNOW_TEST_BOOL", "true")
print(env.int("SNOW_TEST_NUM"))
print(env.bool("SNOW_TEST_BOOL"))
print(env.int("SNOW_TEST_MISSING", 100))
`, "42\ntrue\n100")
}

func TestCSVModule(t *testing.T) {
	check(t, `
using csv
raw = "id,name,score\n1,Alice,95\n2,Bob,88"
data = csv.parse(raw)
print(len(data))
print(data[1][1])
`, "3\nAlice")

	check(t, `
using csv
raw = "id,name,score\n1,Alice,95\n2,Bob,88"
dicts = csv.dicts(raw)
print(len(dicts))
print(dicts[0].name)
print(dicts[1].score)
`, "2\nAlice\n88")

	check(t, `
using csv
rows = [["col1", "col2"], ["a", "b"]]
s = csv.stringify(rows)
print(trim(s))
`, "col1,col2\na,b")

	check(t, `
using csv
raw = "a;b\n1;2"
print(csv.parse(raw, ";")[1][0])
dicts = csv.dicts("n,v\nx,9")
print(csv.stringify(dicts) != "")
`, "1\ntrue")
}

func TestTryCatch(t *testing.T) {
	check(t, `
try:
    fail("boom")
    print("no")
catch err:
    print(err)
print("ok")
`, "boom\nok")

	check(t, `
try:
    fail({mensaje: "no encontrado", linea: 12})
catch err:
    print(err.mensaje)
    print(err.linea)
`, "no encontrado\n12")

	check(t, `
fn validar(edad):
    if edad < 18:
        fail("Debe ser mayor de edad")
    return true

try:
    validar(15)
    print("no")
catch err:
    print(err)
`, "Debe ser mayor de edad")

	check(t, `
fn inner():
    fail("desde inner")

fn mid():
    inner()

try:
    mid()
catch err:
    print(err)
`, "desde inner")

	check(t, `
try:
    print(1 // 0)
    print("no")
catch err:
    print("caught")
`, "caught")

	check(t, `
try:
    try:
        fail("inner")
    catch e:
        print(e)
        fail("outer")
catch e:
    print(e)
`, "inner\nouter")

	check(t, `
n = 0
for i in range(5):
    try:
        if i == 2:
            break
        n += 1
    catch err:
        print(err)
print(n)
`, "2")

	check(t, `
print(nil ?? "nada")
try:
    fail("error")
catch err:
    print(type(err))
`, "nada\nstr")

	if _, err := run(t, `fail("sin catch")`); err == nil {
		t.Fatal("expected uncaught fail to stop the program")
	}
	if _, err := run(t, "try:\n    print(1)\n"); err == nil {
		t.Fatal("expected try without catch to fail")
	}
	if _, err := run(t, "catch err:\n    print(err)\n"); err == nil {
		t.Fatal("expected catch without try to fail")
	}
}

func TestFormat(t *testing.T) {
	src := `
nombres: str[] = ["Julian", "Ana", "Luis"]
num: int[]=[1,2,3]
fn  validar( edad ):
  if edad<18:
   fail("menor")
  return true

try:
  x=validar( 15 )
catch err:
  print( err )
`
	out, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := `nombres: str[] = ["Julian", "Ana", "Luis"]
num: int[] = [1, 2, 3]
fn validar(edad):
    if edad < 18:
        fail("menor")
    return true

try:
    x = validar(15)
catch err:
    print(err)
`
	if out != want {
		t.Fatalf("Format got:\n%q\nwant:\n%q", out, want)
	}
	again, err := Format(out)
	if err != nil {
		t.Fatalf("Format second pass: %v", err)
	}
	if again != out {
		t.Fatalf("Format is not stable:\nfirst:\n%q\nsecond:\n%q", out, again)
	}
}

func TestEnvIntInvalid(t *testing.T) {
	check(t, `
using env
env.set("SNOW_BAD_INT", "abc")
try:
    env.int("SNOW_BAD_INT")
    print("no")
catch err:
    print("bad")
print(env.int("SNOW_BAD_INT", 7))
`, "bad\n7")
}

func TestTypedLists(t *testing.T) {
	check(t, `
nombres: str[] = ["Julian", "Ana", "Luis"]
print(nombres[0])
print(nombres[2])
`, "Julian\nLuis")

	check(t, `
edades: int[] = [25, 30, 40]
print(edades[1])
`, "30")

	check(t, `
alturas: float[] = [1.75, 1.8]
print(alturas[0])
`, "1.75")

	check(t, `
flags: bool[] = [true, false]
print(flags[0])
`, "true")

	check(t, `
matriz: list[] = [[1, 2], [3]]
print(matriz[1][0])
`, "3")

	check(t, `
personas: dict[] = [{nombre: "Ana"}, {nombre: "Luis"}]
print(personas[0].nombre)
print(personas[1].nombre)
`, "Ana\nLuis")

	check(t, `
cosas: any[] = [1, "a", nil, false]
print(len(cosas))
`, "4")

	check(t, `
n: int[] = range(3)
print(n[2])
`, "2")

	check(t, `
try:
    edades: int[] = ["a", "b"]
    print("no")
catch err:
    print(err)
`, "cannot hold str in int[] (element 0)")

	check(t, `
try:
    p: str[] = "no es lista"
    print("no")
catch err:
    print(err)
`, "expected a list, got str")

	check(t, `
using json
try:
    e: int[] = json.parse("[1, \"x\"]")
    print("no")
catch err:
    print(err)
`, "cannot hold str in int[] (element 1)")

	if _, err := run(t, "n: strx[] = [1]"); err == nil {
		t.Fatal("expected unknown list type error")
	}
	if _, err := run(t, "a, b: str[] = [1, 2]"); err == nil {
		t.Fatal("expected typed declaration on a single name")
	}
}

func TestFnTypes(t *testing.T) {
	check(t, `
fn sumar(a: int, b: int) -> int:
    return a + b
print(sumar(2, 3))
`, "5")

	check(t, `
fn juntar(nombres: str[]) -> str:
    return join(nombres, ",")
print(juntar(["a", "b", "c"]))
`, "a,b,c")

	check(t, `
fn procesar(datos: str[]) -> int:
    return len(datos)
print(procesar(["x", "y", "z"]))
`, "3")

	check(t, `
fn config() -> dict:
    return {modo: "prod"}
print(config().modo)
`, "prod")

	check(t, `
doppel = fn(n: int) -> int: n * 2
print(map(doppel, [1, 2, 3]))
`, "[2, 4, 6]")

	check(t, `
fn f(x: int) -> int:
    return x * 2
try:
    f("a")
catch err:
    print(err)
`, "parameter 'x': expected int, got str")

	check(t, `
fn f() -> int:
    return "hi"
try:
    f()
catch err:
    print(err)
`, "return of f: expected int, got str")

	check(t, `
fn f(v: int[]) -> int:
    return len(v)
try:
    f(["a"])
catch err:
    print(err)
`, "parameter 'v': cannot hold str in int[] (element 0)")

	check(t, `
fn f(x: any) -> any:
    return x
print(f(1))
print(f("a"))
print(f(nil))
print(type(f([1, 2])))
`, "1\na\nnil\nlist")

	check(t, `
fn f(a, b):
    return a + b
print(f(1, 2))
print(f("x", "y"))
`, "3\nxy")

	check(t, `
fn f() -> int:
    return 1, 2
a, b = f()
print(a, b)
`, "1 2")

	out, err := Format(`fn  procesar( datos:str[] , n:int ) -> int :
  return len( datos ) + n
`)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := `fn procesar(datos: str[], n: int) -> int:
    return len(datos) + n
`
	if out != want {
		t.Fatalf("Format got:\n%q\nwant:\n%q", out, want)
	}
	again, err := Format(out)
	if err != nil {
		t.Fatalf("Format second pass: %v", err)
	}
	if again != out {
		t.Fatalf("Format not stable:\n%q\nvs\n%q", out, again)
	}

	if _, err := run(t, "fn f(x: strx) -> int: return 1"); err == nil {
		t.Fatal("expected unknown type error")
	}
}

func TestSafeChain(t *testing.T) {
	check(t, `u = nil
print(u?.perfil.nombre)
`, "nil")

	check(t, `d = {a: {b: 42}}
print(d?.a.b)
`, "42")

	check(t, `d = {a: {b: 42}}
print(d?.x.y ?? "missing")
`, "missing")

	check(t, `d = {a: {b: 42}}
print(d?.a?["x"] ?? "missing")
`, "missing")

	if _, err := run(t, `d = {a: {b: 42}}
print(d?.a.x)`); err == nil {
		t.Fatal("expected key not found error for plain dot access in a chain")
	}

	check(t, `p = {nombre: "Ana"}
print(p?.nombre)
print(p?.edad ?? "sin edad")
`, "Ana\nsin edad")

	check(t, `d = {a: {b: 42}}
print(d?["a"]?["b"] ?? "missing")
print(d?["x"]?["y"] ?? "missing")
`, "42\nmissing")

	check(t, `l = [1, 2, 3]
print(l?[5] ?? "fuera")
`, "fuera")

	check(t, `d = {a: [10, 20]}
print(d?.a?[1])
`, "20")

	check(t, `l = [[1, 2], [3]]
print(l?[0]?[1] ?? "!")
print(l?[9]?[0] ?? "!")
`, "2\n!")

	check(t, `s = "abc"
print(s?[1])
`, "b")
}

func TestSlices(t *testing.T) {
	check(t, `l = [1, 2, 3, 4, 5]
print(l[1:3])
`, "[2, 3]")

	check(t, `l = [1, 2, 3, 4, 5]
print(l[:2])
print(l[3:])
print(l[:])
print(l[0:0])
`, "[1, 2]\n[4, 5]\n[1, 2, 3, 4, 5]\n[]")

	check(t, `l = [1, 2, 3, 4, 5]
print(l[-2:])
print(l[:-3])
print(l[-4:-1])
`, "[4, 5]\n[1, 2]\n[2, 3, 4]")

	check(t, `s = "hello"
print(s[1:3])
print(s[:])
print(s[2:])
`, "el\nhello\nllo")

	check(t, `print("hello"[1:4])
`, "ell")

	check(t, `print([1, 2, 3]?[0:2])
print(nil?[0:2] ?? "nil-sliced")
`, "[1, 2]\nnil-sliced")

	check(t, `l = []
print(l[:])
`, "[]")

	if _, err := run(t, "print([1,2][5:])"); err == nil {
		t.Fatal("expected slice start out of range error")
	}
}

func TestForTwoNames(t *testing.T) {
	check(t, `d = {a: 1, b: 2}
for k, v in d:
    print(k + "=" + str(v))
`, "a=1\nb=2")

	check(t, `for i, v in ["x", "y", "z"]:
    print(i, v)
`, "0 x\n1 y\n2 z")

	check(t, `for i, c in "abc":
    print(i, c)
`, "0 a\n1 b\n2 c")

	check(t, `d = {x: 10}
suma = 0
for k, v in d:
    suma += v
print(suma)
`, "10")

	check(t, `d = {a: 1, b: 2}
total = ""
for k in d:
    total += k
print(total)
`, "ab")

	check(t, `for c in "snow":
    print(c)
`, "s\nn\no\nw")
}

func TestQQEq(t *testing.T) {
	check(t, `x = nil
x ??= 5
print(x)
x ??= 9
print(x)
`, "5\n5")

	check(t, `x = 3
x ??= 7
print(x)
`, "3")

	check(t, `
y = nil
while true:
    y ??= 1
    break
print(y)
`, "1")

	check(t, `
z = nil
if true:
    z ??= "valor"
print(z)
`, "valor")
}

func TestNotIn(t *testing.T) {
	check(t, `print("z" not in "abc")
print("b" not in "abc")
`, "true\nfalse")

	check(t, `print(3 not in [1, 2])
print(2 not in [1, 2])
`, "true\nfalse")

	check(t, `print("q" not in {a: 1})
print("a" not in {a: 1})
`, "true\nfalse")

	check(t, `print(4 not in [1, 2] and "x" not in "abc")
`, "true")
}

func TestBaseLiterals(t *testing.T) {
	check(t, `print(0b1010)
print(0B11)
print(0o17)
print(0O10)
print(0xFF)
print(0b0)
`, "10\n3\n15\n8\n255\n0")

	if _, err := run(t, "print(0b)"); err == nil {
		t.Fatal("expected invalid base-2 literal error")
	}
	if _, err := run(t, "print(0b102)"); err == nil {
		t.Fatal("expected invalid base-2 literal error")
	}
	if _, err := run(t, "print(0o8)"); err == nil {
		t.Fatal("expected invalid base-8 literal error")
	}
}

func TestMatch(t *testing.T) {
	check(t, `x = 2
match x:
    case 1, 2:
        print("small")
    case 3:
        print("three")
    case _:
        print("other")
`, "small")

	check(t, `x = 7
match x:
    case 1:
        print("one")
    case 2:
        print("two")
    case _:
        print("many")
`, "many")

	check(t, `match "hi":
    case "yo":
        print("A")
    case "hi", "hola":
        print("B")
`, "B")

	check(t, `fn clasificar(n):
    match n:
        case 0:
            return "cero"
        case 1, 2:
            return "pocos"
        case _:
            return "muchos"
print(clasificar(0))
print(clasificar(2))
print(clasificar(9))
`, "cero\npocos\nmuchos")

	check(t, `x = 5
match x:
    case 1:
        print("uno")
    case 5:
        print("cinco")
`, "cinco")

	check(t, `fn start(): print("starting")
fn stop(): print("stopping")
command = "reboot"
match command:
    case "start":
        start()
    case "stop":
        stop()
    case _:
        print("Unknown command")
`, "Unknown command")

	for _, src := range []string{
		`match 1:
    case _, 1:
        print("invalid")
`,
		`match 1:
    case _:
        print("default")
    case 1:
        print("invalid")
`,
		`match 1:
    else:
        print("invalid")
`,
	} {
		if _, err := Parse(src, "test.snow"); err == nil {
			t.Fatalf("expected invalid match syntax to fail parsing:\n%s", src)
		}
	}
}

func TestTypedListNil(t *testing.T) {
	check(t, `l: int[] = [1, nil, 3]
print(len(l))
print(l[1])
`, "3\nnil")

	check(t, `nombres: str[] = ["a", nil]
print(len(nombres))
`, "2")

	check(t, `fn f(v: list[]) -> int:
    return len(v)
print(f([[1, nil], [2]]))
`, "2")

	check(t, `fn f(v: any[]) -> int:
    return len(v)
print(f([1, nil, 2]))
`, "3")
}

func TestStackTraces(t *testing.T) {
	_, err := run(t, `fn b():
    x = [1]
    return x[5]
fn a():
    return b()
print(a())
`)
	if err == nil {
		t.Fatal("expected an error")
	}
	msg := FormatError(err)
	if !strings.Contains(msg, "stack trace:") {
		t.Fatalf("expected a stack trace, got: %s", msg)
	}
	if !strings.Contains(msg, "in b (") || !strings.Contains(msg, "in a (") {
		t.Fatalf("expected frames for a and b, got:\n%s", msg)
	}

	// fails inside a function also get a trace
	_, err = run(t, `fn falla():
    fail("boom")
fn top():
    falla()
top()
`)
	if err == nil {
		t.Fatal("expected an error")
	}
	if s := FormatError(err); !strings.Contains(s, "in falla (") {
		t.Fatalf("expected frame for falla, got:\n%s", s)
	}

	// a caught failure shows only the plain message
	out, err := run(t, `fn falla():
    fail("boom")
try:
    falla()
catch err:
    print(err)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "boom" {
		t.Fatalf("expected 'boom', got %q", out)
	}
}

func TestFormatterNewSyntax(t *testing.T) {
	src := `x  =  nil
x ??= 1
d = {a: {b: 42}}
a = d?.a.b
l = d?["a"]?["b"]
s = l[1:3]
v = l[:]
w = [1, 2, 3]
for i, it in w:
    print(i, "x" not in "xyz", it)
match x:
    case 1:
        print ( "one" )
    case _:
        print ( "other" )
`
	out, err := Format(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	again, err := Format(out)
	if err != nil {
		t.Fatalf("Format second pass: %v", err)
	}
	if again != out {
		t.Fatalf("Format not stable:\n%q\nvs\n%q", out, again)
	}
	if !strings.Contains(out, "x ??= 1") {
		t.Fatalf("expected preserved ??=, got:\n%s", out)
	}
	if !strings.Contains(out, "d?.a.b") {
		t.Fatalf("expected preserved ?. chain, got:\n%s", out)
	}
	if !strings.Contains(out, `d?["a"]?["b"]`) {
		t.Fatalf("expected preserved ?[ chain, got:\n%s", out)
	}
	if !strings.Contains(out, "l[1:3]") || !strings.Contains(out, "l[:]") {
		t.Fatalf("expected preserved slices, got:\n%s", out)
	}
	if !strings.Contains(out, `"x" not in "xyz"`) {
		t.Fatalf("expected preserved 'not in', got:\n%s", out)
	}
	if !strings.Contains(out, "print(\"one\")") {
		t.Fatalf("expected reformatted print, got:\n%s", out)
	}
}

func TestCheck(t *testing.T) {
	issues, err := Check("x = 1\nprint(x + z)\n", "a.snow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var hasUndefined bool
	for _, is := range issues {
		if is.IsErr && strings.Contains(is.Msg, "undefined name 'z'") {
			hasUndefined = true
		}
	}
	if !hasUndefined {
		t.Fatalf("expected undefined name error, got %+v", issues)
	}
}

func TestCheckUnused(t *testing.T) {
	issues, err := Check("x = 1\ny = 2\nprint(x)\n", "a.snow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var hasUnused bool
	for _, is := range issues {
		if !is.IsErr && is.Msg == "variable 'y' is assigned but never used" {
			hasUnused = true
		}
	}
	if !hasUnused {
		t.Fatalf("expected unused y warning, got %+v", issues)
	}
	for _, is := range issues {
		if strings.Contains(is.Msg, "'x'") {
			t.Fatalf("x is used, should not warn: %+v", issues)
		}
	}
}

func TestCheckUnreachable(t *testing.T) {
	issues, err := Check("fn f():\n    return 1\n    print(2)\n", "a.snow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var hasUnreachable bool
	for _, is := range issues {
		if !is.IsErr && is.Msg == "unreachable code" {
			hasUnreachable = true
		}
	}
	if !hasUnreachable {
		t.Fatalf("expected unreachable warning, got %+v", issues)
	}
}

func TestCheckUsageWithoutDefs(t *testing.T) {
	issues, err := Check("print(1)\n", "a.snow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("clean program should have no issues, got %+v", issues)
	}
}

func TestCheckUnknownModule(t *testing.T) {
	issues, err := Check("using snow.unknown\n", "a.snow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var hasUnknown bool
	for _, is := range issues {
		if is.IsErr && strings.Contains(is.Msg, "unknown standard module") {
			hasUnknown = true
		}
	}
	if !hasUnknown {
		t.Fatalf("expected unknown-module error, got %+v", issues)
	}
}
