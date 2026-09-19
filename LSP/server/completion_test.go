package main

import (
	"encoding/json"
	"testing"

	"github.com/JDVA0/snow/lsp/protocol"
)

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

func TestHoverDefinitionAndRename(t *testing.T) {
	s := NewServer()
	doc := &Document{URI: "file:///tmp/test.snow", Text: "fn greet(name):\n    return name\ngreet(\"Snow\")\n"}
	s.documents[doc.URI] = doc
	params, _ := json.Marshal(map[string]interface{}{"textDocument": map[string]string{"uri": doc.URI}, "position": map[string]int{"line": 2, "character": 2}})
	hover := s.handleHover(&Request{ID: 1, Params: params})
	if hover == nil || hover.Result == nil {
		t.Fatal("expected hover result")
	}
	definition := s.handleDefinition(&Request{ID: 2, Params: params})
	location, ok := definition.Result.(protocol.Location)
	if !ok || location.Range.Start.Line != 0 {
		t.Fatalf("expected definition on line 0, got %#v", definition.Result)
	}

	renameParams, _ := json.Marshal(map[string]interface{}{"textDocument": map[string]string{"uri": doc.URI}, "position": map[string]int{"line": 2, "character": 2}, "newName": "welcome"})
	rename := s.handleRename(&Request{ID: 3, Params: renameParams})
	edit, ok := rename.Result.(protocol.WorkspaceEdit)
	if !ok || len(edit.Changes[doc.URI]) != 2 {
		t.Fatalf("expected two rename edits, got %#v", rename.Result)
	}
}
