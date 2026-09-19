package main

import "testing"

func TestModuleCompletionForHTTP(t *testing.T) {
	s := NewServer()
	doc := &Document{
		URI:  "file:///tmp/test.snow",
		Text: "using http\nhttp.",
	}

	items := s.getCompletionItems(doc, 1, 5)
	if len(items) == 0 {
		t.Fatal("expected completion items for http.")
	}

	labels := map[string]bool{}
	for _, item := range items {
		labels[item.Label] = true
	}

	for _, want := range []string{"get", "post", "put", "delete", "patch", "request"} {
		if !labels[want] {
			t.Fatalf("expected http completion to include %q, got %#v", want, labels)
		}
	}
}

func TestCompletionIncludesBuiltinsAndDocumentNames(t *testing.T) {
	s := NewServer()
	doc := &Document{URI: "file:///tmp/test.snow", Text: "frutas = [1, 2]\nfru"}
	items := s.getCompletionItems(doc, 1, 3)
	if len(items) != 1 || items[0].Label != "frutas" {
		t.Fatalf("expected filtered document symbol completion, got %#v", items)
	}

	doc.Text = "sum"
	items = s.getCompletionItems(doc, 0, 3)
	for _, item := range items {
		if item.Label == "sum" {
			return
		}
	}
	t.Fatalf("expected sum builtin completion, got %#v", items)
}
