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
