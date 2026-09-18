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



