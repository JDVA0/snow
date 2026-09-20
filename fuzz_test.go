package blizzard

import "testing"

func FuzzTokenizeNeverPanics(f *testing.F) {
	for _, seed := range []string{
		"let value = 1\nprint(value)",
		"fn main() {\n    if true { print(1) }\n}\nmain()",
		"let data = {outer: {items: [1, 2, 3]}}",
		"f\"hello {name}\"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		_, _ = Tokenize(source, "fuzz.blizz")
	})
}

func FuzzParseNeverPanics(f *testing.F) {
	for _, seed := range []string{
		"let value = 1",
		"fn add(a, b) { return a + b }",
		"match value { case 1 { print(1) } case _ { print(0) } }",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		_, _ = Parse(source, "fuzz.blizz")
	})
}
